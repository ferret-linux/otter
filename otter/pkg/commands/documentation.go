package commands

import (
	"embed"
	"fmt"
	"io"
	"io/fs"
	"path"
	"regexp"
	"sort"
	"strings"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/glamour/v2"
	"charm.land/glamour/v2/styles"
	"charm.land/lipgloss/v2"

	"github.com/charmbracelet/x/ansi"
)

// documentationFS is the embedded contents of assets/docs (the repository's
// docs/ directory, relocated under pkg/commands/assets/ so it can be
// embedded directly alongside the command that serves it), rooted at
// "assets/docs" (i.e. paths look like "assets/docs/getting-started.md",
// "assets/docs/commands/otter-list.md").
//
//go:embed assets/docs
var documentationFS embed.FS

// glamourGutter accounts for the left-side margin glamour applies to
// rendered output, on top of the viewport's own border/padding — see
// bubbletea's examples/glamour, which calls out this exact adjustment.
const glamourGutter = 2

// treePaneFraction is the fraction of the terminal width the left tree
// pane is always given. The split is fixed (25%/75%) so both panes always
// add up to the full terminal width — content always reaches the right
// edge, and any tree entry too long to fit is ellipsized by the delegate
// rather than resizing the panes. This is deliberately non-dynamic: no
// measurement, no resizing on expand/collapse.
const treePaneFraction = 0.25

// minTreeWidth ensures the tree pane never collapses below a usable width
// on extremely narrow terminals, where 25% of the window could otherwise
// be a few cells wide.
const minTreeWidth = 12

//nolint:gochecknoglobals // package-level styles are the idiomatic lipgloss pattern
var (
	// Palette mirrors otter's existing house colors from pkg/ui/colors.go
	// and internal/cli/print-file.go's help-screen colorSlots, so the
	// docs viewer reads as the same app rather than a one-off theme.
	colorTeal   = lipgloss.Color("14") // bright cyan — matches ui.Teal / print-file {1}
	colorDim    = lipgloss.Color("7")  // matches ui.Dim / print-file {5}
	colorBg     = lipgloss.Color("0")
	colorGreen  = lipgloss.Color("2") // matches ui.Green / print-file {0} — command/dir names
	colorCyan   = lipgloss.Color("6") // matches ui.Cyan / print-file {2} — box borders
	colorYellow = lipgloss.Color("3") // matches ui.Yellow / print-file {3} — headers
	colorBlue   = lipgloss.Color("4") // matches print-file {4} — flags/extras

	// selectedTreeStyle highlights the focused row with a solid
	// background fill (not a left-edge bar — see treeDelegate.Render for
	// why the bar glyph specifically would clash here).
	selectedTreeStyle   = lipgloss.NewStyle().Background(colorTeal).Foreground(colorBg)
	unselectedTreeStyle = lipgloss.NewStyle().Foreground(colorDim)

	// dirNameStyle colors directory names/expand-indicators, kept apart
	// from unselectedTreeStyle so folders read as more prominent than
	// files, matching ordinary file-tree conventions.
	dirNameStyle = lipgloss.NewStyle().Bold(true).Foreground(colorGreen)

	// treeConnectorStyle colors just the ├──/└──/│ connector glyphs
	// drawn by docTreePrefix, kept separate from unselectedTreeStyle so
	// row labels/icons aren't affected.
	treeConnectorStyle = lipgloss.NewStyle().Foreground(colorCyan)

	// folderIconStyle/fileIconStyle color just the 🗀/🗟 glyphs in
	// docTreeDelegate.Render, independently of nameStyle so the icon's
	// color doesn't change with selection the way the name text does.
	folderIconStyle = lipgloss.NewStyle().Foreground(colorYellow)
	fileIconStyle   = lipgloss.NewStyle().Foreground(colorBlue)

	paneBorderStyle        = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(colorDim)
	focusedPaneBorderStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(colorTeal)

	helpStyle = lipgloss.NewStyle().Foreground(colorDim)
)

// docKeyMap defines the keybindings for the documentation viewer. Keeping
// bindings here (rather than matching raw key strings in Update) means the
// help line below is generated from the same source of truth that Update
// actually checks against — the two can't drift apart.
type docKeyMap struct {
	Up         key.Binding // covers both up and down movement, labeled "↑/↓ move"
	SwitchPane key.Binding
	Open       key.Binding
	Quit       key.Binding
}

