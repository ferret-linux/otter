package docsui

import (
	"fmt"
	"io"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
)

// treeItem adapts an entry to list.Item.
type treeItem struct {
	entry entry
}

func (i treeItem) FilterValue() string { return i.entry.FilterValue() }

// treeDelegate renders treeItem rows with tree connector glyphs
// (├──/└──/│) and highlights the selected row with a solid background
// fill rather than a left-edge bar, since the bar glyph (│) would be
// visually ambiguous next to the tree's own continuation columns.
type treeDelegate struct{}

func (d treeDelegate) Height() int  { return 1 }
func (d treeDelegate) Spacing() int { return 0 }

func (d treeDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d treeDelegate) Render(w io.Writer, m list.Model, index int, it list.Item) {
	ti, ok := it.(treeItem)
	if !ok {
		return
	}
	e := ti.entry

	line := treePrefix(e) + e.name

	style := unselectedTreeStyle
	if index == m.Index() {
		style = selectedTreeStyle
	}
	if e.isDir {
		style = style.Bold(true)
	}

	fmt.Fprint(w, style.Render(line))
}
