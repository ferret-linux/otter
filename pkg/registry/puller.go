package registry

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math"
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

	style      lipgloss.Style // bordered box style, sized responsively to the terminal (see scaledSize)
	innerWidth int            // content width inside the box's border and padding
}

// scaledSize returns the box dimension (in cells) to use for a terminal
// dimension of x cells, along one axis (width or height independently).
//
// The box occupies a fraction of x that decreases smoothly as x grows,
// asymptotically approaching floor, while the box's absolute size in
// cells never decreases as x grows. This is achieved by scaling the
// fraction itself, rather than the absolute size, with a saturating
// arctangent curve: fraction(x) = floor + (ceiling-floor)*(2/pi)*atan(k/x).
// atan(k/x) is 1 as x -> 0 and decays like k/x for large x, so
// x*fraction(x) trends toward x*floor (still growing) instead of settling
// or shrinking, unlike a plain inverse (k/x) fraction, which would let the
// absolute size shrink as the terminal grows past a certain point.
//
// k and floor are fit per-axis (see callers) so that small terminals get a
// large fraction of their size and large terminals settle toward floor.
func scaledSize(x int, k, floor float64) int {
	if x <= 0 {
		return 0
	}
	const ceiling = 1.0
	fraction := floor + (ceiling-floor)*(2/math.Pi)*math.Atan(k/float64(x))
	return int(math.Round(float64(x) * fraction))
}

func newScrollWindow(w io.Writer) *scrollWindow {
	width, height := 80, 24
	if ww, hh, err := term.GetSize(int(os.Stderr.Fd())); err == nil {
		width = ww
		height = hh
	}

	// k and floor below are fit so that fraction(120 cols) = 50% (width)
	// and fraction(20 rows) ≈ 70%, fraction(40 rows) ≈ 35% (height),
	// while keeping the box's absolute size non-decreasing as the
	// terminal grows; see scaledSize.
	const (
		widthK      = 110.47
		widthFloor  = 0.05
		heightK     = 28.3
		heightFloor = 0.05
	)

	windowLines := scaledSize(height, heightK, heightFloor)
	if windowLines < 1 {
		windowLines = 1
	}

	boxWidth := scaledSize(width, widthK, widthFloor)
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
	for i := range total {
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
//
// If the pull replaces the image previously behind imageRef (e.g. a mutable
// tag was refreshed to a new build), the old image is removed afterward
// provided no otter container still references it. This runs for every
// caller of Pull — otter create's auto-pull/always-pull/staleness paths and
// otter registry pull alike — so a refreshed tag never leaves an orphaned
// build behind regardless of which command triggered the pull. Any failure
// in this cleanup step is logged as a warning only; it never fails Pull,
// since the pull itself already succeeded by the time cleanup runs.
func Pull(
	ctx context.Context,
	cm containermanager.ContainerManager,
	imageRef string,
	platform string,
	progress *ui.Progress,
) error {
	oldID, hadOld := cm.ImageID(ctx, imageRef)

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

	if hadOld {
		if newID, ok := cm.ImageID(ctx, imageRef); ok && newID != oldID {
			cleanupDanglingImage(ctx, cm, oldID)
		}
	}

	return nil
}

// cleanupDanglingImage removes oldID if no otter container references it.
// Only called after a pull has already replaced oldID's tag with a new
// build, so any failure here is logged as a warning rather than returned.
// Uses ContainerImageID (each container's frozen image binding) rather than
// resolving Container.Image through ImageID, since oldID by definition is
// no longer what the tag currently resolves to.
func cleanupDanglingImage(ctx context.Context, cm containermanager.ContainerManager, oldID string) {
	containers, err := cm.ListContainers(ctx)
	if err != nil {
		ui.DefaultLogger.Warn("failed to list containers, leaving old image in place", "image", oldID, "error", err)
		return
	}

	for _, c := range containers {
		if !c.IsOtterContainer() {
			continue
		}
		containerImageID, ok := cm.ContainerImageID(ctx, c.Name)
		if ok && containerImageID == oldID {
			ui.DefaultLogger.Info("old image still in use, leaving in place", "image", oldID, "container", c.Name)
			return
		}
	}

	if err := cm.RemoveImage(ctx, oldID, false); err != nil {
		ui.DefaultLogger.Warn("failed to remove old image", "image", oldID, "error", err)
		return
	}
	ui.DefaultLogger.Info("removed old image", "image", oldID)
}
