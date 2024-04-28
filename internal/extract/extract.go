package extract

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mattn/go-zglob"
	"github.com/mstcl/pher.git/internal/listing"
	"github.com/mstcl/pher.git/internal/parse"
	"github.com/mstcl/pher.git/internal/tags"
	"github.com/mstcl/pher.git/internal/util"
)

// Process files to build content c, metadata m, links l.
func Extract(files []string, inDir string, isHighlight bool) (
	map[string]parse.Metadata,
	map[string][]byte,
	map[string][]listing.Listing,
	[]tags.Tag,
	map[string][]listing.Listing,
	map[string]string,
	error,
) {
	// Metadata, content, backlinks and tags storing
	m := make(map[string]parse.Metadata)
	c := make(map[string][]byte)
	l := make(map[string][]listing.Listing)
	tc := make(map[string]int)
	tl := make(map[string][]listing.Listing)
	rl := make(map[string][]listing.Listing)
	h := make(map[string]string)

	for _, f := range files {
		// Read input file
		b, err := os.ReadFile(f)
		if err != nil {
			return nil,
				nil,
				nil,
				nil,
				nil,
				nil,
				fmt.Errorf("reading file: %w", err)
		}

		// Append to metadata
		md := parse.ParseMetadata(b)
		m[f] = md

		// Don't proceed if file is draft
		if md.Draft {
			continue
		}

		// Resolve basic vars
		path := util.GetFilePath(f)
		base := util.GetFileBase(f)
		href := util.ResolveHref(f, inDir)
		title := util.ResolveTitle(md.Title, base)

		// Append to hrefs

		h[f] = href

		// Parse for content
		html, err := parse.ParseFile(b, md.TOC, isHighlight)
		if err != nil {
			return nil,
				nil,
				nil,
				nil,
				nil,
				nil,
				fmt.Errorf("parse body: %w", err)
		}

		// Append to content
		c[f] = html

		// Append to tags
		for _, i := range md.Tags {
			tc[i] += 1
			tl[i] = append(tl[i], listing.Listing{
				Href:        href,
				Title:       title,
				Description: md.Description,
				IsDir:       false,
			})
		}

		// Extract wiki links
		links, err := parse.ParseWikilinks(b)
		if err != nil {
			return nil,
				nil,
				nil,
				nil,
				nil,
				nil,
				fmt.Errorf("extract links: %w", err)
		}

		// Append to backlinks l
		// Backlinks can be relative e.g. [[../blah]]
		for _, v := range links {
			// Reconstruct wikilink into full input path
			ref, err := filepath.Abs(path + "/" + v + ".md")
			if err != nil {
				return nil,
					nil,
					nil,
					nil,
					nil,
					nil,
					fmt.Errorf("resolving path: %w", err)
			}
			l[ref] = append(
				l[ref],
				listing.Listing{
					Href:        href,
					Title:       title,
					Description: md.Description,
					IsDir:       false,
				},
			)
		}
	}

	// Get related links
	for _, f := range files {
		if parse.IsDraft(m[f]) || len(m[f].Tags) == 0 {
			continue
		}
		rl[f] = getUniqueRelLinks(f, inDir, m[f], tl)
	}

	// Transform tags to right struct
	t := tags.GatherTags(tc, tl)
	return m, c, l, t, rl, h, nil
}

// For file f, find related links based on tags (populated in tl)
func getUniqueRelLinks(
	f string,
	inDir string,
	m parse.Metadata,
	tl map[string][]listing.Listing,
) ([]listing.Listing) {
	l := []listing.Listing{}
	u := []listing.Listing{}

	// get all links under f's tags
	for _, t := range m.Tags {
		l = append(l, tl[t]...)
	}

	// remove self from l
	for _, j := range l {
		if inDir+j.Href+".md" == f {
			continue
		}
		u = append(u, j)
	}
	return u
}

// For every index file present under subdir, populate its children to render
// the listings
func ExtractIndexListing(
	inDir string,
	m map[string]parse.Metadata,
) (map[string][]listing.Listing, error) {
	ls := make(map[string][]listing.Listing)
	files, err := zglob.Glob(inDir + "/**/*")
	files = append(files, inDir)
	if err != nil {
		_ = fmt.Errorf("glob files: %w", err)
	}
	// Go through everything that aren't files
	// Glob those directories for both files and directories
	for _, dir := range files {
		// Stat files/directories
		st, err := os.Stat(dir)
		if err != nil {
			return nil, fmt.Errorf("stat files: %w", err)
		}

		// Only process directories
		if st.Mode().IsRegular() {
			continue
		}

		// Glob under directory
		subfiles, err := filepath.Glob(dir + "/*")
		if err != nil {
			return nil, fmt.Errorf("glob files: %w", err)
		}

		// Use source file as key for consistency
		dir = dir + "/" + "index.md"

		// Now go through all globbed files/directories
		for _, f := range subfiles {
			// Don't need to process index files or unlisted ones
			if util.GetFileBase(f) == "index" || m[f].Unlisted {
				continue
			}

			// Create the entry
			ld, err := makeListingEntry(f, inDir, m[f])
			if err != nil {
				return nil, fmt.Errorf("creating listing entry: %w", err)
			}

			// Append to ls
			if m[f].Pinned {
				ls[dir] = append([]listing.Listing{ld}, ls[dir]...)
				continue
			}
			ls[dir] = append(ls[dir], ld)
		}
	}
	return ls, err
}

// Create a listing entry for f
func makeListingEntry(f string, inDir string, m parse.Metadata) (listing.Listing, error) {
	// Stat files/directories
	fst, err := os.Stat(f)
	if err != nil {
		return listing.Listing{}, fmt.Errorf("stat files: %w", err)
	}

	ld := listing.Listing{}

	// Grab href target, different for file vs. dir
	ld.IsDir = fst.Mode().IsDir()
	switch mode := fst.Mode(); {
	case mode.IsDir():
		target := util.GetRelativeFilePath(f, inDir)
		ld.Href = target
		ld.Title = target
		// Switch target to index for title & description
		f += "/index.md"
	case mode.IsRegular():
		ld.Href = util.ResolveHref(f, inDir)
	}

	// Grab titles and description.
	// If metadata has title -> use that.
	// If not -> use filename only if entry is not a directory
	if len(m.Title) > 0 {
		ld.Title = m.Title
	} else if !ld.IsDir {
		ld.Title = util.GetFileBase(f)
	}
	ld.Description = m.Description

	return ld, nil
}
