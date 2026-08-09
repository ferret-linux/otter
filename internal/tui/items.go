package tui

import (
	"fmt"
	"io"
	"strings"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"

	"github.com/ferret-linux/otter/pkg/containermanager"
)

// containerItem adapts a containermanager.Container to list.Item so it can
// be shown in the containers list.
type containerItem struct {
	container containermanager.Container
}

func (i containerItem) FilterValue() string { return i.container.Name }

// actionItem adapts an action label (e.g. "Start", "Remove") to list.Item
// so the actions pane can reuse bubbles/list instead of a hand-rolled
// cursor. FilterValue is unused since the actions list has filtering
// disabled, but is required to satisfy list.Item.
type actionItem string

func (i actionItem) FilterValue() string { return string(i) }

// actionDelegate renders actionItem rows as plain labels, matching
// containerDelegate's selected/unselected row styling.
type actionDelegate struct{}

func (d actionDelegate) Height() int  { return 1 }
func (d actionDelegate) Spacing() int { return 0 }

func (d actionDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d actionDelegate) Render(w io.Writer, m list.Model, index int, it list.Item) {
	label, ok := it.(actionItem)
	if !ok {
		return
	}

	rowStyle := unselectedRowStyle
	labelStyle := actionLabelStyle
	if index == m.Index() {
		rowStyle = selectedRowStyle
		labelStyle = actionActiveStyle
	}

	fmt.Fprint(w, rowStyle.Render(labelStyle.Render(string(label))))
}

// containerDelegate renders containerItem rows as a status dot plus name.
// The dot's shape and color already convey running/paused/exited state
// (mirroring the ●/○ convention in pkg/commands/list.go's PrintList), so
// the raw status text isn't repeated here — this sidebar is narrow.
type containerDelegate struct{}

func (d containerDelegate) Height() int  { return 1 }
func (d containerDelegate) Spacing() int { return 0 }

func (d containerDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d containerDelegate) Render(w io.Writer, m list.Model, index int, it list.Item) {
	ci, ok := it.(containerItem)
	if !ok {
		return
	}
	c := ci.container

	line := fmt.Sprintf("%s %s", statusDot(c), nameStyle.Render(c.Name))

	style := unselectedRowStyle
	if index == m.Index() {
		style = selectedRowStyle
	}

	fmt.Fprint(w, style.Render(line))
}

// statusDot returns the colored status dot for a container, mirroring the
// ●/○ + color convention in pkg/commands/list.go's PrintList, minus the
// status text.
func statusDot(c containermanager.Container) string {
	switch {
	case c.IsRunning():
		return runningStyle.Render("●")
	case strings.Contains(strings.ToLower(c.Status), "paused"):
		return pausedStyle.Render("●")
	case strings.Contains(strings.ToLower(c.Status), "exited"):
		return exitedStyle.Render("○")
	default:
		return pausedStyle.Render("○")
	}
}
