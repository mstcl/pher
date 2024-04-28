// Parses markdown to html
package parse

import (
	"bytes"
	"fmt"

	"github.com/mstcl/pher.git/internal/frontmatter"
	"github.com/mstcl/pher.git/internal/toc"
	"github.com/mstcl/pher.git/internal/wikilink"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
	"go.abhg.dev/goldmark/anchor"
)

// Allowed frontmatter in unmarshalled YAML.
//
// # Default values
//
// * Pinned: false
//
// * Unlisted: false
//
// * ShowHeader: true
//
// * Layout: "list"
//
// * Draft: false
//
// * TOC: false
type Metadata struct {
	Title       string   `yaml:"title"`
	Description string   `yaml:"description"`
	Tags        []string `yaml:"tags"`
	Date        string   `yaml:"date"`
	DateUpdated string   `yaml:"dateUpdated"`
	Pinned      bool     `yaml:"pinned"`
	Unlisted    bool     `yaml:"unlisted"`
	Draft       bool     `yaml:"draft"`
	TOC         bool     `yaml:"toc"`
	ShowHeader  bool     `yaml:"showHeader"`
	Layout      string   `yaml:"layout"`
}

func DefaultMetadata() Metadata {
	return Metadata{
		Pinned:     false,
		Unlisted:   false,
		ShowHeader: true,
		Layout:     "list",
		Draft:      false,
		TOC:        false,
	}
}

// Custom texter for heading anchors.
type customTexter struct{}

// Custom Texter function to hide anchor for level 1 headings.
func (*customTexter) AnchorText(h *anchor.HeaderInfo) []byte {
	if h.Level == 1 {
		return nil
	}
	return []byte("#")
}

// Convert source b with renderer r to give html and Metadata.
//
// Requires r to have the fronmatter extension.
func convert(b []byte, r goldmark.Markdown) ([]byte, Metadata, error) {
	w := new(bytes.Buffer)

	// Get context
	ctx := parser.NewContext()

	// err := r.Renderer().Render(w, b, p)
	err := r.Convert(b, w, parser.WithContext(ctx))
	if err != nil {
		return nil,
			Metadata{},
			fmt.Errorf("converting to markdown: %w", err)
	}

	md := DefaultMetadata()

	// Decode frontmatter
	d := frontmatter.Get(ctx)
	err = d.Decode(&md)
	if err != nil {
		return nil,
			Metadata{},
			fmt.Errorf("decoding frontmatter: %w", err)
	}

	return w.Bytes(), md, nil
}

// Parse metadata (frontmatter) from source b.
func ParseMetadata(b []byte) Metadata {
	r := goldmark.New(goldmark.WithExtensions(&frontmatter.Extender{}))
	_, md, err := convert(b, r)
	if err != nil {
		_ = fmt.Errorf("convert markdown: %w", err)
	}
	return md
}

// Is the file a draft?
func IsDraft(md Metadata) bool {
	return md.Draft
}

// Read in b and parse body.
// Frontmatter is reprocessed to strip it.
func ParseSource(b []byte, isToc bool, isHighlight bool) ([]byte, error) {
	ext := []goldmark.Extender{
		&anchor.Extender{
			Texter: &customTexter{},
			Attributer: anchor.Attributes{
				"class": "h-anchor",
			},
			Position: anchor.Before,
		},
		&wikilink.Extender{},
		&frontmatter.Extender{},
		extension.GFM,
		extension.Table,
		extension.TaskList,
		extension.Strikethrough,
		extension.Linkify,
		extension.DefinitionList,
		extension.Footnote,
		extension.Typographer,
	}
	if isToc {
		ext = append(ext, &toc.Extender{})
	}
	if isHighlight {
		ext = append(ext, highlighting.NewHighlighting(
			highlighting.WithStyle("friendly"),
		))
	}
	r := goldmark.New(
		goldmark.WithExtensions(ext...),
		goldmark.WithParserOptions(parser.WithAutoHeadingID()),
		goldmark.WithRendererOptions(html.WithUnsafe()),
	)
	body, _, err := convert(b, r)
	if err != nil {
		return nil,
			fmt.Errorf("convert markdown: %w", err)
	}

	return body, nil
}

// Parse b to find all wikilinks within the document.
func ParseWikilinks(b []byte) ([]string, error) {
	var links []string
	t := text.NewReader(b)
	wp := wikilink.Parser{}

	// Construct custom parser with lighter options
	wikilinkParser := util.Prioritized(&wp, 199)
	paragraphParser := util.Prioritized(parser.NewParagraphParser(), 1000)
	listParser := util.Prioritized(parser.NewListParser(), 300)
	listItemParser := util.Prioritized(parser.NewListItemParser(), 400)
	BlockquoteParser := util.Prioritized(parser.NewBlockquoteParser(), 800)
	z := parser.NewParser(parser.WithBlockParsers([]util.PrioritizedValue{
		paragraphParser,
		listItemParser,
		listParser,
		BlockquoteParser}...),
		parser.WithInlineParsers(wikilinkParser),
		parser.WithParagraphTransformers(),
	)

	// Parse and walk through nodes to find wikilinks
	p := z.Parse(t)
	walker := func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch n.Kind() {
		case wikilink.Kind:
		default:
			return ast.WalkContinue, nil
		}
		if l, ok := n.(*wikilink.Node); ok {
			t := string(l.Target)
			links = append(links, t)
		}
		return ast.WalkContinue, nil
	}
	err := ast.Walk(p, walker)
	if err != nil {
		return nil, fmt.Errorf("extracting links: %w", err)
	}
	return links, nil
}

// Parse b to find all other links within the document.
func ParseInternalLinks(b []byte) ([]string, []string, error) {
	var ilinks []string
	var blinks []string
	t := text.NewReader(b)
	wp := wikilink.Parser{}

	// Construct custom parser with lighter options
	wikilinkParser := util.Prioritized(&wp, 199)
	linkParser := util.Prioritized(parser.NewLinkParser(), 200)
	paragraphParser := util.Prioritized(parser.NewParagraphParser(), 1000)
	listParser := util.Prioritized(parser.NewListParser(), 300)
	listItemParser := util.Prioritized(parser.NewListItemParser(), 400)
	BlockquoteParser := util.Prioritized(parser.NewBlockquoteParser(), 800)
	z := parser.NewParser(parser.WithBlockParsers([]util.PrioritizedValue{
		paragraphParser,
		listItemParser,
		listParser,
		BlockquoteParser}...),
		parser.WithInlineParsers(
			[]util.PrioritizedValue{
				wikilinkParser,
				linkParser}...),
		parser.WithParagraphTransformers(),
	)

	// Parse and walk through nodes to find wikilinks
	p := z.Parse(t)
	walker := func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch n := n.(type) {
		case *ast.Image:
			t := string(n.Destination)
			ilinks = append(ilinks, t)
		case *wikilink.Node:
			t := string(n.Target)
			blinks = append(blinks, t)
		default:
			return ast.WalkContinue, nil
		}
		return ast.WalkContinue, nil
	}
	err := ast.Walk(p, walker)
	if err != nil {
		return nil, nil, fmt.Errorf("extracting images: %w", err)
	}
	return blinks, ilinks, nil
}
