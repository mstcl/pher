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

// copyFile copies inPath to outPath using ioReader and ioWriter
func copyFile(inPath string, outPath string, permission os.FileMode) error {
	inFile, err := os.Open(inPath)
	if err != nil {
		return fmt.Errorf("os.Open %s: %w", inPath, err)
	}
	defer inFile.Close()

	outFile, err := os.OpenFile(outPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, permission)
	if err != nil {
		return fmt.Errorf("os.OpenFile %s: %w", outPath, err)
	}
	defer outFile.Close()

	// Copy the content using a stream
	_, err = io.Copy(outFile, inFile)
	if err != nil {
		return fmt.Errorf("io.Copy %s to %s: %w", inPath, outPath, err)
	}

	return nil
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
			outputPath := filepath.Join(s.OutDir, relToInputDir)
			parentOutputDir := filepath.Dir(outputPath)

			// Make equivalent directory in output directory
			if err := os.MkdirAll(parentOutputDir, 0o755); err != nil {
				return fmt.Errorf("os.MkdirAll %s: %v", parentOutputDir, err)
			}

			// Copy file to target directory
			return copyFile(assetPath, outputPath, 0o644)
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
