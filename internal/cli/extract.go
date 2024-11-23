package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mstcl/pher/internal/convert"
	"github.com/mstcl/pher/internal/listing"
	"github.com/mstcl/pher/internal/source"
	"github.com/mstcl/pher/internal/state"
	"github.com/mstcl/pher/internal/tag"
)

// Process files to build up the entry data for all files, the tags data, and
// the linked internal asset.
// Exclusive calls to parse.* are made here.
// Calls source.ExtractMetadata()
// Calls source.ToHTML()
// Calls source.ExtractLinks()
// Further business logic to construct the backlinks, relatedlinks, asset map and tags slice
func extractExtras(s *state.State) error {
	// tagsCount: tags count (key: tag name)
	tagsCount := make(map[string]int)

	// tagsListing: tags listing - files with this tag (key: tag name)
	tagsListing := make(map[string][]listing.Listing)

	// First loop, can do most things
	for _, f := range s.Files {
		entry := s.Entries[f]

		// Read input file
		file, err := os.Open(f)
		if err != nil {
			return err
		}

		buf := new(bytes.Buffer)
		if _, err := buf.ReadFrom(file); err != nil {
			return err
		}

		file.Close()

		// Extract source and save metadata
		src := source.Source{Body: buf.Bytes(), RendersHighlight: s.Config.CodeHighlight}

		md, err := src.ExtractMetadata()
		if err != nil {
			return err
		}

		src.RendersTOC = md.TOC

		// Don't proceed if file is draft
		if md.Draft {
			continue
		}

		// Extract and parse html body
		html, err := src.ToHTML()
		if err != nil {
			return err
		}

		// Extract wiki backlinks (blinks) and image links (internalLinks)
		links, err := src.ExtractLinks()
		if err != nil {
			return err
		}

		// Resolve basic vars
		path := filepath.Dir(f)
		base := convert.FileBase(f)
		title := convert.Title(md.Title, base)
		href := convert.Href(f, s.InDir, true)
		isDir := base == "index"

		if s.Config.IsExt {
			href += ".html"
		}

		// Update entry
		entry.Metadata = *md
		entry.Body = html
		entry.Href = href
		s.Entries[f] = entry

		// Update assets from internal links
		for _, v := range links.InternalLinks {
			// Absolutize image links
			ref, err := filepath.Abs(path + "/" + v)
			if err != nil {
				return nil
			}

			s.Assets[ref] = true
		}

		// Update assets and wikilinks from backlinks
		for _, v := range links.BackLinks {
			// Reconstruct wikilink into full input path
			ref, err := filepath.Abs(path + "/" + v)
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
	}

	// Second loop for related links
	//
	// NOTE: Entries that share tags are related
	// Hence dependent on tags listing (tl)
	for _, f := range s.Files {
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
			if s.InDir+filename+".md" == f {
				continue
			}

			relatedListings = append(relatedListings, l)
		}

		entry.Relatedlinks = relatedListings

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

	return nil
}
