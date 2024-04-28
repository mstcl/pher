// Render html according to html templates
package render

import (
	"bytes"
	"fmt"
	"html/template"
	"os"

	"github.com/mstcl/pher.git/internal/config"
	"github.com/mstcl/pher.git/internal/listing"
	"github.com/mstcl/pher.git/internal/parse"
	"github.com/mstcl/pher.git/internal/tags"
	"github.com/mstcl/pher.git/internal/util"
)

// All fields used in the html templates.
//
// * Title: page title (set in title tags and in document)
//
// * Description: body description
//
// * Filename: has no extension. Used for navigation crumb.
type RenderData struct {
	Title              string
	Description        string
	Tags               []string
	TOC                bool
	ShowHeader         bool
	Layout             string
	RootCrumb          string
	Body               template.HTML
	Head               template.HTML
	Footer             []config.FooterLink
	Backlinks          []listing.Listing
	Relatedlinks       []listing.Listing
	Crumbs             []listing.Listing
	Filename           string
	Date               string
	DateUpdated        string
	MachineDate        string
	MachineDateUpdated string
	Listing            []listing.Listing
	TagsListing        []tags.Tag
}

// Recombine all the data from different places before rendering.
func ConstructData(
	cfg config.Config,
	html []byte,
	md parse.Metadata,
	backlinks []listing.Listing,
	indexes []listing.Listing,
	related []listing.Listing,
	cr []string,
	cl []string,
	filename string,
) (RenderData, error) {

	d := RenderData{}
	d.Title = util.ResolveTitle(md.Title, filename)
	d.Description = md.Description
	d.Tags = md.Tags
	d.TOC = md.TOC
	d.ShowHeader = md.ShowHeader
	d.Layout = md.Layout
	d.RootCrumb = cfg.RootCrumb
	d.Body = template.HTML(html)
	d.Head = template.HTML(cfg.Head)
	d.Footer = cfg.Footer
	d.Backlinks = backlinks
	d.Relatedlinks = related
	d.Filename = filename
	d.Listing = indexes

	// Populate crumbs
	for i, v := range cr {
		d.Crumbs = append(d.Crumbs, listing.Listing{Href: cl[i], Title: v})
	}

	// Use date only if given
	var err error
	if len(md.Date) > 0 {
		d.Date, d.MachineDate, err = util.ResolveDate(md.Date)
		if err != nil {
			return RenderData{}, fmt.Errorf("time parse: %w", err)
		}
	}

	if len(md.DateUpdated) > 0 {
		d.DateUpdated, d.MachineDateUpdated, err = util.ResolveDate(md.DateUpdated)
		if err != nil {
			return RenderData{}, fmt.Errorf("time parse: %w", err)
		}
	}

	return d, nil
}

// Template html with data d.
func Render(o string, t *template.Template, d RenderData, isDry bool) error {
	// Template the current file
	w := new(bytes.Buffer)
	err := t.ExecuteTemplate(w, "index", d)
	if err != nil {
		return fmt.Errorf("templating: %w", err)
	}

	// Ensure output directory exists
	err = os.MkdirAll(util.GetFilePath(o), os.ModePerm)
	if err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	// Save output html to disk
	if !isDry {
		err = os.WriteFile(o, w.Bytes(), 0644)
		if err != nil {
			return fmt.Errorf("writing article: %w", err)
		}
	}
	return nil
}

// Template tags page
func RenderTags(o string, t *template.Template, d RenderData, isDry bool) error {
	// Template the current file
	w := new(bytes.Buffer)
	err := t.ExecuteTemplate(w, "tags", d)
	if err != nil {
		return fmt.Errorf("templating: %w", err)
	}

	// Ensure output directory exists
	err = os.MkdirAll(util.GetFilePath(o), os.ModePerm)
	if err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	// Save output html to disk
	if !isDry {
		err = os.WriteFile(o, w.Bytes(), 0644)
		if err != nil {
			return fmt.Errorf("writing article: %w", err)
		}
	}
	return nil
}

// Render all files, including tags page, to html.
func RenderAll(
	m map[string]parse.Metadata,
	c map[string][]byte,
	b map[string][]listing.Listing,
	l map[string][]listing.Listing,
	t []tags.Tag,
	rl map[string][]listing.Listing,
	inDir string,
	outDir string,
	tpl *template.Template,
	cfg *config.Config,
	files []string,
	isDry bool,
) error {
	// depth := GetDepth(inDir)
	for _, f := range files {
		// Don't render drafts
		if parse.IsDraft(m[f]) {
			continue
		}

		// Get output path
		chunks := util.GetRelativeFilePath(f, inDir)
		chunks = util.GetFilePath(chunks)
		if len(chunks) > 0 {
			chunks += "/"
		}
		base := util.GetFileBase(f) + ".html"

		// Get navigation crumbs
		cr, cl := util.GetCrumbs(chunks)

		o := outDir + "/" + chunks + base

		// Construct data before rendering
		d, err := ConstructData(
			*cfg,
			c[f],
			m[f],
			b[f],
			l[f],
			rl[f],
			cr,
			cl,
			util.GetFileBase(f),
		)
		if err != nil {
			return fmt.Errorf("construct render data: %w", err)
		}

		// Add tags only to root index
		if f == inDir+"/index.md" {
			d.TagsListing = t
		}

		// Render
		err = Render(o, tpl, d, isDry)
		if err != nil {
			return fmt.Errorf("render html: %w", err)
		}
	}
	// Render tags page
	d := RenderData{
		RootCrumb:   cfg.RootCrumb,
		Footer:      cfg.Footer,
		TagsListing: t,
	}
	err := RenderTags(outDir+"/tags.html", tpl, d, isDry)
	if err != nil {
		return fmt.Errorf("render html: %w", err)
	}
	return nil
}
