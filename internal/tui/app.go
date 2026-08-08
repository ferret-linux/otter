package tui

import (
	"context"

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
var sectionNames = []string{"Home", "Shell", "Registry", "Create", "Docs"}

// These must stay in sync with windowStyle's Border/Padding (styles.go) and
// tabStyle's Border/Padding, since they're used to compute how much space
// is left over for a section's own content.
const (
	windowBorderCols  = 2 // left + right border columns (top is unset)
	windowPaddingCols = 4 // Padding(1, 2) → 2 left + 2 right
	windowBorderRows  = 1 // bottom border row only; top is unset
	windowPaddingRows = 2 // Padding(1, 2) → 1 top + 1 bottom
	tabRowHeight      = 3 // tab box: top border + label line + bottom border
	footerGapRows     = 1 // blank line between the frame and the help text
	footerRows        = 1 // the help line itself
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
// after accounting for the tab row, the window frame, and the footer.
func (a App) contentSize() (width, height int) {
	width = a.width - windowBorderCols - windowPaddingCols
	height = a.height - tabRowHeight - windowBorderRows - windowPaddingRows - footerGapRows - footerRows
	if width < 0 {
		width = 0
	}
	if height < 0 {
		height = 0
	}
	return width, height
}

func (a App) View() tea.View {
	tabs := a.renderTabs()

	var body string
	if a.active == sectionHome {
		body = a.home.View()
	} else {
		cw, ch := a.contentSize()
		body = lipgloss.NewStyle().Width(cw).Height(ch).
			Render(helpStyle.Render(sectionNames[a.active] + " — coming soon"))
	}

	frame := windowStyle.Render(body)
	help := helpStyle.Render("←/→ switch section · ↑/↓ move · tab switch pane · ↵ select · q quit")

	v := tea.NewView(tabs + "\n" + frame + "\n\n" + help)
	v.AltScreen = true
	return v
}

// renderTabs draws the section tab bar as connected pill tabs: the active
// tab's bottom border opens straight into the window frame below it, while
// inactive tabs sit on a closed ridge. See styles.go's tabBorderWithBottom.
func (a App) renderTabs() string {
	rendered := make([]string, len(sectionNames))
	last := len(sectionNames) - 1

	for i, name := range sectionNames {
		isActive := section(i) == a.active

		style := tabStyle
		if isActive {
			style = activeTabStyle
		}

		border, _, _, _, _ := style.GetBorder()
		switch {
		case i == 0 && isActive:
			border.BottomLeft = "│"
		case i == 0 && !isActive:
			border.BottomLeft = "├"
		case i == last && isActive:
			border.BottomRight = "│"
		case i == last && !isActive:
			border.BottomRight = "┤"
		}
		style = style.Border(border)

		rendered[i] = style.Render(name)
	}

	return lipgloss.JoinHorizontal(lipgloss.Bottom, rendered...)
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
