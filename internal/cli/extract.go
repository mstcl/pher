package cli

import (
	"bytes"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mstcl/pher/v2/internal/convert"
	"github.com/mstcl/pher/v2/internal/listing"
	"github.com/mstcl/pher/v2/internal/source"
	"github.com/mstcl/pher/v2/internal/state"
	"github.com/mstcl/pher/v2/internal/tag"
)

// Process files to build up the entry data for all files, the tags data, and
// the linked internal asset.
// Exclusive calls to parse.* are made here.
// Calls source.ExtractMetadata()
// Calls source.ToHTML()
// Calls source.ExtractLinks()
// Further business logic to construct the backlinks, relatedlinks, asset map and tags slice
func extractExtras(s *state.State, logger *slog.Logger) error {
	// tagsCount: tags count (key: tag name)
	tagsCount := make(map[string]int)

	// tagsListing: tags listing - files with this tag (key: tag name)
	tagsListing := make(map[string][]listing.Listing)

	// First loop, can do most things
	for _, f := range s.Files {
		child := logger.With(slog.String("filepath", f), slog.String("context", "extracting extras"))

		entry := s.Entries[f]

		file, err := os.Open(f)
		if err != nil {
			return err
		}

		defer file.Close()

		buf := new(bytes.Buffer)
		if _, err := buf.ReadFrom(file); err != nil {
			return err
		}

		child.Debug("read in file")

		src := source.Source{
			Body:          buf.Bytes(),
			CodeHighlight: s.Config.CodeHighlight,
			CodeTheme:     s.Config.CodeTheme,
		}

		md, err := src.ExtractMetadata()
		if err != nil {
			return err
		}

		child.Debug("extracted metadata", slog.Any("metadata", md))

		src.TOC = md.TOC

		// Don't proceed if file is draft
		if md.Draft {
			child.Debug("skipping: file is draft")

			continue
		}

		// Extract and parse html body
		rendered, err := src.ToHTML()
		if err != nil {
			return err
		}

		child.Debug("extracted html")

		// Extract wiki backlinks (blinks) and image links (internalLinks)
		links, err := src.ExtractLinks()
		if err != nil {
			return err
		}

		child.Debug("extracted links", slog.Any("links", links))

		// Resolve basic vars
		path := filepath.Dir(f)
		base := convert.FileBase(f)
		title := convert.Title(md.Title, base)
		href := convert.Href(f, s.InDir, false)
		isDir := base == "index"

		if s.Config.IsExt {
			href += ".html"
		}

		// Update entry
		entry.Metadata = *md
		entry.Body = rendered.HTML
		entry.Href = href
		entry.ChromaCSS = rendered.ChromaCSS
		s.Entries[f] = entry

		// Update assets from internal links
		for _, v := range links.InternalLinks {
			// Absolutize image links
			ref, err := filepath.Abs(filepath.Join(path, v))
			if err != nil {
				return nil
			}

			s.Assets[ref] = true
		}

		child.Debug("updated assets with internal links paths", slog.Any("assets", s.Assets))

		// Update assets and wikilinks from backlinks
		for _, v := range links.BackLinks {
			// Reconstruct wikilink into full input path
			ref, err := filepath.Abs(filepath.Join(path, v))
			if err != nil {
				return err
			}

			// Process links with extensions as external files
			// like images/gifs
			if len(filepath.Ext(ref)) > 0 {
				s.Assets[ref] = true
			}

			ref += ".md"

			// Save backlinks
			linkedEntry := s.Entries[ref]
			linkedEntry.Backlinks = append(
				linkedEntry.Backlinks,
				listing.Listing{
					Href:        href,
					Title:       title,
					Description: entry.Metadata.Description,
					IsDir:       isDir,
				},
			)
			s.Entries[ref] = linkedEntry
		}

		child.Debug("updated assets and wiklinks from backlinks")

		// Grab tags count and tags listing
		// We process the final tags later - this is
		// for the tags page
		for _, v := range md.Tags {
			tagsCount[v] += 1

			tagsListing[v] = append(tagsListing[v], listing.Listing{
				Href:        href,
				Title:       title,
				Description: entry.Metadata.Description,
				IsDir:       isDir,
			})
		}

		child.Debug("updated tags", slog.Any("tagsCount", tagsCount), slog.Any("tagsListing", tagsListing))
	}

	logger.Debug("proceeding to second loop")

	// Second loop for related links
	//
	// NOTE: Entries that share tags are related
	// Hence dependent on tags listing (tl)
	for _, f := range s.Files {
		child := logger.With(slog.String("filepath", f), slog.String("context", "extracting extras"))

		entry := s.Entries[f]
		if entry.Metadata.Draft || len(entry.Metadata.Tags) == 0 {
			continue
		}

		// listings: all related links
		listings := []listing.Listing{}

		// relatedListings: unique related links
		relatedListings := []listing.Listing{}

		// Get all files with similar tags
		for _, t := range entry.Metadata.Tags {
			listings = append(listings, tagsListing[t]...)
		}

		// Remove self from l to ensure uniqueness
		for _, l := range listings {
			filename := strings.TrimSuffix(l.Href, filepath.Ext(l.Href))
			if filepath.Join(s.InDir, filename) == strings.TrimSuffix(f, ".md") {
				continue
			}

			relatedListings = append(relatedListings, l)
		}

		entry.Relatedlinks = relatedListings

		child.Debug("extracted related links", slog.Any("relatedlinks", relatedListings))

		// Update entry
		s.Entries[f] = entry
	}

	// Transform maps of tags count and tags listing to give a sorted slice of tags.
	// Used by render.RenderTags exclusively.
	tags := []tag.Tag{}
	keys := make([]string, 0, len(tagsCount))

	for k := range tagsCount {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	for _, k := range keys {
		tags = append(tags, tag.Tag{Name: k, Count: tagsCount[k], Links: tagsListing[k]})
	}
	s.Tags = append(s.Tags, tags...)

	logger.Debug("extracted tags", slog.Any("tags", s.Tags))

	return nil
}
