// Parses markdown to html
package parse

import (
	"bytes"
	"fmt"

	"github.com/mstcl/pher/internal/frontmatter"
	"github.com/mstcl/pher/internal/toc"
	"github.com/mstcl/pher/internal/wikilink"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
	"go.abhg.dev/goldmark/anchor"
)

type Source struct {
	Body             []byte
	RendersTOC       bool
	RendersHighlight bool
}

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
	Date        string   `yaml:"date"`
	DateUpdated string   `yaml:"dateUpdated"`
	Layout      string   `yaml:"layout"`
	Tags        []string `yaml:"tags"`
	Pinned      bool     `yaml:"pinned"`
	Unlisted    bool     `yaml:"unlisted"`
	Draft       bool     `yaml:"draft"`
	TOC         bool     `yaml:"toc"`
	ShowHeader  bool     `yaml:"showHeader"`
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
func (s *Source) convert(r goldmark.Markdown) ([]byte, Metadata, error) {
	w := new(bytes.Buffer)

	// Get context
	context := parser.NewContext()

	if err := r.Convert(s.Body, w, parser.WithContext(context)); err != nil {
		return nil,
			Metadata{},
			fmt.Errorf("converting to markdown: %w", err)
	}

	md := DefaultMetadata()

	// Decode frontmatter
	d := frontmatter.Get(context)
	if err := d.Decode(&md); err != nil {
		return nil,
			Metadata{},
			fmt.Errorf("decoding frontmatter: %w", err)
	}

	return w.Bytes(), md, nil
}

// Parse metadata (frontmatter) from source b.
func (s *Source) ParseMetadata() (Metadata, error) {
	r := goldmark.New(goldmark.WithExtensions(&frontmatter.Extender{}))
	_, md, err := s.convert(r)
	if err != nil {
		return Metadata{}, fmt.Errorf("convert markdown: %w", err)
	}
	return md, nil
}

// Read in b and parse body.
// Frontmatter is reprocessed to strip it.
func (s *Source) ParseSource() ([]byte, error) {
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
	if s.RendersTOC {
		ext = append(ext, &toc.Extender{})
	}
	if s.RendersHighlight {
		ext = append(ext, highlighting.NewHighlighting(
			highlighting.WithStyle("friendly"),
		))
	}
	r := goldmark.New(
		goldmark.WithExtensions(ext...),
		goldmark.WithParserOptions(parser.WithAutoHeadingID()),
		goldmark.WithRendererOptions(html.WithUnsafe()),
	)
	body, _, err := s.convert(r)
	if err != nil {
		return nil,
			fmt.Errorf("convert markdown: %w", err)
	}

	return body, nil
}

// Parse b to find all other links within the document.
func (s *Source) ParseInternalLinks() ([]string, []string, error) {
	// il: internal links
	var il []string
	// bl: back links
	var bl []string

	r := text.NewReader(s.Body)
	wikiLinkParser := wikilink.Parser{}

	// Construct custom parser with lighter options
	wikilinkParser := util.Prioritized(&wikiLinkParser, 199)
	linkParser := util.Prioritized(parser.NewLinkParser(), 200)
	paragraphParser := util.Prioritized(parser.NewParagraphParser(), 1000)
	listParser := util.Prioritized(parser.NewListParser(), 300)
	listItemParser := util.Prioritized(parser.NewListItemParser(), 400)
	BlockquoteParser := util.Prioritized(parser.NewBlockquoteParser(), 800)
	p := parser.NewParser(parser.WithBlockParsers([]util.PrioritizedValue{
		paragraphParser,
		listItemParser,
		listParser,
		BlockquoteParser,
	}...),
		parser.WithInlineParsers(
			[]util.PrioritizedValue{
				wikilinkParser,
				linkParser,
			}...),
		parser.WithParagraphTransformers(),
	)

	// Parse and walk through nodes to find internal links
	nodes := p.Parse(r)
	walker := func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch n := n.(type) {
		case *ast.Image:
			dest := string(n.Destination)
			il = append(il, dest)
		case *wikilink.Node:
			target := string(n.Target)
			bl = append(bl, target)
		default:
			return ast.WalkContinue, nil
		}
		return ast.WalkContinue, nil
	}
	err := ast.Walk(nodes, walker)
	if err != nil {
		return nil, nil, fmt.Errorf("error extracting internal links: %w", err)
	}
	return bl, il, nil
}
