package cli

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/mstcl/pher/v2/internal/convert"
	"github.com/mstcl/pher/v2/internal/state"
	"golang.org/x/sync/errgroup"

	"github.com/mstcl/pher/v2/internal/checks"
)

// Move all index.md from files to the end so they are processed last
func reorderFiles(files []string) []string {
	var notIndex []string

	var index []string

	for _, i := range files {
		base := convert.FileBase(i)
		if base == "index" {
			index = append(index, i)
			continue
		}

		notIndex = append(notIndex, i)
	}

	return append(notIndex, index...)
}

// Delete files
func removeFiles(files []string) error {
	for _, c := range files {
		if err := os.RemoveAll(c); err != nil {
			return fmt.Errorf("removing old output files: %w", err)
		}
	}

	return nil
}

// copyFile
func copyFile(inPath string, outPath string, permission os.FileMode) error {
	content, err := os.ReadFile(inPath)
	if err != nil {
		return fmt.Errorf("read file %s: %w", inPath, err)
	}

	if err = os.WriteFile(outPath, content, permission); err != nil {
		return fmt.Errorf("write file %s: %w", outPath, err)
	}

	return nil
}

// mkdirIfNotExists takes in a directory path, checks if it exists, and
// create it if not
func mkdirIfNotExists(dir string) error {
	if err := os.Mkdir(dir, 0o755); err == nil {
		return nil
	} else if os.IsExist(err) {
		// check that the existing path is a directory
		info, err := os.Stat(dir)
		if err != nil {
			return err
		}

		if !info.IsDir() {
			return fmt.Errorf("%s exists but is not a directory", dir)
		}

		return nil
	} else {
		return fmt.Errorf("mkdir %s: %w", dir, err)
	}
}

// Move extra files like assets (images, fonts, css) over to output, preserving
// the file structure.
func syncAssets(ctx context.Context, s *state.State, logger *slog.Logger) error {
	eg, _ := errgroup.WithContext(ctx)

	for assetPath := range s.Assets {
		child := logger.With(
			slog.String("filepath", assetPath),
			slog.String("context", "copying asset"),
		)

		child.Debug("submitting goroutine")

		eg.Go(func() error {
			// NOTE: want our assets to go from inDir/a/b/c/image.png -> outDir/a/b/c/image.png
			relToInputDir, _ := filepath.Rel(s.InDir, assetPath)
			parentOutputDir := filepath.Join(s.OutDir, relToInputDir)

			// Make equivalent directory in output directory
			if err := mkdirIfNotExists(filepath.Dir(parentOutputDir)); err != nil {
				return err
			}

			// Copy file to target directory
			return copyFile(assetPath, parentOutputDir, 0o644)
		})
	}

	return eg.Wait()
}

// Filter hidden files from files
func filterHiddenFiles(inDir string, files []string) []string {
	newFiles := []string{}

	for _, f := range files {
		if rel, _ := filepath.Rel(inDir, f); rel[0] == 46 {
			continue
		}

		newFiles = append(newFiles, f)
	}

	return newFiles
}
