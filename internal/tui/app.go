package tui

import (
	"context"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/ferret-linux/otter/pkg/containermanager"
)

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

// App is otter tui's top-level model. It owns the section tab bar and
// delegates rendering/input to whichever section is active. Only Home is
// implemented so far; the rest render as placeholders.
type App struct {
	ctx context.Context
	cm  containermanager.ContainerManager

	active section
	home   homeModel

	width, height int
}

// NewApp builds the initial top-level app model.
func NewApp(ctx context.Context, cm containermanager.ContainerManager) App {
	return App{
		ctx:  ctx,
		cm:   cm,
		home: newHomeModel(ctx, cm),
	}
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

	help := helpStyle.Render("←/→ switch section · ↑/↓ move · tab switch pane · ↵ select · q quit")

	screen := tabs + "\n" + divider + "\n\n" + body + "\n\n" + help

	v := tea.NewView(appStyle.Render(screen))
	v.AltScreen = true
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
