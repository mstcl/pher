package nodepath

import (
	"path/filepath"
	"strings"

	"github.com/mattn/go-zglob"
)

// NOTE: a nodepath is the path of a nodegroup or node (note this
// non-distinction)

type NodePath string

func (np NodePath) String() string {
	return string(np)
}

// Base given a path /path/to/filename.ext, returns filename
func (np NodePath) Base() string {
	fn := filepath.Base(np.String())
	return strings.TrimSuffix(fn, filepath.Ext(fn))
}

func (np NodePath) IsNodegroup() (bool, error) {
	// we want to check all nested files
	files, err := zglob.Glob(filepath.Join(np.String(), "**", "*.md"))
	if err != nil {
		return false, err
	}

	if len(files) == 0 {
		return false, nil
	}

	return true, nil
}

// Href function returns the href, which is defined as follows:
// inputDir/a/b/c/file.md -> a/b/c/file
func (np NodePath) Href(inputDir string, prefixSlash bool) string {
	// inDir/a/b/c/file.md -> a/b/c/file.md
	rel, _ := filepath.Rel(inputDir, np.String())

	// a/b/c/file.md -> a/b/c/file
	href := strings.TrimSuffix(rel, filepath.Ext(rel))

	// a/b/c/file -> /a/b/c/file (for web rooting)
	if prefixSlash {
		href = "/" + href
	}

	return href
}
