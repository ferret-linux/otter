package webui

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"
	"strings"

	"github.com/alecthomas/chroma/v2"
	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/util"

	"github.com/ferret-linux/otter/pkg/commands"
)

// docsCodeChromaStyle is the chroma color scheme used for fenced/indented
// code blocks on the Docs page. Code blocks render on a fixed dark
// background regardless of the page's own light/dark theme (see
// style.css's .markdown-body pre rule) — the usual convention for
// documentation code blocks, and the closest visual match to `otter
// documentation`'s glamour dark theme.
const docsCodeChromaStyle = "dracula"

// docsMarkdown is the Docs page's shared markdown-to-HTML converter:
// goldmark with GitHub-flavored-markdown extensions (tables,
// strikethrough, autolinking) plus docsCodeRenderer overriding fenced
// and indented code block rendering with chroma syntax highlighting.
var docsMarkdown = goldmark.New(
	goldmark.WithExtensions(extension.GFM),
	goldmark.WithRendererOptions(
		html.WithXHTML(),
		renderer.WithNodeRenderers(util.Prioritized(docsCodeRenderer{}, 100)),
	),
)

// docsCodeRenderer overrides goldmark's default code block rendering
// (which just HTML-escapes the raw text into a plain <pre><code>) with
// chroma-highlighted HTML. It's registered at priority 100, below
// goldmark's own default html.NewRenderer (registered at priority 1000
// by goldmark.New — see yuin/goldmark's markdown.go), so these two funcs
// take over rendering for their two node kinds specifically while every
// other node kind still falls through to the default renderer.
type docsCodeRenderer struct{}

func (docsCodeRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindFencedCodeBlock, renderFencedCodeBlock)
	reg.Register(ast.KindCodeBlock, renderCodeBlock)
	reg.Register(ast.KindHeading, renderHeading)
}

// renderHeading overrides goldmark's default heading rendering to prepend
// the literal "#"-prefix (e.g. "## ") before the heading text, matching
// how `otter documentation`'s glamour renderer displays headings in the
// terminal instead of relying on size/weight alone.
func renderHeading(w util.BufWriter, source []byte, n ast.Node, entering bool) (ast.WalkStatus, error) {
	node := n.(*ast.Heading)
	if entering {
		fmt.Fprintf(w, "<h%d>", node.Level)
		w.WriteString(strings.Repeat("#", node.Level))
		w.WriteString(" ")
	} else {
		fmt.Fprintf(w, "</h%d>\n", node.Level)
	}
	return ast.WalkContinue, nil
}

func renderFencedCodeBlock(w util.BufWriter, source []byte, n ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	node := n.(*ast.FencedCodeBlock)
	lang := ""
	if l := node.Language(source); l != nil {
		lang = string(l)
	}
	return ast.WalkContinue, writeHighlighted(w, codeBlockText(source, n), lang)
}

func renderCodeBlock(w util.BufWriter, source []byte, n ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	return ast.WalkContinue, writeHighlighted(w, codeBlockText(source, n), "")
}

// codeBlockText reassembles a code block node's raw text from its source
// line segments — goldmark stores code block content as spans into the
// original source buffer rather than as child AST nodes, so there's
// nothing to walk into; renderFencedCodeBlock/renderCodeBlock write the
// whole element on the "entering" pass and do nothing on the way out.
func codeBlockText(source []byte, n ast.Node) string {
	var buf bytes.Buffer
	lines := n.Lines()
	for i := 0; i < lines.Len(); i++ {
		line := lines.At(i)
		buf.Write(line.Value(source))
	}
	return buf.String()
}

// writeHighlighted tokenizes code with chroma — using lang if given,
// otherwise guessing from content, otherwise falling back to
// unhighlighted plain text — and writes the resulting syntax-highlighted
// <pre><code>...</code></pre> HTML to w.
func writeHighlighted(w util.BufWriter, code, lang string) error {
	lexer := lexers.Get(lang)
	if lexer == nil {
		lexer = lexers.Analyse(code)
	}
	if lexer == nil {
		lexer = lexers.Fallback
	}
	lexer = chroma.Coalesce(lexer)

	style := styles.Get(docsCodeChromaStyle)
	if style == nil {
		style = styles.Fallback
	}

	iterator, err := lexer.Tokenise(nil, code)
	if err != nil {
		return err
	}

	formatter := chromahtml.New(chromahtml.WithLineNumbers(false))
	return formatter.Format(w, style, iterator)
}

// renderDoc reads and renders the markdown file at path (a
// commands.DocEntry.Path) to HTML, wrapped as template.HTML so
// html/template embeds it verbatim instead of escaping it — safe here
// because the only source is otter's own embedded docs, never
// user-supplied content.
func renderDoc(path string) (template.HTML, error) {
	raw, err := commands.ReadDoc(path)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := docsMarkdown.Convert([]byte(raw), &buf); err != nil {
		return "", err
	}
	return template.HTML(buf.String()), nil //nolint:gosec // trusted embedded content, see doc comment
}

// docsPageData is the Docs page's view model: the full documentation
// tree, rendered as the nav list in templates/docs.html.
type docsPageData struct {
	pageData
	Entries []commands.DocEntry
}

// docsPage handles GET /docs: the Docs page, listing every entry in
// otter's embedded documentation tree with no doc selected yet. A doc's
// rendered content is loaded into #docs-body by docsContentFragment when
// its tree item is clicked — the same htmx-swap-on-click pattern Console
// uses for #picker-detail (see templates/console.html).
func (s *server) docsPage(w http.ResponseWriter, r *http.Request) {
	entries, err := commands.DocsTree()
	if err != nil {
		s.notify("error", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := docsPageData{pageData: pageData{Nav: "docs", PageTitle: "documentation"}, Entries: entries}
	if err := s.templates.ExecuteTemplate(w, "layout", data); err != nil {
		s.notify("error", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// docsContentFragment handles GET /docs/content?path=...: the rendered
// markdown fragment for one doc, htmx-swapped into #docs-body when its
// tree item is clicked. path must match a non-directory entry's Path
// from commands.DocsTree() — anything else (an unrecognized path, or a
// directory's path) gets a 404 instead of being passed straight to
// commands.ReadDoc.
func (s *server) docsContentFragment(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")

	entries, err := commands.DocsTree()
	if err != nil {
		s.notify("error", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	valid := false
	for _, e := range entries {
		if !e.IsDir && e.Path == path {
			valid = true
			break
		}
	}
	if !valid {
		http.NotFound(w, r)
		return
	}

	content, err := renderDoc(path)
	if err != nil {
		s.notify("error", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := s.templates.ExecuteTemplate(w, "docs_body", content); err != nil {
		s.notify("error", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
