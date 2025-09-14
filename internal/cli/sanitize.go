package cli

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/mstcl/pher/v2/internal/convert"
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
func reorderNodeFiles(nodes []string) []string {
	var notIndex []string

	var index []string

	for _, i := range nodes {
		base := convert.FileBase(i)
		if base == "index" {
			index = append(index, i)
			continue
		}

		notIndex = append(notIndex, i)
	}

	return append(notIndex, index...)
}

// dropHiddenFiles goes through files slice and drop those started with a dot
func dropHiddenFiles(nodes []string) []string {
	newFiles := []string{}

	for _, f := range nodes {
		base := filepath.Base(f)
		if strings.HasPrefix(base, ".") {
			continue
		}

		newFiles = append(newFiles, f)
	}

	return newFiles
}

func sanitizeNodeFiles(nodes []string, logger *slog.Logger) []string {
	// sanitize by removing all hidden files
	nodes = dropHiddenFiles(nodes)
	logger.Debug("dropped hidden files")

	// reorder the list so indexes are processed last
	nodes = reorderNodeFiles(nodes)
	logger.Debug("finalized list of files to process")

	return nodes
}