// ShortHelp satisfies help.KeyMap.
func (k docKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.SwitchPane, k.Open, k.Quit}
}

// FullHelp satisfies help.KeyMap. The docs viewer has no expanded help
// view distinct from the short one, so both return the same bindings.
func (k docKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{k.ShortHelp()}
}

//nolint:gochecknoglobals // keymap is the idiomatic bubbles/key pattern
var docKeys = docKeyMap{
	Up:         key.NewBinding(key.WithKeys("up", "down"), key.WithHelp("↑/↓", "move")),
	SwitchPane: key.NewBinding(key.WithKeys("left", "right"), key.WithHelp("←/→", "switch pane")),
	Open:       key.NewBinding(key.WithKeys("enter"), key.WithHelp("↵", "open/toggle")),
	Quit:       key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
}

// docEntry is one row of the docs tree: either a directory (selectable —
// pressing enter toggles it collapsed/expanded) or a markdown file
// (selectable, previewable).
type docEntry struct {
	// name is the display label — the file/dir name with any .md
	// extension stripped, e.g. "getting-started" or "commands".
	name string

	// embedPath is the full path within documentationFS, e.g.
	// "assets/docs/getting-started.md", used to load content on selection.
	embedPath string

	// depth is 0 for top-level assets/docs entries, 1 for entries inside
	// one subdirectory, and so on. assets/docs is currently only two
	// levels deep, but this isn't assumed anywhere below.
	depth int

	isDir bool

	// isLastAtDepth reports whether this entry is the last child among
	// its siblings at each ancestor depth, indexed by depth (0-based).
	// len(isLastAtDepth) == depth+1. This drives which connector glyph
	// (├── vs └──) and which continuation glyph (│ vs blank) each
	// ancestor column renders at this row — the same bookkeeping the
	// classic Unix `tree` command does.
	isLastAtDepth []bool
}

// FilterValue satisfies list.Item. The docs tree has no filtering enabled,
// but list.Item requires this method.
func (e docEntry) FilterValue() string { return e.name }

// buildDocsTree walks documentationFS and returns a flat, depth-ordered
// slice of entries suitable for rendering as a connector-drawn tree.
// Directories and files are sorted together alphabetically within each
// directory (matching the layout the tree is meant to mirror), with
// directories walked depth-first so a directory's children immediately
// follow it.
func buildDocsTree() ([]docEntry, error) {
	const root = "assets/docs"

	children, err := docChildrenOf(root)
	if err != nil {
		return nil, err
	}

	var out []docEntry
	walkDocs(root, 0, nil, children, &out)
	return out, nil
}

// srNumberPattern matches the leading "SR number" prefix on doc file and
// directory names (e.g. "01." in "01.otter.md", "08." in "08.commands")
// that exists only to control sort/tree order — see docChildrenOf — and
// is never meant to be shown to the user.
var srNumberPattern = regexp.MustCompile(`^\d+\.`)

// stripSRPrefix removes a leading SR number prefix from name, if present,
// so the ordering-only prefix never leaks into the displayed label.
func stripSRPrefix(name string) string {
	return srNumberPattern.ReplaceAllString(name, "")
}

// docChildrenOf returns the sorted, immediate directory entries of dir
// within documentationFS.
func docChildrenOf(dir string) ([]fs.DirEntry, error) {
	children, err := fs.ReadDir(documentationFS, dir)
	if err != nil {
		return nil, err
	}
	sort.Slice(children, func(i, j int) bool {
		return children[i].Name() < children[j].Name()
	})
	return children, nil
}

