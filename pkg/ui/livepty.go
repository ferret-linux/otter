package ui

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/creack/pty"
	"golang.org/x/term"
)

const (
	liveBoxMinOuterW = 24
	liveBoxMinOuterH = 5
)

// LiveBox is a bordered, live-updating terminal-in-terminal region. It
// implements io.Writer: bytes written to it (typically the master side of
// a pseudo-terminal) are interpreted as terminal output — carriage
// returns, newlines, a small subset of cursor-movement/erase escape
// sequences, and SGR color codes are all honoured — and rendered inside a
// rounded box drawn on the real terminal. LiveBox has no knowledge of what
// produced the bytes; it just renders them.
//
// LiveBox sizes itself relative to the real terminal: outer width is
// ~50% of terminal columns, outer height ~25% of terminal rows, rounded
// to the nearest integer and floored at liveBoxMinOuterW/H. Sizing is
// computed once, when the box is created/started — it does not track
// terminal resizes while running.
type LiveBox struct {
	w io.Writer // real terminal LiveBox draws into (e.g. os.Stderr)

	mu             sync.Mutex
	cols           int // interior content width, in cells
	rows           int // interior content height, in lines
	screen         []string
	cursor         int
	pendingAdvance bool
	drawn          bool
	drawnH         int // total lines currently occupied on screen (border + content)
	pending        []byte
}

// NewLiveBox creates a LiveBox that renders into w, which must be the
// real terminal (e.g. os.Stderr) so it can query the terminal size.
func NewLiveBox(w io.Writer) *LiveBox {
	b := &LiveBox{w: w}
	b.mu.Lock()
	b.recalculateSizeLocked()
	b.mu.Unlock()
	return b
}

// Start draws the initial empty box.
func (b *LiveBox) Start() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.recalculateSizeLocked()
	b.screen = make([]string, b.rows)
	b.cursor = 0
	b.redrawLocked()
}

// Close erases the box entirely, leaving the terminal exactly as it was
// before Start was called (cursor back at the same row/column).
func (b *LiveBox) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.eraseLocked()
}

// Write implements io.Writer, interpreting p as terminal output.
func (b *LiveBox) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if len(b.screen) == 0 {
		// Write called before Start (shouldn't normally happen, but
		// don't panic on a nil/empty screen).
		b.screen = make([]string, b.rows)
	}

	data := p
	if len(b.pending) > 0 {
		data = append(append([]byte(nil), b.pending...), p...)
		b.pending = nil
	}

	i := 0
	for i < len(data) {
		c := data[i]
		switch {
		case c == 0x1b:
			n, complete := parseEscape(data[i:])
			if !complete {
				b.pending = append([]byte(nil), data[i:]...)
				i = len(data)
				continue
			}
			b.applyEscapeLocked(data[i : i+n])
			i += n
		case c == '\r':
			if i+1 == len(data) {
				// Could be a bare redraw-in-place \r, or the first half
				// of a \r\n line ending split across two Write calls —
				// can't tell without the next byte, so hold it back.
				b.pending = append([]byte(nil), data[i])
				i++
				continue
			}
			if data[i+1] == '\n' {
				// \r\n is a normal line ending (pty ONLCR rewrites every
				// \n the child writes into \r\n), not a redraw-in-place:
				// don't clear, just fall through to \n's handling.
				i++
				continue
			}
			b.flushPendingAdvanceLocked()
			b.screen[b.cursor] = ""
			i++
		case c == '\n':
			if b.pendingAdvance {
				// A second consecutive \n (blank line in the stream):
				// flush the first pending advance for real before
				// scheduling another one, so blank lines aren't
				// collapsed away.
				b.flushPendingAdvanceLocked()
			}
			b.pendingAdvance = true
			i++
		case c == '\b':
			// Backspace-driven spinners aren't emulated; docker/podman/
			// nerdctl pull progress uses carriage-return full-line
			// rewrites almost exclusively, not backspace.
			i++
		default:
			r, size := utf8.DecodeRune(data[i:])
			if r == utf8.RuneError && size <= 1 {
				i++
				continue
			}
			b.flushPendingAdvanceLocked()
			b.screen[b.cursor] += string(r)
			i += size
		}
	}

	b.redrawLocked()
	return len(p), nil
}

