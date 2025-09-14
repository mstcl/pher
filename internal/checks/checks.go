// Package checks [TODO]
package checks

import (
	"path/filepath"

	"github.com/mattn/go-zglob"
)

// EntryPresent checks for presence of markdown files under a directory
func EntryPresent(dir string) (bool, error) {
	// we want to check all nested files
	files, err := zglob.Glob(filepath.Join(dir, "/**/*.md"))
	if err != nil {
		return false, err
	}

	if len(files) == 0 {
		return false, nil
	}

	return true, nil
}
