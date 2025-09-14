package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"slices"

	"github.com/mattn/go-zglob"
	"github.com/mstcl/pher/v2/internal/state"
	"golang.org/x/sync/errgroup"
)

func createDir(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("os.MkdirAll %s: %w", dir, err)
	}

	return nil
}

// getSrcFiles return the files we need to process by recursively glob for all
// markdown files, then run sanitizeSrcFiles on them
func getSrcFiles(inputDir string, logger *slog.Logger) ([]string, error) {
	files, err := zglob.Glob(filepath.Join(inputDir, "**", "*.md"))
	if err != nil {
		return nil, fmt.Errorf("glob files: %w", err)
	}

	// sanitize files found
	files = sanitizeSrcFiles(files, logger)
	logger.Debug("sanitized source files", slog.Any("paths", files))

	return files, nil
}

// cleanOutput removes all files and directories in outputDir,
// except for the ones listed in the exceptions list.
// WARN: on error keep deleting everything, and report errors at the end
// this is to ensure a clean state.
func cleanOutputDir(outputDir string, exceptions []string) error {
	entries, err := os.ReadDir(outputDir)
	if err != nil {
		return fmt.Errorf("os.ReadDir %s: %w", outputDir, err)
	}

	var removeErrors []error
	for _, entry := range entries {
		// skip those in the exceptions
		if slices.Contains(exceptions, entry.Name()) {
			continue
		}

		pathToRemove := filepath.Join(outputDir, entry.Name())
		if err := os.RemoveAll(pathToRemove); err != nil {
			removeErrors = append(
				removeErrors,
				fmt.Errorf("os.RemoveAll %s: %w", pathToRemove, err),
			)
		}
	}

	if len(removeErrors) > 0 {
		return errors.Join(removeErrors...)
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
func copyUserAssets(ctx context.Context, s *state.State, logger *slog.Logger) error {
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

// copyStatic
func copyStatic(s *state.State, logger *slog.Logger) error {
	outputDir := filepath.Join(s.OutDir, relStaticOutputDir)

	// make static directory in output directory
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("os.MkdirAll %s: %w", outputDir, err)
	}

	logger.Debug("created static output directory", slog.String("dir", outputDir))

	staticFS, err := fs.Sub(EmbedFS, relStaticDir)
	if err != nil {
		return fmt.Errorf("create subfilesystem %s: %w", relStaticDir, err)
	}

	logger.Debug("created static subfilesystem", slog.String("dir", relStaticDir))

	// walk through all files and directories in the `staticfs`.
	// starting at the root of the sub-filesystem.
	if err := fs.WalkDir(staticFS, ".", func(inputPath string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// skip directories and only process files
		if d.IsDir() {
			return nil
		}

		// construct the destination path for the file
		outputPath := filepath.Join(outputDir, inputPath)
		parentOutputDir := filepath.Dir(outputPath)

		// create the destination directory if it doesn't exist
		if err := os.MkdirAll(parentOutputDir, 0o755); err != nil {
			return fmt.Errorf("os.MkdirAll %s: %w", parentOutputDir, err)
		}

		// open the input file from fs
		inputFile, err := staticFS.Open(inputPath)
		if err != nil {
			return err
		}
		defer inputFile.Close()

		// create output file and copy content from inputFile to outputFile
		outputFile, err := os.Create(outputPath)
		if err != nil {
			return err
		}
		defer outputFile.Close()

		if _, err := io.Copy(outputFile, inputFile); err != nil {
			return err
		}

		return nil
	}); err != nil {
		return fmt.Errorf("fs.WalkDir: %w", err)
	}

	logger.Debug("walked static subfilesystem", slog.String("outputDir", outputDir))

	return nil
}
