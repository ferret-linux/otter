package tui

import (
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// keyMap describes otter tui's top-level keybindings and satisfies
// help.KeyMap, so the footer help line is generated from the same
// bindings Update actually switches on rather than a hand-written string
// that can drift out of sync.
//
//nolint:gochecknoglobals // see justification in styles.go
var keys = keyMap{
	switchSection: key.NewBinding(
		key.WithKeys("left", "right", "h", "l"),
		key.WithHelp("←/→", "switch section"),
	),
	move: key.NewBinding(
		key.WithKeys("up", "down", "k", "j"),
		key.WithHelp("↑/↓", "move"),
	),
	switchPane: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "switch pane"),
	),
	selectItem: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("↵", "select"),
	),
	quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
}

type keyMap struct {
	switchSection key.Binding
	move          key.Binding
	switchPane    key.Binding
	selectItem    key.Binding
	quit          key.Binding
}

// ShortHelp satisfies help.KeyMap.
func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.switchSection, k.move, k.switchPane, k.selectItem, k.quit}
}

// FullHelp satisfies help.KeyMap. otter tui only ever renders the short
// help line, but the interface requires this.
func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{k.ShortHelp()}
}

// section identifies one of otter tui's top-level screens.
type section int

const (
	sectionHome section = iota
	sectionShell
	sectionRegistry
	sectionCreate
	sectionDocs
)

// sectionNames are the tab bar labels, in display order, matching the
// section consts above.
//
//nolint:gochecknoglobals // see justification in styles.go
var sectionNames = []string{"Home", "Shell", "Registry", "Create", "Docs"}

// These must stay in sync with appStyle's Padding (styles.go) and with how
// View assembles the tab/divider/body/help rows, since they're used to
// compute how much space is left over for a section's own content.
const (
	appPaddingCols = 4 // appStyle's Padding(1, 2) → 2 left + 2 right
	appPaddingRows = 2 // appStyle's Padding(1, 2) → 1 top + 1 bottom
	tabRowHeight   = 1 // plain text tab line, no border
	dividerRows    = 1 // the "─" rule under the tabs
	gapRows        = 2 // blank line after the divider + blank line before help
	footerRows     = 1 // the help line itself
)

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, keys.quit):
			return a, tea.Quit
		case msg.String() == "right" || msg.String() == "l":
			a.active = nextSection(a.active)
			return a, nil
		case msg.String() == "left" || msg.String() == "h":
			a.active = prevSection(a.active)
			return a, nil
		case msg.String() == "1":
			a.active = sectionHome
			return a, nil
		case msg.String() == "2":
			a.active = sectionShell
			return a, nil
		case msg.String() == "3":
			a.active = sectionRegistry
			return a, nil
		case msg.String() == "4":
			a.active = sectionCreate
			return a, nil
		case msg.String() == "5":
			a.active = sectionDocs
			return a, nil
		}

	case tea.WindowSizeMsg:
		a.width, a.height = msg.Width, msg.Height
		cw, ch := a.contentSize()
		a.home = a.home.SetSize(cw, ch)
		a.help.SetWidth(cw)
		return a, nil
	}

	if a.active == sectionHome {
		var cmd tea.Cmd
		a.home, cmd = a.home.Update(msg)
		return a, cmd
	}

	return a, nil
}

func (a App) Init() tea.Cmd {
	return a.home.Init()
}

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return a, tea.Quit
		case "right", "l":
			a.active = nextSection(a.active)
			return a, nil
		case "left", "h":
			a.active = prevSection(a.active)
			return a, nil
		case "1":
			a.active = sectionHome
			return a, nil
		case "2":
			a.active = sectionShell
			return a, nil
		case "3":
			a.active = sectionRegistry
			return a, nil
		case "4":
			a.active = sectionCreate
			return a, nil
		case "5":
			a.active = sectionDocs
			return a, nil
		}

	case tea.WindowSizeMsg:
		a.width, a.height = msg.Width, msg.Height
		cw, ch := a.contentSize()
		a.home = a.home.SetSize(cw, ch)
		return a, nil
	}

	if a.active == sectionHome {
		var cmd tea.Cmd
		a.home, cmd = a.home.Update(msg)
		return a, cmd
	}

	return a, nil
}

// contentSize returns the space available for a section's own content,
// after accounting for the tab row, the divider, the footer, and
// appStyle's padding — the only things in this app that consume space.
func (a App) contentSize() (int, int) {
	width := a.width - appPaddingCols
	height := a.height - appPaddingRows - tabRowHeight - dividerRows - gapRows - footerRows
	if width < 0 {
		width = 0
	}
	if height < 0 {
		height = 0
	}
	return width, height
}

func (a App) View() tea.View {
	cw, ch := a.contentSize()

	tabs := a.renderTabs()
	divider := dimStyle.Render(strings.Repeat("─", cw))

	var body string
	if a.active == sectionHome {
		body = a.home.View()
	} else {
		body = lipgloss.NewStyle().Width(cw).Height(ch).
			Render(helpStyle.Render(sectionNames[a.active] + " — coming soon"))
	}

	footer := helpStyle.Render(a.help.View(keys))

	screen := tabs + "\n" + divider + "\n\n" + body + "\n\n" + footer

	v := tea.NewView(appStyle.Render(screen))
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	return v
}

// renderTabs draws the section tab bar as plain, evenly-spaced text — no
// borders, no boxes, nothing that needs to visually line up with a
// neighboring tab.
func (a App) renderTabs() string {
	rendered := make([]string, len(sectionNames))
	for i, name := range sectionNames {
		style := inactiveTabStyle
		if section(i) == a.active {
			style = activeTabStyle
		}
		rendered[i] = style.Render(name)
	}
	return strings.Join(rendered, "   ")
}

func nextSection(s section) section {
	if int(s) >= len(sectionNames)-1 {
		return s
	}
	return s + 1
}

func prevSection(s section) section {
	if s <= 0 {
		return s
	}
	return s - 1
}
