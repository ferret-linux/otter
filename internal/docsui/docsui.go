package docsui

import (
	"fmt"

	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/glamour/v2"
	"charm.land/glamour/v2/styles"
	"charm.land/lipgloss/v2"

	"github.com/ferret-linux/otter/docsfs"
)

// focusPane identifies which pane currently has input focus.
type focusPane int

const (
	focusTree focusPane = iota
	focusContent
)

// glamourGutter accounts for the left-side margin glamour applies to
// rendered output, on top of the viewport's own border/padding — see
// bubbletea's examples/glamour, which calls out this exact adjustment.
const glamourGutter = 2

// treePaneWidth is the fixed width of the left tree pane.
const treePaneWidth = 28

// Model is otter docs' top-level bubbletea model.
type Model struct {
	tree     list.Model
	content  viewport.Model
	entries  []entry
	renderer *glamour.TermRenderer
	focus    focusPane

	err error

	width, height int
}

// New builds the initial docs UI model, walking the embedded docs/ tree
// and preparing the markdown renderer. Returns an error if docs/ itself
// can't be walked (a build-time embedding problem, not something a user
// action can trigger) or if the renderer can't be constructed.
func New() (Model, error) {
	entries, err := buildTree()
	if err != nil {
		return Model{}, fmt.Errorf("failed to read embedded docs: %w", err)
	}

	items := make([]list.Item, len(entries))
	for i, e := range entries {
		items[i] = treeItem{entry: e}
	}

	l := list.New(items, treeDelegate{}, 0, 0)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)
	l.SetShowPagination(false)
	l.SetFilteringEnabled(false)

	renderer, err := glamour.NewTermRenderer(glamour.WithStyles(styles.DarkStyleConfig))
	if err != nil {
		return Model{}, fmt.Errorf("failed to build markdown renderer: %w", err)
	}

	vp := viewport.New()

	m := Model{
		tree:     l,
		content:  vp,
		entries:  entries,
		renderer: renderer,
	}
	return m.loadSelected(), nil
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "tab":
			m.focus = toggleFocus(m.focus)
			return m, nil
		case "enter":
			if m.focus == focusTree {
				m.focus = focusContent
			}
			return m, nil
		}

	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m.setSize(msg.Width, msg.Height), nil
	}

	if m.focus == focusTree {
		prevIndex := m.tree.Index()
		var cmd tea.Cmd
		m.tree, cmd = m.tree.Update(msg)
		if m.tree.Index() != prevIndex {
			m = m.loadSelected()
		}
		return m, cmd
	}

	var cmd tea.Cmd
	m.content, cmd = m.content.Update(msg)
	return m, cmd
}

// loadSelected renders the currently-selected tree entry's markdown into
// the content viewport. Directory rows and read errors show an inline
// message in the content pane rather than failing the whole program,
// since either can happen from ordinary navigation (selecting a
// directory heading) rather than a program-level fault.
func (m Model) loadSelected() Model {
	item, ok := m.tree.SelectedItem().(treeItem)
	if !ok {
		return m
	}
	e := item.entry

	if e.isDir {
		m.content.SetContent("(directory — select a file to view it)")
		return m
	}

	raw, err := docsfs.FS.ReadFile(e.embedPath)
	if err != nil {
		m.content.SetContent(fmt.Sprintf("failed to read %s: %s", e.embedPath, err))
		return m
	}

	rendered, err := m.renderer.Render(string(raw))
	if err != nil {
		m.content.SetContent(fmt.Sprintf("failed to render %s: %s", e.embedPath, err))
		return m
	}

	m.content.SetContent(rendered)
	return m
}

func (m Model) setSize(width, height int) Model {
	const helpRows = 1

	contentWidth := width - treePaneWidth
	if contentWidth < 0 {
		contentWidth = 0
	}
	paneHeight := height - helpRows
	if paneHeight < 0 {
		paneHeight = 0
	}

	m.tree.SetSize(treePaneWidth-paneFrameSize(), paneHeight-paneFrameSize())
	m.content.SetWidth(contentWidth - paneFrameSize())
	m.content.SetHeight(paneHeight - paneFrameSize())

	renderWidth := contentWidth - paneFrameSize() - glamourGutter
	if renderWidth < 0 {
		renderWidth = 0
	}
	if renderer, err := glamour.NewTermRenderer(
		glamour.WithStyles(styles.DarkStyleConfig),
		glamour.WithWordWrap(renderWidth),
	); err == nil {
		m.renderer = renderer
	}

	return m.loadSelected()
}

// paneFrameSize is the width/height a RoundedBorder adds on each pane —
// both panes use the same border style, so this is one constant rather
// than a per-pane lipgloss.Style.GetHorizontalFrameSize() call.
func paneFrameSize() int { return 2 }

func (m Model) View() tea.View {
	if m.err != nil {
		return tea.NewView(fmt.Sprintf("error: %s", m.err))
	}

	treeBorder := paneBorderStyle
	contentBorder := paneBorderStyle
	if m.focus == focusTree {
		treeBorder = focusedPaneBorderStyle
	} else {
		contentBorder = focusedPaneBorderStyle
	}

	treePane := treeBorder.Render(m.tree.View())
	contentPane := contentBorder.Render(m.content.View())

	body := lipgloss.JoinHorizontal(lipgloss.Top, treePane, contentPane)
	help := helpStyle.Render("↑/↓ move · tab switch pane · ↵ open · q quit")

	v := tea.NewView(body + "\n" + help)
	v.AltScreen = true
	return v
}

func toggleFocus(f focusPane) focusPane {
	if f == focusTree {
		return focusContent
	}
	return focusTree
}
