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
	actions actionsModel
	focus   focusPane

	loading bool
	err     error

	width, height int
}

// newHomeModel builds the initial Home section model.
func newHomeModel(ctx context.Context, cm containermanager.ContainerManager) homeModel {
	l := list.New(nil, containerDelegate{}, 0, 0)
	// Lip Gloss v2 dropped adaptive light/dark colors, so list styles must
	// be chosen explicitly. We assume a dark terminal background, which is
	// the common default; see bubbles' UPGRADE_GUIDE_V2.md.
	l.Styles = list.DefaultStyles(true)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)

	return homeModel{
		ctx:     ctx,
		cm:      cm,
		list:    l,
		actions: newActionsModel(),
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
		case "up", "k":
			if h.focus == focusActions {
				h.actions = h.actions.moveUp()
				return h, nil
			}
		case "down", "j":
			if h.focus == focusActions {
				h.actions = h.actions.moveDown()
				return h, nil
			}
		case "enter":
			// Actions don't execute yet — navigation-only until the
			// results/error feedback mechanism is designed.
			return h, nil
		}
	}

	if h.focus == focusList {
		var cmd tea.Cmd
		h.list, cmd = h.list.Update(msg)
		return h, cmd
	}

	return h, nil
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

	var rows []string
	for i, label := range labelsFor(&c) {
		prefix := "  "
		style := actionLabelStyle
		if h.focus == focusActions && i == h.actions.cursor {
			prefix = cursorMark.Render("▶ ")
			style = actionActiveStyle
		}
		rows = append(rows, prefix+style.Render(label))
	}

	body := header + "\n\n" + strings.Join(rows, "\n")
	return lipgloss.NewStyle().Width(actionsPaneWidth).Render(body)
}
