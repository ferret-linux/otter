package ui

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"
	"unicode/utf8"

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
// to the nearest integer and floored at liveBoxMinOuterW/H.
type LiveBox struct {
	w io.Writer // real terminal LiveBox draws into (e.g. os.Stderr)

	mu      sync.Mutex
	cols    int // interior content width, in cells
	rows    int // interior content height, in lines
	screen  []string
	cursor  int
	drawn   bool
	drawnH  int // total lines currently occupied on screen (border + content)
	pending []byte
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

// Size implements ptyrun.Sizer: it reports the box's current interior
// size in terminal cells, so whatever is writing into the box (e.g. a
// pseudo-terminal's child process) wraps its own output to fit.
func (b *LiveBox) Size() (cols, rows int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.recalculateSizeLocked()
	return b.cols, b.rows
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
			b.screen[b.cursor] = ""
			i++
		case c == '\n':
			b.advanceLineLocked()
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
			b.screen[b.cursor] += string(r)
			i += size
		}
	}

	b.redrawLocked()
	return len(p), nil
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
		b.cursor -= firstParam(1)
		if b.cursor < 0 {
			b.cursor = 0
		}
	case 'B':
		b.cursor += firstParam(1)
		if b.cursor >= len(b.screen) {
			b.cursor = len(b.screen) - 1
		}
	case 'K':
		b.screen[b.cursor] = ""
	case 'J':
		for i := range b.screen {
			b.screen[i] = ""
		}
		b.cursor = 0
	case 'H', 'f':
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
// box's interior dimensions from it. If the terminal size changes (e.g.
// SIGWINCH), the in-progress screen content is dropped rather than
// reflowed — pull-progress output refreshes frequently enough that this
// is a brief, low-cost cosmetic blip rather than something worth the
// added complexity of preserving/reflowing old lines across a dimension
// change.
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

	cols := outerW - 4
	rows := outerH - 2
	if cols < 1 {
		cols = 1
	}
	if rows < 1 {
		rows = 1
	}

	if cols == b.cols && rows == b.rows {
		return
	}
	b.cols = cols
	b.rows = rows
	b.screen = make([]string, b.rows)
	b.cursor = 0
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
