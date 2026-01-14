package visualizer

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xuanyiying/cleanup-cli/internal/output"
)

// TestDiffStateCaptureRoundTrip validates Property 3: Diff State Capture Round-Trip
// For any directory state, capturing the state before and after an operation SHALL produce 
// accurate TreeNode representations that, when compared, correctly identify all changes.
// Feature: enhanced-output-cleanup, Property 3: Diff State Capture Round-Trip
// Validates: Requirements 2.1, 2.2
func TestDiffStateCaptureRoundTrip(t *testing.T) {
	// Create temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "diff-capture-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create diff renderer
	console := output.NewConsole(&bytes.Buffer{})
	renderer := NewDiffRenderer(console)

	// Create initial structure
	file1Path := filepath.Join(tmpDir, "file1.txt")
	err = os.WriteFile(file1Path, []byte("content1"), 0644)
	if err != nil {
		t.Fatalf("failed to create file1: %v", err)
	}

	// Capture initial state
	beforeState, err := renderer.CaptureState(tmpDir)
	if err != nil {
		t.Fatalf("failed to capture before state: %v", err)
	}

	// PROPERTY: Captured state should accurately represent the directory structure
	if beforeState.Name != filepath.Base(tmpDir) {
		t.Fatalf("root node name mismatch: expected %q, got %q", 
			filepath.Base(tmpDir), beforeState.Name)
	}

	if !beforeState.IsDir {
		t.Fatalf("root node should be a directory")
	}

	// Make changes to the directory
	file2Path := filepath.Join(tmpDir, "file2.txt")
	err = os.WriteFile(file2Path, []byte("content2"), 0644)
	if err != nil {
		t.Fatalf("failed to create file2: %v", err)
	}

	// Capture state after changes
	afterState, err := renderer.CaptureState(tmpDir)
	if err != nil {
		t.Fatalf("failed to capture after state: %v", err)
	}

	// Compare states
	diff := renderer.Compare(beforeState, afterState)

	// PROPERTY: All diff entries should be valid
	for _, entry := range diff.Entries {
		if entry.Path == "" {
			t.Fatalf("diff entry has empty path")
		}
		
		if entry.Type == DiffMoved || entry.Type == DiffRenamed {
			if entry.NewPath == "" {
				t.Fatalf("move/rename entry missing NewPath: %+v", entry)
			}
		}
		
		if entry.Size < 0 {
			t.Fatalf("diff entry has negative size: %+v", entry)
		}
	}

	// PROPERTY: Should detect the added file
	foundAddedFile := false
	for _, entry := range diff.Entries {
		if entry.Type == DiffAdded && strings.Contains(entry.Path, "file2.txt") {
			foundAddedFile = true
			break
		}
	}

	if !foundAddedFile {
		t.Fatalf("should detect added file2.txt in diff")
	}

	// PROPERTY: Summary counts should match entry counts
	verifySummaryCounts(diff)
}

// TestDiffHighlightingCorrectness validates Property 4: Diff Highlighting Correctness
// For any diff result containing added, removed, or moved files, the rendered output SHALL 
// contain the correct color codes and symbols (green/+, red/-, yellow/→) for each change type.
// Feature: enhanced-output-cleanup, Property 4: Diff Highlighting Correctness
// Validates: Requirements 2.3, 2.4, 2.5
func TestDiffHighlightingCorrectness(t *testing.T) {
	// Create diff renderer
	console := output.NewConsole(&bytes.Buffer{})
	renderer := NewDiffRenderer(console)

	// Create test diff result with various entry types
	diff := &DiffResult{
		Entries: []*DiffEntry{
			{
				Type:  DiffAdded,
				Path:  "added.txt",
				Size:  100,
				IsDir: false,
			},
			{
				Type:  DiffRemoved,
				Path:  "removed.txt",
				Size:  200,
				IsDir: false,
			},
			{
				Type:    DiffMoved,
				Path:    "old/moved.txt",
				NewPath: "new/moved.txt",
				Size:    300,
				IsDir:   false,
			},
			{
				Type:    DiffRenamed,
				Path:    "oldname.txt",
				NewPath: "newname.txt",
				Size:    400,
				IsDir:   false,
			},
		},
		AddedCount:   1,
		RemovedCount: 1,
		MovedCount:   1,
		RenamedCount: 1,
	}

	// Render the diff
	rendered := renderer.Render(diff)

	// PROPERTY: Added files should have + symbol
	if !strings.Contains(rendered, SymbolAdded) {
		t.Fatalf("rendered output missing + symbol for added file")
	}
	if !strings.Contains(rendered, "added.txt") {
		t.Fatalf("rendered output missing added file path")
	}

	// PROPERTY: Removed files should have - symbol
	if !strings.Contains(rendered, SymbolRemoved) {
		t.Fatalf("rendered output missing - symbol for removed file")
	}
	if !strings.Contains(rendered, "removed.txt") {
		t.Fatalf("rendered output missing removed file path")
	}

	// PROPERTY: Moved files should have → symbol
	if !strings.Contains(rendered, SymbolMoved) {
		t.Fatalf("rendered output missing → symbol for moved file")
	}
	if !strings.Contains(rendered, "old/moved.txt") {
		t.Fatalf("rendered output missing moved file source path")
	}
	if !strings.Contains(rendered, "new/moved.txt") {
		t.Fatalf("rendered output missing moved file destination path")
	}

	// PROPERTY: Renamed files should have ~ symbol
	if !strings.Contains(rendered, SymbolRenamed) {
		t.Fatalf("rendered output missing ~ symbol for renamed file")
	}
	if !strings.Contains(rendered, "oldname.txt") {
		t.Fatalf("rendered output missing renamed file source name")
	}
	if !strings.Contains(rendered, "newname.txt") {
		t.Fatalf("rendered output missing renamed file destination name")
	}
}

