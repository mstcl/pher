// Render html according to html templates
package render

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/mstcl/pher/v2/internal/config"
	"github.com/mstcl/pher/v2/internal/convert"
	"github.com/mstcl/pher/v2/internal/listing"
	"github.com/mstcl/pher/v2/internal/state"
	"github.com/mstcl/pher/v2/internal/tag"
	"golang.org/x/sync/errgroup"
)

// All fields used in the html templates.
//
// * Title: page title (set in title tags and in document)
//
// * Description: body description
//
// * Filename: has no extension. Used for navigation crumb.
type data struct {
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
	Path                                     string
	Tags                                     []string
	TagsListing                              []tag.Tag
	Footer                                   []config.FooterLink
	Backlinks, Relatedlinks, Crumbs, Listing []listing.Listing
	TOC                                      bool
	ShowHeader                               bool
}

type renderInput struct {
	template     *template.Template
	data         *data
	templateName string
	dryRun       bool
}

// Template html with data d.
func render(i *renderInput) error {
	// Template the current file
	w := new(bytes.Buffer)
	if err := i.template.ExecuteTemplate(w, i.templateName, i.data); err != nil {
		return err
	}

	// Ensure output directory exists
	if err := os.MkdirAll(filepath.Dir(i.data.OutFilename), 0o755); err != nil {
		return fmt.Errorf("error mkdir: %w", err)
	}

	// Save output html to disk
	if !i.dryRun {
		if err := os.WriteFile(i.data.OutFilename, w.Bytes(), 0o644); err != nil {
			return fmt.Errorf("error writing entry to disk: %w", err)
		}
	}

	return nil
}

// Render all files, including tags page, to html.
func Render(ctx context.Context, s *state.State, logger *slog.Logger) error {
	var err error

	eg, _ := errgroup.WithContext(ctx)

	for _, f := range s.Files {
		f := f

		child := logger.With(slog.String("filepath", f), slog.String("context", "templating"))

		child.Debug("submitting goroutine")

		eg.Go(func() error {
			// Don't render drafts or skipped files
			entry := s.Entries[f]
			if entry.Metadata.Draft || s.Skip[f] {
				return nil
			}

			// Get navigation crumbs
			crumbsTitle, crumbsLink := convert.NavCrumbs(f, s.InDir, s.Config.IsExt)

			// Populate navigation crumbs
			crumbs := []listing.Listing{}
			for i, t := range crumbsTitle {
				crumbs = append(crumbs, listing.Listing{Href: crumbsLink[i], Title: t})
			}

			// The output path outDir/{a/b/c/file}.html (part in curly brackets is the href)
			outPath := s.OutDir + convert.Href(f, s.InDir, true) + ".html"

			// Construct rendering data (entryData) from config, entry data, listing, nav
			// crumbs, etc.
			entryData := data{
				OutFilename:  outPath,
				Listing:      s.Listings[f],
				Filename:     convert.FileBase(f),
				Description:  entry.Metadata.Description,
				Tags:         entry.Metadata.Tags,
				TOC:          entry.Metadata.TOC,
				ShowHeader:   entry.Metadata.ShowHeader,
				Layout:       entry.Metadata.Layout,
				Backlinks:    entry.Backlinks,
				Relatedlinks: entry.Relatedlinks,
				Body:         template.HTML(entry.Body),
				Head:         template.HTML(s.Config.Head),
				RootCrumb:    s.Config.RootCrumb,
				Footer:       s.Config.Footer,
				WikiTitle:    s.Config.Title,
				Url:          s.Config.Url + entry.Href,
				Path:         s.Config.Path,
				Crumbs:       crumbs,
			}
			entryData.Title = convert.Title(entry.Metadata.Title, entryData.Filename)

			if s.Config.IsExt {
				entryData.Ext = ".html"
			} else {
				entryData.Ext = ""
			}

			// Use date only if given
			entryData.Date, entryData.MachineDate, err = convert.Date(entry.Metadata.Date)
			if err != nil {
				return err
			}

			// Use data updated only if given
			entryData.DateUpdated, entryData.MachineDateUpdated, err = convert.Date(entry.Metadata.DateUpdated)
			if err != nil {
				return err
			}

			// Add tags only to root index
			if f == s.InDir+"/index.md" {
				entryData.TagsListing = s.Tags
			}

			// Render
			if err = render(&renderInput{
				template:     s.Templates,
				dryRun:       s.DryRun,
				templateName: "index",
				data:         &entryData,
			}); err != nil {
				return err
			}

			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return err
	}

	logger.Debug("finished rendering all files")

	// Render tags page
	if err := render(&renderInput{
		template:     s.Templates,
		dryRun:       s.DryRun,
		templateName: "tags",
		data: &data{
			RootCrumb:   s.Config.RootCrumb,
			Footer:      s.Config.Footer,
			TagsListing: s.Tags,
			OutFilename: s.OutDir + "/tags.html",
			Path:        s.Config.Path,
		},
	}); err != nil {
		return err
	}

	logger.Debug("finished rendering tags page")

	return nil
}
