package organizer

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xuanyiying/cleanup-cli/internal/transaction"
	"pgregory.net/rapid"
)

// TestExtensionPreservationOnRename validates Property 2
// Feature: cleanup-cli, Property 2: Extension Preservation on Rename
// For any rename operation on a file with an extension, the resulting filename
// SHALL preserve the original file extension unchanged.
// Validates: Requirements 4.3
func TestExtensionPreservationOnRename(t *testing.T) {
	tmpDir := t.TempDir()
	txnLogPath := filepath.Join(tmpDir, "transactions.json")
	txnManager := transaction.NewManager(txnLogPath)
	organizer := NewOrganizer(txnManager)

	rapid.Check(t, func(t *rapid.T) {
		// Generate random filename with extension
		baseName := rapid.StringMatching(`[a-z]{3,10}`).Draw(t, "baseName")
		ext := rapid.StringMatching(`\.(txt|md|go|py|js|pdf|doc)`).Draw(t, "ext")
		originalName := baseName + ext

		// Generate new name without extension
		newBaseName := rapid.StringMatching(`[a-z]{3,10}`).Draw(t, "newBaseName")

		// Create source file
		sourcePath := filepath.Join(tmpDir, originalName)
		err := os.WriteFile(sourcePath, []byte("test content"), 0644)
		require.NoError(t, err)
		defer os.Remove(sourcePath)

		// Rename with PreserveExtension=true
		opts := &RenameOptions{
			DryRun:            false,
			PreserveExtension: true,
			ConflictStrategy:  ConflictSkip,
		}

		result, err := organizer.Rename(context.Background(), sourcePath, newBaseName, opts)
		require.NoError(t, err)
		require.True(t, result.Success)

		// Verify the extension is preserved
		resultExt := filepath.Ext(result.Target)
		if resultExt != ext {
			t.Errorf("extension not preserved: expected %s, got %s", ext, resultExt)
		}

		// Verify file exists at new location
		_, err = os.Stat(result.Target)
		require.NoError(t, err)
	})
}

// TestDryRunModeSafety validates Property 3
// Feature: cleanup-cli, Property 3: Dry-Run Mode Safety
// For any sequence of operations executed in dry-run mode, the file system state
// SHALL remain unchanged (no files moved, renamed, or deleted).
// Validates: Requirements 4.5
func TestDryRunModeSafety(t *testing.T) {
	tmpDir := t.TempDir()
	txnLogPath := filepath.Join(tmpDir, "transactions.json")
	txnManager := transaction.NewManager(txnLogPath)
	organizer := NewOrganizer(txnManager)

	rapid.Check(t, func(t *rapid.T) {
		// Create source file
		sourcePath := filepath.Join(tmpDir, "test_file.txt")
		err := os.WriteFile(sourcePath, []byte("test content"), 0644)
		require.NoError(t, err)
		defer os.Remove(sourcePath)

		// Get original file info
		originalInfo, err := os.Stat(sourcePath)
		require.NoError(t, err)

		// Perform rename in dry-run mode
		opts := &RenameOptions{
			DryRun:            true,
			PreserveExtension: true,
			ConflictStrategy:  ConflictSkip,
		}

		result, err := organizer.Rename(context.Background(), sourcePath, "new_name", opts)
		require.NoError(t, err)
		require.True(t, result.Success)

		// Verify original file still exists at original location
		currentInfo, err := os.Stat(sourcePath)
		require.NoError(t, err)

		// Verify file was not modified
		if currentInfo.ModTime() != originalInfo.ModTime() {
			t.Error("file was modified during dry-run")
		}

		// Verify new file does not exist
		_, err = os.Stat(result.Target)
		if err == nil {
			t.Error("new file was created during dry-run")
		}
	})
}

