package ui

import (
	"fmt"
	"io"
	"strings"

	"image/color"

	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
)

// themeColor returns c unless color is disabled, in which case it
// falls back to the terminal's default color (see NoColor in colors.go).
func themeColor(c color.Color) color.Color {
	if NoColor() {
		return lipgloss.NoColor{}
	}
	return c
}

// tableBorderColorStyle is the color-only style for table.Table.BorderStyle;
// the border shape itself is set separately via table.Table.Border.
func tableBorderColorStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(themeColor(lipgloss.Cyan))
}

// BorderStyle returns a themed bordered box style of the given total width
// (including border and padding), used to frame live terminal output such
// as streamed pull progress.
func BorderStyle(width int) lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(themeColor(lipgloss.Cyan)).
		Padding(0, 1).
		Width(width)
}

// Table is a multi-column table renderer, rendered as a single
// square-cornered box via lipgloss/table.
type Table struct {
	w       io.Writer
	headers []string
	rows    []tableRow

	// pendingSeparator is consumed by the next AddRow call.
	pendingSeparator bool
}

type tableRow struct {
	cols   []string
	colors []func(string) string

	// sepBefore draws a divider above this row in Render.
	sepBefore bool
}

func NewTable(w io.Writer, headers ...string) *Table {
	return &Table{w: w, headers: headers}
}

// AddRow adds a row. colors, if non-nil, supplies one coloring function
// per column; a nil entry leaves that cell uncolored.
func (t *Table) AddRow(cols []string, colors []func(string) string) {
	t.rows = append(t.rows, tableRow{cols: cols, colors: colors, sepBefore: t.pendingSeparator})
	t.pendingSeparator = false
}

// AddSeparator marks the next AddRow as the start of a new group, so
// Render draws a divider above it (table.Table only supports a divider
// between every row or none, not specific rows).
func (t *Table) AddSeparator() {
	t.pendingSeparator = true
}

// Render prints the table to t.w as a single square-cornered box.
func (t *Table) Render() {
	tbl := table.New().
		Border(lipgloss.NormalBorder()).
		BorderStyle(tableBorderColorStyle()).
		StyleFunc(func(row, col int) lipgloss.Style {
			return lipgloss.NewStyle().Padding(0, 1)
		})

	headerCells := make([]string, len(t.headers))
	for i, h := range t.headers {
		headerCells[i] = Yellow(h)
	}
	tbl.Headers(headerCells...)

	for _, r := range t.rows {
		cells := make([]string, len(r.cols))
		for i, c := range r.cols {
			if i < len(r.colors) && r.colors[i] != nil {
				cells[i] = r.colors[i](c)
			} else {
				cells[i] = c
			}
		}
		tbl.Row(cells...)
	}

	// Splice a divider (reusing the rendered header-separator line, so
	// it matches the real column widths) before each sepBefore row.
	lines := strings.Split(tbl.String(), "\n")
	out := append([]string{}, lines[:3]...)
	for i, r := range t.rows {
		if r.sepBefore && i > 0 {
			out = append(out, lines[2])
		}
		out = append(out, lines[3+i])
	}
	out = append(out, lines[len(lines)-1])

	fmt.Fprintln(t.w, strings.Join(out, "\n"))
}