// TestDiffSummaryAccuracy validates Property 5: Diff Summary Accuracy
// For any diff result, the summary counts (added, removed, moved, renamed) SHALL exactly 
// match the number of entries of each type in the diff entries list.
// Feature: enhanced-output-cleanup, Property 5: Diff Summary Accuracy
// Validates: Requirements 2.7
func TestDiffSummaryAccuracy(t *testing.T) {
	// Create diff renderer
	console := output.NewConsole(&bytes.Buffer{})
	renderer := NewDiffRenderer(console)

	// Test case 1: Mixed changes
	diff := &DiffResult{
		Entries: []*DiffEntry{
			{Type: DiffAdded, Path: "added1.txt", Size: 100, IsDir: false},
			{Type: DiffAdded, Path: "added2.txt", Size: 200, IsDir: false},
			{Type: DiffRemoved, Path: "removed1.txt", Size: 150, IsDir: false},
			{Type: DiffMoved, Path: "old/moved.txt", NewPath: "new/moved.txt", Size: 300, IsDir: false},
			{Type: DiffRenamed, Path: "old.txt", NewPath: "new.txt", Size: 250, IsDir: false},
		},
		AddedCount:   2,
		RemovedCount: 1,
		MovedCount:   1,
		RenamedCount: 1,
		TotalSize:    450, // added files total
	}

	// PROPERTY: Summary counts should match actual entries
	verifySummaryCounts(diff)

	// Render summary and verify it contains correct counts
	summary := renderer.RenderSummary(diff)

	if !strings.Contains(summary, "2 files added") {
		t.Fatalf("summary missing added count: %q", summary)
	}
	if !strings.Contains(summary, "1 files removed") {
		t.Fatalf("summary missing removed count: %q", summary)
	}
	if !strings.Contains(summary, "1 files moved") {
		t.Fatalf("summary missing moved count: %q", summary)
	}
	if !strings.Contains(summary, "1 files renamed") {
		t.Fatalf("summary missing renamed count: %q", summary)
	}

	// Test case 2: Empty diff
	emptyDiff := &DiffResult{
		Entries:      []*DiffEntry{},
		AddedCount:   0,
		RemovedCount: 0,
		MovedCount:   0,
		RenamedCount: 0,
		TotalSize:    0,
	}

	emptySummary := renderer.RenderSummary(emptyDiff)
	if !strings.Contains(emptySummary, "No changes made") {
		t.Fatalf("empty diff summary should show 'No changes made': %q", emptySummary)
	}
}

// verifySummaryCounts verifies that summary counts match actual entries
func verifySummaryCounts(diff *DiffResult) {
	actualAdded := 0
	actualRemoved := 0
	actualMoved := 0
	actualRenamed := 0

	for _, entry := range diff.Entries {
		switch entry.Type {
		case DiffAdded:
			actualAdded++
		case DiffRemoved:
			actualRemoved++
		case DiffMoved:
			actualMoved++
		case DiffRenamed:
			actualRenamed++
		}
	}

	if diff.AddedCount != actualAdded {
		panic(fmt.Sprintf("added count mismatch: expected %d, got %d", actualAdded, diff.AddedCount))
	}

	if diff.RemovedCount != actualRemoved {
		panic(fmt.Sprintf("removed count mismatch: expected %d, got %d", actualRemoved, diff.RemovedCount))
	}

	if diff.MovedCount != actualMoved {
		panic(fmt.Sprintf("moved count mismatch: expected %d, got %d", actualMoved, diff.MovedCount))
	}

	if diff.RenamedCount != actualRenamed {
		panic(fmt.Sprintf("renamed count mismatch: expected %d, got %d", actualRenamed, diff.RenamedCount))
	}
}