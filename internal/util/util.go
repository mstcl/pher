package util

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Get the nesting-depth of a filepath
func GetDepth(dir string) int {
	return len(strings.Split(dir, "/"))
}

// Given /abs/path/to/fileOrDir, where root is /abs/path, return
// to/fileOrDir
func GetRelativeFilePath(f string, root string) string {
	depth := GetDepth(root)
	return strings.Join(strings.Split(f, "/")[depth:], "/")
}

// Given /path/to/filename.ext, return filename.ext
func GetFileName(f string) string {
	chunks := strings.Split(f, "/")
	base := chunks[len(chunks)-1]
	return base
}

// Given /path/to/filename.ext, return filename
func GetFileBase(f string) string {
	chunks := strings.Split(f, "/")
	base := chunks[len(chunks)-1]
	return strings.TrimSuffix(base, filepath.Ext(base))
}

// Given /path/to/filename.ext, return /path/to
func GetFilePath(f string) string {
	chunks := strings.Split(f, "/")
	return strings.Join(chunks[:len(chunks)-1], "/")
}

// Given /path/to/filename.ext, return .ext
func GetFileExt(f string) string {
	chunks := strings.Split(f, "/")
	base := chunks[len(chunks)-1]
	return filepath.Ext(base)
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

// If link is "/foo/bar/hello.md"
//
// crumbs is {"foo", "bar"}
//
// crumbLinks is {"/foo", "/foo/bar"}
func GetCrumbs(f string, inDir string, isExt bool) ([]string, []string) {
	chunks := GetRelativeFilePath(f, inDir)
	crumbs := strings.Split(chunks, "/")
	crumbLinks := []string{}
	for i := range crumbs {
		if i == len(crumbs)-1 {
			break
		}
		if i != len(crumbs) {
			cl := strings.Join(crumbs[:i+1], "/")
			if isExt {
				cl += "/index.html"
			}
			crumbLinks = append(crumbLinks, cl)
		}
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
func ResolveHref(f string, inDir string, prefixSlash bool) string {
	var href string
	rel := GetFilePath(GetRelativeFilePath(f, inDir))
	if len(rel) > 0 {
		href = rel + "/" + GetFileBase(f)
	} else {
		href = GetFileBase(f)
	}
	if prefixSlash {
		href = "/" + href
	}
	return href
}

// Resolve out path of file or directory given absolute path f
func ResolveOutPath(f string, inDir string, outDir string, newExt string) string {
	chunks := GetRelativeFilePath(f, inDir)
	chunks = GetFilePath(chunks)
	if len(chunks) > 0 {
		chunks += "/"
	}

	// Leave me and my extension alone
	var fn string
	if len(newExt) > 0 {
		fn = GetFileBase(f) + newExt
	} else {
		fn = GetFileName(f)
	}

	o := outDir + "/" + chunks + fn
	return o
}

// Resolve the date d from format YYYY-MM-DD
// Returns a pretty date and a machine date
func ResolveDate(d string) (string, string, error) {
	dt, err := time.Parse("2006-01-02", d)
	if err != nil {
		return "", "", fmt.Errorf("time parse: %w", err)
	}
	return dt.Format("02 Jan 2006"), dt.Format(time.RFC3339), nil
}

// Move extra files like assets (images, fonts, css) over to output, preserving
// the structure.
func CopyExtraFiles(inDir string, outDir string, files map[string]bool) error {
	// Copy keys of i (internal image links) to outDir
	for k := range files {
		out := ResolveOutPath(k, inDir, outDir, "")

		// Make dir to preserver structure
		dirOut := GetFilePath(out)
		if err := os.MkdirAll(dirOut, 0o755); err != nil {
			return fmt.Errorf("mkdir: %w", err)
		}

		// Copy from in to out
		b, err := os.ReadFile(k)
		if err != nil {
			return fmt.Errorf("read file: %w", err)
		}
		if err = os.WriteFile(out, b, 0o644); err != nil {
			return fmt.Errorf("write file: %w", err)
		}
	}
	return nil
}

func IsEntryPresent(f string) (bool, error) {
	res, err := filepath.Glob(f + "/*.md")
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

// Ensure directory exists
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
