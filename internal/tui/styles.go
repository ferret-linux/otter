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

var (
	// activeTabStyle and inactiveTabStyle render the section tab bar as
	// plain colored text — no borders, no per-tab boxes, nothing that needs
	// to visually line up with a neighboring tab or a frame below it.
	activeTabStyle   = lipgloss.NewStyle().Bold(true).Foreground(colorTeal)
	inactiveTabStyle = lipgloss.NewStyle().Foreground(colorDim)

	// appStyle is the only spacing applied anywhere in the app: it wraps
	// the entire rendered output (tabs + divider + section content + help)
	// in one padded block. There is no border and no other style in this
	// file adds width/height — see App.contentSize, which is the single
	// place that has to agree with this.
	appStyle = lipgloss.NewStyle().Padding(1, 2)
)
