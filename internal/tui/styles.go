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

	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(colorCyan)
	cursorMark = lipgloss.NewStyle().Foreground(colorCyan).Bold(true)
	helpStyle  = lipgloss.NewStyle().Foreground(colorDim)

	detailBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder()).
				BorderForeground(colorDim).
				Padding(0, 1)
	detailKeyStyle   = lipgloss.NewStyle().Foreground(colorDim)
	detailValueStyle = lipgloss.NewStyle().Foreground(colorTeal)
)
