package cli

import (
	"fmt"
	"html/template"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/mattn/go-zglob"
	"github.com/mstcl/pher/v2/internal/convert"
	"github.com/mstcl/pher/v2/internal/metadata"
	"github.com/mstcl/pher/v2/internal/nodepath"
	"github.com/mstcl/pher/v2/internal/nodepathlink"
	"github.com/mstcl/pher/v2/internal/state"
)

type populateNodePathLinksHelperInput struct {
	nodegroup        string
	childrenNodePath []nodepath.NodePath
}

// populateNodePathLinks finds all nodegroups. For each of them, find the children
// nodepaths (can either be nodes or nodegroups), and calls
// populateNodesListEntryHelper() on children nodepaths to populate the
// children's nodepath links
//
// * listing: listing entries of parents.
//
// * missing: bool map of parent index paths that are missing.
//
// * skip: bool map of files that should not be rendered (because its parents
// is displaying a log)
func populateNodePathLinks(s *state.State, logger *slog.Logger) error {
	// Initialize missing map
	s.NodegroupWithoutIndexMap = make(map[nodepath.NodePath]bool)

	files, err := zglob.Glob(filepath.Join(s.InputDir, "**", "*"))
	if err != nil {
		return err
	}

	files = append(files, s.InputDir)

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
		childrenRaw, err := filepath.Glob(f + "/*")
		if err != nil {
			return err
		}

		child.Debug("found children files", slog.Any("files", childrenRaw))

		var children []nodepath.NodePath
		for _, child := range childrenRaw {
			children = append(children, nodepath.NodePath(child))
		}

		if err := populateNodePathLinksHelper(s, &populateNodePathLinksHelperInput{
			nodegroup:        f,
			childrenNodePath: children,
		}, logger); err != nil {
			return err
		}
	}

	// Update files
	for f := range s.NodegroupWithoutIndexMap {
		entry := s.NodeMap[f]

		// add index to our files to render
		s.NodePaths = append(s.NodePaths, f)
		md := metadata.Default()

		// we have inDir/a/b/c/index.md
		// want to extract c
		// i.e. title is the folder name

		// inDir/a/b/c/index.md -> a/b/c/index.md
		rel, _ := filepath.Rel(s.InputDir, f.String())

		// a/b/c/index.md -> a/b/c -> a/b, c
		_, dir := filepath.Split(filepath.Dir(rel))

		// title is c
		md.Title = dir

		// update Metadata
		entry.Metadata = *md

		// update record
		s.NodeMap[f] = entry
	}

	return nil
}

// Sub-function to loop through depth 1 children inside the current parent
// (parentDir) to populate and return the nodepathlink map, the missing map, and the
// skip map. Additional calls constructListingEntry() to make individual listing
// entry.
func populateNodePathLinksHelper(
	s *state.State,
	i *populateNodePathLinksHelperInput,
	logger *slog.Logger,
) error {
	// Whether to render children
	// Use source file as key for consistency
	nodegroupIndexPath := nodepath.NodePath(filepath.Join(i.nodegroup, "index.md"))
	isLog := s.NodeMap[nodegroupIndexPath].Metadata.Layout == "log"

	for _, np := range i.childrenNodePath {
		child := logger.With(slog.Any("filepath", np), slog.String("context", "child listing"))

		// Stat files/directories
		info, err := os.Stat(np.String())
		if err != nil {
			return err
		}

		IsDir := info.Mode().IsDir()

		// Skip hidden files
		if rel, _ := filepath.Rel(s.InputDir, np.String()); rel[0] == 46 {
			child.Debug("skipped hidden file")

			continue
		}

		// Skip non-markdon files
		if !IsDir && filepath.Ext(np.String()) != ".md" {
			child.Debug("skipped non markdown file")

			continue
		}

		// Skip index files, unlisted ones
		if np.Base() == "index" || s.NodeMap[np].Metadata.Unlisted {
			child.Debug("skip index files and unlisted files")

			continue
		}

		// Don't render these files later
		s.SkippedNodePathMap[np] = isLog

		// Throw error if parent's view is Log but child is subdirectory
		if IsDir && isLog {
			child.Error("found a directory in log parent - this is unexpected")

			return err
		}

		// check if the nodepath is actually a node group
		isNodeGroup, err := np.IsNodegroup()
		if err != nil {
			return err
		}

		if IsDir && !isNodeGroup {
			child.Debug("empty directory found - skipping")
			continue
		}

		// Append to missing index if index doesn't exist for a directory
		if IsDir {
			indexFile := filepath.Join(np.String(), "/index.md")

			_, err := os.Stat(indexFile)
			if os.IsNotExist(err) {
				s.NodegroupWithoutIndexMap[np+"/index.md"] = true
				child.Debug("index doesn't exist, added to missing index state")
			} else if err != nil {
				return fmt.Errorf("stat %s: %w", s.ConfigFile, err)
			}
		}

		// Prepare the listing
		l := nodepathlink.NodePathLink{}

		// Grab href target, different for file vs. dir
		l.IsDir = IsDir

		// Construct the rest of the listing entry fields. Additionally add in
		// relevant rendering data fields like html body and tags for parents
		// with log view configured.
		if l.IsDir {
			target, err := filepath.Rel(i.nodegroup, np.String())
			if err != nil {
				return err
			}

			l.Title = target

			if s.Config.IsExt {
				target += "/index.html"
			}

			l.Href = target
			// Switch target to index for title & description
			np = nodepath.NodePath(filepath.Join(np.String(), "index.md"))
		} else {
			target := np.Href(i.nodegroup, false)
			if s.Config.IsExt {
				target += ".html"
			}

			l.Href = target
		}

		// Grab titles and description.
		// If metadata has title -> use that.
		// If not -> use filename only if entry is not a directory
		title := s.NodeMap[np].Metadata.Title
		if len(title) > 0 {
			l.Title = title
		} else if !l.IsDir {
			l.Title = np.Base()
		}

		l.Description = s.NodeMap[np].Metadata.Description

		// Log entries for log layout

		if isLog {
			l.Body = template.HTML(s.NodeMap[np].Body)

			date := s.NodeMap[np].Metadata.Date
			if len(date) > 0 {
				l.Date, l.MachineDate, err = convert.Date(date)
				if err != nil {
					return err
				}
			}

			dateUpdated := s.NodeMap[np].Metadata.DateUpdated
			if len(dateUpdated) > 0 {
				l.DateUpdated, l.MachineDateUpdated, err = convert.Date(dateUpdated)
				if err != nil {
					return err
				}
			}

			l.Tags = s.NodeMap[np].Metadata.Tags
		}

		// Now we act on the index files
		if IsDir {
			np += "/index.md"
		}

		// Append to listing map
		if s.NodeMap[np].Metadata.Pinned {
			s.NodePathLinkMap[nodegroupIndexPath] = append(
				[]nodepathlink.NodePathLink{l},
				s.NodePathLinkMap[nodegroupIndexPath]...)
			continue
		}

		s.NodePathLinkMap[nodegroupIndexPath] = append(s.NodePathLinkMap[nodegroupIndexPath], l)
	}

	return nil
}
