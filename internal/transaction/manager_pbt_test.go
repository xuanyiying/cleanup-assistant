package transaction

import (
	"os"
	"path/filepath"
	"testing"

	"pgregory.net/rapid"
)

// TestTransactionUndoRoundTrip validates Property 9: Transaction Undo Round-Trip
// For any committed file operation (move or rename), executing undo SHALL restore
// the file to its original path and name.
// Feature: cleanup-cli, Property 9: Transaction Undo Round-Trip
// Validates: Requirements 7.2
func TestTransactionUndoRoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		tmpDir, tmpErr := os.MkdirTemp("", "txn-test-*")
		if tmpErr != nil {
			t.Fatalf("failed to create temp dir: %v", tmpErr)
		}
		defer os.RemoveAll(tmpDir)

		logPath := filepath.Join(tmpDir, "transactions.json")

		// Generate random filenames
		originalName := rapid.StringMatching(`[a-z]{3,10}\.txt`).Draw(t, "originalName")
		newName := rapid.StringMatching(`[a-z]{3,10}\.txt`).Draw(t, "newName")
		fileContent := rapid.StringMatching(`[a-zA-Z0-9 ]{10,100}`).Draw(t, "fileContent")

		// Create original file
		originalPath := filepath.Join(tmpDir, originalName)
		err := os.WriteFile(originalPath, []byte(fileContent), 0644)
		if err != nil {
			t.Fatalf("failed to create original file: %v", err)
		}

		// Verify original file exists
		if _, err := os.Stat(originalPath); err != nil {
			t.Fatalf("original file does not exist: %v", err)
		}

		// Simulate a move/rename operation
		newPath := filepath.Join(tmpDir, newName)
		err = os.Rename(originalPath, newPath)
		if err != nil {
			t.Fatalf("failed to rename file: %v", err)
		}

		// Verify new file exists and original doesn't
		if _, err := os.Stat(newPath); err != nil {
			t.Fatalf("new file does not exist: %v", err)
		}
		if _, err := os.Stat(originalPath); err == nil {
			t.Fatalf("original file still exists after rename")
		}

		// Create transaction manager and commit the operation
		manager := NewManager(logPath)
		tx := manager.Begin()

		op := &ExecutedOperation{
			Type:   OpMove,
			Source: originalPath,
			Target: newPath,
			Backup: originalPath,
		}
		manager.AddOperation(tx, op)

		err = manager.Commit(tx)
		if err != nil {
			t.Fatalf("failed to commit transaction: %v", err)
		}

		// Verify transaction is committed
		if tx.Status != StatusCommitted {
			t.Fatalf("transaction status is not committed: %s", tx.Status)
		}

		// Execute undo
		err = manager.Undo(tx.ID)
		if err != nil {
			t.Fatalf("failed to undo transaction: %v", err)
		}

		// ROUND-TRIP PROPERTY: After undo, the file should be restored to original path
		// Verify original file exists again
		if _, err := os.Stat(originalPath); err != nil {
			t.Fatalf("original file was not restored after undo: %v", err)
		}

		// Verify new file no longer exists
		if _, err := os.Stat(newPath); err == nil {
			t.Fatalf("new file still exists after undo")
		}

		// Verify file content is preserved
		restoredContent, err := os.ReadFile(originalPath)
		if err != nil {
			t.Fatalf("failed to read restored file: %v", err)
		}

		if string(restoredContent) != fileContent {
			t.Fatalf("file content was not preserved: expected %q, got %q", fileContent, string(restoredContent))
		}

		// Verify transaction status is updated to rolledback
		history, err := manager.GetHistory(1)
		if err != nil {
			t.Fatalf("failed to get history: %v", err)
		}

		if len(history) == 0 {
			t.Fatalf("transaction not found in history")
		}

		if history[0].Status != StatusRolledback {
			t.Fatalf("transaction status is not rolledback: %s", history[0].Status)
		}
	})
}

