package cli

import (
	"bytes"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mstcl/pher/v2/internal/assetpath"
	"github.com/mstcl/pher/v2/internal/convert"
	"github.com/mstcl/pher/v2/internal/nodepath"
	"github.com/mstcl/pher/v2/internal/nodepathlink"
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
	tagsListing := make(map[string][]nodepathlink.NodePathLink)

	// First loop, can do most things
	for _, np := range s.NodePaths {
		child := logger.With(
			slog.Any("nodepath", np),
			slog.String("context", "extracting extras"),
		)

		entry := s.NodeMap[np]

		file, err := os.Open(np.String())
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
		path := filepath.Dir(np.String())
		base := np.Base()
		title := convert.Title(md.Title, base)
		href := np.Href(s.InputDir, false)
		isDir := base == "index"

		if s.Config.IsExt {
			href += ".html"
		}

		// Update entry
		entry.Metadata = *md
		entry.Body = rendered.HTML
		entry.Href = href
		entry.ChromaCSS = rendered.ChromaCSS
		s.NodeMap[np] = entry

		// Update assets from internal links
		for _, v := range links.InternalLinks {
			// Absolutize image links
			ref, err := filepath.Abs(filepath.Join(path, v))
			if err != nil {
				return nil
			}

			s.UserAssetMap[assetpath.AssetPath(ref)] = true
		}

		child.Debug("updated assets with internal links paths", slog.Any("assets", s.UserAssetMap))

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
				s.UserAssetMap[assetpath.AssetPath(ref)] = true
			}

			ref += ".md"

			// Save backlinks
			linkedEntry := s.NodeMap[nodepath.NodePath(ref)]
			linkedEntry.Backlinks = append(
				linkedEntry.Backlinks,
				nodepathlink.NodePathLink{
					Href:        href,
					Title:       title,
					Description: entry.Metadata.Description,
					IsDir:       isDir,
				},
			)
			s.NodeMap[nodepath.NodePath(ref)] = linkedEntry
		}

		child.Debug("updated assets and wiklinks from backlinks")

		// Grab tags count and tags listing
		// We process the final tags later - this is
		// for the tags page
		for _, v := range md.Tags {
			tagsCount[v] += 1

			tagsListing[v] = append(tagsListing[v], nodepathlink.NodePathLink{
				Href:        href,
				Title:       title,
				Description: entry.Metadata.Description,
				IsDir:       isDir,
			})
		}

		child.Debug(
			"updated tags",
			slog.Any("tagsCount", tagsCount),
			slog.Any("tagsListing", tagsListing),
		)
	}

	logger.Debug("proceeding to second loop")

	// Second loop for related links
	//
	// NOTE: Entries that share tags are related
	// Hence dependent on tags listing (tl)
	for _, np := range s.NodePaths {
		child := logger.With(
			slog.Any("nodepath", np),
			slog.String("context", "extracting extras"),
		)

		entry := s.NodeMap[np]
		if entry.Metadata.Draft || len(entry.Metadata.Tags) == 0 {
			continue
		}

		// listings: all related links
		listings := []nodepathlink.NodePathLink{}

		// relatedListings: unique related links
		relatedListings := []nodepathlink.NodePathLink{}

		// Get all files with similar tags
		for _, t := range entry.Metadata.Tags {
			listings = append(listings, tagsListing[t]...)
		}

		// Remove self from l to ensure uniqueness
		// Also deduplicate listing for items that share more than two tags
		uniqueFilenames := make(map[string]bool)

		for _, l := range listings {
			filename := strings.TrimSuffix(l.Href, filepath.Ext(l.Href))
			if filepath.Join(s.InputDir, filename) == strings.TrimSuffix(np.String(), ".md") {
				continue
			}

			if _, ok := uniqueFilenames[filename]; ok {
				continue
			}

			relatedListings = append(relatedListings, l)
			uniqueFilenames[filename] = true
		}

		entry.Relatedlinks = relatedListings

		child.Debug("extracted related links", slog.Any("relatedlinks", relatedListings))

		// Update entry
		s.NodeMap[np] = entry
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
	s.NodeTags = append(s.NodeTags, tags...)

	logger.Debug("extracted tags", slog.Any("tags", s.NodeTags))

	return nil
}
