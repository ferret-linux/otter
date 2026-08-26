package registry

import (
	"context"
	"fmt"
	"io"
	"math"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/vt"
	"golang.org/x/term"

	"github.com/ferret-linux/otter/pkg/containermanager"
	"github.com/ferret-linux/otter/pkg/ttyutil"
	"github.com/ferret-linux/otter/pkg/ui"
)

// scrollWindow is an io.Writer that renders the pty bytes written to it in
// a fixed-size, bordered region of the real terminal. Bytes are fed into a
// vt.Emulator, which interprets cursor movement, carriage returns, and
// other ANSI control sequences the way a real terminal would; the box is
// then redrawn (cursor up + clear + reprint) from the emulator's current
// screen state on every Write. This lets a pty-driven process (e.g.
// docker/podman/nerdctl doing its own in-place, multi-line pull-progress
// animation) render correctly inside the box, instead of its control
// sequences being treated as opaque line content.
//
// scrollWindow assumes exclusive control of the terminal region it
// draws into between Start and Close — nothing else should write to w
// concurrently during that span.
type scrollWindow struct {
	w io.Writer

	mu    sync.Mutex
	emu   *vt.Emulator // virtual terminal; owns the current screen content and dimensions
	drawn bool

	style lipgloss.Style // bordered box style, sized responsively to the terminal (see scaledSize)

	resizeCh   chan os.Signal // SIGWINCH notifications, live between Start and Close
	resizeDone chan struct{}  // closed to stop the resize-watching goroutine

	onResize func(rows, cols int) // optional; see OnResize
}

// Size implements PullOutputSizer. It returns the box's current content
// area, in cells — the same dimensions a process writing into the box
// should assume it has to work with.
func (s *scrollWindow) Size() (rows, cols int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.emu.Height(), s.emu.Width()
}

