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
	C       *config.Config
	D       map[string]Entry
	Missing map[string]bool
	Skip    map[string]bool
	L       map[string][]listing.Listing
	InDir   string
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
func (m *ListDeps) listEntries(
	inDir string,
) error {
	// Initialize maps
	m.L = make(map[string][]listing.Listing)
	m.Missing = make(map[string]bool)
	m.Skip = make(map[string]bool)

	files, err := zglob.Glob(inDir + "/**/*")
	if err != nil {
		return err
	}
	files = append(files, inDir)

	// Go through everything that aren't files
	// Glob those directories for both files and directories
	// These are PARENTS with listings
	for _, dir := range files {
		// Stat files/directories
		st, err := os.Stat(dir)
		if err != nil {
			return err
		}

		// Only process directories
		if st.Mode().IsRegular() {
			continue
		}

		// Glob under directory
		children, err := filepath.Glob(dir + "/*")
		if err != nil {
			return err
		}
		if err := m.listEntriesChildren(inDir, dir, children); err != nil {
			return err
		}
	}
	return nil
}

// Sub-function to loop through depth 1 children inside the current parent
// (parentDir) to populate and return the listing map, the missing map, and the
// skip map. Additional calls constructListingEntry() to make individual listing
// entry.
func (m *ListDeps) listEntriesChildren(
	inDir string,
	parentDir string,
	files []string,
) error {
	// Whether to render children
	// Use source file as key for consistency
	dirI := parentDir + "/" + "index.md"
	isLog := m.D[dirI].Metadata.Layout == "log"
	for _, f := range files {
		// Stat files/directories
		fst, err := os.Stat(f)
		if err != nil {
			return err
		}
		IsDir := fst.Mode().IsDir()

		// Skip hidden files
		if rel, _ := filepath.Rel(inDir, f); rel[0] == 46 {
			continue
		}

		// Skip non-markdon files
		if !IsDir && filepath.Ext(f) != ".md" {
			continue
		}

		// Skip index files, unlisted ones
		if convert.FileBase(f) == "index" || m.D[f].Metadata.Unlisted {
			continue
		}

		// Don't render these files later
		m.Skip[f] = isLog

		// Throw error if parent's view is Log but child is subdirectory
		if IsDir && isLog {
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
				m.Missing[f+"/index.md"] = true
			}
		}

		// Prepare the listing
		ld := listing.Listing{}

		// Grab href target, different for file vs. dir
		ld.IsDir = IsDir

		// Construct the rest of the listing entry fields
		ld, err = m.constructListing(ld, f, parentDir, isLog)
		if err != nil {
			return err
		}

		// Now we act on the index files
		if IsDir {
			f = f + "/index.md"
		}

		// Append to ls
		if m.D[f].Metadata.Pinned {
			m.L[dirI] = append([]listing.Listing{ld}, m.L[dirI]...)
			continue
		}
		m.L[dirI] = append(m.L[dirI], ld)
	}
	return nil
}

// Complete a listing entry for a given child. Additionally add in relevant
// rendering data fields like html body and tags for parents with log view
// configured. Returns just a single listing corresponding to the given child.
func (m *ListDeps) constructListing(
	ld listing.Listing,
	f string,
	inDir string,
	isLog bool,
) (listing.Listing, error) {
	// Get Href
	if ld.IsDir {
		target, err := filepath.Rel(inDir, f)
		if err != nil {
			return listing.Listing{}, err
		}
		ld.Title = target
		if m.C.IsExt {
			target += "/index.html"
		}
		ld.Href = target
		// Switch target to index for title & description
		f += "/index.md"
	} else {
		target := convert.Href(f, inDir, false)
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
		ld.Title = convert.FileBase(f)
	}
	ld.Description = m.D[f].Metadata.Description

	// Log entries for log layout

	var err error
	if isLog {
		ld.Body = template.HTML(m.D[f].Body)
		date := m.D[f].Metadata.Date
		if len(date) > 0 {
			ld.Date, ld.MachineDate, err = convert.Date(date)
			if err != nil {
				return listing.Listing{}, err
			}
		}
		dateUpdated := m.D[f].Metadata.DateUpdated
		if len(dateUpdated) > 0 {
			ld.DateUpdated, ld.MachineDateUpdated, err = convert.Date(dateUpdated)
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
func (m *ListDeps) List(files []string) (
	[]string,
	error,
) {
	if err := m.listEntries(m.InDir); err != nil {
		return nil, err
	}

	// Update files
	for f := range m.Missing {
		entry := m.D[f]

		// add index to our files to render
		files = append(files, f)
		md := parse.DefaultMetadata()

		// we have inDir/a/b/c/index.md
		// want to extract c
		// i.e. title is the folder name

		// inDir/a/b/c/index.md -> a/b/c/index.md
		fn, _ := filepath.Rel(m.InDir, f)

		// a/b/c/index.md -> a/b/c -> a/b, c
		_, dir := filepath.Split(filepath.Dir(fn))

		// title is c
		md.Title = dir

		// update Metadata
		entry.Metadata = md

		// update record
		m.D[f] = entry
	}
	return files, nil
}
