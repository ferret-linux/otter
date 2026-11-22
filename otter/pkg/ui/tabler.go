package ui

import (
	"fmt"
	"io"
	"strings"

	"image/color"

	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"

	"github.com/ferret-linux/otter/pkg/ttyutil"
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
		Border(lipgloss.RoundedBorder()).
		BorderForeground(themeColor(lipgloss.Cyan)).
		Padding(0, 1).
		Width(width)
}

type separatorKind uint8

const (
	sepNone separatorKind = iota
	sepGroup
	sepRow
)

// Table is a multi-column table renderer, rendered as a single
// rounded-corner box via lipgloss/table.
type Table struct {
	w       io.Writer
	headers []string
	rows    []tableRow

	// pendingSeparator is consumed by the next AddRow call.
	pendingSeparator separatorKind
}

type tableRow struct {
	cols   []string
	colors []func(string) string

	// sepBefore selects the divider drawn above this row in Render.
	sepBefore separatorKind
}

func NewTable(w io.Writer, headers ...string) *Table {
	return &Table{w: w, headers: headers}
}

// AddRow adds a row. colors, if non-nil, supplies one coloring function
// per column; a nil entry leaves that cell uncolored.
func (t *Table) AddRow(cols []string, colors []func(string) string) {
	t.rows = append(t.rows, tableRow{cols: cols, colors: colors, sepBefore: t.pendingSeparator})
	t.pendingSeparator = sepNone
}

// AddSeparator marks the next AddRow as the start of a new group, so
// Render draws a full divider above it (table.Table only supports a
// divider between every row or none, not specific rows).
func (t *Table) AddSeparator() {
	t.pendingSeparator = sepGroup
}

// AddRowSeparator marks the next AddRow as a continuation row inside the
// current group, so Render draws a light divider above it.
func (t *Table) AddRowSeparator() {
	t.pendingSeparator = sepRow
}

// Render prints the table to t.w as a single rounded-corner box. When the
// final output is wider than a known terminal, it warns and prints nothing
// rather than emitting a broken table.
func (t *Table) Render() {
	lines := t.renderLines()
	if !fitsTerminal(lines, ttyutil.TerminalWidth(t.w)) {
		DefaultLogger.Warn("terminal window is too small to display table")
		return
	}
	fmt.Fprintln(t.w, strings.Join(lines, "\n"))
}

func (t *Table) renderLines() []string {
	tbl := table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(tableBorderColorStyle()).
		StyleFunc(func(_, _ int) lipgloss.Style {
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

	rendered := strings.Split(tbl.String(), "\n")
	if len(rendered) < 4 {
		return rendered
	}

	hardDivider := rendered[2]
	softDivider := softDividerLine(hardDivider)

	out := append([]string{}, rendered[:3]...)
	for i, r := range t.rows {
		if i > 0 {
			switch r.sepBefore {
			case sepGroup:
				out = append(out, hardDivider)
			case sepRow:
				if softDivider != "" {
					out = append(out, softDivider)
				}
			case sepNone:
			}
		}
		out = append(out, rendered[3+i])
	}
	out = append(out, rendered[len(rendered)-1])
	return out
}

// softDividerLine derives a lighter divider from the rendered header divider
// so column widths stay exact. The first column run is blanked to keep the
// group label visually continuous.
func softDividerLine(hard string) string {
	if !strings.Contains(hard, "├") || !strings.Contains(hard, "┼") {
		return ""
	}
	soft := strings.ReplaceAll(hard, "─", "-")
	soft = strings.Replace(soft, "├", "│", 1)
	soft = strings.Replace(soft, "┼", "├", 1)
	idx := strings.Index(soft, "├")
	if idx < 0 {
		return ""
	}
	return strings.ReplaceAll(soft[:idx], "-", " ") + soft[idx:]
}

func fitsTerminal(lines []string, terminalWidth int) bool {
	if terminalWidth <= 0 {
		return true
	}
	for _, line := range lines {
		if lipgloss.Width(line) > terminalWidth {
			return false
		}
	}
	return true
}