// TestConflictResolutionConsistency validates Property 4
// Feature: cleanup-cli, Property 4: Conflict Resolution Consistency
// For any file operation (rename or move) that would result in a name conflict,
// the Organizer SHALL either skip, append a unique suffix, or prompt based on
// the configured strategy, and the resulting filename SHALL be unique in the
// target directory.
// Validates: Requirements 4.2, 5.3
func TestConflictResolutionConsistency(t *testing.T) {
	tmpDir := t.TempDir()
	txnLogPath := filepath.Join(tmpDir, "transactions.json")
	txnManager := transaction.NewManager(txnLogPath)
	organizer := NewOrganizer(txnManager)

	// Test suffix strategy
	t.Run("suffix_strategy", func(t *testing.T) {
		// Create two files with same name
		file1 := filepath.Join(tmpDir, "conflict_test_1.txt")
		file2 := filepath.Join(tmpDir, "conflict_test_2.txt")

		err := os.WriteFile(file1, []byte("content1"), 0644)
		require.NoError(t, err)
		defer os.Remove(file1)

		err = os.WriteFile(file2, []byte("content2"), 0644)
		require.NoError(t, err)
		defer os.Remove(file2)

		// Rename file2 to same name as file1 with suffix strategy
		opts := &RenameOptions{
			DryRun:            false,
			PreserveExtension: true,
			ConflictStrategy:  ConflictSuffix,
		}

		result, err := organizer.Rename(context.Background(), file2, "conflict_test_1", opts)
		require.NoError(t, err)
		require.True(t, result.Success)

		// Verify both files exist and have unique names
		_, err = os.Stat(file1)
		require.NoError(t, err)

		_, err = os.Stat(result.Target)
		require.NoError(t, err)

		// Verify they have different names
		if file1 == result.Target {
			t.Error("files have the same name after conflict resolution")
		}

		// Clean up
		os.Remove(result.Target)
	})

	// Test skip strategy
	t.Run("skip_strategy", func(t *testing.T) {
		// Create two files with same name
		file1 := filepath.Join(tmpDir, "skip_test_1.txt")
		file2 := filepath.Join(tmpDir, "skip_test_2.txt")

		err := os.WriteFile(file1, []byte("content1"), 0644)
		require.NoError(t, err)
		defer os.Remove(file1)

		err = os.WriteFile(file2, []byte("content2"), 0644)
		require.NoError(t, err)
		defer os.Remove(file2)

		// Rename file2 to same name as file1 with skip strategy
		opts := &RenameOptions{
			DryRun:            false,
			PreserveExtension: true,
			ConflictStrategy:  ConflictSkip,
		}

		result, err := organizer.Rename(context.Background(), file2, "skip_test_1", opts)
		require.NoError(t, err)
		require.True(t, result.Success)

		// Verify file2 still exists at original location (was skipped)
		_, err = os.Stat(file2)
		require.NoError(t, err)

		// Verify file1 still exists
		_, err = os.Stat(file1)
		require.NoError(t, err)
	})
}

// TestFolderAutoCreation validates Property 5
// Feature: cleanup-cli, Property 5: Folder Auto-Creation
// For any move operation targeting a non-existent directory, the Organizer SHALL
// create the target directory before moving the file, and the move SHALL succeed.
// Validates: Requirements 5.2
func TestFolderAutoCreation(t *testing.T) {
	tmpDir := t.TempDir()
	txnLogPath := filepath.Join(tmpDir, "transactions.json")
	txnManager := transaction.NewManager(txnLogPath)
	organizer := NewOrganizer(txnManager)

	rapid.Check(t, func(t *rapid.T) {
		// Create source file
		sourcePath := filepath.Join(tmpDir, "source_file.txt")
		err := os.WriteFile(sourcePath, []byte("test content"), 0644)
		require.NoError(t, err)
		defer os.Remove(sourcePath)

		// Generate random target directory path that doesn't exist
		dirName := rapid.StringMatching(`[a-z]{3,10}`).Draw(t, "dirName")
		targetDir := filepath.Join(tmpDir, dirName)

		// Verify target directory doesn't exist
		_, err = os.Stat(targetDir)
		if err == nil {
			t.Skip("target directory already exists")
		}

		// Move file to non-existent directory
		opts := &MoveOptions{
			DryRun:           false,
			CreateTargetDir:  true,
			ConflictStrategy: ConflictSkip,
		}

		result, err := organizer.Move(context.Background(), sourcePath, targetDir, opts)
		require.NoError(t, err)
		require.True(t, result.Success)

		// Verify target directory was created
		_, err = os.Stat(targetDir)
		require.NoError(t, err)

		// Verify file exists at target location
		_, err = os.Stat(result.Target)
		require.NoError(t, err)

		// Clean up
		os.Remove(result.Target)
		os.Remove(targetDir)
	})
}

