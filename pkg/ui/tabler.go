package ui

import (
	"fmt"
	"io"
	"strings"

	"image/color"

	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
)

const colGap = 3

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

// borderStyle is the shared *lipgloss.Style* used for the outer border
// of Panel (and the color of Table's border, applied separately via
// table.Table.BorderStyle — see Table.Render): a rounded border colored
// with lipgloss's named Cyan constant (one of the standard 16 ANSI
// colors), matching the cyan otter has always used for box borders
// elsewhere. Because it's a named ANSI-16 color rather than a
// hex/truecolor value, the terminal's own theme still determines the
// exact shade — it isn't hardcoded to a fixed RGB.
func borderStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(themeColor(lipgloss.Cyan))
}

// tableBorderColorStyle is the color-only style passed to
// table.Table.BorderStyle. Unlike borderStyle (used for Panel, a plain
// lipgloss.Style with its own border shape), this must NOT itself carry
// a border shape: table.Table.Border sets the shape and
// table.Table.BorderStyle sets only the color of that shape. Passing a
// style that also declares a border (as borderStyle does) here would
// conflict with Table.Border's own shape setting.
func tableBorderColorStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(themeColor(lipgloss.Cyan))
}

func runeLen(s string) int {
	return len([]rune(s))
}

func padRight(s string, w int) string {
	l := runeLen(s)
	if l >= w {
		return s
	}
	return s + strings.Repeat(" ", w-l)
}

// Table is a multi-column table renderer, rendered as a single rounded
// box via lipgloss/table.
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

// Render prints the table to t.w as a single rounded-corner box.
func (t *Table) Render() {
	tbl := table.New().
		Border(lipgloss.RoundedBorder()).
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

// Panel is a key-value panel renderer with sections, rendered as a
// single rounded box.
type Panel struct {
	w        io.Writer
	sections []panelSection
}

type panelSection struct {
	title string
	rows  []panelRow
}

type panelRow struct {
	key   string
	value string
}

func NewPanel(w io.Writer) *Panel {
	return &Panel{w: w}
}

func (p *Panel) AddSection(title string, rows ...panelRow) {
	p.sections = append(p.sections, panelSection{title: title, rows: rows})
}

//nolint:revive // unexported return is intentional; callers always pass the result directly into AddSection
func PanelRow(key, value string) panelRow {
	return panelRow{key: key, value: value}
}

// Render prints the panel to p.w as a single rounded-corner box, with a
// horizontal divider between sections.
func (p *Panel) Render() {
	keyWidth := 0
	for _, s := range p.sections {
		for _, r := range s.rows {
			if runeLen(r.key) > keyWidth {
				keyWidth = runeLen(r.key)
			}
		}
	}

	valueWidth := 0
	for _, s := range p.sections {
		if runeLen("▸ "+s.title+":") > valueWidth {
			valueWidth = runeLen("▸ " + s.title + ":")
		}
		for _, r := range s.rows {
			if runeLen(r.value) > valueWidth {
				valueWidth = runeLen(r.value)
			}
		}
	}

	contentWidth := keyWidth + colGap + valueWidth

	dividerStyle := lipgloss.NewStyle().Foreground(themeColor(lipgloss.Cyan))
	var body strings.Builder
	for i, s := range p.sections {
		if i > 0 {
			body.WriteString(dividerStyle.Render(strings.Repeat("─", contentWidth)))
			body.WriteString("\n")
		}
		label := "▸ " + s.title + ":"
		body.WriteString(Yellow(label))
		body.WriteString(strings.Repeat(" ", contentWidth-runeLen(label)))
		body.WriteString("\n")
		for _, r := range s.rows {
			body.WriteString(Teal(padRight(r.key, keyWidth)))
			body.WriteString(strings.Repeat(" ", colGap))
			body.WriteString(Dim(padRight(r.value, valueWidth)))
			body.WriteString("\n")
		}
	}

	box := borderStyle().
		Padding(0, 1).
		Render(strings.TrimRight(body.String(), "\n"))

	fmt.Fprintln(p.w, box)
}
