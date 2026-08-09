package commands

import (
	"embed"
	"fmt"
	"io"
	"io/fs"
	"path"
	"sort"
	"strings"

	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/glamour/v2"
	"charm.land/glamour/v2/styles"
	"charm.land/lipgloss/v2"
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

// treePaneWidth is the fixed width of the left tree pane.
const treePaneWidth = 28

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

// docEntry is one row of the docs tree: either a directory heading (not
// selectable) or a markdown file (selectable, previewable).
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
				name:          c.Name(),
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
			name:          strings.TrimSuffix(c.Name(), ".md"),
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
type docTreeDelegate struct{}

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
	// glyphs. Directory headings never highlight at all, since they're
	// inert (see skipDirEntries): the cursor can't actually stop on one,
	// so a highlighted dir row would be a visual lie.
	nameStyle := unselectedTreeStyle
	if index == m.Index() && !e.isDir {
		nameStyle = selectedTreeStyle
	}
	if e.isDir {
		nameStyle = nameStyle.Bold(true)
	}

	fmt.Fprint(w, unselectedTreeStyle.Render(docTreePrefix(e))+nameStyle.Render(e.name))
}

// skipDirEntries nudges l's cursor off any directory-heading row it may be
// sitting on, moving further in the given direction until it lands on a
// file row (or exhausts the list). Directory rows are pure headings — see
// docTreePrefix and docTreeDelegate.Render — so the cursor should never
// rest on one.
func skipDirEntries(l list.Model, entries []docEntry, movingDown bool) list.Model {
	for len(entries) > 0 && entries[l.Index()].isDir {
		prev := l.Index()
		if movingDown {
			l.CursorDown()
		} else {
			l.CursorUp()
		}
		if l.Index() == prev {
			// Hit the end of the list without finding a file row; stop
			// rather than loop forever.
			break
		}
	}
	return l
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
	tree     list.Model
	content  viewport.Model
	entries  []docEntry
	renderer *glamour.TermRenderer
	focus    docFocusPane

	err error

	width, height int
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

	items := make([]list.Item, len(entries))
	for i, e := range entries {
		items[i] = docTreeItem{entry: e}
	}

	l := list.New(items, docTreeDelegate{}, 0, 0)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)
	l.SetShowPagination(false)
	l.SetFilteringEnabled(false)
	// The list defaults to selecting index 0, which may be a directory
	// heading (e.g. "commands" sorts first) — nudge forward onto the
	// first real file so the initial selection is never a dir row.
	l = skipDirEntries(l, entries, true)

	renderer, err := glamour.NewTermRenderer(glamour.WithStyles(styles.DarkStyleConfig))
	if err != nil {
		return DocumentationModel{}, fmt.Errorf("failed to build markdown renderer: %w", err)
	}

	vp := viewport.New()

	m := DocumentationModel{
		tree:     l,
		content:  vp,
		entries:  entries,
		renderer: renderer,
	}
	return m.loadSelected(), nil
}

func (m DocumentationModel) Init() tea.Cmd { return nil }

func (m DocumentationModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "left", "right":
			m.focus = toggleDocFocus(m.focus)
			return m, nil
		case "enter":
			if m.focus == docFocusTree {
				m.focus = docFocusContent
			}
			return m, nil
		}

	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m.setSize(msg.Width, msg.Height), nil
	}

	if m.focus == docFocusTree {
		prevIndex := m.tree.Index()
		var cmd tea.Cmd
		m.tree, cmd = m.tree.Update(msg)
		if newIndex := m.tree.Index(); newIndex != prevIndex {
			m.tree = skipDirEntries(m.tree, m.entries, newIndex > prevIndex)
			m = m.loadSelected()
		}
		return m, cmd
	}

	var cmd tea.Cmd
	m.content, cmd = m.content.Update(msg)
	return m, cmd
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
		m.content.SetContent("(directory — select a file to view it)")
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

	contentWidth := width - treePaneWidth
	if contentWidth < 0 {
		contentWidth = 0
	}
	paneHeight := height - helpRows
	if paneHeight < 0 {
		paneHeight = 0
	}

	m.tree.SetSize(treePaneWidth-docPaneFrameSize(), paneHeight-docPaneFrameSize())
	m.content.SetWidth(contentWidth - docPaneFrameSize())
	m.content.SetHeight(paneHeight - docPaneFrameSize())

	renderWidth := contentWidth - docPaneFrameSize() - glamourGutter
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
func docPaneFrameSize() int { return 2 }

func (m DocumentationModel) View() tea.View {
	if m.err != nil {
		return tea.NewView(fmt.Sprintf("error: %s", m.err))
	}

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
	help := helpStyle.Render("↑/↓ move · ←/→ switch pane · ↵ open · q quit")

	v := tea.NewView(body + "\n" + help)
	v.AltScreen = true
	return v
}

func toggleDocFocus(f docFocusPane) docFocusPane {
	if f == docFocusTree {
		return docFocusContent
	}
	return docFocusTree
}