// walkDocs appends dir's children to out, recursing into subdirectories
// depth-first. parentIsLast tracks, for each ancestor depth, whether that
// ancestor was the last child of its own parent — new entries extend it
// by one element for their own position among dir's children.
func walkDocs(dir string, depth int, parentIsLast []bool, children []fs.DirEntry, out *[]docEntry) {
	for i, c := range children {
		isLast := i == len(children)-1
		isLastAtDepth := append(append([]bool(nil), parentIsLast...), isLast)

		if c.IsDir() {
			*out = append(*out, docEntry{
				name:          stripSRPrefix(c.Name()),
				embedPath:     path.Join(dir, c.Name()),
				depth:         depth,
				isDir:         true,
				isLastAtDepth: isLastAtDepth,
			})
			// Errors reading a subdirectory are treated as "no children"
			// rather than failing the whole tree — buildDocsTree's caller
			// (NewDocumentationModel) already surfaces a hard failure if
			// assets/docs itself can't be read at all, which is the only
			// case that should stop the command outright.
			if grandchildren, err := docChildrenOf(path.Join(dir, c.Name())); err == nil {
				walkDocs(path.Join(dir, c.Name()), depth+1, isLastAtDepth, grandchildren, out)
			}
			continue
		}

		if !strings.HasSuffix(c.Name(), ".md") {
			continue
		}
		*out = append(*out, docEntry{
			name:          stripSRPrefix(strings.TrimSuffix(c.Name(), ".md")),
			embedPath:     path.Join(dir, c.Name()),
			depth:         depth,
			isDir:         false,
			isLastAtDepth: isLastAtDepth,
		})
	}
}

// docTreePrefix returns the connector glyphs (ancestor continuation columns
// plus this row's own ├──/└──) that should precede e.name when rendered,
// matching the classic Unix `tree` layout — but only for entries below the
// top level. Depth-0 entries (both the "commands" heading and the flat
// top-level docs) render with no prefix at all, since there's no visible
// root node above them to connect to.
func docTreePrefix(e docEntry) string {
	if e.depth == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString(" ")
	// One column per ancestor depth below the root, then this entry's own
	// connector. Ancestor depth 0 is skipped since depth-0 entries render
	// no visible connector for their own row (see above), so there's no
	// vertical bar to continue down from them.
	for i := 1; i < e.depth; i++ {
		if e.isLastAtDepth[i] {
			b.WriteString("    ")
		} else {
			b.WriteString("│   ")
		}
	}
	if e.isLastAtDepth[e.depth] {
		b.WriteString("└── ")
	} else {
		b.WriteString("├── ")
	}
	return b.String()
}

// docTreeItem adapts a docEntry to list.Item.
type docTreeItem struct {
	entry docEntry
}

func (i docTreeItem) FilterValue() string { return i.entry.FilterValue() }

// docTreeDelegate renders docTreeItem rows with tree connector glyphs
// (├──/└──/│) and highlights the selected row with a solid background
// fill rather than a left-edge bar, since the bar glyph (│) would be
// visually ambiguous next to the tree's own continuation columns.
// collapsed is shared with DocumentationModel.collapsed (the same map,
// not a copy), so a toggle there is immediately visible here without
// having to reconstruct the delegate.
type docTreeDelegate struct {
	collapsed map[string]bool
	// rowWidth is the tree pane's interior width — the number of display
	// cells a fully-rendered row may occupy before the border. The fixed
	// leading pieces (connector prefix, indicator, icon, gap) never
	// change with the entry, so any overrun comes from the name, which is
	// ellipsized to fit. Keeping it exactly inside rowWidth means the
	// border box always matches the width set in setSize, so the panes
	// always sum to the full terminal width.
	rowWidth int
}

func (d docTreeDelegate) Height() int  { return 1 }
func (d docTreeDelegate) Spacing() int { return 0 }