// flushPendingAdvanceLocked performs a deferred line advance, if one is
// owed. \n only schedules an advance (sets pendingAdvance) instead of
// moving the cursor immediately, so that a newline at the very end of
// the stream — which is indistinguishable from any other newline at the
// time it's seen — doesn't permanently leave an empty row pre-allocated
// below the last real content. The advance actually happens here, lazily,
// right before something is about to occupy the next line.
func (b *LiveBox) flushPendingAdvanceLocked() {
	if !b.pendingAdvance {
		return
	}
	b.pendingAdvance = false
	b.advanceLineLocked()
}

func (b *LiveBox) advanceLineLocked() {
	b.cursor++
	if b.cursor >= len(b.screen) {
		copy(b.screen, b.screen[1:])
		b.screen[len(b.screen)-1] = ""
		b.cursor = len(b.screen) - 1
	}
}

// parseEscape scans data (which starts with the ESC byte 0x1b) and
// returns the length of one complete escape sequence and whether it was
// complete within data. CSI ("\x1b[...") sequences are recognised in
// full; OSC ("\x1b]...") sequences are consumed up to their BEL/ST
// terminator and otherwise ignored; anything else is treated as a
// two-byte sequence.
func parseEscape(data []byte) (n int, complete bool) {
	if len(data) < 2 {
		return 0, false
	}
	switch data[1] {
	case '[':
		for i := 2; i < len(data); i++ {
			c := data[i]
			if c >= 0x40 && c <= 0x7e {
				return i + 1, true
			}
		}
		return 0, false
	case ']':
		for i := 2; i < len(data); i++ {
			if data[i] == 0x07 {
				return i + 1, true
			}
			if data[i] == 0x1b && i+1 < len(data) && data[i+1] == '\\' {
				return i + 2, true
			}
		}
		return 0, false
	default:
		return 2, true
	}
}

// applyEscapeLocked interprets one complete escape sequence returned by
// parseEscape. SGR (color/style, final byte 'm') sequences are preserved
// verbatim in the current line's content so colors render inside the box.
// Cursor-movement and erase sequences update box state instead. Anything
// else is dropped: it carries no useful visual state for a fixed-size
// box.
func (b *LiveBox) applyEscapeLocked(seq []byte) {
	if len(seq) < 3 || seq[1] != '[' {
		return
	}
	final := seq[len(seq)-1]
	params := string(seq[2 : len(seq)-1])

	if final == 'm' {
		b.flushPendingAdvanceLocked()
		b.screen[b.cursor] += string(seq)
		return
	}

	firstParam := func(def int) int {
		if params == "" {
			return def
		}
		v, err := strconv.Atoi(strings.SplitN(params, ";", 2)[0])
		if err != nil || v <= 0 {
			return def
		}
		return v
	}

	switch final {
	case 'A':
		b.pendingAdvance = false
		b.cursor -= firstParam(1)
		if b.cursor < 0 {
			b.cursor = 0
		}
	case 'B':
		b.pendingAdvance = false
		b.cursor += firstParam(1)
		if b.cursor >= len(b.screen) {
			b.cursor = len(b.screen) - 1
		}
	case 'K':
		b.flushPendingAdvanceLocked()
		b.screen[b.cursor] = ""
	case 'J':
		b.pendingAdvance = false
		for i := range b.screen {
			b.screen[i] = ""
		}
		b.cursor = 0
	case 'H', 'f':
		b.pendingAdvance = false
		row := firstParam(1) - 1
		if row < 0 {
			row = 0
		}
		if row >= len(b.screen) {
			row = len(b.screen) - 1
		}
		b.cursor = row
	default:
		// cursor-forward/back, scroll regions, private modes, etc. —
		// not interpreted.
	}
}

// recalculateSizeLocked queries the real terminal size and derives the
// box's interior dimensions from it.
func (b *LiveBox) recalculateSizeLocked() {
	termW, termH, err := term.GetSize(int(os.Stderr.Fd()))
	if err != nil || termW <= 0 || termH <= 0 {
		termW, termH = 80, 24
	}

	outerW := termW / 2
	outerH := termH / 4
	if outerW < liveBoxMinOuterW {
		outerW = liveBoxMinOuterW
	}
	if outerH < liveBoxMinOuterH {
		outerH = liveBoxMinOuterH
	}

	cols := outerW - 4 // "│ " + content + " │"
	rows := outerH - 2 // top + bottom border
	if cols < 1 {
		cols = 1
	}
	if rows < 1 {
		rows = 1
	}

	b.cols = cols
	b.rows = rows
}

