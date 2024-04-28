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
func Extract(files []string, inDir string, isHighlight bool, isExt bool) (
	map[string]parse.Metadata,
	map[string][]byte,
	map[string][]listing.Listing,
	[]tags.Tag,
	map[string][]listing.Listing,
	map[string]string,
	map[string]bool,
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
	i := make(map[string]bool)

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
		href := util.ResolveHref(f, inDir, true)
		if isExt {
			href += ".html"
		}
		title := util.ResolveTitle(md.Title, base)

		// Append to hrefs

		h[f] = href

		// Parse for content
		html, err := parse.ParseSource(b, md.TOC, isHighlight)
		if err != nil {
			return nil,
				nil,
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
		for _, v := range md.Tags {
			tc[v] += 1
			tl[v] = append(tl[v], listing.Listing{
				Href:        href,
				Title:       title,
				Description: md.Description,
				IsDir:       false,
			})
		}

		// Extract wiki links
		// links, err := parse.ParseWikilinks(b)
		// if err != nil {
		// 	return nil,
		// 		nil,
		// 		nil,
		// 		nil,
		// 		nil,
		// 		nil,
		// 		fmt.Errorf("extract links: %w", err)
		// }

		// Extract wiki blinks
		blinks, ilinks, err := parse.ParseInternalLinks(b)
		if err != nil {
			return nil,
				nil,
				nil,
				nil,
				nil,
				nil,
				nil,
				fmt.Errorf("extract links: %w", err)
		}

		// Absolutize image links
		for _, v := range ilinks {
			// Reconstruct link into full input path
			ref, err := filepath.Abs(path + "/" + v)
			if err != nil {
				return nil,
					nil,
					nil,
					nil,
					nil,
					nil,
					nil,
					fmt.Errorf("resolving path: %w", err)
			}
			i[ref] = true
		}

		// Append to backlinks l
		// Backlinks can be relative e.g. [[../blah]]
		for _, v := range blinks {
			// Reconstruct wikilink into full input path
			ref, err := filepath.Abs(path + "/" + v)
			// Process links with extensions as external files
			// like images/gifs
			if len(util.GetFileExt(ref)) > 0 {
				i[ref] = true
			}
			ref += ".md"
			if err != nil {
				return nil,
					nil,
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
	return m, c, l, t, rl, h, i, nil
}

// For file f, find related links based on tags (populated in tl)
func getUniqueRelLinks(
	f string,
	inDir string,
	m parse.Metadata,
	tl map[string][]listing.Listing,
) []listing.Listing {
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
// the listings, also return those with missing listing with format
// `map[string]bool` where key is the index path that is missing.
func ExtractIndexListing(
	inDir string,
	m map[string]parse.Metadata,
	isExt bool,
) (
	map[string][]listing.Listing,
	map[string]bool,
	error,) {
	ls := make(map[string][]listing.Listing)
	missing := make(map[string]bool)
	files, err := zglob.Glob(inDir + "/**/*")
	files = append(files, inDir)
	if err != nil {
		return nil, nil, fmt.Errorf("glob files: %w", err)
	}
	// Go through everything that aren't files
	// Glob those directories for both files and directories
	for _, dir := range files {
		// Stat files/directories
		st, err := os.Stat(dir)
		if err != nil {
			return nil, nil, fmt.Errorf("stat files: %w", err)
		}

		// Only process directories
		if st.Mode().IsRegular() {
			continue
		}

		// Glob under directory
		subfiles, err := filepath.Glob(dir + "/*")
		if err != nil {
			return nil, nil, fmt.Errorf("glob files: %w", err)
		}

		// Use source file as key for consistency
		dirI := dir + "/" + "index.md"

		// Now go through all globbed files/directories
		for _, f := range subfiles {
			// Stat files/directories
			fst, err := os.Stat(f)
			if err != nil {
				return nil, nil, fmt.Errorf("stat files: %w", err)
			}
			IsDir := fst.Mode().IsDir()

			// Skip hidden files
			if util.GetRelativeFilePath(f, inDir)[0] == 46 {
				continue
			}

			// Skip non-markdon files
			if util.GetFileExt(f) != ".md" && !IsDir {
				continue
			}

			// Skip index files or unlisted ones
			if util.GetFileBase(f) == "index" || m[f].Unlisted {
				continue
			}

			// Skip directories without any markdown files
			// Append to missing if index doesn't exist
			if IsDir {
				entryExists, err := util.IsEntryPresent(f)
				if err != nil {
					return nil, nil, fmt.Errorf("glob files: %w", err)
				}
				if !entryExists {
					continue
				}
				mf := f+"/index.md"
				indexExists, err := util.IsFileExist(mf)
				if err != nil {
					return nil, nil, fmt.Errorf("stat file: %w", err)
				}
				if !indexExists {
					missing[mf] = true
				}
			}

			ld := listing.Listing{}

			// Grab href target, different for file vs. dir
			ld.IsDir = IsDir

			// Create the entry
			ld, err = makeListingEntry(ld, f, dir, m, isExt)
			if err != nil {
				return nil, nil, fmt.Errorf("creating listing entry: %w", err)
			}

			// Append to ls
			if m[f].Pinned {
				ls[dirI] = append([]listing.Listing{ld}, ls[dirI]...)
				continue
			}
			ls[dirI] = append(ls[dirI], ld)
		}
	}
	return ls, missing, err
}

// Create a listing entry for f
func makeListingEntry(
	ld listing.Listing,
	f string,
	inDir string,
	m map[string]parse.Metadata,
	isExt bool,
) (listing.Listing, error) {
	// Get Href
	if ld.IsDir {
		target := util.GetRelativeFilePath(f, inDir)
		ld.Title = target
		if isExt {
			target += "/index.html"
		}
		ld.Href = target
		// Switch target to index for title & description
		f += "/index.md"
	} else {
		target := util.ResolveHref(f, inDir, false)
		if isExt {
			target += ".html"
		}
		ld.Href = target
	}

	// Grab titles and description.
	// If metadata has title -> use that.
	// If not -> use filename only if entry is not a directory
	if len(m[f].Title) > 0 {
		ld.Title = m[f].Title
	} else if !ld.IsDir {
		ld.Title = util.GetFileBase(f)
	}
	ld.Description = m[f].Description

	return ld, nil
}