func (d docTreeDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d docTreeDelegate) Render(w io.Writer, m list.Model, index int, it list.Item) {
	ti, ok := it.(docTreeItem)
	if !ok {
		return
	}
	e := ti.entry

	// Highlighting only ever applies to the name itself, not the prefix —
	// otherwise the selection background bleeds over the tree connector
	// glyphs. Directories highlight just like files now that they're
	// selectable rows (enter toggles them collapsed/expanded instead of
	// switching focus to the content pane).
	nameStyle := unselectedTreeStyle
	if e.isDir {
		nameStyle = dirNameStyle
	}
	if index == m.Index() {
		nameStyle = selectedTreeStyle
		if e.isDir {
			nameStyle = nameStyle.Bold(true)
		}
	}

	indicator := ""
	icon := "🗟"
	iconStyle := fileIconStyle
	if e.isDir {
		indicator = "▸ "
		if !d.collapsed[e.embedPath] {
			indicator = "▾ "
		}
		icon = "🗀"
		iconStyle = folderIconStyle
	}
	// On a selected row, match the icon's background to the highlight
	// bar so the icon doesn't leave a gap in the solid fill — its
	// foreground color still stays distinct from the rest of the row.
	if index == m.Index() {
		iconStyle = iconStyle.Background(colorTeal)
	}

	prefix := docTreePrefix(e)
	// Budget is what's left for the name after the fixed leading pieces
	// (connector prefix, expand indicator, icon, and the one-space gap
	// before the name), all measured in display cells so the finished row
	// is exactly comparable to d.rowWidth. Names that don't fit are
	// ellipsized with a trailing "…" rather than pushing the row — and
	// therefore the whole border box — wider than the pane was sized to.
	budget := d.rowWidth - lipgloss.Width(prefix) - lipgloss.Width(indicator) - lipgloss.Width(icon) - 1
	name := e.name
	if budget > 0 {
		name = ansi.Truncate(name, budget, "…")
	} else {
		name = ""
	}

	row := treeConnectorStyle.Render(prefix) +
		nameStyle.Render(indicator) +
		iconStyle.Render(icon) +
		nameStyle.Render(" "+name)

	// list.Model doesn't pad rows to its set width, so left alone the
	// border box would shrink to the widest on-screen row and stop
	// matching the 25% allocation setSize computed — leaving the content
	// pane short of the terminal edge. Pad each row out to the full pane
	// interior. The pad inherits nameStyle, so the selected row's
	// highlight bar runs all the way to the border like a filled row.
	if pad := d.rowWidth - lipgloss.Width(row); pad > 0 {
		row += nameStyle.Render(strings.Repeat(" ", pad))
	}

	fmt.Fprint(w, row)
}

// visibleDocEntries returns the subset of entries that should currently be
// rendered, given which directories are collapsed. Collapsing a directory
// hides everything nested under it — every following entry whose depth is
// greater than the directory's own depth — but never hides its siblings or
// ancestors. Nested collapsed directories don't need special-casing: once
// an ancestor's subtree is being skipped, entries inside a collapsed
// descendant are already being skipped too.
func visibleDocEntries(entries []docEntry, collapsed map[string]bool) []docEntry {
	var out []docEntry
	skipBelowDepth := -1
	for _, e := range entries {
		if skipBelowDepth != -1 {
			if e.depth > skipBelowDepth {
				continue
			}
			skipBelowDepth = -1
		}
		out = append(out, e)
		if e.isDir && collapsed[e.embedPath] {
			skipBelowDepth = e.depth
		}
	}
	return out
}

// docTreeItems adapts a slice of docEntry into the []list.Item shape the
// list widget expects.
func docTreeItems(entries []docEntry) []list.Item {
	items := make([]list.Item, len(entries))
	for i, e := range entries {
		items[i] = docTreeItem{entry: e}
	}
	return items
}

// docFocusPane identifies which pane currently has input focus.
type docFocusPane int

const (
	docFocusTree docFocusPane = iota
	docFocusContent
)

// DocumentationModel is `otter documentation`'s top-level bubbletea model:
// a terminal UI for browsing otter's documentation, with a tree of
// assets/docs on the left and the selected file's rendered markdown on
// the right.
type DocumentationModel struct {
	tree      list.Model
	content   viewport.Model
	entries   []docEntry
	visible   []docEntry
	collapsed map[string]bool
	renderer  *glamour.TermRenderer
	focus     docFocusPane
	help      help.Model

	width, height int
	// treeWidth is the tree pane's box width (interior plus border),
	// computed in setSize as 25% of the terminal width. Used by the mouse
	// handler to decide which pane a click lands on.
	treeWidth int
}

