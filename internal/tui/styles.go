// Package tui implements otter's interactive terminal UI.
package tui

import "charm.land/lipgloss/v2"

// Colors mirror the ANSI palette in pkg/ui/colors.go (red/green/yellow/cyan/
// teal/dim) so the TUI's visual language matches otter's existing CLI
// output. They're ANSI 16/256-color indices rather than hex so they resolve
// to the exact same terminal colors as the \033[31m-style codes in pkg/ui.
var (
	colorRed    = lipgloss.Color("1")  // matches \033[31m in pkg/ui/colors.go
	colorGreen  = lipgloss.Color("2")  // matches \033[32m
	colorYellow = lipgloss.Color("3")  // matches \033[33m
	colorCyan   = lipgloss.Color("6")  // matches \033[36m
	colorTeal   = lipgloss.Color("14") // matches \033[96m (bright cyan)
	colorDim    = lipgloss.Color("7")  // matches \033[37m
)

var (
	nameStyle    = lipgloss.NewStyle().Foreground(colorTeal)
	imageStyle   = lipgloss.NewStyle().Foreground(colorDim)
	runningStyle = lipgloss.NewStyle().Foreground(colorGreen)
	pausedStyle  = lipgloss.NewStyle().Foreground(colorYellow)
	exitedStyle  = lipgloss.NewStyle().Foreground(colorRed)

	cursorMark = lipgloss.NewStyle().Foreground(colorCyan).Bold(true)
	helpStyle  = lipgloss.NewStyle().Foreground(colorDim)
	dimStyle   = lipgloss.NewStyle().Foreground(colorDim)

	dividerStyle = lipgloss.NewStyle().Foreground(colorDim)

	actionLabelStyle  = lipgloss.NewStyle().Foreground(colorDim)
	actionActiveStyle = lipgloss.NewStyle().Foreground(colorTeal).Bold(true)
)

// tabBorderWithBottom returns a rounded border whose bottom edge uses the
// given corner/fill runes. Used to make inactive tabs sit on a continuous
// ridge while the active tab "opens" straight into the window below it —
// the technique from charmbracelet/bubbletea's examples/tabs.
func tabBorderWithBottom(left, middle, right string) lipgloss.Border {
	b := lipgloss.RoundedBorder()
	b.BottomLeft = left
	b.Bottom = middle
	b.BottomRight = right
	return b
}

var (
	inactiveTabBorder = tabBorderWithBottom("┴", "─", "┴")
	activeTabBorder   = tabBorderWithBottom("┘", " ", "└")

	tabStyle = lipgloss.NewStyle().
			Border(inactiveTabBorder).
			BorderForeground(colorCyan).
			Padding(0, 2)
	activeTabStyle = tabStyle.
			Border(activeTabBorder).
			Bold(true).
			Foreground(colorTeal)

	// windowStyle wraps the active section's content. Its top border is
	// unset because the tab row's bottom border serves as the seam between
	// the two — see App.renderTabs / App.View.
	windowStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorCyan).
			UnsetBorderTop().
			Padding(1, 2)
)
