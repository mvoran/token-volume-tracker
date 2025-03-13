package utils

import (
	"os"
	"path/filepath"
)

// GetProjectRoot returns the absolute path to the project root directory.
// It looks for the go.mod file to determine the root.
func GetProjectRoot() (string, error) {
	// Start from the current directory
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// Walk up the directory tree until we find go.mod
	for {
		// Check if go.mod exists in current directory
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}

		// Get parent directory
		parent := filepath.Dir(dir)
		if parent == dir {
			// We've reached the root without finding go.mod
			return "", os.ErrNotExist
		}
		dir = parent
	}
}

// GetTestDataDir returns the absolute path to the testdata directory.
func GetTestDataDir() (string, error) {
	root, err := GetProjectRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "pkg", "analysis", "testdata"), nil
}
