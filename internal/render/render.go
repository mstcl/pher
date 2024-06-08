// Render html according to html templates
package render

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"path/filepath"

	"github.com/mstcl/pher/internal/config"
	"github.com/mstcl/pher/internal/convert"
	"github.com/mstcl/pher/internal/entry"
	"github.com/mstcl/pher/internal/listing"
	"github.com/mstcl/pher/internal/parse"
	"github.com/mstcl/pher/internal/tag"
)

type RenderDeps struct {
	Config    *config.Config
	Templates *template.Template
	Entries   map[string]entry.Entry
	Skip      map[string]bool
	Listings  map[string][]listing.Listing
	InDir     string
	OutDir    string
	Files     []string
	Tags      []tag.Tag
	Metadata  parse.Metadata
	DryRun    bool
}

// All fields used in the html templates.
//
// * Title: page title (set in title tags and in document)
//
// * Description: body description
//
// * Filename: has no extension. Used for navigation crumb.
type RenderData struct {
	Body                                     template.HTML
	Head                                     template.HTML
	WikiTitle                                string
	Url                                      string
	Title                                    string
	Description                              string
	Layout                                   string
	RootCrumb                                string
	Filename                                 string
	Date                                     string
	DateUpdated                              string
	MachineDate                              string
	MachineDateUpdated                       string
	Ext                                      string
	OutFilename                              string
	Tags                                     []string
	TagsListing                              []tag.Tag
	Footer                                   []config.FooterLink
	Backlinks, Relatedlinks, Crumbs, Listing []listing.Listing
	TOC                                      bool
	ShowHeader                               bool
}

// Template html with data d.
func (rd *RenderData) Render(t *template.Template, dryRun bool, templateName string) error {
	// Template the current file
	w := new(bytes.Buffer)
	if err := t.ExecuteTemplate(w, templateName, rd); err != nil {
		return err
	}

	// Ensure output directory exists
	if err := os.MkdirAll(filepath.Dir(rd.OutFilename), 0o755); err != nil {
		return fmt.Errorf("error mkdir: %w", err)
	}

	// Save output html to disk
	if !dryRun {
		if err := os.WriteFile(rd.OutFilename, w.Bytes(), 0o644); err != nil {
			return fmt.Errorf("error writing entry to disk: %w", err)
		}
	}
	return nil
}

// Render all files, including tags page, to html.
func (d *RenderDeps) RenderAll() error {
	for _, f := range d.Files {
		// Don't render drafts or skipped files
		entry := d.Entries[f]
		if entry.Metadata.Draft || d.Skip[f] {
			continue
		}

		// Get navigation crumbs
		crumbsTitle, crumbsLink := convert.NavCrumbs(f, d.InDir, d.Config.IsExt)

		// Populate crumbs
		crumbs := []listing.Listing{}
		for i, t := range crumbsTitle {
			crumbs = append(crumbs, listing.Listing{Href: crumbsLink[i], Title: t})
		}

		// The output path outDir/{a/b/c/file}.html (part in curly brackets is the href)
		outPath := d.OutDir + convert.Href(f, d.InDir, true) + ".html"

		// Construct rendering data (rd) from config, entry data, listing, nav
		// crumbs, etc.
		rd := RenderData{
			OutFilename:  outPath,
			Listing:      d.Listings[f],
			Filename:     convert.FileBase(f),
			Description:  entry.Metadata.Description,
			Tags:         entry.Metadata.Tags,
			TOC:          entry.Metadata.TOC,
			ShowHeader:   entry.Metadata.ShowHeader,
			Layout:       entry.Metadata.Layout,
			Backlinks:    entry.Backlinks,
			Relatedlinks: entry.Relatedlinks,
			Body:         template.HTML(entry.Body),
			Head:         template.HTML(d.Config.Head),
			RootCrumb:    d.Config.RootCrumb,
			Footer:       d.Config.Footer,
			WikiTitle:    d.Config.Title,
			Url:          d.Config.Url + entry.Href,
			Crumbs:       crumbs,
		}
		rd.Title = convert.Title(entry.Metadata.Title, rd.Filename)
		if d.Config.IsExt {
			rd.Ext = ".html"
		} else {
			rd.Ext = ""
		}

		// Use date only if given
		var err error
		rd.Date, rd.MachineDate, err = convert.Date(entry.Metadata.Date)
		if err != nil {
			return err
		}

		// Use data updated only if given
		rd.DateUpdated, rd.MachineDateUpdated, err = convert.Date(entry.Metadata.DateUpdated)
		if err != nil {
			return err
		}

		// Add tags only to root index
		if f == d.InDir+"/index.md" {
			rd.TagsListing = d.Tags
		}

		// Render
		if err = rd.Render(d.Templates, d.DryRun, "index"); err != nil {
			return err
		}
	}
	// Construct tags data (td) to render tags page
	td := RenderData{
		RootCrumb:   d.Config.RootCrumb,
		Footer:      d.Config.Footer,
		TagsListing: d.Tags,
		OutFilename: d.OutDir + "/tags.html",
	}
	if err := td.Render(d.Templates, d.DryRun, "tags"); err != nil {
		return err
	}
	return nil
}
