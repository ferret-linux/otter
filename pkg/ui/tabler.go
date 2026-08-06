package ui

import (
	"fmt"
	"io"
	"strings"

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

	// pendingSeparator is set by AddSeparator and consumed by the next
	// AddRow call, marking that row as the start of a new group.
	pendingSeparator bool
}

type tableRow struct {
	cols   []string
	colors []func(string) string

	// sepBefore marks this row as the first row of a new group: Render
	// draws a horizontal divider above it, reusing the table's own
	// header-separator line so the divider lines up with the real
	// column widths.
	sepBefore bool
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
	t.rows = append(t.rows, tableRow{cols: cols, colors: colors, sepBefore: t.pendingSeparator})
	t.pendingSeparator = false
}

// AddSeparator marks the next row added via AddRow as the start of a new
// group, so Render draws a horizontal divider above it. Useful for
// grouping related rows (e.g. by section) within a single table, since
// lipgloss's table.Table only supports a divider between every row
// (BorderRow) or none at all, not a divider between specific rows.
func (t *Table) AddSeparator() {
	t.pendingSeparator = true
}

// Render prints the table to t.w as a single square-cornered box. If any
// row was marked via AddSeparator, a horizontal divider is drawn above
// it by reusing the table's own rendered header-separator line, so the
// divider's column boundaries always match the real (auto-sized) column
// widths instead of being computed separately.
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

	// table.Table only supports a divider between every row (BorderRow)
	// or none — nothing in between. Rows marked via AddSeparator get one
	// spliced in here, reusing the table's own header-separator line
	// (lines[2]) so it matches the real column widths. Assumes the
	// default top border + header, both always set above.
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
