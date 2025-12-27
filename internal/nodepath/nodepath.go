// Package nodepath defines the NodeProup struct
package nodepath

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mattn/go-zglob"
)

// NodePath is the path of a nodegroup or node (note this
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
	// Stat nodepath
	npStat, err := os.Stat(np.String())
	if err != nil {
		return false, fmt.Errorf("os.Stat %s: %w", np, err)
	}

	// Only process nodegroups
	if npStat.Mode().IsRegular() {
		return false, nil
	}

	return true, nil
}

func (np NodePath) HasChildren() (bool, error) {
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
