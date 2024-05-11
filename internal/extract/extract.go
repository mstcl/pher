package extract

import (
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"sort"

	"github.com/mattn/go-zglob"
	"github.com/mstcl/pher/internal/entry"
	"github.com/mstcl/pher/internal/listing"
	"github.com/mstcl/pher/internal/parse"
	"github.com/mstcl/pher/internal/tag"
	"github.com/mstcl/pher/internal/util"
)

// Process files to build up the entry data for all files, the tags data, and
// the linked internal asset.
// Exclusive calls to parse.* are made here.
// Calls parse.ParseMetadata() to grab metadata.
// Calls parse.ParseSource() to grab html body.
// Calls parse.ParseInternalLinks() to grab backlinks and internal links.
// Additionally construct the backlinks, relatedlinks, asset map and tags slice
//
// These are the returned data:
//
// d: entry.Entry (key: entry filename)
//
// i: linked internal assets (key: asset filename)
func ExtractEntry(files []string, inDir string, isHighlight bool, isExt bool) (
	map[string]entry.Entry,
	[]tag.Tag,
	map[string]bool,
	error,
) {
	d := make(map[string]entry.Entry)
	i := make(map[string]bool)

	// These are local data:
	// tc: tags count (key: tag name)
	// tl: tags listing - files with this tag (key: tag name)
	tc := make(map[string]int)
	tl := make(map[string][]listing.Listing)

	for _, f := range files {
		entry := d[f]
		// Read input file
		b, err := os.ReadFile(f)
		if err != nil {
			return nil,
				nil,
				nil,
				fmt.Errorf("error reading file %s: %w", f, err)
		}

		// Extract and save metadata
		md, err := parse.ParseMetadata(b)
		if err != nil {
			return nil,
				nil,
				nil,
				err
		}
		entry.Metadata = md

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

		// Save href
		entry.Href = href

		// Extract and parse html body
		html, err := parse.ParseSource(b, md.TOC, isHighlight)
		if err != nil {
			return nil,
				nil,
				nil,
				err
		}

		// Save html body
		entry.Body = html

		// Grab tags count (tc) and tags listing (tl)
		// We process the final tags later - this is
		// for the tags page
		for _, v := range md.Tags {
			tc[v] += 1
			tl[v] = append(tl[v], listing.Listing{
				Href:        href,
				Title:       title,
				Description: md.Description,
				IsDir:       false,
			})
		}

		// Extract wiki backlinks (blinks) and image links (ilinks)
		blinks, ilinks, err := parse.ParseInternalLinks(b)
		if err != nil {
			return nil,
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
					fmt.Errorf("error get absolute paths: %w", err)
			}
			i[ref] = true
		}

		// Construct backlinks
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
					fmt.Errorf("error get absolute paths: %w", err)
			}
			// Save backlinks
			linkedEntry := d[ref]
			linkedEntry.Backlinks = append(
				linkedEntry.Backlinks,
				listing.Listing{
					Href:        href,
					Title:       title,
					Description: md.Description,
					IsDir:       false,
				},
			)
			d[ref] = linkedEntry
		}
		d[f] = entry
	}

	// Construct and save related links
	// Entries that share tags are related
	// Hence dependent on tags listing (tl)
	for _, f := range files {
		entry := d[f]
		if parse.IsDraft(entry.Metadata) || len(entry.Metadata.Tags) == 0 {
			continue
		}
		entry.Relatedlinks = constructUniqueRelLinks(f, inDir, entry.Metadata, tl)
		d[f] = entry
	}

	// Transform tags to right format
	// a list of listing.Tag
	t := extractTags(tc, tl)

	return d, t, i, nil
}

// For a file f, find slice related links based on tags (populated in tl) such
// that the slice is unique and does not contain f
func constructUniqueRelLinks(
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

// Get all nested directories (the parents), and call
// extractChildrenEntries() to populate the children.
// Returns:
//
// * listing: listing entries of parents.
//
// * missing: bool map of parent index paths that are missing.
//
// * skip: bool map of files that should not be rendered (because its parents
// is displaying a log)
func extractParentListings(
	inDir string,
	d map[string]entry.Entry,
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
		ls, missing, skip, err = extractChildrenEntries(
			inDir, dir, isExt, d, children, missing, ls, skip,
		)
		if err != nil {
			return nil, nil, nil, err
		}
	}
	return ls, missing, skip, err
}

// Sub-function to loop through depth 1 children inside the current parent
// (parentDir) to populate and return the listing map, the missing map, and the
// skip map. Additional calls constructListingEntry() to make individual listing
// entry.
func extractChildrenEntries(
	inDir string,
	parentDir string,
	isExt bool,
	d map[string]entry.Entry,
	files []string,
	missing map[string]bool,
	ls map[string][]listing.Listing,
	skip map[string]bool,
) (map[string][]listing.Listing, map[string]bool, map[string]bool, error) {
	// Whether to render children
	// Use source file as key for consistency
	dirI := parentDir + "/" + "index.md"
	isLog := d[dirI].Metadata.Layout == "log"
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
		if util.GetFileBase(f) == "index" || d[f].Metadata.Unlisted {
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

		// Construct the listing entry
		ld, err = constructListingEntry(ld, f, parentDir, d, isExt, isLog)
		if err != nil {
			return nil, nil, nil,
				fmt.Errorf("error creating listing entry for %s: %w", f, err)
		}

		if IsDir {
			f = f + "/index.md"
		}

		// Append to ls
		if d[f].Metadata.Pinned {
			ls[dirI] = append([]listing.Listing{ld}, ls[dirI]...)
			continue
		}
		ls[dirI] = append(ls[dirI], ld)
	}
	return ls, missing, skip, nil
}

// Complete a listing entry for a given child. Additionally add in relevant
// rendering data fields like html body and tags for parents with log view
// configured. Returns just a single listing corresponding to the given child.
func constructListingEntry(
	ld listing.Listing,
	f string,
	inDir string,
	d map[string]entry.Entry,
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
	title := d[f].Metadata.Title
	if len(title) > 0 {
		ld.Title = title
	} else if !ld.IsDir {
		ld.Title = util.GetFileBase(f)
	}
	ld.Description = d[f].Metadata.Description

	var err error
	if isLog {
		ld.Body = template.HTML(d[f].Body)
		date := d[f].Metadata.Date
		if len(date) > 0 {
			ld.Date, ld.MachineDate, err = util.ResolveDate(date)
			if err != nil {
				return listing.Listing{}, err
			}
		}
		dateUpdated := d[f].Metadata.DateUpdated
		if len(dateUpdated) > 0 {
			ld.DateUpdated, ld.MachineDateUpdated, err = util.ResolveDate(dateUpdated)
			if err != nil {
				return listing.Listing{}, err
			}
		}
		ld.Tags = d[f].Metadata.Tags
	}

	return ld, nil
}

// The main callable function for extract.go to call all the relevant functions
// to populate listing, missing indexes, and skipped files
func ExtractAllListings(
	files []string,
	inDir string,
	d map[string]entry.Entry,
	isExt bool,
) ([]string, map[string][]listing.Listing, map[string]bool, error) {
	l, missing, skip, err := extractParentListings(inDir, d, isExt)
	if err != nil {
		return nil, nil, nil, err
	}
	for f := range missing {
		entry := d[f]
		files = append(files, f)
		md := parse.DefaultMetadata()
		md.Title = util.GetFilePath(util.GetRelativeFilePath(f, inDir))
		entry.Metadata = md
		d[f] = entry
	}
	return files, l, skip, nil
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
