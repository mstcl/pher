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

type Meta struct {
	C         *config.Config
	Templates *template.Template
	D         map[string]entry.Entry
	Skip      map[string]bool
	L         map[string][]listing.Listing
	InDir     string
	OutDir    string
	Files     []string
	T         []tag.Tag
	M         parse.Metadata
	IsDry     bool
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
func (rd *RenderData) Render(t *template.Template, isDry bool, templateName string) error {
	// Template the current file
	w := new(bytes.Buffer)
	if err := t.ExecuteTemplate(w, templateName, rd); err != nil {
		return err
	}

	// Ensure output directory exists
	if err := os.MkdirAll(util.GetFilePath(rd.OutFilename), 0o755); err != nil {
		return fmt.Errorf("error mkdir: %w", err)
	}

	// Save output html to disk
	if !isDry {
		if err := os.WriteFile(rd.OutFilename, w.Bytes(), 0o644); err != nil {
			return fmt.Errorf("error writing entry to disk: %w", err)
		}
	}
	return nil
}

// Render all files, including tags page, to html.
func (m *Meta) RenderAll() error {
	for _, f := range m.Files {
		// Don't render drafts or skipped files
		e := m.D[f]
		if e.Metadata.Draft || m.Skip[f] {
			continue
		}

		// Get navigation crumbs
		cr, cl := util.GetCrumbs(f, m.InDir, m.C.IsExt)

		// Populate crumbs
		crumbs := []listing.Listing{}
		for i, v := range cr {
			crumbs = append(crumbs, listing.Listing{Href: cl[i], Title: v})
		}

		// Get output path
		o := util.ResolveOutPath(f, m.InDir, m.OutDir, ".html")

		// Construct rendering data (rd) from config, entry data, listing, nav
		// crumbs, etc.
		rd := RenderData{
			OutFilename:  o,
			Listing:      m.L[f],
			Filename:     util.GetFileBase(f),
			Description:  e.Metadata.Description,
			Tags:         e.Metadata.Tags,
			TOC:          e.Metadata.TOC,
			ShowHeader:   e.Metadata.ShowHeader,
			Layout:       e.Metadata.Layout,
			Backlinks:    e.Backlinks,
			Relatedlinks: e.Relatedlinks,
			Body:         template.HTML(e.Body),
			Head:         template.HTML(m.C.Head),
			RootCrumb:    m.C.RootCrumb,
			Footer:       m.C.Footer,
			Crumbs:       crumbs,
		}
		rd.Title = util.ResolveTitle(e.Metadata.Title, rd.Filename)
		if m.C.IsExt {
			rd.Ext = ".html"
		} else {
			rd.Ext = ""
		}

		// Use date only if given
		var err error
		rd.Date, rd.MachineDate, err = util.ResolveDate(e.Metadata.Date)
		if err != nil {
			return err
		}

		// Use data updated only if given
		rd.DateUpdated, rd.MachineDateUpdated, err = util.ResolveDate(e.Metadata.DateUpdated)
		if err != nil {
			return err
		}

		// Add tags only to root index
		if f == m.InDir+"/index.md" {
			rd.TagsListing = m.T
		}

		// Render
		if err = rd.Render(m.Templates, m.IsDry, "index"); err != nil {
			return err
		}
	}
	// Construct tags data (td) to render tags page
	td := RenderData{
		RootCrumb:   m.C.RootCrumb,
		Footer:      m.C.Footer,
		TagsListing: m.T,
		OutFilename: m.OutDir + "/tags.html",
	}
	if err := td.Render(m.Templates, m.IsDry, "tags"); err != nil {
		return err
	}
	return nil
}
