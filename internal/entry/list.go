// Format the list of sub entries for any given entry
package entry

import (
	"html/template"
	"os"
	"path/filepath"

	"github.com/mattn/go-zglob"
	"github.com/mstcl/pher/internal/config"
	"github.com/mstcl/pher/internal/convert"
	"github.com/mstcl/pher/internal/ioutil"
	"github.com/mstcl/pher/internal/listing"
	"github.com/mstcl/pher/internal/parse"
)

// Required dependencies
type ListDeps struct {
	Config   *config.Config
	Entries  map[string]Entry
	Missing  map[string]bool
	Skip     map[string]bool
	Listings map[string][]listing.Listing
	InDir    string
}

// Get all nested directories (the parents), and call
// extractChildrenEntries() to populate the children.
//
// * listing: listing entries of parents.
//
// * missing: bool map of parent index paths that are missing.
//
// * skip: bool map of files that should not be rendered (because its parents
// is displaying a log)
func (d *ListDeps) listEntries(
	inDir string,
) error {
	// Initialize maps
	d.Listings = make(map[string][]listing.Listing)
	d.Missing = make(map[string]bool)
	d.Skip = make(map[string]bool)

	files, err := zglob.Glob(inDir + "/**/*")
	if err != nil {
		return err
	}
	files = append(files, inDir)

	// Go through everything that aren't files
	// Glob those directories for both files and directories
	// These are PARENTS with listings
	for _, f := range files {
		// Stat files/directories
		info, err := os.Stat(f)
		if err != nil {
			return err
		}

		// Only process directories
		if info.Mode().IsRegular() {
			continue
		}

		// Glob under directory
		children, err := filepath.Glob(f + "/*")
		if err != nil {
			return err
		}
		if err := d.listEntriesChildren(inDir, f, children); err != nil {
			return err
		}
	}
	return nil
}

// Sub-function to loop through depth 1 children inside the current parent
// (parentDir) to populate and return the listing map, the missing map, and the
// skip map. Additional calls constructListingEntry() to make individual listing
// entry.
func (d *ListDeps) listEntriesChildren(
	inDir string,
	parentDir string,
	files []string,
) error {
	// Whether to render children
	// Use source file as key for consistency
	dirIndex := parentDir + "/" + "index.md"
	asLog := d.Entries[dirIndex].Metadata.Layout == "log"
	for _, f := range files {
		// Stat files/directories
		info, err := os.Stat(f)
		if err != nil {
			return err
		}
		IsDir := info.Mode().IsDir()

		// Skip hidden files
		if rel, _ := filepath.Rel(inDir, f); rel[0] == 46 {
			continue
		}

		// Skip non-markdon files
		if !IsDir && filepath.Ext(f) != ".md" {
			continue
		}

		// Skip index files, unlisted ones
		if convert.FileBase(f) == "index" || d.Entries[f].Metadata.Unlisted {
			continue
		}

		// Don't render these files later
		d.Skip[f] = asLog

		// Throw error if parent's view is Log but child is subdirectory
		if IsDir && asLog {
			return err
		}

		// Skip directories without any entry (markdown files)
		entryPresent, err := ioutil.IsEntryPresent(f)
		if err != nil {
			return err
		}
		if IsDir && !entryPresent {
			continue
		}

		// Append to missing if index doesn't exist
		if IsDir {
			indexExists, err := ioutil.IsFileExist(f + "/index.md")
			if err != nil {
				return err
			}
			if !indexExists {
				d.Missing[f+"/index.md"] = true
			}
		}

		// Prepare the listing
		l := listing.Listing{}

		// Grab href target, different for file vs. dir
		l.IsDir = IsDir

		// Construct the rest of the listing entry fields
		l, err = d.getListing(l, f, parentDir, asLog)
		if err != nil {
			return err
		}

		// Now we act on the index files
		if IsDir {
			f = f + "/index.md"
		}

		// Append to listing map
		if d.Entries[f].Metadata.Pinned {
			d.Listings[dirIndex] = append([]listing.Listing{l}, d.Listings[dirIndex]...)
			continue
		}
		d.Listings[dirIndex] = append(d.Listings[dirIndex], l)
	}
	return nil
}

// Complete a listing entry for a given child. Additionally add in relevant
// rendering data fields like html body and tags for parents with log view
// configured. Returns just a single listing corresponding to the given child.
func (d *ListDeps) getListing(
	l listing.Listing,
	f string,
	inDir string,
	isLog bool,
) (listing.Listing, error) {
	// Get Href
	if l.IsDir {
		target, err := filepath.Rel(inDir, f)
		if err != nil {
			return listing.Listing{}, err
		}
		l.Title = target
		if d.Config.IsExt {
			target += "/index.html"
		}
		l.Href = target
		// Switch target to index for title & description
		f += "/index.md"
	} else {
		target := convert.Href(f, inDir, false)
		if d.Config.IsExt {
			target += ".html"
		}
		l.Href = target
	}

	// Grab titles and description.
	// If metadata has title -> use that.
	// If not -> use filename only if entry is not a directory
	title := d.Entries[f].Metadata.Title
	if len(title) > 0 {
		l.Title = title
	} else if !l.IsDir {
		l.Title = convert.FileBase(f)
	}
	l.Description = d.Entries[f].Metadata.Description

	// Log entries for log layout

	var err error
	if isLog {
		l.Body = template.HTML(d.Entries[f].Body)
		date := d.Entries[f].Metadata.Date
		if len(date) > 0 {
			l.Date, l.MachineDate, err = convert.Date(date)
			if err != nil {
				return listing.Listing{}, err
			}
		}
		dateUpdated := d.Entries[f].Metadata.DateUpdated
		if len(dateUpdated) > 0 {
			l.DateUpdated, l.MachineDateUpdated, err = convert.Date(dateUpdated)
			if err != nil {
				return listing.Listing{}, err
			}
		}
		l.Tags = d.Entries[f].Metadata.Tags
	}

	return l, nil
}

// The main callable function for extract.go to call all the relevant functions
// to populate listing, update files to add missing indexes, and skipped files
func (d *ListDeps) List(files []string) (
	[]string,
	error,
) {
	if err := d.listEntries(d.InDir); err != nil {
		return nil, err
	}

	// Update files
	for f := range d.Missing {
		entry := d.Entries[f]

		// add index to our files to render
		files = append(files, f)
		md := parse.DefaultMetadata()

		// we have inDir/a/b/c/index.md
		// want to extract c
		// i.e. title is the folder name

		// inDir/a/b/c/index.md -> a/b/c/index.md
		rel, _ := filepath.Rel(d.InDir, f)

		// a/b/c/index.md -> a/b/c -> a/b, c
		_, dir := filepath.Split(filepath.Dir(rel))

		// title is c
		md.Title = dir

		// update Metadata
		entry.Metadata = md

		// update record
		d.Entries[f] = entry
	}
	return files, nil
}
