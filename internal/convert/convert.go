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
func Date(date string) (string, string, error) {
	if len(date) == 0 {
		return "", "", nil
	}

	dateTime, err := time.Parse("2006-01-02", date)
	if err != nil {
		return "", "", err
	}

	return dateTime.Format("02 Jan 2006"), dateTime.Format(time.RFC3339), nil
}

// If link is "/a/b/c/file.md"
//
// crumbsTitle: {"a", "b", "c"}
//
// crumbsLink: {"a/index.html", "a/b/index.html", "a/b/c/index.html"}
func NavCrumbs(f string, inDir string, isExt bool) ([]string, []string) {
	// inDir/a/b/c/file.md -> a/b/c/file.md
	rel, _ := filepath.Rel(inDir, f)

	// a/b/c/file.md -> {a, b, c, file.md}
	crumbsTitle := strings.Split(rel, "/")

	// make the crumbsLink
	crumbsLink := []string{}

	for i := range crumbsTitle {
		// don't process last item
		if i == len(crumbsTitle)-1 {
			break
		}

		cl := strings.Join(crumbsTitle[:i+1], "/")
		if isExt {
			cl += "/index.html"
		}

		crumbsLink = append(crumbsLink, cl)
	}

	return crumbsTitle[:len(crumbsTitle)-1], crumbsLink
}

// Return title mt else fn
func Title(metadataTitle string, filename string) string {
	var title string
	if len(metadataTitle) > 0 {
		title = metadataTitle
	} else {
		title = filename
	}

	return title
}

// Given /path/to/filename.ext, return filename
func FileBase(f string) string {
	fn := filepath.Base(f)
	return strings.TrimSuffix(fn, filepath.Ext(fn))
}
