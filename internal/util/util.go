package util

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mattn/go-zglob"
)

// Given /path/to/filename.ext, return filename
func GetFileBase(f string) string {
	fn := filepath.Base(f)
	return strings.TrimSuffix(fn, filepath.Ext(fn))
}

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
		base := GetFileBase(i)
		if base == "index" {
			yi = append(yi, i)
			continue
		}
		ni = append(ni, i)
	}
	return append(ni, yi...)
}

// If link is "/a/b/c/file.md"
//
// crumbs is {"a", "b", "c"}
//
// crumbLinks is {"/a/index.html", "/a/b/index.html", "a/b/c/index.html"}
func GetNavCrumbs(f string, inDir string, isExt bool) ([]string, []string) {
	// inDir/a/b/c/file.md -> a/b/c/file.md
	rel, _ := filepath.Rel(inDir, f)

	// a/b/c/file.md -> {a, b, c, file.md}
	crumbs := strings.Split(rel, "/")

	// make the crumbLinks
	crumbLinks := []string{}
	for i := range crumbs {
		// don't process last item
		if i == len(crumbs)-1 {
			break
		}
		cl := strings.Join(crumbs[:i+1], "/")
		if isExt {
			cl += "/index.html"
		}
		crumbLinks = append(crumbLinks, cl)
	}
	return crumbs[:len(crumbs)-1], crumbLinks
}

// Return title mt else fn
func ResolveTitle(mt string, fn string) string {
	var title string
	if len(mt) > 0 {
		title = mt
	} else {
		title = fn
	}
	return title
}

// Return the href
// inDir/a/b/c/file.md -> a/b/c/file
func ResolveHref(f string, inDir string, prefixSlash bool) string {
	// inDir/a/b/c/file.md -> a/b/c/file.md
	rel, _ := filepath.Rel(inDir, f)

	// a/b/c/file.md -> a/b/c/file
	href := strings.TrimSuffix(rel, filepath.Ext(rel))

	// a/b/c/file -> /a/b/c/file (for web rooting)
	if prefixSlash {
		href = "/" + href
	}
	return href
}

// Resolve the date d from format YYYY-MM-DD
// Returns a pretty date and a machine date
func ResolveDate(d string) (string, string, error) {
	if len(d) == 0 {
		return "", "", nil
	}
	dt, err := time.Parse("2006-01-02", d)
	if err != nil {
		return "", "", fmt.Errorf("time parse: %w", err)
	}
	return dt.Format("02 Jan 2006"), dt.Format(time.RFC3339), nil
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

// Given filename.ext, return filename
func RemoveExtension(f string) string {
	return strings.TrimSuffix(f, filepath.Ext(f))
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
