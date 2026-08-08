package tui

import "github.com/ferret-linux/otter/pkg/containermanager"

// focusPane identifies which pane of the Home section currently has input
// focus.
type focusPane int

const (
	focusList focusPane = iota
	focusActions
)

func toggleFocus(f focusPane) focusPane {
	if f == focusList {
		return focusActions
	}
	return focusList
}

// actionsModel is the Home section's action menu: a small, static list of
// operations for the currently selected container. It's hand-rolled rather
// than a second bubbles/list.Model since it's a fixed set of rows with no
// filtering/search needs.
type actionsModel struct {
	cursor int
}

func newActionsModel() actionsModel {
	return actionsModel{}
}

// defaultActionLabels lists the actions offered for a container, in
// display order. Index 1 ("Start") is swapped for "Stop" by labelsFor when
// the selected container is running.
//
// NOTE: unlike Start/Stop, Lock does not flip to "Unlock" dynamically yet.
// Determining lock state requires an extra I/O round-trip per container
// (pkg/commands/lock.go's isLocked does a CopyFromContainer call), which
// this navigation-only phase deliberately doesn't perform. Revisit once
// actions actually execute.
func defaultActionLabels() []string {
	return []string{"Enter", "Start", "Restart", "Pause", "Lock", "Upgrade", "Logs", "Remove"}
}

// labelsFor returns the action labels for the given container, adjusting
// dynamic entries (currently just Start/Stop) based on its state.
func labelsFor(c *containermanager.Container) []string {
	labels := append([]string(nil), defaultActionLabels()...)
	if c != nil && c.IsRunning() {
		labels[1] = "Stop"
	}
	return labels
}

func (a actionsModel) moveUp() actionsModel {
	if a.cursor > 0 {
		a.cursor--
	}
	return a
}

func (a actionsModel) moveDown() actionsModel {
	if a.cursor < len(defaultActionLabels())-1 {
		a.cursor++
	}
	return a
}