// TestTransactionUndoMultipleOperations validates that undo works correctly
// with multiple operations in a single transaction, reversing them in reverse order.
// Feature: cleanup-cli, Property 9: Transaction Undo Round-Trip
// Validates: Requirements 7.2
func TestTransactionUndoMultipleOperations(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		tmpDir, tmpErr := os.MkdirTemp("", "txn-test-*")
		if tmpErr != nil {
			t.Fatalf("failed to create temp dir: %v", tmpErr)
		}
		defer os.RemoveAll(tmpDir)

		logPath := filepath.Join(tmpDir, "transactions.json")

		// Generate random filenames
		file1Name := rapid.StringMatching(`[a-z]{3,10}\.txt`).Draw(t, "file1Name")
		file2Name := rapid.StringMatching(`[a-z]{3,10}\.txt`).Draw(t, "file2Name")
		file1Content := rapid.StringMatching(`[a-zA-Z0-9 ]{10,50}`).Draw(t, "file1Content")
		file2Content := rapid.StringMatching(`[a-zA-Z0-9 ]{10,50}`).Draw(t, "file2Content")

		// Create original files
		file1Path := filepath.Join(tmpDir, file1Name)
		file2Path := filepath.Join(tmpDir, file2Name)

		err := os.WriteFile(file1Path, []byte(file1Content), 0644)
		if err != nil {
			t.Fatalf("failed to create file1: %v", err)
		}

		err = os.WriteFile(file2Path, []byte(file2Content), 0644)
		if err != nil {
			t.Fatalf("failed to create file2: %v", err)
		}

		// Simulate rename operations
		newFile1Path := filepath.Join(tmpDir, "renamed_"+file1Name)
		newFile2Path := filepath.Join(tmpDir, "renamed_"+file2Name)

		err = os.Rename(file1Path, newFile1Path)
		if err != nil {
			t.Fatalf("failed to rename file1: %v", err)
		}

		err = os.Rename(file2Path, newFile2Path)
		if err != nil {
			t.Fatalf("failed to rename file2: %v", err)
		}

		// Create transaction with multiple operations
		manager := NewManager(logPath)
		tx := manager.Begin()

		op1 := &ExecutedOperation{
			Type:   OpMove,
			Source: file1Path,
			Target: newFile1Path,
			Backup: file1Path,
		}
		op2 := &ExecutedOperation{
			Type:   OpMove,
			Source: file2Path,
			Target: newFile2Path,
			Backup: file2Path,
		}

		manager.AddOperation(tx, op1)
		manager.AddOperation(tx, op2)

		err = manager.Commit(tx)
		if err != nil {
			t.Fatalf("failed to commit transaction: %v", err)
		}

		// Execute undo
		err = manager.Undo(tx.ID)
		if err != nil {
			t.Fatalf("failed to undo transaction: %v", err)
		}

		// ROUND-TRIP PROPERTY: Both files should be restored to original paths
		if _, err := os.Stat(file1Path); err != nil {
			t.Fatalf("file1 was not restored after undo: %v", err)
		}

		if _, err := os.Stat(file2Path); err != nil {
			t.Fatalf("file2 was not restored after undo: %v", err)
		}

		// Verify renamed files no longer exist
		if _, err := os.Stat(newFile1Path); err == nil {
			t.Fatalf("renamed file1 still exists after undo")
		}

		if _, err := os.Stat(newFile2Path); err == nil {
			t.Fatalf("renamed file2 still exists after undo")
		}

		// Verify file contents are preserved
		content1, err := os.ReadFile(file1Path)
		if err != nil {
			t.Fatalf("failed to read restored file1: %v", err)
		}

		content2, err := os.ReadFile(file2Path)
		if err != nil {
			t.Fatalf("failed to read restored file2: %v", err)
		}

		if string(content1) != file1Content {
			t.Fatalf("file1 content was not preserved")
		}

		if string(content2) != file2Content {
			t.Fatalf("file2 content was not preserved")
		}
	})
}
