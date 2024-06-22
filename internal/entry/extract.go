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
	Config   *config.Config
	InDir    string
	OutDir   string
	Entries  map[string]Entry
	Assets   map[string]bool
	Tags     []tag.Tag
	Metadata parse.Metadata
}

// Process files to build up the entry data for all files, the tags data, and
// the linked internal asset.
// Exclusive calls to parse.* are made here.
// Calls parse.ParseMetadata() to grab metadata.
// Calls parse.ParseSource() to grab html body.
// Calls parse.ParseInternalLinks() to grab backlinks and internal links.
// Additionally construct the backlinks, relatedlinks, asset map and tags slice
func (d *ExtractDeps) ExtractEntries(files []string) error {
	d.Entries = make(map[string]Entry) // entry data
	d.Assets = make(map[string]bool)   // internal assets

	// tagsCount: tags count (key: tag name)
	tagsCount := make(map[string]int)

	// tagsListing: tags listing - files with this tag (key: tag name)
	tagsListing := make(map[string][]listing.Listing)

	// First loop, can do most things
	for _, f := range files {
		entry := d.Entries[f]

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
		src := parse.Source{Body: buf.Bytes(), RendersHighlight: d.Config.CodeHighlight}
		md, err := src.ParseMetadata()
		if err != nil {
			return err
		}
		src.RendersTOC = md.TOC

		// Don't proceed if file is draft
		if md.Draft {
			continue
		}

		// Extract and parse html body
		html, err := src.ParseSource()
		if err != nil {
			return err
		}

		// Extract wiki backlinks (blinks) and image links (il)
		var il internalLinks
		var bl backLinks

		bl, il, err = src.ParseInternalLinks()
		if err != nil {
			return err
		}

		// Resolve basic vars
		path := filepath.Dir(f)
		base := convert.FileBase(f)
		title := convert.Title(md.Title, base)
		href := convert.Href(f, d.InDir, true)
		if d.Config.IsExt {
			href += ".html"
		}

		// Update entry
		entry.Metadata = md
		entry.Body = html
		entry.Href = href
		d.Entries[f] = entry

		// Update assets from internal links
		d.Assets, err = il.extract(internalLinksDeps{path: path}, d.Assets)
		if err != nil {
			return err
		}

		// Update assets and wikilinks from backlinks
		d.Assets, d.Entries, err = bl.extract(backLinksDeps{
			href:        href,
			title:       title,
			description: md.Description,
			path:        path,
		}, d.Assets, d.Entries)
		if err != nil {
			return err
		}

		// entryTags: store updated tags count and tags listing
		var entryTags tags = md.Tags

		tagsCount, tagsListing = entryTags.split(tagsDeps{
			href:        href,
			title:       title,
			description: md.Description,
		}, tagsCount, tagsListing)
	}

	// Second loop for related links
	//
	// NOTE: Entries that share tags are related
	// Hence dependent on tags listing (tl)
	for _, f := range files {
		entry := d.Entries[f]
		if entry.Metadata.Draft || len(entry.Metadata.Tags) == 0 {
			continue
		}

		// l: all related links
		l := []listing.Listing{}

		// rl: unique related links
		rl := []listing.Listing{}

		// Get all links under f's tags
		for _, t := range entry.Metadata.Tags {
			l = append(l, tagsListing[t]...)
		}

		// Remove self from l to ensure uniqueness
		for _, j := range l {
			fn := strings.TrimSuffix(j.Href, filepath.Ext(j.Href))
			if d.InDir+fn+".md" == f {
				continue
			}
			rl = append(rl, j)
		}

		entry.Relatedlinks = rl

		// Update entry
		d.Entries[f] = entry
	}

	// Transform tags to right format
	// a list of listing.Tag
	d.Tags = extractTags(tagsCount, tagsListing)

	return nil
}

// Transform maps of tags count and tags listing to give a sorted slice of tags.
// Used by render.RenderTags exclusively.
func extractTags(tagsCount map[string]int, tagsListing map[string][]listing.Listing) []tag.Tag {
	tags := []tag.Tag{}
	keys := make([]string, 0, len(tagsCount))
	for k := range tagsCount {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		tags = append(tags, tag.Tag{Name: k, Count: tagsCount[k], Links: tagsListing[k]})
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
func (i *internalLinks) extract(deps internalLinksDeps, assets map[string]bool) (map[string]bool, error) {
	for _, v := range *i {
		// Absolutize image links
		ref, err := filepath.Abs(deps.path + "/" + v)
		if err != nil {
			return nil, err
		}
		assets[ref] = true
	}
	return assets, nil
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
func (b *backLinks) extract(deps backLinksDeps, assets map[string]bool, entry map[string]Entry,
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
			assets[ref] = true
		}
		ref += ".md"

		// Save backlinks
		linkedEntry := entry[ref]
		linkedEntry.Backlinks = append(
			linkedEntry.Backlinks,
			listing.Listing{
				Href:        deps.href,
				Title:       deps.title,
				Description: deps.description,
				IsDir:       false,
			},
		)
		entry[ref] = linkedEntry
	}
	return assets, entry, nil
}

type (
	tags     []string
	tagsDeps struct {
		href        string
		title       string
		description string
	}
)

// Grab tags count and tags listing
// We process the final tags later - this is
// for the tags page
func (t *tags) split(
	deps tagsDeps, tagsCount map[string]int, tagsListing map[string][]listing.Listing,
) (map[string]int, map[string][]listing.Listing) {
	for _, v := range *t {
		tagsCount[v] += 1
		tagsListing[v] = append(tagsListing[v], listing.Listing{
			Href:        deps.href,
			Title:       deps.title,
			Description: deps.description,
			IsDir:       false,
		})
	}
	return tagsCount, tagsListing
}
