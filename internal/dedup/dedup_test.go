package dedup

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDeduplicator_FindDuplicates(t *testing.T) {
	// Create temp directory with test files
	tmpDir := t.TempDir()

	// Create duplicate files
	content1 := []byte("This is test content")
	content2 := []byte("This is different content")

	// Create duplicates of content1
	files := []string{
		filepath.Join(tmpDir, "file1.txt"),
		filepath.Join(tmpDir, "file2.txt"),
		filepath.Join(tmpDir, "subdir", "file3.txt"),
	}

	os.Mkdir(filepath.Join(tmpDir, "subdir"), 0755)

	for _, file := range files {
		if err := os.WriteFile(file, content1, 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Create a unique file
	uniqueFile := filepath.Join(tmpDir, "unique.txt")
	if err := os.WriteFile(uniqueFile, content2, 0644); err != nil {
		t.Fatal(err)
	}

	// Find duplicates
	dedup := NewDeduplicator()
	dedup.MinSize = 1 // Lower threshold for testing

	groups, err := dedup.FindDuplicates(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("FindDuplicates failed: %v", err)
	}

	// Should find one group with 3 duplicates
	if len(groups) != 1 {
		t.Errorf("Expected 1 duplicate group, got %d", len(groups))
	}

	if len(groups) > 0 && len(groups[0].Files) != 3 {
		t.Errorf("Expected 3 files in group, got %d", len(groups[0].Files))
	}
}

func TestDeduplicator_MinSize(t *testing.T) {
	tmpDir := t.TempDir()

	// Create small file
	smallFile := filepath.Join(tmpDir, "small.txt")
	if err := os.WriteFile(smallFile, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}

	dedup := NewDeduplicator()
	dedup.MinSize = 1024 // 1KB minimum

	groups, err := dedup.FindDuplicates(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("FindDuplicates failed: %v", err)
	}

	// Should find no duplicates (file too small)
	if len(groups) != 0 {
		t.Errorf("Expected 0 duplicate groups for small files, got %d", len(groups))
	}
}

func TestCreateRemovalPlan_Newest(t *testing.T) {
	now := time.Now()
	groups := []*DuplicateGroup{
		{
			Hash: "abc123",
			Size: 1000,
			Files: []*FileInfo{
				{Path: "/path/new.txt", Size: 1000, ModTime: now},
				{Path: "/path/old.txt", Size: 1000, ModTime: now.Add(-time.Hour)},
			},
		},
	}

	dedup := NewDeduplicator()
	plan := dedup.CreateRemovalPlan(groups, "newest")

	if len(plan.ToKeep) != 1 {
		t.Errorf("Expected 1 file to keep, got %d", len(plan.ToKeep))
	}

	if len(plan.ToRemove) != 1 {
		t.Errorf("Expected 1 file to remove, got %d", len(plan.ToRemove))
	}

	if plan.ToKeep[0].Path != "/path/new.txt" {
		t.Errorf("Expected to keep newest file, got %s", plan.ToKeep[0].Path)
	}

	if plan.SpaceSaved != 1000 {
		t.Errorf("Expected space saved 1000, got %d", plan.SpaceSaved)
	}
}

func TestCreateRemovalPlan_Oldest(t *testing.T) {
	now := time.Now()
	groups := []*DuplicateGroup{
		{
			Hash: "abc123",
			Size: 1000,
			Files: []*FileInfo{
				{Path: "/path/new.txt", Size: 1000, ModTime: now},
				{Path: "/path/old.txt", Size: 1000, ModTime: now.Add(-time.Hour)},
			},
		},
	}

	dedup := NewDeduplicator()
	plan := dedup.CreateRemovalPlan(groups, "oldest")

	if plan.ToKeep[0].Path != "/path/old.txt" {
		t.Errorf("Expected to keep oldest file, got %s", plan.ToKeep[0].Path)
	}
}

func TestIsBackupLocation(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"/home/user/backup/file.txt", true},
		{"/home/user/.trash/file.txt", true},
		{"/home/user/documents/file.txt", false},
		{"/tmp/file.txt", true},
		{"/home/user/old/file.txt", true},
	}

	for _, tt := range tests {
		result := isBackupLocation(tt.path)
		if result != tt.expected {
			t.Errorf("isBackupLocation(%s) = %v, want %v", tt.path, result, tt.expected)
		}
	}
}

func TestGetStats(t *testing.T) {
	groups := []*DuplicateGroup{
		{
			Hash: "abc",
			Size: 1000,
			Files: []*FileInfo{
				{Path: "/a.txt", Size: 1000},
				{Path: "/b.txt", Size: 1000},
				{Path: "/c.txt", Size: 1000},
			},
		},
		{
			Hash: "def",
			Size: 500,
			Files: []*FileInfo{
				{Path: "/d.txt", Size: 500},
				{Path: "/e.txt", Size: 500},
			},
		},
	}

	stats := GetStats(groups)

	if stats.TotalGroups != 2 {
		t.Errorf("Expected 2 groups, got %d", stats.TotalGroups)
	}

	if stats.TotalFiles != 5 {
		t.Errorf("Expected 5 files, got %d", stats.TotalFiles)
	}

	if stats.TotalDuplicates != 3 {
		t.Errorf("Expected 3 duplicates, got %d", stats.TotalDuplicates)
	}

	// Wasted space: 2*1000 + 1*500 = 2500
	if stats.WastedSpace != 2500 {
		t.Errorf("Expected wasted space 2500, got %d", stats.WastedSpace)
	}

	if stats.LargestDuplicate != 1000 {
		t.Errorf("Expected largest duplicate 1000, got %d", stats.LargestDuplicate)
	}
}

func TestExecuteRemovalPlan_DryRun(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	plan := &RemovalPlan{
		ToRemove: []*FileInfo{
			{Path: testFile},
		},
	}

	dedup := NewDeduplicator()
	err := dedup.ExecuteRemovalPlan(context.Background(), plan, true)
	if err != nil {
		t.Fatalf("ExecuteRemovalPlan failed: %v", err)
	}

	// File should still exist (dry run)
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Error("File was removed in dry run mode")
	}
}

func TestExecuteRemovalPlan_Actual(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	plan := &RemovalPlan{
		ToRemove: []*FileInfo{
			{Path: testFile},
		},
	}

	dedup := NewDeduplicator()
	err := dedup.ExecuteRemovalPlan(context.Background(), plan, false)
	if err != nil {
		t.Fatalf("ExecuteRemovalPlan failed: %v", err)
	}

	// File should be removed
	if _, err := os.Stat(testFile); !os.IsNotExist(err) {
		t.Error("File was not removed")
	}
}
