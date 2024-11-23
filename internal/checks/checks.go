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

// Ensure a directory exists
func DirExist(dir string) error {
	if err := os.Mkdir(dir, 0o755); err == nil {
		return nil
	} else if os.IsExist(err) {
		// check that the existing path is a directory
		info, err := os.Stat(dir)
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
