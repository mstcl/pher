package entry

import (
	"bytes"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mstcl/pher/internal/config"
	"github.com/mstcl/pher/internal/convert"
	"github.com/mstcl/pher/internal/listing"
	"github.com/mstcl/pher/internal/parse"
	"github.com/mstcl/pher/internal/tag"
)

// Required dependencies
type ExtractDeps struct {
	C      *config.Config
	InDir  string
	OutDir string
	D      map[string]Entry
	A      map[string]bool
	T      []tag.Tag
	M      parse.Metadata
}

// Process files to build up the entry data for all files, the tags data, and
// the linked internal asset.
// Exclusive calls to parse.* are made here.
// Calls parse.ParseMetadata() to grab metadata.
// Calls parse.ParseSource() to grab html body.
// Calls parse.ParseInternalLinks() to grab backlinks and internal links.
// Additionally construct the backlinks, relatedlinks, asset map and tags slice
func (m *ExtractDeps) ExtractEntries(files []string) error {
	m.D = make(map[string]Entry) // entry data
	m.A = make(map[string]bool)  // internal assets

	// These are local data:
	// tc: tags count (key: tag name)
	// tl: tags listing - files with this tag (key: tag name)
	tc := make(map[string]int)
	tl := make(map[string][]listing.Listing)

	// First loop, can do most things
	for _, f := range files {
		entry := m.D[f]

		// Read input file
		file, err := os.Open(f)
		if err != nil {
			return err
		}
		defer file.Close()

		buf := new(bytes.Buffer)
		if _, err := buf.ReadFrom(file); err != nil {
			return err
		}

		// Extract source and save metadata
		s := parse.Source{B: buf.Bytes(), IsHighlight: m.C.CodeHighlight}
		md, err := s.ParseMetadata()
		if err != nil {
			return err
		}
		s.IsTOC = md.TOC

		// Don't proceed if file is draft
		if md.Draft {
			continue
		}

		// Extract and parse html body
		html, err := s.ParseSource()
		if err != nil {
			return err
		}

		// Extract wiki backlinks (blinks) and image links (ilinks)
		var ilinks internalLinks
		var blinks backLinks

		blinks, ilinks, err = s.ParseInternalLinks()
		if err != nil {
			return err
		}

		// Resolve basic vars
		path := filepath.Dir(f)
		base := convert.FileBase(f)
		title := convert.Title(md.Title, base)
		href := convert.Href(f, m.InDir, true)
		if m.C.IsExt {
			href += ".html"
		}

		// Update entry
		entry.Metadata = md
		entry.Body = html
		entry.Href = href
		m.D[f] = entry

		// Update assets from internal links
		m.A, err = ilinks.extract(internalLinksDeps{path: path}, m.A)
		if err != nil {
			return err
		}

		// Update assets and wikilinks from backlinks
		m.A, m.D, err = blinks.extract(backLinksDeps{
			href:        href,
			title:       title,
			description: md.Description,
			path:        path,
		}, m.A, m.D)
		if err != nil {
			return err
		}

		// Update tags count and tags listing
		var entryTags tags = md.Tags

		tc, tl = entryTags.split(tagsDeps{
			href:        href,
			title:       title,
			description: md.Description,
		}, tc, tl)
	}

	// Second loop for related links
	//
	// NOTE: Entries that share tags are related
	// Hence dependent on tags listing (tl)
	for _, f := range files {
		entry := m.D[f]
		if entry.Metadata.Draft || len(entry.Metadata.Tags) == 0 {
			continue
		}

		l := []listing.Listing{}
		u := []listing.Listing{}

		// Get all links under f's tags
		for _, t := range entry.Metadata.Tags {
			l = append(l, tl[t]...)
		}

		// Remove self from l to ensure uniqueness
		for _, j := range l {
			fn := strings.TrimSuffix(j.Href, filepath.Ext(j.Href))
			if m.InDir+fn+".md" == f {
				continue
			}
			u = append(u, j)
		}

		entry.Relatedlinks = u

		// Update entry
		m.D[f] = entry
	}

	// Transform tags to right format
	// a list of listing.Tag
	m.T = extractTags(tc, tl)

	// Update meta data structs

	return nil
}

// Transform maps of tags count and tags listing to give a sorted slice of tags.
// Used by render.RenderTags exclusively.
func extractTags(tc map[string]int, tl map[string][]listing.Listing) []tag.Tag {
	tags := []tag.Tag{}
	keys := make([]string, 0, len(tc))
	for k := range tc {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		tags = append(tags, tag.Tag{Name: k, Count: tc[k], Links: tl[k]})
	}
	return tags
}

type (
	internalLinks     []string
	internalLinksDeps struct {
		path string
	}
)

// Reconstruct link into full input path and record them
func (i *internalLinks) extract(deps internalLinksDeps, a map[string]bool) (map[string]bool, error) {
	for _, v := range *i {
		// Absolutize image links
		ref, err := filepath.Abs(deps.path + "/" + v)
		if err != nil {
			return nil, err
		}
		a[ref] = true
	}
	return a, nil
}

type (
	backLinks     []string
	backLinksDeps struct {
		href        string
		title       string
		description string
		path        string
	}
)

// Construct backlinks
// NOTE: Backlinks can be relative e.g. [[../blah]]
func (b *backLinks) extract(deps backLinksDeps, a map[string]bool, d map[string]Entry,
) (map[string]bool, map[string]Entry, error) {
	for _, v := range *b {
		// Reconstruct wikilink into full input path
		ref, err := filepath.Abs(deps.path + "/" + v)
		if err != nil {
			return nil, nil, err
		}

		// Process links with extensions as external files
		// like images/gifs
		if len(filepath.Ext(ref)) > 0 {
			a[ref] = true
		}
		ref += ".md"

		// Save backlinks
		linkedEntry := d[ref]
		linkedEntry.Backlinks = append(
			linkedEntry.Backlinks,
			listing.Listing{
				Href:        deps.href,
				Title:       deps.title,
				Description: deps.description,
				IsDir:       false,
			},
		)
		d[ref] = linkedEntry
	}
	return a, d, nil
}

type (
	tags     []string
	tagsDeps struct {
		href        string
		title       string
		description string
	}
)

// Grab tags count (tc) and tags listing (tl)
// We process the final tags later - this is
// for the tags page
func (t *tags) split(
	deps tagsDeps, tc map[string]int, tl map[string][]listing.Listing,
) (map[string]int, map[string][]listing.Listing) {
	for _, v := range *t {
		tc[v] += 1
		tl[v] = append(tl[v], listing.Listing{
			Href:        deps.href,
			Title:       deps.title,
			Description: deps.description,
			IsDir:       false,
		})
	}
	return tc, tl
}
