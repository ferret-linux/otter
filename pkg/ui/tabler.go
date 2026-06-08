package ui

import (
	"fmt"
	"io"
	"strings"
)

const tableWidth = 63

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

func hline(left, right string) string {
	return left + strings.Repeat(horizontal, tableWidth-2) + right
}

// Table is a multi-column table renderer.
type Table struct {
	w       io.Writer
	headers []string
	rows    []tableRow
}

type tableRow struct {
	cols  []string
	color func(string) string
}

func NewTable(w io.Writer, headers ...string) *Table {
	return &Table{w: w, headers: headers}
}

func (t *Table) AddRow(colorFn func(string) string, cols ...string) {
	t.rows = append(t.rows, tableRow{cols: cols, color: colorFn})
}

func (t *Table) Render() {
	// Calculate column widths
	widths := make([]int, len(t.headers))
	for i, h := range t.headers {
		widths[i] = len(h)
	}
	for _, r := range t.rows {
		for i, c := range r.cols {
			if i < len(widths) && len(c) > widths[i] {
				widths[i] = len(c)
			}
		}
	}

	pad := func(s string, w int) string {
		if len(s) >= w {
			return s
		}
		return s + strings.Repeat(" ", w-len(s))
	}

	renderRow := func(cols []string) string {
		var sb strings.Builder
		sb.WriteString(vertical)
		sb.WriteString(" ")
		for i, h := range cols {
			sb.WriteString(pad(h, widths[i]))
			if i < len(cols)-1 {
				sb.WriteString("   ")
			}
		}
		// pad to table width
		content := sb.String()
		remaining := tableWidth - 1 - len([]rune(content))
		if remaining > 0 {
			content += strings.Repeat(" ", remaining)
		}
		return content + vertical
	}

	//nolint:forbidigo
	fmt.Fprintln(t.w, hline(topLeft, topRight))
	//nolint:forbidigo
	fmt.Fprintln(t.w, Bold(renderRow(t.headers)))
	//nolint:forbidigo
	fmt.Fprintln(t.w, hline(middleLeft, middleRight))
	for _, r := range t.rows {
		line := renderRow(r.cols)
		if r.color != nil {
			line = r.color(line)
		}
		//nolint:forbidigo
		fmt.Fprintln(t.w, line)
	}
	//nolint:forbidigo
	fmt.Fprintln(t.w, hline(bottomLeft, bottomRight))
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
	pad := func(s string, w int) string {
		if len(s) >= w {
			return s
		}
		return s + strings.Repeat(" ", w-len(s))
	}

	renderKV := func(key, value string) string {
		content := vertical + " " + pad(key, 12) + " " + value
		runes := []rune(content)
		remaining := tableWidth - 1 - len(runes)
		if remaining > 0 {
			content += strings.Repeat(" ", remaining)
		}
		return content + vertical
	}

	renderSection := func(title string) string {
		content := vertical + " " + Teal("▸ "+title+":")
		// Teal adds escape codes, calculate visible length
		visible := vertical + "  ▸ " + title + ":"
		remaining := tableWidth - 1 - len([]rune(visible))
		if remaining > 0 {
			content += strings.Repeat(" ", remaining)
		}
		return content + vertical
	}

	//nolint:forbidigo
	fmt.Fprintln(p.w, hline(topLeft, topRight))
	for i, s := range p.sections {
		if i > 0 {
			//nolint:forbidigo
			fmt.Fprintln(p.w, hline(middleLeft, middleRight))
		}
		//nolint:forbidigo
		fmt.Fprintln(p.w, renderSection(s.title))
		for _, r := range s.rows {
			//nolint:forbidigo
			fmt.Fprintln(p.w, renderKV(r.key, r.value))
		}
	}
	//nolint:forbidigo
	fmt.Fprintln(p.w, hline(bottomLeft, bottomRight))
}
