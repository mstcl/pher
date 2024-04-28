package util

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func GetDepth(dir string) int {
	return len(strings.Split(dir, "/"))
}

// Given /abs/path/to/fileOrDir, where root is /abs/path, return
// to/fileOrDir
func GetRelativeFilePath(f string, root string) string {
	depth := GetDepth(root)
	return strings.Join(strings.Split(f, "/")[depth:], "/")
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

// OS delete given files
func RemoveContents(contents []string) error {
	for _, c := range contents {
		err := os.RemoveAll(c)
		if err != nil {
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

func GetCrumbs(relDir string) ([]string, []string) {
	crumbs := strings.Split(relDir, "/")
	crumbLinks := []string{}
	for i := range crumbs {
		if i == len(crumbs)-1 {
			break
		}
		if i != len(crumbs) {
			cl := strings.Join(crumbs[:i+1], "/")
			crumbLinks = append(crumbLinks, cl)
		}
	}
	return crumbs[:len(crumbs)-1], crumbLinks
}

func ResolveTitle(mt string, fn string) string {
	var title string
	if len(mt) > 0 {
		title = mt
	} else {
		title = fn
	}
	return title
}

func ResolveHref(f string, inDir string) string {
	var href string
	rel := GetFilePath(GetRelativeFilePath(f, inDir))
	if len(rel) > 0 {
		href = "/" + rel + "/" + GetFileBase(f)
	} else {
		href = "/" + GetFileBase(f)
	}
	return href
}

func ResolveDate(d string) (string, string, error) {
	dt, err := time.Parse("2006-01-02", d)
	if err != nil {
		return "", "", fmt.Errorf("time parse: %w", err)
	}
	return dt.Format("02 Jan 2006"), dt.Format(time.RFC3339), nil
}
