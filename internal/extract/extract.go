package extract

import (
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"sort"

	"github.com/mattn/go-zglob"
	"github.com/mstcl/pher/internal/listing"
	"github.com/mstcl/pher/internal/parse"
	"github.com/mstcl/pher/internal/util"
)

// Process files to build content c, metadata m, links l.
func Extract(files []string, inDir string, isHighlight bool, isExt bool) (
	map[string]parse.Metadata,
	map[string][]byte,
	map[string][]listing.Listing,
	[]listing.Tag,
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
				fmt.Errorf("error reading file %s: %w", f, err)
		}

		// Append to metadata
		md, err := parse.ParseMetadata(b)
		if err != nil {
			return nil,
				nil,
				nil,
				nil,
				nil,
				nil,
				nil,
				err
		}
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
				err
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
				err
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
					fmt.Errorf("error resolving path: %w", err)
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
	t := extractTags(tc, tl)
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
		if inDir+util.RemoveExtension(j.Href)+".md" == f {
			continue
		}
		u = append(u, j)
	}
	return u
}

// For every index file present under subdir, populate its children to render
// the listings, also return those with missing listing with format
// `map[string]bool` where key is the index path that is missing.
func extractIndexListing(
	inDir string,
	m map[string]parse.Metadata,
	c map[string][]byte,
	isExt bool,
) (
	map[string][]listing.Listing,
	map[string]bool,
	map[string]bool,
	error) {
	ls := make(map[string][]listing.Listing)
	missing := make(map[string]bool)
	skip := make(map[string]bool)
	files, err := zglob.Glob(inDir + "/**/*")
	files = append(files, inDir)
	if err != nil {
		return nil, nil, nil,
			fmt.Errorf("error globbing files %s/**/*: %w", inDir, err)
	}
	// Go through everything that aren't files
	// Glob those directories for both files and directories
	// These are PARENTS with listings
	for _, dir := range files {
		// Stat files/directories
		st, err := os.Stat(dir)
		if err != nil {
			return nil, nil, nil,
				fmt.Errorf("error when stat file or directory %s: %w", dir, err)
		}

		// Only process directories
		if st.Mode().IsRegular() {
			continue
		}

		// Glob under directory
		children, err := filepath.Glob(dir + "/*")
		if err != nil {
			return nil, nil, nil,
				fmt.Errorf("error globbing files %s/*: %w", dir, err)
		}
		ls, missing, skip, err = processChildrenEntries(
			inDir, dir, isExt, m, c, children, missing, ls, skip,
		)
		if err != nil {
			return nil, nil, nil, err
		}
	}
	return ls, missing, skip, err
}

// Now go through all children under a parent directory
func processChildrenEntries(
	inDir string,
	parentDir string,
	isExt bool,
	m map[string]parse.Metadata,
	c map[string][]byte,
	files []string,
	missing map[string]bool,
	ls map[string][]listing.Listing,
	skip map[string]bool,
) (map[string][]listing.Listing, map[string]bool, map[string]bool, error) {
	// Whether to render children
	// Use source file as key for consistency
	dirI := parentDir + "/" + "index.md"
	isLog := m[dirI].Layout == "log"
	for _, f := range files {
		// Stat files/directories
		fst, err := os.Stat(f)
		if err != nil {
			return nil, nil, nil,
				fmt.Errorf("error when stat file or directory %s: %w", f, err)
		}
		IsDir := fst.Mode().IsDir()

		// Skip hidden files
		if util.GetRelativeFilePath(f, inDir)[0] == 46 {
			continue
		}

		// Skip non-markdon files
		if !IsDir && util.GetFileExt(f) != ".md" {
			continue
		}

		// Skip index files, unlisted ones, and ones that utilize
		// log listing
		if util.GetFileBase(f) == "index" || m[f].Unlisted {
			continue
		}

		// Skip directories without any entry (markdown files)
		// Throw error if parent's view is Log but child is subdirectory
		// Append to missing if index doesn't exist
		if IsDir {
			if isLog {
				return nil, nil, nil,
					fmt.Errorf("subdirectory detected in log directory! abort")
			}
			entryPresent, err := util.IsEntryPresent(f)
			if err != nil {
				return nil, nil, nil, fmt.Errorf("glob files: %w", err)
			}
			if !entryPresent {
				continue
			}
			indexExists, err := util.IsFileExist(f + "/index.md")
			if err != nil {
				return nil, nil, nil,
					fmt.Errorf(
						"error when stat file or directory %s/index.md: %w",
						f, err,
					)
			}
			if !indexExists {
				missing[f+"/index.md"] = true
			}
		} else {
			skip[f] = isLog
		}

		ld := listing.Listing{}

		// Grab href target, different for file vs. dir
		ld.IsDir = IsDir

		// Create the entry
		ld, err = makeListingEntry(ld, f, parentDir, m, c, isExt, isLog)
		if err != nil {
			return nil, nil, nil,
				fmt.Errorf("error creating listing entry for %s: %w", f, err)
		}

		if IsDir {
			f = f + "/index.md"
		}

		// Append to ls
		if m[f].Pinned {
			ls[dirI] = append([]listing.Listing{ld}, ls[dirI]...)
			continue
		}
		ls[dirI] = append(ls[dirI], ld)
	}
	return ls, missing, skip, nil
}

// Create a listing entry for f
func makeListingEntry(
	ld listing.Listing,
	f string,
	inDir string,
	m map[string]parse.Metadata,
	c map[string][]byte,
	isExt bool,
	isLog bool,
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

	var err error
	if isLog {
		ld.Body = template.HTML(c[f])
		if len(m[f].Date) > 0 {
			ld.Date, ld.MachineDate, err = util.ResolveDate(m[f].Date)
			if err != nil {
				return listing.Listing{}, fmt.Errorf("time parse: %w", err)
			}
		}
		if len(m[f].DateUpdated) > 0 {
			ld.DateUpdated, ld.MachineDateUpdated, err = util.ResolveDate(m[f].DateUpdated)
			if err != nil {
				return listing.Listing{}, fmt.Errorf("time parse: %w", err)
			}
		}
		ld.Tags = m[f].Tags
	}

	return ld, nil
}

// Populate listing for indexes
// Additionally make listings if there are none
func FetchListingsCreateMissing(files []string, inDir string, m map[string]parse.Metadata, c map[string][]byte, isExt bool) ([]string, map[string][]listing.Listing, map[string]bool, error) {
	l, missing, skip, err := extractIndexListing(inDir, m, c, isExt)
	if err != nil {
		return nil, nil, nil, err
	}
	for k := range missing {
		files = append(files, k)
		md := parse.DefaultMetadata()
		md.Title = util.GetFilePath(util.GetRelativeFilePath(k, inDir))
		m[k] = md
	}
	return files, l, skip, nil
}

// Transform map t to list of sorted tags
func extractTags(tc map[string]int, tl map[string][]listing.Listing) []listing.Tag {
	tags := []listing.Tag{}
	keys := make([]string, 0, len(tc))
	for k := range tc {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		tags = append(tags, listing.Tag{Name: k, Count: tc[k], Links: tl[k]})
	}
	return tags
}
