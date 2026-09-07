package registry

import (
	"context"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/taigrr/bubbleterm"
	"golang.org/x/term"

	"github.com/ferret-linux/otter/pkg/containermanager"
	"github.com/ferret-linux/otter/pkg/ttyutil"
	"github.com/ferret-linux/otter/pkg/ui"
)

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
// see scaledSize. Shared by newPullModel and pullModel's WindowSizeMsg
// handling so both use the same curve.
const (
	widthK      = 110.47
	widthFloor  = 0.05
	heightK     = 28.3
	heightFloor = 0.05
)

// scaledDimensions returns the box's window-line count and total width for
// a terminal of the given size, per scaledSize.
func scaledDimensions(width, height int) (windowLines, boxWidth int) {
	windowLines = scaledSize(height, heightK, heightFloor)
	if windowLines < 1 {
		windowLines = 1
	}
	boxWidth = scaledSize(width, widthK, widthFloor)
	return windowLines, boxWidth
}

// initialTerminalSize returns the real terminal size, for use before the
// bubbletea program starts (and thus before any tea.WindowSizeMsg has
// arrived to report it).
func initialTerminalSize() (width, height int) {
	width, height = 80, 24
	if ww, hh, err := term.GetSize(int(os.Stderr.Fd())); err == nil {
		width, height = ww, hh
	}
	return width, height
}

// pullExitedMsg reports that the pull command has exited. bubbleterm has no
// built-in "process exited" message of its own (see pullModel.exited), so
// pullModel synthesizes one.
type pullExitedMsg struct{}

// pullModel is a small bubbletea wrapper around a *bubbleterm.Model that
// reproduces the pull box's previous look and responsive sizing (see
// scaledSize/scaledDimensions) on top of bubbleterm's own pty handling.
// Unlike the box it replaces, tea.KeyMsg is forwarded into child — that's
// what actually makes the pty bidirectional and fixes the --root
// password-prompt bug, rather than only ever copying pty output out.
type pullModel struct {
	child *bubbleterm.Model
	style lipgloss.Style // bordered box style, sized responsively to the terminal (see scaledDimensions)

	cmd     *exec.Cmd     // the pull command; ProcessState is read from it once exited
	exited  chan struct{} // signaled once, from child.GetEmulator()'s exit callback or immediately if already exited
	pullErr error         // the pull's result, set once pullExitedMsg is handled
	boxRows int           // rows the box occupies on screen (windowLines + 2 borders), used to erase it on exit
}

// newPullModel creates the wrapper model and starts cmd inside it. cmd must
// not have been started yet.
func newPullModel(cmd *exec.Cmd) (*pullModel, error) {
	width, height := initialTerminalSize()
	windowLines, boxWidth := scaledDimensions(width, height)
	style := ui.BorderStyle(boxWidth)
	innerWidth := boxWidth - style.GetHorizontalFrameSize()

	child, err := bubbleterm.New(innerWidth, windowLines)
	if err != nil {
		return nil, err
	}

	// Buffered so the callback below never blocks the process's own exit
	// handling, and so a send that races ahead of the immediate
	// IsProcessExited check just below isn't lost.
	exited := make(chan struct{}, 1)
	child.GetEmulator().SetOnExit(func(string) {
		select {
		case exited <- struct{}{}:
		default:
		}
	})

	if err := child.GetEmulator().StartCommand(cmd); err != nil {
		child.Close()
		return nil, err
	}

	// The command may already have exited (or the exit callback may have
	// already fired) by the time StartCommand returns; make sure exited is
	// signaled either way.
	if child.GetEmulator().IsProcessExited() {
		select {
		case exited <- struct{}{}:
		default:
		}
	}

	return &pullModel{child: child, style: style, cmd: cmd, exited: exited, boxRows: windowLines + 2}, nil
}

