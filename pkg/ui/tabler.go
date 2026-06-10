package ui

import (
	"fmt"
	"io"
	"strings"
)

const colGap = 3

//nolint:gochecknoglobals // box-drawing character set is effectively a constant
var (
	topLeft     = "╭"
	topRight    = "╮"
	bottomLeft  = "╰"
	bottomRight = "╯"
	horizontal  = "─"
	vertical    = "│"
	middleLeft  = "├"
	middleRight = "┤"
)

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

func hline(width int, left, right string) string {
	return Cyan(left + strings.Repeat(horizontal, width-2) + right)
}

// Table is a multi-column table renderer.
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

func (t *Table) AddRow(cols []string, colors []func(string) string) {
	t.rows = append(t.rows, tableRow{cols: cols, colors: colors})
}

//nolint:gocognit // ignore cognitive complexity here, Render orchestrates table layout and drawing
func (t *Table) Render() {
	// Calculate column widths from content
	widths := make([]int, len(t.headers))
	for i, h := range t.headers {
		widths[i] = runeLen(h)
	}
	for _, r := range t.rows {
		for i, c := range r.cols {
			if i < len(widths) && runeLen(c) > widths[i] {
				widths[i] = runeLen(c)
			}
		}
	}

	// Calculate total table width from column widths
	inner := 1 // leading space after │
	for i, w := range widths {
		inner += w
		if i < len(widths)-1 {
			inner += colGap
		}
	}
	inner++                 // trailing space before │
	tableWidth := inner + 2 // +2 for the two │ borders

	renderRow := func(cols []string, colored bool) string {
		var sb strings.Builder
		sb.WriteString(Cyan(vertical))
		sb.WriteString(" ")
		for i, col := range cols {
			w := 0
			if i < len(widths) {
				w = widths[i]
			}
			if colored {
				sb.WriteString(Yellow(padRight(col, w)))
			} else {
				sb.WriteString(padRight(col, w))
			}
			if i < len(cols)-1 {
				sb.WriteString(strings.Repeat(" ", colGap))
			}
		}
		sb.WriteString(" ")
		sb.WriteString(Cyan(vertical))
		return sb.String()
	}

	fmt.Fprintln(t.w, hline(tableWidth, topLeft, topRight))
	fmt.Fprintln(t.w, renderRow(t.headers, true))
	fmt.Fprintln(t.w, hline(tableWidth, middleLeft, middleRight))
	for _, r := range t.rows {
		var sb strings.Builder
		sb.WriteString(Cyan(vertical))
		sb.WriteString(" ")
		for i, col := range r.cols {
			w := 0
			if i < len(widths) {
				w = widths[i]
			}
			cell := padRight(col, w)
			if i < len(r.colors) && r.colors[i] != nil {
				cell = r.colors[i](cell)
			}
			sb.WriteString(cell)
			if i < len(r.cols)-1 {
				sb.WriteString(strings.Repeat(" ", colGap))
			}
		}
		sb.WriteString(" ")
		sb.WriteString(Cyan(vertical))
		fmt.Fprintln(t.w, sb.String())
	}
	fmt.Fprintln(t.w, hline(tableWidth, bottomLeft, bottomRight))
}

// Panel is a key-value panel renderer with sections.
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

func PanelRow(key, value string) panelRow {
	return panelRow{key: key, value: value}
}

func (p *Panel) Render() {
	// Calculate key column width and table width from content
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

	// inner = │ + space + key + gap + value + space + │
	tableWidth := 1 + 1 + keyWidth + colGap + valueWidth + 1 + 1

	renderKV := func(key, value string) string {
		var sb strings.Builder
		sb.WriteString(Cyan(vertical))
		sb.WriteString(" ")
		sb.WriteString(Teal(padRight(key, keyWidth)))
		sb.WriteString(strings.Repeat(" ", colGap))
		sb.WriteString(Dim(padRight(value, valueWidth)))
		sb.WriteString(" ")
		sb.WriteString(Cyan(vertical))
		return sb.String()
	}

	renderSection := func(title string) string {
		label := "▸ " + title + ":"
		colored := Yellow(label)
		// pad using visible width only
		padding := strings.Repeat(" ", keyWidth+colGap+valueWidth-runeLen(label))
		return Cyan(vertical) + " " + colored + padding + " " + Cyan(vertical)
	}

	//nolint:forbidigo // writing to an io.Writer, not stdout directly
	fmt.Fprintln(p.w, hline(tableWidth, topLeft, topRight))
	for i, s := range p.sections {
		if i > 0 {
			//nolint:forbidigo // writing to an io.Writer, not stdout directly
			fmt.Fprintln(p.w, hline(tableWidth, middleLeft, middleRight))
		}
		//nolint:forbidigo // writing to an io.Writer, not stdout directly
		fmt.Fprintln(p.w, renderSection(s.title))
		for _, r := range s.rows {
			//nolint:forbidigo // writing to an io.Writer, not stdout directly
			fmt.Fprintln(p.w, renderKV(r.key, r.value))
		}
	}
	//nolint:forbidigo // writing to an io.Writer, not stdout directly
	fmt.Fprintln(p.w, hline(tableWidth, bottomLeft, bottomRight))
}
