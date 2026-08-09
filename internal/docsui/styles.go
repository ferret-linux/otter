package docsui

import "charm.land/lipgloss/v2"

//nolint:gochecknoglobals // package-level styles are the idiomatic lipgloss pattern
var (
	colorTeal = lipgloss.Color("14")
	colorDim  = lipgloss.Color("7")
	colorBg   = lipgloss.Color("0")

	// selectedTreeStyle highlights the focused row with a solid
	// background fill (not a left-edge bar — see treeDelegate.Render for
	// why the bar glyph specifically would clash here).
	selectedTreeStyle   = lipgloss.NewStyle().Background(colorTeal).Foreground(colorBg)
	unselectedTreeStyle = lipgloss.NewStyle().Foreground(colorDim)

	paneBorderStyle        = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(colorDim)
	focusedPaneBorderStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(colorTeal)

	helpStyle = lipgloss.NewStyle().Foreground(colorDim)
)