// TestSafeDeletion validates Property 11
// Feature: cleanup-cli, Property 11: Safe Deletion
// For any delete operation, the file SHALL be moved to the configured trash
// directory rather than permanently deleted, unless explicitly overridden.
// Validates: Requirements 7.4, 7.5
func TestSafeDeletion(t *testing.T) {
	tmpDir := t.TempDir()
	txnLogPath := filepath.Join(tmpDir, "transactions.json")
	txnManager := transaction.NewManager(txnLogPath)
	organizer := NewOrganizer(txnManager)

	rapid.Check(t, func(t *rapid.T) {
		// Create source file
		fileName := rapid.StringMatching(`[a-z]{3,10}\.txt`).Draw(t, "fileName")
		sourcePath := filepath.Join(tmpDir, fileName)
		err := os.WriteFile(sourcePath, []byte("test content"), 0644)
		require.NoError(t, err)
		defer os.Remove(sourcePath)

		// Create trash directory
		trashDir := filepath.Join(tmpDir, "trash")

		// Delete file (move to trash)
		result, err := organizer.Delete(context.Background(), sourcePath, trashDir)
		require.NoError(t, err)
		require.True(t, result.Success)

		// Verify original file no longer exists at original location
		_, err = os.Stat(sourcePath)
		if err == nil {
			t.Error("original file still exists after delete")
		}

		// Verify file exists in trash
		_, err = os.Stat(result.Target)
		require.NoError(t, err)

		// Verify trash directory was created
		_, err = os.Stat(trashDir)
		require.NoError(t, err)

		// Clean up
		os.Remove(result.Target)
		os.Remove(trashDir)
	})
}

// Unit tests for basic functionality

func TestRenameBasic(t *testing.T) {
	tmpDir := t.TempDir()
	txnLogPath := filepath.Join(tmpDir, "transactions.json")
	txnManager := transaction.NewManager(txnLogPath)
	organizer := NewOrganizer(txnManager)

	// Create source file
	sourcePath := filepath.Join(tmpDir, "original.txt")
	err := os.WriteFile(sourcePath, []byte("test content"), 0644)
	require.NoError(t, err)

	// Rename file
	opts := &RenameOptions{
		DryRun:            false,
		PreserveExtension: true,
		ConflictStrategy:  ConflictSkip,
	}

	result, err := organizer.Rename(context.Background(), sourcePath, "renamed", opts)
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.True(t, strings.HasSuffix(result.Target, "renamed.txt"))

	// Verify file exists at new location
	_, err = os.Stat(result.Target)
	require.NoError(t, err)

	// Verify original file no longer exists
	_, err = os.Stat(sourcePath)
	assert.Error(t, err)
}

func TestMoveBasic(t *testing.T) {
	tmpDir := t.TempDir()
	txnLogPath := filepath.Join(tmpDir, "transactions.json")
	txnManager := transaction.NewManager(txnLogPath)
	organizer := NewOrganizer(txnManager)

	// Create source file
	sourcePath := filepath.Join(tmpDir, "source.txt")
	err := os.WriteFile(sourcePath, []byte("test content"), 0644)
	require.NoError(t, err)

	// Create target directory
	targetDir := filepath.Join(tmpDir, "target")
	err = os.MkdirAll(targetDir, 0755)
	require.NoError(t, err)

	// Move file
	opts := &MoveOptions{
		DryRun:           false,
		CreateTargetDir:  false,
		ConflictStrategy: ConflictSkip,
	}

	result, err := organizer.Move(context.Background(), sourcePath, targetDir, opts)
	require.NoError(t, err)
	assert.True(t, result.Success)

	// Verify file exists at target location
	_, err = os.Stat(result.Target)
	require.NoError(t, err)

	// Verify original file no longer exists
	_, err = os.Stat(sourcePath)
	assert.Error(t, err)
}

func TestMoveWithAutoCreateDir(t *testing.T) {
	tmpDir := t.TempDir()
	txnLogPath := filepath.Join(tmpDir, "transactions.json")
	txnManager := transaction.NewManager(txnLogPath)
	organizer := NewOrganizer(txnManager)

	// Create source file
	sourcePath := filepath.Join(tmpDir, "source.txt")
	err := os.WriteFile(sourcePath, []byte("test content"), 0644)
	require.NoError(t, err)

	// Target directory doesn't exist
	targetDir := filepath.Join(tmpDir, "nonexistent", "target")

	// Move file with auto-create
	opts := &MoveOptions{
		DryRun:           false,
		CreateTargetDir:  true,
		ConflictStrategy: ConflictSkip,
	}

	result, err := organizer.Move(context.Background(), sourcePath, targetDir, opts)
	require.NoError(t, err)
	assert.True(t, result.Success)

	// Verify target directory was created
	_, err = os.Stat(targetDir)
	require.NoError(t, err)

	// Verify file exists at target location
	_, err = os.Stat(result.Target)
	require.NoError(t, err)
}

