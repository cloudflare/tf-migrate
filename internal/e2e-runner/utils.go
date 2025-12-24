// utils.go provides general utility functions for file and directory operations.
//
// This file implements helper functions used throughout the e2e test suite:
//   - Repository root detection (finding go.mod)
//   - File copying with proper error handling and resource cleanup
//   - Recursive directory copying with permission preservation
//
// These utilities ensure consistent file handling across all test operations.
package e2e

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// getRepoRoot finds the repository root by looking for go.mod
func getRepoRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		return "."
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root
			return "."
		}
		dir = parent
	}
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}

	// Copy contents
	_, err = io.Copy(dstFile, srcFile)

	// Check close error - important for ensuring data is flushed to disk
	if closeErr := dstFile.Close(); closeErr != nil && err == nil {
		err = closeErr
	}

	return err
}

// copyDir recursively copies a directory
func copyDir(src, dst string) error {
	if err := os.MkdirAll(dst, permDir); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dst, err)
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}
