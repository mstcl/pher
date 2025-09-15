package cli

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/mstcl/pher/v2/internal/nodepath"
	"github.com/mstcl/pher/v2/internal/state"
)

func sanitize(s *state.State, logger *slog.Logger) error {
	var err error

	// Sanitize configuration file
	s.ConfigFile, err = filepath.Abs(s.ConfigFile)
	if err != nil {
		return fmt.Errorf("absolute path: %w", err)
	}

	// Check whether configuration file exists
	_, err = os.Stat(s.ConfigFile)
	if os.IsNotExist(err) {
		return fmt.Errorf("missing: %s", s.ConfigFile)
	} else if err != nil {
		return fmt.Errorf("os.Stat %s: %w", s.ConfigFile, err)
	}

	logger.Debug("sanitized config file", slog.String("path", s.ConfigFile))

	// Sanitize input directory
	s.InputDir, err = filepath.Abs(s.InputDir)
	if err != nil {
		return fmt.Errorf("filepath.Abs: %w", err)
	}

	logger.Debug("sanitized input directory", slog.String("path", s.InputDir))

	// Sanitize output directory
	s.OutputDir, err = filepath.Abs(s.OutputDir)
	if err != nil {
		return fmt.Errorf("filepath.Abs: %w", err)
	}

	logger.Debug("sanitized output directory", slog.String("path", s.OutputDir))

	return nil
}

// reorderNodeFiles resorts nodes slice so that all group index are moved to the
// end so they are processed last
func reorderNodeFiles(nodepaths []nodepath.NodePath) []nodepath.NodePath {
	var notIndex []nodepath.NodePath
	var index []nodepath.NodePath

	for _, i := range nodepaths {
		base := i.Base()
		if base == "index" {
			index = append(index, i)
			continue
		}

		notIndex = append(notIndex, i)
	}

	return append(notIndex, index...)
}

// dropHiddenFiles drops files where any path component is hidden (starts with a dot).
func dropHiddenFiles(nodepaths []nodepath.NodePath) []nodepath.NodePath {
	var newFiles []nodepath.NodePath

	for _, np := range nodepaths {
		if !isPathHidden(np.String()) {
			newFiles = append(newFiles, np)
		}
	}

	return newFiles
}

// isPathHidden checks if any component of a path starts with a dot.
func isPathHidden(p string) bool {
	// split the path into components.
	parts := strings.Split(p, string(filepath.Separator))

	// iterate through each part and check for a leading dot.
	for _, part := range parts {
		if strings.HasPrefix(part, ".") && part != "." && part != ".." {
			return true
		}
	}

	return false
}

func sanitizeNodePaths(nodepaths []nodepath.NodePath, logger *slog.Logger) []nodepath.NodePath {
	// sanitize by removing all hidden files
	nodepaths = dropHiddenFiles(nodepaths)
	logger.Debug("dropped hidden files")

	// reorder the list so indexes are processed last
	nodepaths = reorderNodeFiles(nodepaths)
	logger.Debug("finalized list of files to process")

	return nodepaths
}
