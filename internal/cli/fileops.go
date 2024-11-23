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

// Move extra files like assets (images, fonts, css) over to output, preserving
// the file structure.
func syncAssets(ctx context.Context, s *state.State, logger *slog.Logger) error {
	eg, _ := errgroup.WithContext(ctx)

	for f := range s.Assets {
		f := f

		child := logger.With(slog.String("filepath", f), slog.String("context", "copying asset"))

		child.Debug("submitting goroutine")

		eg.Go(func() error {
			// want our assets to go from inDir/a/b/c/image.png -> outDir/a/b/c/image.png
			rel, _ := filepath.Rel(s.InDir, f)
			path := s.OutDir + "/" + rel

			// Make dir on filesystem
			if err := checks.DirExist(filepath.Dir(path)); err != nil {
				return fmt.Errorf("make directory: %w", err)
			}

			// Copy from f to out
			b, err := os.ReadFile(f)
			if err != nil {
				return fmt.Errorf("read file: %w", err)
			}

			if err = os.WriteFile(path, b, 0o644); err != nil {
				return fmt.Errorf("write file: %w", err)
			}

			return nil
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
