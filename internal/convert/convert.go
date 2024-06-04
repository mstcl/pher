// Transform various data strings to other data strings
package convert

import (
	"path/filepath"
	"strings"
	"time"
)

// Return the href
// inDir/a/b/c/file.md -> a/b/c/file
func Href(f string, inDir string, prefixSlash bool) string {
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
func Date(d string) (string, string, error) {
	if len(d) == 0 {
		return "", "", nil
	}
	dt, err := time.Parse("2006-01-02", d)
	if err != nil {
		return "", "", err
	}
	return dt.Format("02 Jan 2006"), dt.Format(time.RFC3339), nil
}

// If link is "/a/b/c/file.md"
//
// crumbs is {"a", "b", "c"}
//
// crumbLinks is {"/a/index.html", "/a/b/index.html", "a/b/c/index.html"}
func NavCrumbs(f string, inDir string, isExt bool) ([]string, []string) {
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
func Title(mt string, fn string) string {
	var title string
	if len(mt) > 0 {
		title = mt
	} else {
		title = fn
	}
	return title
}

// Given /path/to/filename.ext, return filename
func FileBase(f string) string {
	fn := filepath.Base(f)
	return strings.TrimSuffix(fn, filepath.Ext(fn))
}
