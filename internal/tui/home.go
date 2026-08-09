package tui

import (
	"context"
	"fmt"
	"strings"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/ferret-linux/otter/pkg/commands"
	"github.com/ferret-linux/otter/pkg/containermanager"
	"github.com/ferret-linux/otter/pkg/ui"
)

// actionsPaneWidth is the fixed width of the Home section's action menu.
const actionsPaneWidth = 22

// containersLoadedMsg carries the result of fetching the container list.
type containersLoadedMsg struct {
	result *commands.ListResult
}

// errMsg carries an error that occurred while loading the container list.
type errMsg struct {
	err error
}

// homeModel is otter tui's Home section: a container list on the left and
// an action menu for the selected container on the right.
type homeModel struct {
	ctx context.Context
	cm  containermanager.ContainerManager

	list    list.Model
	actions list.Model
	focus   focusPane

	loading bool
	err     error

	width, height int
}

// newList builds a bare bubbles/list.Model with all chrome (title, status
// bar, help, pagination, filtering) suppressed — both the container list
// and the actions pane are fixed-content sidebars with no need for any of
// it, so every Show*/SetFilteringEnabled call below is required to avoid
// unwanted UI appearing.
func newList(delegate list.ItemDelegate) list.Model {
	l := list.New(nil, delegate, 0, 0)
	// Lip Gloss v2 dropped adaptive light/dark colors, so list styles must
	// be chosen explicitly. We assume a dark terminal background, which is
	// the common default; see bubbles' UPGRADE_GUIDE_V2.md.
	l.Styles = list.DefaultStyles(true)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)
	l.SetShowPagination(false)
	l.SetFilteringEnabled(false)
	return l
}

// newHomeModel builds the initial Home section model.
func newHomeModel(ctx context.Context, cm containermanager.ContainerManager) homeModel {
	return homeModel{
		ctx:     ctx,
		cm:      cm,
		list:    newList(containerDelegate{}),
		actions: newList(actionDelegate{}),
		loading: true,
	}
}

func (h homeModel) Init() tea.Cmd {
	return fetchContainers(h.ctx, h.cm)
}

func fetchContainers(ctx context.Context, cm containermanager.ContainerManager) tea.Cmd {
	return func() tea.Msg {
		result, err := commands.NewListCommand(cm).Execute(ctx, commands.ListOptions{})
		if err != nil {
			return errMsg{err: err}
		}
		return containersLoadedMsg{result: result}
	}
}

// SetSize resizes the Home section to fit within the given content area —
// the space left over after the app frame's tab bar, border, and padding.
// See App.contentSize.
func (h homeModel) SetSize(width, height int) homeModel {
	h.width, h.height = width, height
	h.list.SetSize(homeListWidth(width), height)
	actionsHeight := height - actionsHeaderRows
	if actionsHeight < 0 {
		actionsHeight = 0
	}
	h.actions.SetSize(actionsPaneWidth, actionsHeight)
	return h
}

// homeListWidth reserves space for the action pane and the divider
// between it and the container list.
func homeListWidth(total int) int {
	const dividerWidth = 1
	w := total - actionsPaneWidth - dividerWidth
	if w < 16 {
		w = 16
	}
	return w
}

func (h homeModel) Update(msg tea.Msg) (homeModel, tea.Cmd) {
	switch msg := msg.(type) {
	case containersLoadedMsg:
		h.loading = false
		items := make([]list.Item, 0, len(msg.result.Containers))
		for _, c := range msg.result.Containers {
			items = append(items, containerItem{container: c})
		}
		cmd := h.list.SetItems(items)
		h.actions.SetItems(h.currentActionItems())
		return h, cmd

	case errMsg:
		h.loading = false
		h.err = msg.err
		return h, nil

	case tea.KeyPressMsg:
		switch msg.String() {
		case "tab":
			h.focus = toggleFocus(h.focus)
			return h, nil
		case "enter":
			// Actions don't execute yet — navigation-only until the
			// results/error feedback mechanism is designed.
			return h, nil
		}

	case tea.MouseClickMsg:
		return h.handleMouseClick(msg), nil

	case tea.MouseWheelMsg:
		return h.handleMouseWheel(msg), nil
	}

	if h.focus == focusList {
		prevIndex := h.list.Index()
		var cmd tea.Cmd
		h.list, cmd = h.list.Update(msg)
		if h.list.Index() != prevIndex {
			h.actions.SetItems(h.currentActionItems())
		}
		return h, cmd
	}

	var cmd tea.Cmd
	h.actions, cmd = h.actions.Update(msg)
	return h, cmd
}

