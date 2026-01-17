// Package fileutil provides safe file operation utilities with automatic backup and rollback support.
//
// This package offers higher-level file operations that handle common edge cases:
//   - Automatic backup creation before overwriting files
//   - Rollback on failure
//   - Directory creation as needed
//   - Permission preservation
//
// Example usage:
//
//	// Safe rename with automatic backup
//	if err := fileutil.SafeRename("old.txt", "new.txt"); err != nil {
//	    log.Fatal(err)
//	}
//
//	// Move file to directory (creates directory if needed)
//	if err := fileutil.SafeMove("file.txt", "/path/to/dir"); err != nil {
//	    log.Fatal(err)
//	}
//
//	// Copy file with permission preservation
//	if err := fileutil.CopyFile("source.txt", "dest.txt"); err != nil {
//	    log.Fatal(err)
//	}
package fileutil

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// SafeRename renames a file with backup and rollback support
func SafeRename(src, dst string) error {
	// Validate paths
	if src == "" || dst == "" {
		return fmt.Errorf("source and destination paths cannot be empty")
	}

	// Check if source exists
	if _, err := os.Stat(src); err != nil {
		return fmt.Errorf("source file does not exist: %w", err)
	}

	// Create backup if destination exists
	var backupPath string
	if _, err := os.Stat(dst); err == nil {
		backupPath = dst + ".backup"
		if err := os.Rename(dst, backupPath); err != nil {
			return fmt.Errorf("failed to create backup: %w", err)
		}
	}

	// Perform rename
	if err := os.Rename(src, dst); err != nil {
		// Restore backup if it was created
		if backupPath != "" {
			os.Rename(backupPath, dst)
		}
		return fmt.Errorf("failed to rename file: %w", err)
	}

	// Clean up backup
	if backupPath != "" {
		os.Remove(backupPath)
	}

	return nil
}

// SafeMove moves a file to a target directory with backup support
func SafeMove(src, dstDir string) error {
	// Validate paths
	if src == "" || dstDir == "" {
		return fmt.Errorf("source and destination paths cannot be empty")
	}

	// Check if source exists
	if _, err := os.Stat(src); err != nil {
		return fmt.Errorf("source file does not exist: %w", err)
	}

	// Ensure destination directory exists
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Construct destination path
	fileName := filepath.Base(src)
	dst := filepath.Join(dstDir, fileName)

	return SafeRename(src, dst)
}

// CopyFile copies a file from src to dst
func CopyFile(src, dst string) error {
	// Open source file
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer srcFile.Close()

	// Create destination file
	dstFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dstFile.Close()

	// Copy content
	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("failed to copy file content: %w", err)
	}

	// Sync to ensure data is written
	if err := dstFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync destination file: %w", err)
	}

	// Copy permissions
	srcInfo, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("failed to get source file info: %w", err)
	}

	if err := os.Chmod(dst, srcInfo.Mode()); err != nil {
		return fmt.Errorf("failed to set file permissions: %w", err)
	}

	return nil
}

// FileExists checks if a file exists
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// DirExists checks if a directory exists
func DirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// EnsureDir ensures a directory exists, creating it if necessary
func EnsureDir(path string) error {
	if DirExists(path) {
		return nil
	}

	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	return nil
}

// GetFileSize returns the size of a file in bytes
func GetFileSize(path string) (int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, fmt.Errorf("failed to get file info: %w", err)
	}

	return info.Size(), nil
}

// IsEmpty checks if a directory is empty
func IsEmpty(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, fmt.Errorf("failed to open directory: %w", err)
	}
	defer f.Close()

	_, err = f.Readdirnames(1)
	if err == io.EOF {
		return true, nil
	}

	return false, err
}
