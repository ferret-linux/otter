package tui

import (
	"context"
	"fmt"
	"io"
	"strings"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/ferret-linux/otter/pkg/commands"
	"github.com/ferret-linux/otter/pkg/containermanager"
	"github.com/ferret-linux/otter/pkg/ui"
)

const containerIDDisplayLength = 12

// containerItem adapts a containermanager.Container to list.Item so it can
// be shown in the containers list.
type containerItem struct {
	container containermanager.Container
}

func (i containerItem) FilterValue() string { return i.container.Name }

// containerDelegate renders containerItem rows, reusing the same status dot
// and color conventions as `otter list` (see pkg/commands/list.go PrintList).
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

	dot, status := statusPresentation(c)
	line := fmt.Sprintf("%s %s  %s  %s",
		dot,
		nameStyle.Render(c.Name),
		status,
		imageStyle.Render(ui.TrimImageRef(c.Image)),
	)

	prefix := "  "
	if index == m.Index() {
		prefix = cursorMark.Render("▶ ")
	}

	fmt.Fprint(w, prefix+line)
}

// statusPresentation returns the colored status dot and status text for a
// container, mirroring the ●/○ + color convention in
// pkg/commands/list.go's PrintList.
func statusPresentation(c containermanager.Container) (dot string, status string) {
	switch {
	case c.IsRunning():
		return runningStyle.Render("●"), runningStyle.Render(c.Status)
	case strings.Contains(strings.ToLower(c.Status), "paused"):
		return pausedStyle.Render("●"), pausedStyle.Render(c.Status)
	case strings.Contains(strings.ToLower(c.Status), "exited"):
		return exitedStyle.Render("○"), exitedStyle.Render(c.Status)
	default:
		return pausedStyle.Render("○"), pausedStyle.Render(c.Status)
	}
}

// containersLoadedMsg carries the result of fetching the container list.
type containersLoadedMsg struct {
	result *commands.ListResult
}

// errMsg carries an error that occurred while loading the container list.
type errMsg struct {
	err error
}

// Model is the containers list screen: otter tui's homepage.
type Model struct {
	ctx context.Context
	cm  containermanager.ContainerManager

	list    list.Model
	loading bool
	err     error

	width, height int
}

// NewModel builds the initial containers list model.
func NewModel(ctx context.Context, cm containermanager.ContainerManager) Model {
	l := list.New(nil, containerDelegate{}, 0, 0)
	// Lip Gloss v2 dropped adaptive light/dark colors, so list styles must
	// be chosen explicitly. We assume a dark terminal background, which is
	// the common default; see bubbles' UPGRADE_GUIDE_V2.md.
	l.Styles = list.DefaultStyles(true)
	l.Styles.Title = titleStyle
	l.Title = "otter"
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)

	return Model{
		ctx:     ctx,
		cm:      cm,
		list:    l,
		loading: true,
	}
}

func (m Model) Init() tea.Cmd {
	return fetchContainers(m.ctx, m.cm)
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

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.list.SetSize(listWidth(msg.Width), msg.Height)

	case containersLoadedMsg:
		m.loading = false
		items := make([]list.Item, 0, len(msg.result.Containers))
		for _, c := range msg.result.Containers {
			items = append(items, containerItem{container: c})
		}
		cmd := m.list.SetItems(items)
		return m, cmd

	case errMsg:
		m.loading = false
		m.err = msg.err
		return m, nil
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m Model) View() tea.View {
	var v tea.View

	switch {
	case m.err != nil:
		v = tea.NewView(exitedStyle.Render(fmt.Sprintf("error: %s", m.err)) +
			"\n\n" + helpStyle.Render("press q to quit"))
	case m.loading:
		v = tea.NewView("loading environments...")
	default:
		body := lipgloss.JoinHorizontal(lipgloss.Top, m.list.View(), m.renderDetail())
		help := helpStyle.Render("↑/↓ navigate · q quit")
		v = tea.NewView(body + "\n" + help)
	}

	v.AltScreen = true
	return v
}

func (m Model) renderDetail() string {
	selected, ok := m.list.SelectedItem().(containerItem)
	if !ok {
		return detailBorderStyle.Render("no environment selected")
	}
	c := selected.container

	lines := []string{
		detailKeyStyle.Render("id      ") + detailValueStyle.Render(shortID(c.ID)),
		detailKeyStyle.Render("image   ") + detailValueStyle.Render(ui.TrimImageRef(c.Image)),
		detailKeyStyle.Render("status  ") + detailValueStyle.Render(c.Status),
	}

	return detailBorderStyle.Render(strings.Join(lines, "\n"))
}

func shortID(id string) string {
	if len(id) > containerIDDisplayLength {
		return id[:containerIDDisplayLength]
	}
	return id
}

func listWidth(total int) int {
	const detailWidth = 32
	w := total - detailWidth
	if w < 20 {
		w = 20
	}
	return w
}
