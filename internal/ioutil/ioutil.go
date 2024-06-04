package ioutil

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mattn/go-zglob"
	"github.com/mstcl/pher/internal/convert"
)

// OS delete given files
func RemoveContents(contents []string) error {
	for _, c := range contents {
		if err := os.RemoveAll(c); err != nil {
			return fmt.Errorf("removing old output files: %w", err)
		}
	}
	return nil
}

// Move all index.md from files to the end so they are processed last
func ReorderFiles(files []string) []string {
	var ni []string
	var yi []string
	for _, i := range files {
		base := convert.FileBase(i)
		if base == "index" {
			yi = append(yi, i)
			continue
		}
		ni = append(ni, i)
	}
	return append(ni, yi...)
}

// Move extra files like assets (images, fonts, css) over to output, preserving
// the file structure.
func CopyExtraFiles(inDir string, outDir string, files map[string]bool) error {
	for f := range files {
		// want our assets to go from inDir/a/b/c/image.png -> outDir/a/b/c/image.png
		rel, _ := filepath.Rel(inDir, f)
		out := outDir + "/" + rel

		// Make dir on filesystem
		if err := EnsureDir(filepath.Dir(out)); err != nil {
			return fmt.Errorf("make directory: %w", err)
		}

		// Copy from f to out
		b, err := os.ReadFile(f)
		if err != nil {
			return fmt.Errorf("read file: %w", err)
		}
		if err = os.WriteFile(out, b, 0o644); err != nil {
			return fmt.Errorf("write file: %w", err)
		}
	}
	return nil
}

// Check for markdown files under directory
func IsEntryPresent(f string) (bool, error) {
	// we want to check all nested files
	res, err := zglob.Glob(f + "/**/*.md")
	if err != nil {
		return false, err
	}
	if len(res) == 0 {
		return false, nil
	}
	return true, nil
}

// Return true/false if a path exists/doesn't exist
func IsFileExist(f string) (bool, error) {
	if _, err := os.Stat(f); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		} else {
			return false, err
		}
	}
	return true, nil
}

// Ensure a directory exists
func EnsureDir(dirName string) error {
	if err := os.Mkdir(dirName, 0o755); err == nil {
		return nil
	} else if os.IsExist(err) {
		// check that the existing path is a directory
		info, err := os.Stat(dirName)
		if err != nil {
			return err
		}
		if !info.IsDir() {
			return fmt.Errorf("path exists but is not a directory")
		}
		return nil
	}
	return nil
}

// Remove hidden files from slice of files
func RemoveHiddenFiles(inDir string, files []string) []string {
	newFiles := []string{}
	for _, f := range files {
		if rel, _ := filepath.Rel(inDir, f); rel[0] == 46 {
			continue
		}
		newFiles = append(newFiles, f)
	}
	return newFiles
}
