package registry

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"charm.land/lipgloss/v2"
	"golang.org/x/term"

	"github.com/ferret-linux/otter/pkg/containermanager"
	"github.com/ferret-linux/otter/pkg/ttyutil"
	"github.com/ferret-linux/otter/pkg/ui"
)

// scrollWindow is an io.Writer that renders the most recent N lines
// written to it in a fixed-size region of the real terminal, redrawing
// that region in place (cursor up + clear + reprint) each time a new
// line completes. It has no knowledge of what produced the bytes beyond
// splitting them on '\n'; partial (unterminated) lines are held back
// until either a newline arrives or Close flushes them.
//
// scrollWindow assumes exclusive control of the terminal region it
// draws into between Start and Close — nothing else should write to w
// concurrently during that span.
type scrollWindow struct {
	w io.Writer

	mu      sync.Mutex
	lines   []string // fixed-size ring, oldest-to-newest, blank-padded
	partial []byte   // bytes received since the last '\n'
	drawn   bool

	style      lipgloss.Style // bordered box style, fixed at 50% of the terminal width
	innerWidth int            // content width inside the box's border and padding
}

func newScrollWindow(w io.Writer) *scrollWindow {
	width, height := 80, 24
	if ww, hh, err := term.GetSize(int(os.Stderr.Fd())); err == nil {
		width = ww
		height = hh
	}

	windowLines := height * 33 / 100
	if windowLines < 1 {
		windowLines = 1
	}

	boxWidth := width * 40 / 100
	style := ui.BorderStyle(boxWidth)

	return &scrollWindow{
		w:          w,
		lines:      make([]string, windowLines),
		style:      style,
		innerWidth: boxWidth - style.GetHorizontalFrameSize(),
	}
}

// Start draws the initial empty window.
func (s *scrollWindow) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.redrawLocked()
}

// Close finalizes the pull box: any trailing partial line is discarded,
// and the box is erased entirely, with the cursor restored to the row it
// started on. The box is a transient progress indicator, not output
// meant to remain on screen once the pull is done.
func (s *scrollWindow) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.drawn {
		return
	}

	total := len(s.lines) + 2
	fmt.Fprintf(s.w, "\033[%dA", total)
	for i := 0; i < total; i++ {
		fmt.Fprint(s.w, "\033[2K")
		if i < total-1 {
			fmt.Fprint(s.w, "\n")
		}
	}
	fmt.Fprintf(s.w, "\033[%dA", total-1)
	s.drawn = false
}

// Write implements io.Writer. Complete lines (terminated by '\n') are
// pushed into the scrolling window and the window is redrawn; a
// trailing incomplete line is held in s.partial until completed.
func (s *scrollWindow) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	data := p
	if len(s.partial) > 0 {
		data = append(append([]byte(nil), s.partial...), p...)
		s.partial = nil
	}

	pushed := false
	for {
		idx := bytes.IndexByte(data, '\n')
		if idx < 0 {
			s.partial = append([]byte(nil), data...)
			break
		}
		line := strings.TrimRight(string(data[:idx]), "\r")
		s.pushLineLocked(line)
		pushed = true
		data = data[idx+1:]
	}

	if pushed {
		s.redrawLocked()
	}

	return len(p), nil
}

func (s *scrollWindow) pushLineLocked(line string) {
	copy(s.lines, s.lines[1:])
	s.lines[len(s.lines)-1] = line
}

// redrawLocked reprints the current window contents in place: it moves
// the cursor back to the top of the previously-drawn window (if any),
// clears each line, and reprints. Must be called with s.mu held.
func (s *scrollWindow) redrawLocked() {
	if s.drawn {
		fmt.Fprintf(s.w, "\033[%dA", len(s.lines)+2)
	}

	truncated := make([]string, len(s.lines))
	for i, line := range s.lines {
		truncated[i] = truncateLine(line, s.innerWidth)
	}
	box := s.style.Render(strings.Join(truncated, "\n"))

	for _, line := range strings.Split(box, "\n") {
		fmt.Fprint(s.w, "\033[2K")
		fmt.Fprintln(s.w, line)
	}
	s.drawn = true
}

// truncateLine truncates s to at most width terminal cells. Cell width
// (not byte or rune count) is used so wide characters and ANSI sequences
// are measured correctly.
func truncateLine(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= width {
		return s
	}
	r := []rune(s)
	for len(r) > 0 && lipgloss.Width(string(r)) > width {
		r = r[:len(r)-1]
	}
	return string(r)
}

// Pull unconditionally pulls the given image ref using the provided
// container manager. Callers are responsible for deciding whether a pull
// is needed (e.g. via ImageExists or staleness checks) before calling Pull.
func Pull(
	ctx context.Context,
	cm containermanager.ContainerManager,
	imageRef string,
	platform string,
	progress *ui.Progress,
) error {
	ui.DefaultLogger.Info("large images may take a while, please be patient...")
	progress.Next("pulling '%s'...", imageRef)

	var out containermanager.PullOutput
	if ttyutil.IsInteractive() {
		win := newScrollWindow(os.Stderr)
		win.Start()
		defer win.Close()
		out = win
	}

	err := cm.PullImage(ctx, imageRef, platform, out)

	if err != nil {
		progress.Fail()
		return fmt.Errorf("failed to pull image '%s': %w", imageRef, err)
	}

	progress.Done()
	return nil
}