// NewDocumentationModel builds the initial docs UI model, walking the
// embedded assets/docs tree and preparing the markdown renderer. Returns
// an error if assets/docs itself can't be walked (a build-time embedding
// problem, not something a user action can trigger) or if the renderer
// can't be constructed.
func NewDocumentationModel() (DocumentationModel, error) {
	entries, err := buildDocsTree()
	if err != nil {
		return DocumentationModel{}, fmt.Errorf("failed to read embedded docs: %w", err)
	}

	collapsed := map[string]bool{}
	visible := visibleDocEntries(entries, collapsed)

	l := list.New(docTreeItems(visible), docTreeDelegate{collapsed: collapsed, rowWidth: minTreeWidth - docPaneFrameSize}, 0, 0)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)
	l.SetShowPagination(false)
	l.SetFilteringEnabled(false)

	renderer, err := glamour.NewTermRenderer(glamour.WithStyles(styles.DarkStyleConfig))
	if err != nil {
		return DocumentationModel{}, fmt.Errorf("failed to build markdown renderer: %w", err)
	}

	vp := viewport.New()

	h := help.New()
	h.Styles.ShortKey = helpStyle
	h.Styles.ShortDesc = helpStyle
	h.Styles.ShortSeparator = helpStyle

	m := DocumentationModel{
		tree:      l,
		content:   vp,
		entries:   entries,
		visible:   visible,
		collapsed: collapsed,
		renderer:  renderer,
		help:      h,
	}
	return m.loadSelected(), nil
}

func (m DocumentationModel) Init() tea.Cmd { return nil }

func (m DocumentationModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, docKeys.Quit):
			return m, tea.Quit
		case key.Matches(msg, docKeys.SwitchPane):
			m.focus = toggleDocFocus(m.focus)
			return m, nil
		case key.Matches(msg, docKeys.Open):
			if m.focus == docFocusTree {
				if sel, ok := m.tree.SelectedItem().(docTreeItem); ok && sel.entry.isDir {
					return m.toggleCollapsed(sel.entry), nil
				}
				m.focus = docFocusContent
			}
			return m, nil
		}

	case tea.MouseMsg:
		mouse := msg.Mouse()
		if mouse.X < m.treeWidth {
			m.focus = docFocusTree
		} else {
			m.focus = docFocusContent
		}

		if m.focus == docFocusTree {
			switch e := msg.(type) {
			case tea.MouseClickMsg:
				if e.Button == tea.MouseLeft {
					if row := m.treeRowAt(mouse.Y); row != -1 {
						m.tree.Select(row)
						m = m.loadSelected()
						if sel, ok := m.tree.SelectedItem().(docTreeItem); ok && sel.entry.isDir {
							return m.toggleCollapsed(sel.entry), nil
						}
					}
					return m, nil
				}
			case tea.MouseWheelMsg:
				switch e.Button {
				case tea.MouseWheelUp:
					if idx := m.tree.Index() - 1; idx >= 0 {
						m.tree.Select(idx)
					}
				case tea.MouseWheelDown:
					if idx := m.tree.Index() + 1; idx < len(m.visible) {
						m.tree.Select(idx)
					}
				}
				return m.loadSelected(), nil
			}
		}

	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m.setSize(msg.Width, msg.Height), nil
	}

	if m.focus == docFocusTree {
		prevIndex := m.tree.Index()
		var cmd tea.Cmd
		m.tree, cmd = m.tree.Update(msg)
		if m.tree.Index() != prevIndex {
			m = m.loadSelected()
		}
		return m, cmd
	}

	var cmd tea.Cmd
	m.content, cmd = m.content.Update(msg)
	return m, cmd
}

// toggleCollapsed flips dir's collapsed state, rebuilds the tree's visible
// entries and list items to match, and keeps the cursor resting on dir
// itself — so a further enter immediately re-expands it, and the cursor
// never ends up pointing at a row that just became hidden (e.g. a file
// that was selected somewhere inside the subtree being collapsed).
func (m DocumentationModel) toggleCollapsed(dir docEntry) DocumentationModel {
	m.collapsed[dir.embedPath] = !m.collapsed[dir.embedPath]
	m.visible = visibleDocEntries(m.entries, m.collapsed)
	m.tree.SetItems(docTreeItems(m.visible))

	for i, e := range m.visible {
		if e.embedPath == dir.embedPath {
			m.tree.Select(i)
			break
		}
	}

	return m.loadSelected()
}