// currentActionItems returns the action list.Items for the currently
// selected container, or nil if none is selected.
func (h homeModel) currentActionItems() []list.Item {
	selected, ok := h.list.SelectedItem().(containerItem)
	if !ok {
		return nil
	}
	return actionListItems(&selected.container)
}

func (h homeModel) View() string {
	switch {
	case h.err != nil:
		return exitedStyle.Render(fmt.Sprintf("error: %s", h.err))
	case h.loading:
		return "loading environments..."
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, h.list.View(), verticalDivider(h.height), h.renderActions())
}

func verticalDivider(height int) string {
	if height < 1 {
		height = 1
	}
	line := dividerStyle.Render("│")
	lines := make([]string, height)
	for i := range lines {
		lines[i] = line
	}
	return strings.Join(lines, "\n")
}

func (h homeModel) renderActions() string {
	selected, ok := h.list.SelectedItem().(containerItem)
	if !ok {
		return lipgloss.NewStyle().Width(actionsPaneWidth).Render(dimStyle.Render("no environment selected"))
	}
	c := selected.container

	header := nameStyle.Render(c.Name) + "\n" + imageStyle.Render(ui.TrimImageRef(c.Image))
	body := header + "\n\n" + h.actions.View()
	return lipgloss.NewStyle().Width(actionsPaneWidth).Render(body)
}

// listPaneAt reports which pane (if any) contains column x, given the
// container list's rendered width and the single-column divider between
// the two panes. Matches the layout built by View/homeListWidth.
func (h homeModel) listPaneAt(x int) (focusPane, bool) {
	listWidth := homeListWidth(h.width)
	switch {
	case x < listWidth:
		return focusList, true
	case x < listWidth+1:
		return 0, false // the divider itself
	default:
		return focusActions, true
	}
}

// actionsHeaderRows is the number of lines renderActions draws above the
// actions list itself (container name + image + one blank line), which a
// click's row offset must account for.
const actionsHeaderRows = 3

func (h homeModel) handleMouseClick(msg tea.MouseClickMsg) homeModel {
	mouse := msg.Mouse()
	if mouse.Button != tea.MouseLeft {
		return h
	}

	pane, ok := h.listPaneAt(mouse.X)
	if !ok {
		return h
	}
	h.focus = pane

	switch pane {
	case focusList:
		if mouse.Y >= 0 && mouse.Y < len(h.list.Items()) {
			h.list.Select(mouse.Y)
			h.actions.SetItems(h.currentActionItems())
		}
	case focusActions:
		row := mouse.Y - actionsHeaderRows
		if row >= 0 && row < len(h.actions.Items()) {
			h.actions.Select(row)
		}
	}
	return h
}

func (h homeModel) handleMouseWheel(msg tea.MouseWheelMsg) homeModel {
	mouse := msg.Mouse()
	pane, ok := h.listPaneAt(mouse.X)
	if !ok {
		return h
	}

	switch pane {
	case focusList:
		switch mouse.Button {
		case tea.MouseWheelUp:
			h.list.CursorUp()
			h.actions.SetItems(h.currentActionItems())
		case tea.MouseWheelDown:
			h.list.CursorDown()
			h.actions.SetItems(h.currentActionItems())
		}
	case focusActions:
		switch mouse.Button {
		case tea.MouseWheelUp:
			h.actions.CursorUp()
		case tea.MouseWheelDown:
			h.actions.CursorDown()
		}
	}
	return h
}