// waitExited returns a tea.Cmd that blocks until ch is signaled, then
// reports the pull as finished. child.GetEmulator().Done() can't be used
// for this: that channel closes when the emulator itself is closed (i.e.
// when we close it), not when the underlying process exits.
func waitExited(ch <-chan struct{}) tea.Cmd {
	return func() tea.Msg {
		<-ch
		return pullExitedMsg{}
	}
}

func (m *pullModel) Init() tea.Cmd {
	return tea.Batch(m.child.Init(), waitExited(m.exited))
}

func (m *pullModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		windowLines, boxWidth := scaledDimensions(msg.Width, msg.Height)
		m.style = ui.BorderStyle(boxWidth)
		m.boxRows = windowLines + 2
		innerWidth := boxWidth - m.style.GetHorizontalFrameSize()

		updated, cmd := m.child.Update(tea.WindowSizeMsg{Width: innerWidth, Height: windowLines})
		m.child = updated.(*bubbleterm.Model)
		return m, cmd

	case pullExitedMsg:
		if m.cmd.ProcessState != nil && !m.cmd.ProcessState.Success() {
			m.pullErr = &exec.ExitError{ProcessState: m.cmd.ProcessState}
		}
		return m, tea.Quit
	}

	updated, cmd := m.child.Update(msg)
	m.child = updated.(*bubbleterm.Model)
	return m, cmd
}

func (m *pullModel) View() tea.View {
	// Render inline in the terminal (the alternate screen is a whole-terminal
	// takeover, which isn't what a transient pull box should do). The box
	// appears at the current cursor position over the existing terminal
	// content and is cleared when the program quits, restoring the terminal
	// as it was before.
	v := tea.NewView(m.style.Render(m.child.View().Content))
	return v
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

	if err := runPull(ctx, cm, imageRef, platform); err != nil {
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

// eraseInlineBox clears the rows the pull box was drawn over and restores
// the terminal to how it was before the box appeared.
//
// Bubble Tea's inline renderer overwrites its own frame in the main screen
// buffer but, unlike the alternate screen, does not erase it on quit: on
// close it only issues erase-below from the bottom of the screen, which
// leaves the box's rows behind. We therefore clear them ourselves. The box
// occupies the last windowLines+2 rows of the screen (windowLines content
// rows plus the top and bottom borders), so we move the cursor back up to
// the box's first row and erase that whole block, then show the cursor
// again. boxRows is the number of rows to erase (windowLines+2).
func eraseInlineBox(w io.Writer, boxRows int) {
	if boxRows < 1 {
		return
	}
	// Hide the cursor so the block erase doesn't leave it visibly hopping.
	// On return from Bubble Tea the cursor rests at the bottom of the screen
	// on the box's last row; walk it up boxRows-1 rows to the box's top
	// border, clearing each row as we go, then restore the cursor.
	fmt.Fprint(w, "\x1b[?25l")
	for i := 0; i < boxRows; i++ {
		if i > 0 {
			fmt.Fprint(w, "\x1b[1A") // up one row
		}
		fmt.Fprint(w, "\x1b[2K") // erase the whole line
	}
	fmt.Fprint(w, "\x1b[?25h") // show cursor
}

// runPull runs the actual pull: interactively, in a bordered, resizable box
// when attached to a real terminal, or via the provider's own silent,
// buffered pull otherwise.
func runPull(ctx context.Context, cm containermanager.ContainerManager, imageRef, platform string) error {
	interactive := ttyutil.IsInteractive()

	cmd, err := cm.PullImage(ctx, imageRef, platform, interactive)
	if err != nil {
		return err
	}
	if !interactive {
		return nil
	}

	model, err := newPullModel(cmd)
	if err != nil {
		return err
	}

	finalModel, err := tea.NewProgram(model, tea.WithOutput(os.Stderr)).Run()
	if err != nil {
		return err
	}
	pm := finalModel.(*pullModel)

	// Erase the inline box Bubble Tea left in the main screen buffer (it's
	// not on the alternate screen, so it isn't cleaned up on quit).
	eraseInlineBox(os.Stderr, pm.boxRows)

	return pm.pullErr
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