// treeRowAt maps a mouse Y coordinate (0 at the very top of the rendered
// view) to an index into m.visible, or -1 if y falls outside the tree
// pane's rendered item rows (the border, the help line, past the last
// item on the current page, or a padding row on a partially-filled last
// page). Accounts for the list's own scroll/pagination state — i.e. the
// row rendered at the top of the pane is not always m.visible[0]; it's
// whatever index list.Model's Paginator has scrolled to — by resolving
// the page's starting index the same way list.Model itself does when
// rendering (see bubbles/list.Model.populatedView, which computes
// start, end via Paginator.GetSliceBounds and steps by one delegate row
// per item).
func (m DocumentationModel) treeRowAt(y int) int {
	const treeContentTop = 1 // RoundedBorder's top edge occupies screen row 0
	row := y - treeContentTop
	if row < 0 {
		return -1
	}

	rowHeight := docTreeDelegate{}.Height() + docTreeDelegate{}.Spacing()
	offset := row / rowHeight

	start, end := m.tree.Paginator.GetSliceBounds(len(m.visible))
	idx := start + offset
	if idx < start || idx >= end {
		return -1
	}
	return idx
}

// loadSelected renders the currently-selected tree entry's markdown into
// the content viewport. Directory rows and read errors show an inline
// message in the content pane rather than failing the whole program,
// since either can happen from ordinary navigation (selecting a
// directory heading) rather than a program-level fault.
func (m DocumentationModel) loadSelected() DocumentationModel {
	item, ok := m.tree.SelectedItem().(docTreeItem)
	if !ok {
		return m
	}
	e := item.entry

	if e.isDir {
		m.content.SetContent("(directory — press enter to expand or collapse)")
		return m
	}

	raw, err := documentationFS.ReadFile(e.embedPath)
	if err != nil {
		m.content.SetContent(fmt.Sprintf("failed to read %s: %s", e.embedPath, err))
		return m
	}

	rendered, err := m.renderer.Render(string(raw))
	if err != nil {
		m.content.SetContent(fmt.Sprintf("failed to render %s: %s", e.embedPath, err))
		return m
	}

	m.content.SetContent(rendered)
	return m
}

func (m DocumentationModel) setSize(width, height int) DocumentationModel {
	const helpRows = 1

	m.treeWidth = int(float64(width) * treePaneFraction)
	if m.treeWidth < minTreeWidth {
		m.treeWidth = minTreeWidth
	}
	treeRowWidth := m.treeWidth - docPaneFrameSize
	if treeRowWidth < 0 {
		treeRowWidth = 0
	}

	// Rebuild the delegate with the current row width so Render keeps
	// every row ellipsized inside the tree pane. list.Model has no getter
	// for its delegate, so we reconstruct it carrying collapsed through
	// (the same shared map, so live toggles still show).
	m.tree.SetDelegate(docTreeDelegate{collapsed: m.collapsed, rowWidth: treeRowWidth})

	contentWidth := width - m.treeWidth
	if contentWidth < 0 {
		contentWidth = 0
	}
	paneHeight := height - helpRows
	if paneHeight < 0 {
		paneHeight = 0
	}

	m.tree.SetSize(treeRowWidth, paneHeight-docPaneFrameSize)
	m.content.SetWidth(contentWidth - docPaneFrameSize)
	m.content.SetHeight(paneHeight - docPaneFrameSize)

	renderWidth := contentWidth - docPaneFrameSize - glamourGutter
	if renderWidth < 0 {
		renderWidth = 0
	}
	if renderer, err := glamour.NewTermRenderer(
		glamour.WithStyles(styles.DarkStyleConfig),
		glamour.WithWordWrap(renderWidth),
	); err == nil {
		m.renderer = renderer
	}

	return m.loadSelected()
}

// docPaneFrameSize is the width/height a RoundedBorder adds on each pane —
// both panes use the same border style, so this is one constant rather
// than a per-pane lipgloss.Style.GetHorizontalFrameSize() call.
const docPaneFrameSize = 2

func (m DocumentationModel) View() tea.View {
	treeBorder := paneBorderStyle
	contentBorder := paneBorderStyle
	if m.focus == docFocusTree {
		treeBorder = focusedPaneBorderStyle
	} else {
		contentBorder = focusedPaneBorderStyle
	}

	treePane := treeBorder.Render(m.tree.View())
	contentPane := contentBorder.Render(m.content.View())

	body := lipgloss.JoinHorizontal(lipgloss.Top, treePane, contentPane)
	helpView := m.help.View(docKeys)

	v := tea.NewView(body + "\n" + helpView)
	v.AltScreen = true
	v.MouseMode = tea.MouseModeAllMotion
	return v
}

func toggleDocFocus(f docFocusPane) docFocusPane {
	if f == docFocusTree {
		return docFocusContent
	}
	return docFocusTree
}
