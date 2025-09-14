package checks

import (
	"fmt"
	"os"

	"github.com/mattn/go-zglob"
)

// Check for markdown files under directory
func EntryPresent(f string) (bool, error) {
	// we want to check all nested files
	files, err := zglob.Glob(f + "/**/*.md")
	if err != nil {
		return false, err
	}

	if len(files) == 0 {
		return false, nil
	}

	return true, nil
}

// Return true/false if a path exists/doesn't exist
func FileExist(f string) (bool, error) {
	if _, err := os.Stat(f); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		} else {
			return false, err
		}
	}

	return true, nil
}
