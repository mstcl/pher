// Package source handles markdown source and helper methods
package source

import (
	"bytes"
	"fmt"

	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/mstcl/pher/v2/internal/customanchor"
	"github.com/mstcl/pher/v2/internal/frontmatter"
	"github.com/mstcl/pher/v2/internal/metadata"
	"github.com/mstcl/pher/v2/internal/toc"
	"github.com/mstcl/pher/v2/internal/wikilink"
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

type Links struct {
	BackLinks     []string
	InternalLinks []string
}

type Source struct {
	CodeTheme     string
	Body          []byte
	TOC           bool
	CodeHighlight bool
}

type Rendered struct {
	HTML      []byte
	ChromaCSS []byte
}

// ExtractMetadata parses metadata (frontmatter) from source.
func (s *Source) ExtractMetadata() (*metadata.Metadata, error) {
	r := goldmark.New(goldmark.WithExtensions(&frontmatter.Extender{}))

	_, md, err := s.convert(r)
	if err != nil {
		return nil, fmt.Errorf("convert markdown: %w", err)
	}

	return md, nil
}

// Convert transforms source with renderer to give html and Metadata.
// Requires renderer to have the fronmatter extension.
func (s *Source) convert(r goldmark.Markdown) ([]byte, *metadata.Metadata, error) {
	w := new(bytes.Buffer)

	// Get context
	context := parser.NewContext()

	if err := r.Convert(s.Body, w, parser.WithContext(context)); err != nil {
		return nil, nil, fmt.Errorf("converting to markdown: %w", err)
	}

	md := metadata.Default()

	// Decode frontmatter
	d := frontmatter.Get(context)
	if err := d.Decode(&md); err != nil {
		return nil, nil, fmt.Errorf("decoding frontmatter: %w", err)
	}

	return w.Bytes(), md, nil
}

// ToHTML reads in soure code and parse body.
// Frontmatter is reprocessed to strip it.
func (s *Source) ToHTML() (*Rendered, error) {
	ext := []goldmark.Extender{
		&anchor.Extender{
			Texter: &customanchor.Texter{},
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
	if s.TOC {
		ext = append(ext, &toc.Extender{})
	}

	chromaWriter := new(bytes.Buffer)

	if s.CodeHighlight {
		ext = append(ext, highlighting.NewHighlighting(
			highlighting.WithStyle(s.CodeTheme),
			highlighting.WithCSSWriter(chromaWriter),
			highlighting.WithFormatOptions(chromahtml.WithClasses(true)),
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

	return &Rendered{
		HTML:      body,
		ChromaCSS: chromaWriter.Bytes(),
	}, nil
}

// ExtractLinks walks through all files to collect links within the document.
func (s *Source) ExtractLinks() (*Links, error) {
	// internalLinks: internal links
	var internalLinks []string
	// backlinks: back links
	var backlinks []string

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
			internalLinks = append(internalLinks, dest)
		case *wikilink.Node:
			target := string(n.Target)
			backlinks = append(backlinks, target)
		default:
			return ast.WalkContinue, nil
		}

		return ast.WalkContinue, nil
	}

	err := ast.Walk(nodes, walker)
	if err != nil {
		return nil, fmt.Errorf("error extracting internal links: %w", err)
	}

	return &Links{BackLinks: backlinks, InternalLinks: internalLinks}, nil
}
