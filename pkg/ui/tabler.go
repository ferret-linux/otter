package ui

import (
	"fmt"
	"io"

	"image/color"

	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
)

// themeColor returns c unless color output is disabled (NoColor), in
// which case it returns lipgloss.NoColor{} so the element renders in
// the terminal's plain default color instead. This routes lipgloss
// coloring through the same NO_COLOR/non-tty decision the rest of
// otter's output already uses (see NoColor in colors.go), instead of
// introducing a second, separate color-detection path.
func themeColor(c color.Color) color.Color {
	if NoColor() {
		return lipgloss.NoColor{}
	}
	return c
}

// tableBorderColorStyle is the color-only style passed to
// table.Table.BorderStyle. table.Table.Border sets the border shape
// (lipgloss.NormalBorder(), a plain square-cornered single-line border)
// and table.Table.BorderStyle sets only the color of that shape, so this
// style must not itself declare a border.
func tableBorderColorStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(themeColor(lipgloss.Cyan))
}

// Table is a multi-column table renderer, rendered as a single
// square-cornered box via lipgloss/table.
type Table struct {
	w       io.Writer
	headers []string
	rows    []tableRow
}

type tableRow struct {
	cols   []string
	colors []func(string) string
}

func NewTable(w io.Writer, headers ...string) *Table {
	return &Table{w: w, headers: headers}
}

// AddRow adds a row. colors, if non-nil, supplies one coloring function
// per column (e.g. ui.Yellow, ui.Dim) applied to that column's cell
// text; a nil entry (or a colors slice shorter than cols) leaves that
// cell uncolored. Coloring is applied to the cell text itself (via
// these functions, which already respect NoColor — see colors.go)
// before the cell reaches the table renderer, so it composes with
// lipgloss's own column width/padding logic without a second,
// conflicting color layer.
func (t *Table) AddRow(cols []string, colors []func(string) string) {
	t.rows = append(t.rows, tableRow{cols: cols, colors: colors})
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

	fmt.Fprintln(t.w, tbl.String())
}
