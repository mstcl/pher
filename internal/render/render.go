// Render html according to html templates
package render

import (
	"bytes"
	"fmt"
	"html/template"
	"os"

	"github.com/mstcl/pher/internal/config"
	"github.com/mstcl/pher/internal/entry"
	"github.com/mstcl/pher/internal/listing"
	"github.com/mstcl/pher/internal/parse"
	"github.com/mstcl/pher/internal/tag"
	"github.com/mstcl/pher/internal/util"
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
	TagsListing        []tag.Tag
	Ext                string
}

// Recombine all the data from different places before rendering.
func ConstructData(
	cfg config.Config,
	entryData entry.Entry,
	listings []listing.Listing,
	cr []string,
	cl []string,
	filename string,
	isExt bool,
) (RenderData, error) {
	md := entryData.Metadata
	d := RenderData{}
	d.Title = util.ResolveTitle(md.Title, filename)
	d.Description = md.Description
	d.Tags = md.Tags
	d.TOC = md.TOC
	d.ShowHeader = md.ShowHeader
	d.Layout = md.Layout
	d.RootCrumb = cfg.RootCrumb
	d.Body = template.HTML(entryData.Body)
	d.Head = template.HTML(cfg.Head)
	d.Footer = cfg.Footer
	d.Backlinks = entryData.Backlinks
	d.Relatedlinks = entryData.Relatedlinks
	d.Filename = filename
	d.Listing = listings

	// Populate crumbs
	for i, v := range cr {
		d.Crumbs = append(d.Crumbs, listing.Listing{Href: cl[i], Title: v})
	}

	// Use date only if given
	var err error
	if len(md.Date) > 0 {
		d.Date, d.MachineDate, err = util.ResolveDate(md.Date)
		if err != nil {
			return RenderData{}, err
		}
	}

	if len(md.DateUpdated) > 0 {
		d.DateUpdated, d.MachineDateUpdated, err = util.ResolveDate(md.DateUpdated)
		if err != nil {
			return RenderData{}, err
		}
	}

	if isExt {
		d.Ext = ".html"
	} else {
		d.Ext = ""
	}

	return d, nil
}

// Template html with data d.
func Render(o string, t *template.Template, rd RenderData, isDry bool) error {
	// Template the current file
	w := new(bytes.Buffer)
	if err := t.ExecuteTemplate(w, "index", rd); err != nil {
		return err
	}

	// Ensure output directory exists
	if err := os.MkdirAll(util.GetFilePath(o), os.ModePerm); err != nil {
		return fmt.Errorf("error mkdir: %w", err)
	}

	// Save output html to disk
	if !isDry {
		if err := os.WriteFile(o, w.Bytes(), 0644); err != nil {
			return fmt.Errorf("error writing entry to disk: %w", err)
		}
	}
	return nil
}

// Template tags page
func RenderTags(o string, t *template.Template, rd RenderData, isDry bool) error {
	// Template the current file
	w := new(bytes.Buffer)
	if err := t.ExecuteTemplate(w, "tags", rd); err != nil {
		return err
	}

	// Ensure output directory exists
	if err := os.MkdirAll(util.GetFilePath(o), os.ModePerm); err != nil {
		return fmt.Errorf("error mkdir: %w", err)
	}

	// Save output html to disk
	if !isDry {
		if err := os.WriteFile(o, w.Bytes(), 0644); err != nil {
			return fmt.Errorf("error writing entry to disk: %w", err)
		}
	}
	return nil
}

// Render all files, including tags page, to html.
func RenderAll(
	d map[string]entry.Entry,
	l map[string][]listing.Listing,
	t []tag.Tag,
	inDir string,
	outDir string,
	tpl *template.Template,
	cfg *config.Config,
	files []string,
	isDry bool,
	skip map[string]bool,
) error {
	// depth := GetDepth(inDir)
	for _, f := range files {
		// Don't render drafts or skipped files
		if parse.IsDraft(d[f].Metadata) || skip[f] {
			continue
		}

		// Get navigation crumbs
		cr, cl := util.GetCrumbs(f, inDir, cfg.IsExt)

		// Get output path
		o := util.ResolveOutPath(f, inDir, outDir, ".html")

		// Construct rendering data (rd) from config, entry data, listing, nav
		// crumbs, etc.
		rd, err := ConstructData(
			*cfg,
			d[f],
			l[f],
			cr,
			cl,
			util.GetFileBase(f),
			cfg.IsExt,
		)
		if err != nil {
			return err
		}

		// Add tags only to root index
		if f == inDir+"/index.md" {
			rd.TagsListing = t
		}

		// Render
		if err = Render(o, tpl, rd, isDry); err != nil {
			return err
		}
	}
	// Construct tags data (td) to render tags page
	td := RenderData{
		RootCrumb:   cfg.RootCrumb,
		Footer:      cfg.Footer,
		TagsListing: t,
	}
	if err := RenderTags(outDir+"/tags.html", tpl, td, isDry); err != nil {
		return err
	}
	return nil
}
