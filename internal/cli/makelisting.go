package cli

import (
	"fmt"
	"html/template"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/mattn/go-zglob"
	"github.com/mstcl/pher/v2/internal/checks"
	"github.com/mstcl/pher/v2/internal/convert"
	"github.com/mstcl/pher/v2/internal/listing"
	"github.com/mstcl/pher/v2/internal/metadata"
	"github.com/mstcl/pher/v2/internal/state"
)

// Get all directories, and call listChildren() to populate the files within.
//
// * listing: listing entries of parents.
//
// * missing: bool map of parent index paths that are missing.
//
// * skip: bool map of files that should not be rendered (because its parents
// is displaying a log)
func makeFileListing(s *state.State, logger *slog.Logger) error {
	// Initialize missing map
	s.Missing = make(map[string]bool)

	files, err := zglob.Glob(s.InDir + "/**/*")
	if err != nil {
		return err
	}

	files = append(files, s.InDir)

	logger.Debug("found files to process listing", slog.Any("files", files))

	// Go through everything that aren't files
	// Glob those directories for both files and directories
	// These are PARENTS with listings
	for _, f := range files {
		child := logger.With(slog.String("filepath", f), slog.String("context", "file listing"))

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

		child.Debug("found children files", slog.Any("files", children))

		if err := makeFileListingHelper(s, &helperInput{
			parentDir: f,
			files:     children,
		}, logger); err != nil {
			return err
		}
	}

	// Update files
	for f := range s.Missing {
		entry := s.Entries[f]

		// add index to our files to render
		s.Files = append(s.Files, f)
		md := metadata.Default()

		// we have inDir/a/b/c/index.md
		// want to extract c
		// i.e. title is the folder name

		// inDir/a/b/c/index.md -> a/b/c/index.md
		rel, _ := filepath.Rel(s.InDir, f)

		// a/b/c/index.md -> a/b/c -> a/b, c
		_, dir := filepath.Split(filepath.Dir(rel))

		// title is c
		md.Title = dir

		// update Metadata
		entry.Metadata = *md

		// update record
		s.Entries[f] = entry
	}

	return nil
}

type helperInput struct {
	parentDir string
	files     []string
}

// Sub-function to loop through depth 1 children inside the current parent
// (parentDir) to populate and return the listing map, the missing map, and the
// skip map. Additional calls constructListingEntry() to make individual listing
// entry.
func makeFileListingHelper(
	s *state.State,
	i *helperInput,
	logger *slog.Logger,
) error {
	// Whether to render children
	// Use source file as key for consistency
	dirIndex := filepath.Join(i.parentDir, "index.md")
	isLog := s.Entries[dirIndex].Metadata.Layout == "log"

	for _, f := range i.files {
		child := logger.With(slog.String("filepath", f), slog.String("context", "child listing"))

		// Stat files/directories
		info, err := os.Stat(f)
		if err != nil {
			return err
		}

		IsDir := info.Mode().IsDir()

		// Skip hidden files
		if rel, _ := filepath.Rel(s.InDir, f); rel[0] == 46 {
			child.Debug("skipped hidden file")

			continue
		}

		// Skip non-markdon files
		if !IsDir && filepath.Ext(f) != ".md" {
			child.Debug("skipped non markdown file")

			continue
		}

		// Skip index files, unlisted ones
		if convert.FileBase(f) == "index" || s.Entries[f].Metadata.Unlisted {
			child.Debug("skip index files and unlisted files")

			continue
		}

		// Don't render these files later
		s.Skip[f] = isLog

		// Throw error if parent's view is Log but child is subdirectory
		if IsDir && isLog {
			child.Error("found a directory in log parent - this is unexpected")

			return err
		}

		// Skip directories without any entry (markdown files)
		entryPresent, err := checks.EntryPresent(f)
		if err != nil {
			return err
		}

		if IsDir && !entryPresent {
			child.Debug("empty directory found - skipping")

			continue
		}

		// Append to missing index if index doesn't exist for a directory
		if IsDir {
			indexFile := filepath.Join(f, "/index.md")

			_, err := os.Stat(indexFile)
			if os.IsNotExist(err) {
				s.Missing[f+"/index.md"] = true
				child.Debug("index doesn't exist, added to missing index state")
			} else if err != nil {
				return fmt.Errorf("stat %s: %w", s.ConfigFile, err)
			}
		}

		// Prepare the listing
		l := listing.Listing{}

		// Grab href target, different for file vs. dir
		l.IsDir = IsDir

		// Construct the rest of the listing entry fields. Additionally add in
		// relevant rendering data fields like html body and tags for parents
		// with log view configured.
		if l.IsDir {
			target, err := filepath.Rel(i.parentDir, f)
			if err != nil {
				return err
			}

			l.Title = target

			if s.Config.IsExt {
				target += "/index.html"
			}

			l.Href = target
			// Switch target to index for title & description
			f = filepath.Join(f, "index.md")
		} else {
			target := convert.Href(f, i.parentDir, false)
			if s.Config.IsExt {
				target += ".html"
			}

			l.Href = target
		}

		// Grab titles and description.
		// If metadata has title -> use that.
		// If not -> use filename only if entry is not a directory
		title := s.Entries[f].Metadata.Title
		if len(title) > 0 {
			l.Title = title
		} else if !l.IsDir {
			l.Title = convert.FileBase(f)
		}

		l.Description = s.Entries[f].Metadata.Description

		// Log entries for log layout

		if isLog {
			l.Body = template.HTML(s.Entries[f].Body)

			date := s.Entries[f].Metadata.Date
			if len(date) > 0 {
				l.Date, l.MachineDate, err = convert.Date(date)
				if err != nil {
					return err
				}
			}

			dateUpdated := s.Entries[f].Metadata.DateUpdated
			if len(dateUpdated) > 0 {
				l.DateUpdated, l.MachineDateUpdated, err = convert.Date(dateUpdated)
				if err != nil {
					return err
				}
			}

			l.Tags = s.Entries[f].Metadata.Tags
		}

		// Now we act on the index files
		if IsDir {
			f += "/index.md"
		}

		// Append to listing map
		if s.Entries[f].Metadata.Pinned {
			s.Listings[dirIndex] = append([]listing.Listing{l}, s.Listings[dirIndex]...)
			continue
		}

		s.Listings[dirIndex] = append(s.Listings[dirIndex], l)
	}

	return nil
}
