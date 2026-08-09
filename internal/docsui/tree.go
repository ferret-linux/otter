// Package docsui implements `otter docs`: a terminal UI for browsing
// otter's documentation, with a tree of docs/ on the left and the
// selected file's rendered markdown on the right.
package docsui

import (
	"io/fs"
	"path"
	"sort"
	"strings"

	"github.com/ferret-linux/otter/docsfs"
)

// entry is one row of the docs tree: either a directory heading (not
// selectable) or a markdown file (selectable, previewable).
type entry struct {
	// name is the display label — the file/dir name with any .md
	// extension stripped, e.g. "getting-started" or "commands".
	name string

	// embedPath is the full path within docsfs.FS, e.g.
	// "docs/getting-started.md", used to load content on selection.
	embedPath string

	// depth is 0 for top-level docs/ entries, 1 for entries inside one
	// subdirectory, and so on. docs/ is currently only two levels deep,
	// but this isn't assumed anywhere below.
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
func (e entry) FilterValue() string { return e.name }

// buildTree walks docsfs.FS and returns a flat, depth-ordered slice of
// entries suitable for rendering as a connector-drawn tree. Directories
// and files are sorted together alphabetically within each directory
// (matching the layout the tree is meant to mirror), with directories
// walked depth-first so a directory's children immediately follow it.
func buildTree() ([]entry, error) {
	const root = "docs"

	children, err := childrenOf(root)
	if err != nil {
		return nil, err
	}

	var out []entry
	walk(root, 0, nil, children, &out)
	return out, nil
}

// childrenOf returns the sorted, immediate directory entries of dir
// within docsfs.FS.
func childrenOf(dir string) ([]fs.DirEntry, error) {
	children, err := fs.ReadDir(docsfs.FS, dir)
	if err != nil {
		return nil, err
	}
	sort.Slice(children, func(i, j int) bool {
		return children[i].Name() < children[j].Name()
	})
	return children, nil
}

// walk appends dir's children to out, recursing into subdirectories
// depth-first. parentIsLast tracks, for each ancestor depth, whether that
// ancestor was the last child of its own parent — new entries extend it
// by one element for their own position among dir's children.
func walk(dir string, depth int, parentIsLast []bool, children []fs.DirEntry, out *[]entry) {
	for i, c := range children {
		isLast := i == len(children)-1
		isLastAtDepth := append(append([]bool(nil), parentIsLast...), isLast)

		if c.IsDir() {
			*out = append(*out, entry{
				name:          c.Name(),
				embedPath:     path.Join(dir, c.Name()),
				depth:         depth,
				isDir:         true,
				isLastAtDepth: isLastAtDepth,
			})
			// Errors reading a subdirectory are treated as "no children"
			// rather than failing the whole tree — buildTree's caller
			// (docs.go's Action) already surfaces a hard failure if
			// docs/ itself can't be read at all, which is the only
			// case that should stop the command outright.
			if grandchildren, err := childrenOf(path.Join(dir, c.Name())); err == nil {
				walk(path.Join(dir, c.Name()), depth+1, isLastAtDepth, grandchildren, out)
			}
			continue
		}

		if !strings.HasSuffix(c.Name(), ".md") {
			continue
		}
		*out = append(*out, entry{
			name:          strings.TrimSuffix(c.Name(), ".md"),
			embedPath:     path.Join(dir, c.Name()),
			depth:         depth,
			isDir:         false,
			isLastAtDepth: isLastAtDepth,
		})
	}
}

// treePrefix returns the connector glyphs (ancestor continuation columns
// plus this row's own ├──/└──) that should precede e.name when rendered,
// matching the classic Unix `tree` layout.
func treePrefix(e entry) string {
	var b strings.Builder
	// One column per ancestor depth, then this entry's own connector.
	for i := 0; i < e.depth; i++ {
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