func (b *LiveBox) redrawLocked() {
	if b.drawn {
		b.eraseLocked()
	}

	boxWidth := b.cols + 4
	var buf strings.Builder
	buf.WriteString(hline(boxWidth, topLeft, topRight))
	buf.WriteString("\r\n")
	for _, line := range b.screen {
		buf.WriteString(Cyan(vertical))
		buf.WriteString(" ")
		buf.WriteString(padRightANSI(truncateANSI(line, b.cols), b.cols))
		buf.WriteString(" ")
		buf.WriteString(Cyan(vertical))
		buf.WriteString("\r\n")
	}
	buf.WriteString(hline(boxWidth, bottomLeft, bottomRight))
	buf.WriteString("\r\n")

	fmt.Fprint(b.w, buf.String())
	b.drawn = true
	b.drawnH = b.rows + 2
}

// eraseLocked clears the currently-drawn box (if any) and returns the
// cursor to exactly the row it was on before Start was first called,
// column 0.
func (b *LiveBox) eraseLocked() {
	if !b.drawn {
		return
	}
	fmt.Fprintf(b.w, "\033[%dA", b.drawnH)
	for i := 0; i < b.drawnH; i++ {
		fmt.Fprint(b.w, "\033[2K")
		if i < b.drawnH-1 {
			fmt.Fprint(b.w, "\n")
		}
	}
	if b.drawnH > 1 {
		fmt.Fprintf(b.w, "\033[%dA", b.drawnH-1)
	}
	fmt.Fprint(b.w, "\r")
	b.drawn = false
	b.drawnH = 0
}

// ansiVisibleLen returns the number of visible runes in s, skipping over
// ESC-introduced escape sequences (which are zero-width on screen).
func ansiVisibleLen(s string) int {
	n := 0
	inEsc := false
	for _, r := range s {
		if inEsc {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				inEsc = false
			}
			continue
		}
		if r == 0x1b {
			inEsc = true
			continue
		}
		n++
	}
	return n
}

// padRightANSI right-pads s with spaces so its visible width is w,
// ignoring embedded escape sequences when measuring width.
func padRightANSI(s string, w int) string {
	n := ansiVisibleLen(s)
	if n >= w {
		return s
	}
	return s + strings.Repeat(" ", w-n)
}

// truncateANSI truncates s to at most w visible runes, keeping any
// escape sequences encountered along the way intact (they're zero-width
// and don't count against w).
func truncateANSI(s string, w int) string {
	if ansiVisibleLen(s) <= w {
		return s
	}
	var sb strings.Builder
	n := 0
	inEsc := false
	for _, r := range s {
		if inEsc {
			sb.WriteRune(r)
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				inEsc = false
			}
			continue
		}
		if r == 0x1b {
			inEsc = true
			sb.WriteRune(r)
			continue
		}
		if n >= w {
			continue
		}
		sb.WriteRune(r)
		n++
	}
	return sb.String()
}

// RunInBox starts cmd attached to a new pseudo-terminal sized to box's
// current interior dimensions, and streams the command's combined
// stdout/stderr into box until the command exits. Sizing is taken once,
// at the moment the command starts — RunInBox does not track terminal
// resizes while the command runs.
//
// cmd must not have Stdin, Stdout, or Stderr already set; RunInBox
// assigns them to the pseudo-terminal. If cmd was built with
// exec.CommandContext, cancelling that context kills the command exactly
// as it would for any other command in this codebase.
func RunInBox(cmd *exec.Cmd, box *LiveBox) error {
	box.mu.Lock()
	cols, rows := box.cols, box.rows
	box.mu.Unlock()

	//nolint:gosec // cols/rows come from LiveBox's own clamped sizing, not external input
	master, err := pty.StartWithSize(cmd, &pty.Winsize{Cols: uint16(cols), Rows: uint16(rows)})
	if err != nil {
		return err
	}
	defer func() { _ = master.Close() }()

	copyDone := make(chan struct{})
	go func() {
		// Reading from a pty master after the slave side closes returns
		// an error (typically EIO on Linux) as its normal end-of-output
		// signal, not a real failure — it's discarded here on purpose.
		_, _ = io.Copy(box, master)
		close(copyDone)
	}()

	waitErr := cmd.Wait()
	<-copyDone

	return waitErr
}
