package cli

import (
	"fmt"
	"html/template"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/mattn/go-zglob"
	"github.com/mstcl/pher/v3/internal/convert"
	"github.com/mstcl/pher/v3/internal/metadata"
	"github.com/mstcl/pher/v3/internal/nodepath"
	"github.com/mstcl/pher/v3/internal/nodepathlink"
	"github.com/mstcl/pher/v3/internal/state"
)

type populateNodePathLinksHelperInput struct {
	parentNodePath   nodepath.NodePath
	childrenNodePath []nodepath.NodePath
}

// populateNodePathLinks finds all nodegroups. For each of them, find the
// children nodepaths (can either be nodes or nodegroups), and calls
// populateNodesListEntryHelper() on children nodepaths to populate the
// children's nodepath links
func populateNodePathLinks(s *state.State, logger *slog.Logger) error {
	s.NodegroupWithoutIndexMap = make(map[nodepath.NodePath]bool)

	nodepathsRaw, err := zglob.Glob(filepath.Join(s.InputDir, "**", "*"))
	if err != nil {
		return err
	}

	var nodepaths []nodepath.NodePath
	for _, np := range nodepathsRaw {
		nodepaths = append(nodepaths, nodepath.NodePath(np))
	}

	// add the root "." as well
	nodepaths = append(nodepaths, nodepath.NodePath(s.InputDir))

	logger.Debug("found all nodepaths", slog.Any("nodepaths", nodepaths))

	// Go through and drop all nodepaths that aren't nodegroups Glob nodegroups
	// for further nodepaths and call the helper function to populate their
	// NodePathLink slice
	for _, np := range nodepaths {
		childNodePath := logger.With(
			slog.Any("nodepath", np),
			slog.String("context", "populateNodePathLinks"),
		)

		isNodegroup, err := np.IsNodegroup()
		if err != nil {
			return err
		}

		if !isNodegroup {
			continue
		}

		// Find immediate children of the nodepath
		childrenRaw, err := filepath.Glob(filepath.Join(np.String(), "*"))
		if err != nil {
			return err
		}

		childNodePath.Debug("found children files", slog.Any("files", childrenRaw))

		var children []nodepath.NodePath
		for _, child := range childrenRaw {
			children = append(children, nodepath.NodePath(child))
		}

		// Run the helper on the parent and its chidlren
		if err := populateNodePathLinksHelper(s, &populateNodePathLinksHelperInput{
			parentNodePath:   np,
			childrenNodePath: children,
		}, logger); err != nil {
			return err
		}
	}

	// Add index files to NodegroupWithoutIndexMap
	// TODO: refactor this
	for np := range s.NodegroupWithoutIndexMap {
		entry := s.NodeMap[np]

		// add index to our files to render
		s.NodePaths = append(s.NodePaths, np)
		md := metadata.Default()

		// we have inDir/a/b/c/index.md
		// want to extract c
		// i.e. title is the folder name

		// inDir/a/b/c/index.md -> a/b/c/index.md
		rel, _ := filepath.Rel(s.InputDir, np.String())

		// a/b/c/index.md -> a/b/c -> a/b, c
		_, dir := filepath.Split(filepath.Dir(rel))

		// title is c
		md.Title = dir

		// update Metadata
		entry.Metadata = *md

		// update record
		s.NodeMap[np] = entry
	}

	return nil
}

