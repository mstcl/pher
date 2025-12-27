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
	"github.com/mstcl/pher/v2/internal/nodepath"
	"github.com/mstcl/pher/v2/internal/nodepathlink"
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
	ChromaCSS                                template.CSS
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
	Backlinks, Relatedlinks, Crumbs, Listing []nodepathlink.NodePathLink
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

	for _, np := range s.NodePaths {
		child := logger.With(slog.Any("nodepath", np), slog.String("context", "templating"))

		child.Debug("submitting goroutine")

		eg.Go(func() error {
			// Don't render drafts or skipped files
			entry := s.NodeMap[np]
			if entry.Metadata.Draft || s.SkippedNodePathMap[np] {
				return nil
			}

			// Get navigation crumbs
			crumbsTitle, crumbsLink := convert.NavCrumbs(np, s.InputDir, s.Config.IsExt)

			// Populate navigation crumbs
			crumbs := []nodepathlink.NodePathLink{}
			for i, t := range crumbsTitle {
				crumbs = append(crumbs, nodepathlink.NodePathLink{Href: crumbsLink[i], Title: t})
			}

			// The output path outDir/{a/b/c/file}.html (part in curly brackets is the href)
			outPath := s.OutputDir + np.Href(s.InputDir, true) + ".html"

			// Construct rendering data (entryData) from config, entry data, listing, nav
			// crumbs, etc.
			entryData := data{
				OutFilename:  outPath,
				Listing:      s.NodePathLinksMap[np],
				Filename:     np.Base(),
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
				ChromaCSS:    template.CSS(entry.ChromaCSS),
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
			entryData.DateUpdated, entryData.MachineDateUpdated, err = convert.Date(
				entry.Metadata.DateUpdated,
			)
			if err != nil {
				return err
			}

			// Add tags only to root index
			if np == nodepath.NodePath(filepath.Join(s.InputDir, "index.md")) {
				entryData.TagsListing = s.NodeTags
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
			TagsListing: s.NodeTags,
			OutFilename: s.OutputDir + "/tags.html",
			Path:        s.Config.Path,
		},
	}); err != nil {
		return err
	}

	logger.Debug("finished rendering tags page")

	return nil
}