func TestDeleteToTrash(t *testing.T) {
	tmpDir := t.TempDir()
	txnLogPath := filepath.Join(tmpDir, "transactions.json")
	txnManager := transaction.NewManager(txnLogPath)
	organizer := NewOrganizer(txnManager)

	// Create source file
	sourcePath := filepath.Join(tmpDir, "file_to_delete.txt")
	err := os.WriteFile(sourcePath, []byte("test content"), 0644)
	require.NoError(t, err)

	// Create trash directory
	trashDir := filepath.Join(tmpDir, "trash")

	// Delete file
	result, err := organizer.Delete(context.Background(), sourcePath, trashDir)
	require.NoError(t, err)
	assert.True(t, result.Success)

	// Verify original file no longer exists
	_, err = os.Stat(sourcePath)
	assert.Error(t, err)

	// Verify file exists in trash
	_, err = os.Stat(result.Target)
	require.NoError(t, err)
}

// TestBatchErrorResilience validates Property 12
// Feature: cleanup-cli, Property 12: Batch Error Resilience
// For any batch operation where some files fail, the successful operations SHALL
// be completed and logged, and all errors SHALL be collected and reported at the end.
// Validates: Requirements 8.3
func TestBatchErrorResilience(t *testing.T) {
	tmpDir := t.TempDir()
	txnLogPath := filepath.Join(tmpDir, "transactions.json")
	txnManager := transaction.NewManager(txnLogPath)
	organizer := NewOrganizerWithDeps(txnManager, nil, nil)

	rapid.Check(t, func(t *rapid.T) {
		// Generate a mix of valid and invalid files
		numValidFiles := rapid.IntRange(1, 5).Draw(t, "numValidFiles")
		numInvalidFiles := rapid.IntRange(1, 3).Draw(t, "numInvalidFiles")

		// Create valid files
		validFiles := make([]*PlannedOperation, 0)
		for i := 0; i < numValidFiles; i++ {
			fileName := rapid.StringMatching(`[a-z]{3,8}\.txt`).Draw(t, "fileName")
			sourcePath := filepath.Join(tmpDir, fileName)
			err := os.WriteFile(sourcePath, []byte("test content"), 0644)
			require.NoError(t, err)

			targetDir := filepath.Join(tmpDir, "target")
			validFiles = append(validFiles, &PlannedOperation{
				Type:   OpMove,
				Source: sourcePath,
				Target: filepath.Join(targetDir, fileName),
				Reason: "test",
			})
		}

		// Create invalid files (non-existent sources)
		invalidFiles := make([]*PlannedOperation, 0)
		for i := 0; i < numInvalidFiles; i++ {
			fileName := rapid.StringMatching(`[a-z]{3,8}\.txt`).Draw(t, "fileName")
			nonExistentPath := filepath.Join(tmpDir, "nonexistent", fileName)

			targetDir := filepath.Join(tmpDir, "target")
			invalidFiles = append(invalidFiles, &PlannedOperation{
				Type:   OpMove,
				Source: nonExistentPath,
				Target: filepath.Join(targetDir, fileName),
				Reason: "test",
			})
		}

		// Combine all operations
		allOps := append(validFiles, invalidFiles...)

		// Create plan
		plan := &OrganizePlan{
			Operations: allOps,
			Summary: &PlanSummary{
				TotalFiles:      len(allOps),
				TotalOperations: len(allOps),
			},
		}

		// Execute plan
		strategy := &OrganizeStrategy{
			DryRun:           false,
			CreateFolders:    true,
			ConflictStrategy: ConflictSuffix,
			MaxConcurrency:   2,
		}

		result, err := organizer.ExecutePlan(context.Background(), plan, strategy)
		require.NoError(t, err)

		// Verify error resilience properties
		// 1. Total processed should equal total operations
		totalProcessed := result.Successful + result.Failed
		if totalProcessed != len(allOps) {
			t.Errorf("not all operations were processed: expected %d, got %d", len(allOps), totalProcessed)
		}

		// 2. Failed count should match invalid files
		if result.Failed < numInvalidFiles {
			t.Errorf("not all invalid files were reported as failed: expected at least %d, got %d", numInvalidFiles, result.Failed)
		}

		// 3. Successful count should match valid files
		if result.Successful < numValidFiles {
			t.Errorf("not all valid files were processed successfully: expected at least %d, got %d", numValidFiles, result.Successful)
		}

		// 4. All errors should be collected
		if len(result.Errors) < numInvalidFiles {
			t.Errorf("not all errors were collected: expected at least %d, got %d", numInvalidFiles, len(result.Errors))
		}

		// 5. Failed files map should contain all failed operations
		if len(result.FailedFiles) < numInvalidFiles {
			t.Errorf("failed files map incomplete: expected at least %d, got %d", numInvalidFiles, len(result.FailedFiles))
		}

		// Clean up
		for _, op := range validFiles {
			os.Remove(op.Source)
			os.Remove(op.Target)
		}
		os.RemoveAll(filepath.Join(tmpDir, "target"))
	})
}