// OnResize implements PullOutputSizer. fn is called with the box's new
// content-area size after every resize for as long as the window is open.
func (s *scrollWindow) OnResize(fn func(rows, cols int)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onResize = fn
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

// k and floor below are fit so that fraction(120 cols) = 50% (width) and
// fraction(20 rows) ≈ 70%, fraction(40 rows) ≈ 35% (height), while
// keeping the box's absolute size non-decreasing as the terminal grows;
// see scaledSize. Shared by newScrollWindow and resizeLocked so both use
// the same curve.
const (
	widthK      = 110.47
	widthFloor  = 0.05
	heightK     = 28.3
	heightFloor = 0.05
)

// dimensions reads the current terminal size and returns the box's
// window-line count and total width, per scaledSize.
func dimensions() (windowLines, boxWidth int) {
	width, height := 80, 24
	if ww, hh, err := term.GetSize(int(os.Stderr.Fd())); err == nil {
		width = ww
		height = hh
	}

	windowLines = scaledSize(height, heightK, heightFloor)
	if windowLines < 1 {
		windowLines = 1
	}

	boxWidth = scaledSize(width, widthK, widthFloor)
	return windowLines, boxWidth
}

func newScrollWindow(w io.Writer) *scrollWindow {
	windowLines, boxWidth := dimensions()
	style := ui.BorderStyle(boxWidth)
	innerWidth := boxWidth - style.GetHorizontalFrameSize()

	return &scrollWindow{
		w:     w,
		emu:   vt.NewEmulator(innerWidth, windowLines),
		style: style,
	}
}

// Start draws the initial empty window and begins watching for terminal
// resizes (SIGWINCH), redrawing the box at its new size whenever the
// terminal is resized for as long as the window stays open.
func (s *scrollWindow) Start() {
	fmt.Fprint(s.w, "\033[?25l")

	s.mu.Lock()
	s.redrawLocked()
	s.mu.Unlock()

	s.resizeCh = make(chan os.Signal, 1)
	s.resizeDone = make(chan struct{})
	signal.Notify(s.resizeCh, syscall.SIGWINCH)

	go s.watchResize()
}

// watchResize redraws the box at its new size on every SIGWINCH, until
// resizeDone is closed by Close.
func (s *scrollWindow) watchResize() {
	for {
		select {
		case <-s.resizeCh:
			s.mu.Lock()
			s.resizeLocked()
			s.mu.Unlock()
		case <-s.resizeDone:
			return
		}
	}
}

// Close finalizes the pull box: any trailing partial line is discarded,
// and the box is erased entirely, with the cursor restored to the row it
// started on. The box is a transient progress indicator, not output
// meant to remain on screen once the pull is done.
func (s *scrollWindow) Close() {
	if s.resizeCh != nil {
		signal.Stop(s.resizeCh)
		close(s.resizeDone)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.drawn {
		return
	}

	s.eraseLocked()
	s.drawn = false
	fmt.Fprint(s.w, "\033[?25h")
}

// eraseLocked clears the currently-drawn window (len(s.lines)+2 rows: the
// content lines plus the box's top and bottom border) and leaves the
// cursor at the row the window started on. Must be called with s.mu held
// and s.drawn true.
func (s *scrollWindow) eraseLocked() {
	total := s.emu.Height() + 2
	fmt.Fprintf(s.w, "\033[%dA", total)
	for i := range total {
		fmt.Fprint(s.w, "\033[2K")
		if i < total-1 {
			fmt.Fprint(s.w, "\n")
		}
	}
	fmt.Fprintf(s.w, "\033[%dA", total-1)
}

// Write implements io.Writer. Bytes are fed directly into the virtual
// terminal emulator, which interprets cursor movement, carriage returns,
// and other ANSI control sequences the way a real terminal would, then
// the window is redrawn from the emulator's current screen state.
func (s *scrollWindow) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	n, err := s.emu.Write(p)
	if err != nil {
		return n, err
	}

	s.redrawLocked()

	return n, nil
}

// resizeLocked re-reads the terminal size and, if the box's dimensions
// have changed, erases the current (old-sized) box, recomputes the style
// and inner width, resizes the line ring to match, and redraws. The
// content currently visible is preserved across the resize (see
// resizeLinesLocked); only lines that scroll off as a result of the new,
// smaller size are dropped. Must be called with s.mu held.
func (s *scrollWindow) resizeLocked() {
	windowLines, boxWidth := dimensions()

	if windowLines == s.emu.Height() && boxWidth == s.style.GetWidth() {
		return
	}

	if s.drawn {
		s.eraseLocked()
		s.drawn = false
	}

	s.style = ui.BorderStyle(boxWidth)
	innerWidth := boxWidth - s.style.GetHorizontalFrameSize()
	s.emu.Resize(innerWidth, windowLines)

	s.redrawLocked()

	if s.onResize != nil {
		s.onResize(windowLines, innerWidth)
	}
}

// redrawLocked reprints the current window contents in place: it moves
// the cursor back to the top of the previously-drawn window (if any),
// clears each line, and reprints the emulator's current screen state.
// Must be called with s.mu held.
func (s *scrollWindow) redrawLocked() {
	if s.drawn {
		fmt.Fprintf(s.w, "\033[%dA", s.emu.Height()+2)
	}

	box := s.style.Render(s.emu.String())

	for _, line := range strings.Split(box, "\n") {
		fmt.Fprint(s.w, "\033[2K")
		fmt.Fprintln(s.w, line)
	}
	s.drawn = true
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

	var win *scrollWindow
	var out containermanager.PullOutput
	if ttyutil.IsInteractive() {
		win = newScrollWindow(os.Stderr)
		win.Start()
		out = win
	}

	err := cm.PullImage(ctx, imageRef, platform, out)

	// Close the box before anything below writes to the same stream
	// (progress.Fail/Done, cleanupDanglingImage's logging) — Close's
	// erase math assumes the cursor is still exactly where the box's
	// last redraw left it, so any interleaved write here would corrupt it.
	if win != nil {
		win.Close()
	}

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