// populateNodePathLinksHelper loops through all immediate children inside the
// current parent (i.parentNodePath) to populate and return NodePathLinksMap,
// NodegroupWithoutIndexMap, and SkippedNodePathMap.
func populateNodePathLinksHelper(
	s *state.State,
	i *populateNodePathLinksHelperInput,
	logger *slog.Logger,
) error {
	// this is the nodegroup index path, we expect it to be at /path/to/nodegroup/index.md
	nodegroupIndexPath := nodepath.NodePath(filepath.Join(i.parentNodePath.String(), "index.md"))

	// is the parent nodegroup a log type? If so cache this because we will use
	// it later on for further logic
	isLog := s.NodeMap[nodegroupIndexPath].Metadata.Layout == "log"

	// for each child, make some decisions
	for _, np := range i.childrenNodePath {
		childLogger := logger.With(
			slog.Any("filepath", np),
			slog.String("context", "child listing"),
		)

		// stat files/directories
		info, err := os.Stat(np.String())
		if err != nil {
			return err
		}

		IsDir := info.Mode().IsDir()

		// begin filtering invalid/unwated files/directories and/or fast
		// failing unexpected behaviour

		// throw error if parent's view is log type but child is subdirectory
		if IsDir && isLog {
			childLogger.Error("found a directory in log parent -- this is unexpected")

			return err
		}

		// if a nodegroup, skip if it's empty
		if IsDir {
			// check if the nodepath is actually a node group
			nodegroupHasChildren, err := np.HasChildren()
			if err != nil {
				return err
			}

			if !nodegroupHasChildren {
				childLogger.Debug("skipping empty directory found")

				continue
			}
		}

		// Skip hidden files/directories
		relativePath, _ := filepath.Rel(s.InputDir, np.String())
		if strings.HasPrefix(relativePath, ".") {
			childLogger.Debug("skipping hidden file/directory")

			continue
		}

		// Skip non-markdown files
		fileExtension := filepath.Ext(np.String())
		if !IsDir && fileExtension != ".md" {
			childLogger.Debug("skipping non-markdown file")

			continue
		}

		// Skip index files, unlisted ones
		if np.Base() == "index" {
			childLogger.Debug("skipping index file")

			continue
		}

		if s.NodeMap[np].Metadata.Unlisted {
			childLogger.Debug("skipping unlisted file")

			continue
		}

		// checks complete, now we consider only files that are valid

		// don't render these files later
		s.SkippedNodePathMap[np] = isLog

		// append to missing index if index doesn't exist for a directory
		if IsDir {
			indexFile := filepath.Join(np.String(), "index.md")

			_, err := os.Stat(indexFile)
			if os.IsNotExist(err) {
				s.NodegroupWithoutIndexMap[np+"/index.md"] = true // TODO: change the behaviour so we don't have to append the /index.md as the key

				childLogger.Debug("index doesn't exist, added to missing index state")
			} else if err != nil {
				return fmt.Errorf("stat %s: %w", s.ConfigFile, err)
			}
		}

		// prepare the link
		l := nodepathlink.NodePathLink{}

		// grab href target, different for file vs. dir
		l.IsDir = IsDir

		// construct the rest of the NodePathLink fields. Additionally add in
		// relevant rendering data fields like html body and tags for parents
		// with log view configured.
		//
		// we also update np in place as it will be used as the key in further
		// maps, so that if it's a parent the key should have index.md at the
		// end
		if IsDir {
			// isolate the directory name, this is the title and href
			npName, err := filepath.Rel(i.parentNodePath.String(), np.String())
			if err != nil {
				return err
			}

			l.Title = npName

			if s.Config.IsExt {
				l.Href = filepath.Join(npName, "index.html")
			} else {
				l.Href = npName
			}

			// switch nodegroup key to index for title & description
			np = nodepath.NodePath(filepath.Join(np.String(), "index.md"))
			childLogger.Debug(
				"replaced nodegroup key with index path",
				slog.String("np", np.String()),
			)
		} else {
			npName := np.Href(i.parentNodePath.String(), false)

			if s.Config.IsExt {
				l.Href = npName + ".html"
			} else {
				l.Href = npName
			}
		}

		// grab nodepath title
		// if metadata has title -> use that.
		// if not -> use filename only if nodepath is not a directory
		// as directory title is already set above
		title := s.NodeMap[np].Metadata.Title
		if len(title) > 0 {
			l.Title = title
		} else if !l.IsDir {
			l.Title = np.Base()
		}

		// grab nodepath description
		l.Description = s.NodeMap[np].Metadata.Description

		// handle log nodegroup logic
		if isLog {
			l.Body = template.HTML(s.NodeMap[np].Body)

			// if date is present convert it
			date := s.NodeMap[np].Metadata.Date
			if len(date) > 0 {
				l.Date, l.MachineDate, err = convert.Date(date)
				if err != nil {
					return err
				}
			}

			// if dateUpdated is present convert it
			dateUpdated := s.NodeMap[np].Metadata.DateUpdated
			if len(dateUpdated) > 0 {
				l.DateUpdated, l.MachineDateUpdated, err = convert.Date(dateUpdated)
				if err != nil {
					return err
				}
			}

			// set link tags
			l.Tags = s.NodeMap[np].Metadata.Tags
		}

		// if node is pinned we prepend it to the links map slice value, else we append it
		if s.NodeMap[np].Metadata.Pinned {
			s.NodePathLinksMap[nodegroupIndexPath] = append(
				[]nodepathlink.NodePathLink{l},
				s.NodePathLinksMap[nodegroupIndexPath]...)

			continue
		} else {
			s.NodePathLinksMap[nodegroupIndexPath] = append(s.NodePathLinksMap[nodegroupIndexPath], l)
		}
	}

	return nil
}
