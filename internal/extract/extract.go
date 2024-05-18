package extract

import (
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"sort"

	"github.com/mattn/go-zglob"
	"github.com/mstcl/pher/internal/config"
	"github.com/mstcl/pher/internal/entry"
	"github.com/mstcl/pher/internal/listing"
	"github.com/mstcl/pher/internal/parse"
	"github.com/mstcl/pher/internal/tag"
	"github.com/mstcl/pher/internal/util"
)

type Meta struct {
	C      *config.Config
	InDir  string
	OutDir string
	D      map[string]entry.Entry
	M      parse.Metadata
}

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
func (m *Meta) ExtractEntry(files []string) (
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
		s := parse.Source{B: b, IsHighlight: m.C.CodeHighlight}
		md, err := s.ParseMetadata()
		if err != nil {
			return nil,
				nil,
				nil,
				err
		}
		s.IsTOC = md.TOC
		entry.Metadata = md

		// Don't proceed if file is draft
		if md.Draft {
			continue
		}

		// Resolve basic vars
		path := util.GetFilePath(f)
		base := util.GetFileBase(f)
		href := util.ResolveHref(f, m.InDir, true)
		if m.C.IsExt {
			href += ".html"
		}
		title := util.ResolveTitle(md.Title, base)

		// Save href
		entry.Href = href

		// Extract and parse html body
		html, err := s.ParseSource()
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
		blinks, ilinks, err := s.ParseInternalLinks()
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
		e := d[f]
		if e.Metadata.Draft || len(e.Metadata.Tags) == 0 {
			continue
		}
		e.Relatedlinks = m.constructUniqueRelLinks(f, tl)
		d[f] = e
	}

	// Transform tags to right format
	// a list of listing.Tag
	t := extractTags(tc, tl)

	return d, t, i, nil
}

// For a file f, find slice related links based on tags (populated in tl) such
// that the slice is unique and does not contain f
func (m *Meta) constructUniqueRelLinks(
	f string,
	tl map[string][]listing.Listing,
) []listing.Listing {
	l := []listing.Listing{}
	u := []listing.Listing{}

	// get all links under f's tags
	for _, t := range m.M.Tags {
		l = append(l, tl[t]...)
	}

	// remove self from l
	for _, j := range l {
		if m.InDir+util.RemoveExtension(j.Href)+".md" == f {
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
func (m *Meta) extractParentListings(
	inDir string,
) (
	map[string][]listing.Listing,
	map[string]bool,
	map[string]bool,
	error,
) {
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
		ls, missing, skip, err = m.extractChildrenEntries(
			inDir, dir, children, missing, ls, skip,
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
func (m *Meta) extractChildrenEntries(
	inDir string,
	parentDir string,
	files []string,
	missing map[string]bool,
	ls map[string][]listing.Listing,
	skip map[string]bool,
) (map[string][]listing.Listing, map[string]bool, map[string]bool, error) {
	// Whether to render children
	// Use source file as key for consistency
	dirI := parentDir + "/" + "index.md"
	isLog := m.D[dirI].Metadata.Layout == "log"
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
		if util.GetFileBase(f) == "index" || m.D[f].Metadata.Unlisted {
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
		ld, err = m.constructListingEntry(ld, f, parentDir, isLog)
		if err != nil {
			return nil, nil, nil,
				fmt.Errorf("error creating listing entry for %s: %w", f, err)
		}

		if IsDir {
			f = f + "/index.md"
		}

		// Append to ls
		if m.D[f].Metadata.Pinned {
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
func (m *Meta) constructListingEntry(
	ld listing.Listing,
	f string,
	inDir string,
	isLog bool,
) (listing.Listing, error) {
	// Get Href
	if ld.IsDir {
		target := util.GetRelativeFilePath(f, inDir)
		ld.Title = target
		if m.C.IsExt {
			target += "/index.html"
		}
		ld.Href = target
		// Switch target to index for title & description
		f += "/index.md"
	} else {
		target := util.ResolveHref(f, inDir, false)
		if m.C.IsExt {
			target += ".html"
		}
		ld.Href = target
	}

	// Grab titles and description.
	// If metadata has title -> use that.
	// If not -> use filename only if entry is not a directory
	title := m.D[f].Metadata.Title
	if len(title) > 0 {
		ld.Title = title
	} else if !ld.IsDir {
		ld.Title = util.GetFileBase(f)
	}
	ld.Description = m.D[f].Metadata.Description

	var err error
	if isLog {
		ld.Body = template.HTML(m.D[f].Body)
		date := m.D[f].Metadata.Date
		if len(date) > 0 {
			ld.Date, ld.MachineDate, err = util.ResolveDate(date)
			if err != nil {
				return listing.Listing{}, err
			}
		}
		dateUpdated := m.D[f].Metadata.DateUpdated
		if len(dateUpdated) > 0 {
			ld.DateUpdated, ld.MachineDateUpdated, err = util.ResolveDate(dateUpdated)
			if err != nil {
				return listing.Listing{}, err
			}
		}
		ld.Tags = m.D[f].Metadata.Tags
	}

	return ld, nil
}

// The main callable function for extract.go to call all the relevant functions
// to populate listing, update files to add missing indexes, and skipped files
func (m *Meta) ExtractAllListings(files []string) (
	[]string,
	map[string][]listing.Listing,
	map[string]bool,
	error,
) {
	l, missing, skip, err := m.extractParentListings(m.InDir)
	if err != nil {
		return nil, nil, nil, err
	}
	for f := range missing {
		entry := m.D[f]
		files = append(files, f)
		md := parse.DefaultMetadata()
		md.Title = util.GetFilePath(util.GetRelativeFilePath(f, m.InDir))
		entry.Metadata = md
		m.D[f] = entry
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
