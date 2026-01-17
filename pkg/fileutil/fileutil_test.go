package fileutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSafeRename(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()

	// Create source file
	src := filepath.Join(tmpDir, "source.txt")
	if err := os.WriteFile(src, []byte("test content"), 0644); err != nil {
		t.Fatal(err)
	}

	// Test successful rename
	dst := filepath.Join(tmpDir, "dest.txt")
	if err := SafeRename(src, dst); err != nil {
		t.Errorf("SafeRename failed: %v", err)
	}

	// Verify destination exists
	if !FileExists(dst) {
		t.Error("Destination file should exist")
	}

	// Verify source doesn't exist
	if FileExists(src) {
		t.Error("Source file should not exist")
	}
}

func TestSafeRenameWithBackup(t *testing.T) {
	tmpDir := t.TempDir()

	// Create source and existing destination
	src := filepath.Join(tmpDir, "source.txt")
	dst := filepath.Join(tmpDir, "dest.txt")

	os.WriteFile(src, []byte("source content"), 0644)
	os.WriteFile(dst, []byte("dest content"), 0644)

	// Rename should succeed and backup old destination
	if err := SafeRename(src, dst); err != nil {
		t.Errorf("SafeRename with backup failed: %v", err)
	}

	// Verify new content
	content, _ := os.ReadFile(dst)
	if string(content) != "source content" {
		t.Error("Destination should have source content")
	}
}

func TestSafeMove(t *testing.T) {
	tmpDir := t.TempDir()

	// Create source file
	src := filepath.Join(tmpDir, "source.txt")
	os.WriteFile(src, []byte("test"), 0644)

	// Move to new directory
	dstDir := filepath.Join(tmpDir, "subdir")
	if err := SafeMove(src, dstDir); err != nil {
		t.Errorf("SafeMove failed: %v", err)
	}

	// Verify destination exists
	dst := filepath.Join(dstDir, "source.txt")
	if !FileExists(dst) {
		t.Error("Destination file should exist")
	}
}

func TestCopyFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create source file
	src := filepath.Join(tmpDir, "source.txt")
	content := []byte("test content")
	os.WriteFile(src, content, 0644)

	// Copy file
	dst := filepath.Join(tmpDir, "dest.txt")
	if err := CopyFile(src, dst); err != nil {
		t.Errorf("CopyFile failed: %v", err)
	}

	// Verify both exist
	if !FileExists(src) || !FileExists(dst) {
		t.Error("Both files should exist after copy")
	}

	// Verify content
	dstContent, _ := os.ReadFile(dst)
	if string(dstContent) != string(content) {
		t.Error("Content should match")
	}
}

func TestFileExists(t *testing.T) {
	tmpDir := t.TempDir()

	// Non-existent file
	if FileExists(filepath.Join(tmpDir, "nonexistent.txt")) {
		t.Error("Non-existent file should return false")
	}

	// Existing file
	file := filepath.Join(tmpDir, "exists.txt")
	os.WriteFile(file, []byte("test"), 0644)
	if !FileExists(file) {
		t.Error("Existing file should return true")
	}
}

func TestDirExists(t *testing.T) {
	tmpDir := t.TempDir()

	// Existing directory
	if !DirExists(tmpDir) {
		t.Error("Existing directory should return true")
	}

	// Non-existent directory
	if DirExists(filepath.Join(tmpDir, "nonexistent")) {
		t.Error("Non-existent directory should return false")
	}

	// File (not directory)
	file := filepath.Join(tmpDir, "file.txt")
	os.WriteFile(file, []byte("test"), 0644)
	if DirExists(file) {
		t.Error("File should not be considered a directory")
	}
}

func TestEnsureDir(t *testing.T) {
	tmpDir := t.TempDir()

	// Create new directory
	newDir := filepath.Join(tmpDir, "new", "nested", "dir")
	if err := EnsureDir(newDir); err != nil {
		t.Errorf("EnsureDir failed: %v", err)
	}

	if !DirExists(newDir) {
		t.Error("Directory should exist after EnsureDir")
	}

	// Call again on existing directory (should not error)
	if err := EnsureDir(newDir); err != nil {
		t.Error("EnsureDir on existing directory should not error")
	}
}

func TestGetFileSize(t *testing.T) {
	tmpDir := t.TempDir()

	// Create file with known size
	file := filepath.Join(tmpDir, "file.txt")
	content := []byte("test content")
	os.WriteFile(file, content, 0644)

	size, err := GetFileSize(file)
	if err != nil {
		t.Errorf("GetFileSize failed: %v", err)
	}

	if size != int64(len(content)) {
		t.Errorf("Expected size %d, got %d", len(content), size)
	}
}

func TestIsEmpty(t *testing.T) {
	tmpDir := t.TempDir()

	// Empty directory
	emptyDir := filepath.Join(tmpDir, "empty")
	os.Mkdir(emptyDir, 0755)

	isEmpty, err := IsEmpty(emptyDir)
	if err != nil {
		t.Errorf("IsEmpty failed: %v", err)
	}
	if !isEmpty {
		t.Error("Empty directory should return true")
	}

	// Non-empty directory
	os.WriteFile(filepath.Join(emptyDir, "file.txt"), []byte("test"), 0644)
	isEmpty, err = IsEmpty(emptyDir)
	if err != nil {
		t.Errorf("IsEmpty failed: %v", err)
	}
	if isEmpty {
		t.Error("Non-empty directory should return false")
	}
}

func TestSafeRenameErrors(t *testing.T) {
	// Empty paths
	if err := SafeRename("", "dest"); err == nil {
		t.Error("Empty source should error")
	}

	if err := SafeRename("src", ""); err == nil {
		t.Error("Empty destination should error")
	}

	// Non-existent source
	if err := SafeRename("/nonexistent/file.txt", "/tmp/dest.txt"); err == nil {
		t.Error("Non-existent source should error")
	}
}
