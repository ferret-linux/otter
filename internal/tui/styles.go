package tui

import "charm.land/lipgloss/v2"

// Colors mirror the ANSI palette in pkg/ui/colors.go (red/green/yellow/cyan/
// teal/dim) so the TUI's visual language matches otter's existing CLI
// output. They're ANSI 16/256-color indices rather than hex so they resolve
// to the exact same terminal colors as the \033[31m-style codes in pkg/ui.
//
//nolint:gochecknoglobals // package-level styles/colors are the idiomatic lipgloss pattern; threading them through every model would be a needless architecture change for no behavioral gain
var (
	colorRed    = lipgloss.Color("1")  // matches \033[31m in pkg/ui/colors.go
	colorGreen  = lipgloss.Color("2")  // matches \033[32m
	colorYellow = lipgloss.Color("3")  // matches \033[33m
	colorCyan   = lipgloss.Color("6")  // matches \033[36m
	colorTeal   = lipgloss.Color("14") // matches \033[96m (bright cyan)
	colorDim    = lipgloss.Color("7")  // matches \033[37m
	colorBlack  = lipgloss.Color("0")  // used as foreground on activeTabStyle's fill
)

//nolint:gochecknoglobals // see justification above
var (
	nameStyle    = lipgloss.NewStyle().Foreground(colorTeal)
	imageStyle   = lipgloss.NewStyle().Foreground(colorDim)
	runningStyle = lipgloss.NewStyle().Foreground(colorGreen)
	pausedStyle  = lipgloss.NewStyle().Foreground(colorYellow)
	exitedStyle  = lipgloss.NewStyle().Foreground(colorRed)

	// selectedRowStyle marks the focused row in the container list or
	// actions pane with a colored left-edge bar instead of a prefix glyph
	// (the Glow/lipgloss list convention). unselectedRowStyle pads
	// unselected rows by the same width so text stays aligned when the
	// cursor moves.
	selectedRowStyle = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder(), false, false, false, true).
				BorderForeground(colorCyan).
				PaddingLeft(1)
	unselectedRowStyle = lipgloss.NewStyle().PaddingLeft(2)

	helpStyle = lipgloss.NewStyle().Foreground(colorDim)
	dimStyle  = lipgloss.NewStyle().Foreground(colorDim)

	dividerStyle = lipgloss.NewStyle().Foreground(colorDim)

	actionLabelStyle  = lipgloss.NewStyle().Foreground(colorDim)
	actionActiveStyle = lipgloss.NewStyle().Foreground(colorTeal).Bold(true)
)

//nolint:gochecknoglobals // see justification above
var (
	// activeTabStyle marks the active section with a solid background
	// fill (the table/pokemon-example convention) rather than just bold
	// text, so the current section reads as a distinct block in the tab
	// row. inactiveTabStyle stays plain dim text — no border, no box,
	// nothing that needs to visually line up with a neighboring tab.
	activeTabStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorBlack).
			Background(colorTeal).
			Padding(0, 1)
	inactiveTabStyle = lipgloss.NewStyle().Foreground(colorDim).Padding(0, 1)

	// appStyle is the only spacing applied anywhere in the app: it wraps
	// the entire rendered output (tabs + divider + section content + help)
	// in one padded block. There is no border and no other style in this
	// file adds width/height — see App.contentSize, which is the single
	// place that has to agree with this.
	appStyle = lipgloss.NewStyle().Padding(1, 2)
)
