package transaction

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransactionBeginCommit(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "transactions.json")

	manager := NewManager(logPath)

	// Begin a transaction
	tx := manager.Begin()
	assert.NotNil(t, tx)
	assert.Equal(t, StatusPending, tx.Status)
	assert.NotEmpty(t, tx.ID)

	// Add an operation
	op := &ExecutedOperation{
		Type:   OpMove,
		Source: "/path/to/source",
		Target: "/path/to/target",
		Backup: "/path/to/backup",
	}
	manager.AddOperation(tx, op)
	assert.Len(t, tx.Operations, 1)

	// Commit the transaction
	err := manager.Commit(tx)
	require.NoError(t, err)
	assert.Equal(t, StatusCommitted, tx.Status)

	// Verify log file was created
	assert.FileExists(t, logPath)
}

func TestTransactionPersistence(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "transactions.json")

	manager1 := NewManager(logPath)

	// Create and commit a transaction
	tx := manager1.Begin()
	op := &ExecutedOperation{
		Type:   OpRename,
		Source: "/path/to/old",
		Target: "/path/to/new",
	}
	manager1.AddOperation(tx, op)
	err := manager1.Commit(tx)
	require.NoError(t, err)

	// Create a new manager and load history
	manager2 := NewManager(logPath)
	history, err := manager2.GetHistory(10)
	require.NoError(t, err)
	assert.Len(t, history, 1)
	assert.Equal(t, tx.ID, history[0].ID)
	assert.Equal(t, StatusCommitted, history[0].Status)
	assert.Len(t, history[0].Operations, 1)
}

func TestGetHistory(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "transactions.json")

	manager := NewManager(logPath)

	// Create multiple transactions
	for i := 0; i < 5; i++ {
		tx := manager.Begin()
		op := &ExecutedOperation{
			Type:   OpMove,
			Source: "/source",
			Target: "/target",
		}
		manager.AddOperation(tx, op)
		err := manager.Commit(tx)
		require.NoError(t, err)
		time.Sleep(10 * time.Millisecond) // Ensure different timestamps
	}

	// Get all history
	history, err := manager.GetHistory(0)
	require.NoError(t, err)
	assert.Len(t, history, 5)

	// Get limited history
	history, err = manager.GetHistory(3)
	require.NoError(t, err)
	assert.Len(t, history, 3)
}

func TestUndoOperation(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "transactions.json")

	// Create source and backup files
	sourceFile := filepath.Join(tmpDir, "source.txt")
	backupFile := filepath.Join(tmpDir, "backup.txt")
	targetFile := filepath.Join(tmpDir, "target.txt")

	// Create source file
	err := os.WriteFile(sourceFile, []byte("content"), 0644)
	require.NoError(t, err)

	// Simulate a move operation by creating backup
	err = os.Rename(sourceFile, targetFile)
	require.NoError(t, err)

	// Create backup
	err = os.WriteFile(backupFile, []byte("content"), 0644)
	require.NoError(t, err)

	manager := NewManager(logPath)

	// Create and commit a transaction
	tx := manager.Begin()
	op := &ExecutedOperation{
		Type:   OpMove,
		Source: sourceFile,
		Target: targetFile,
		Backup: backupFile,
	}
	manager.AddOperation(tx, op)
	err = manager.Commit(tx)
	require.NoError(t, err)

	// Verify target file exists
	assert.FileExists(t, targetFile)

	// Undo the transaction
	err = manager.Undo(tx.ID)
	require.NoError(t, err)

	// Verify source file is restored
	assert.FileExists(t, sourceFile)
}

func TestUndoNonExistentTransaction(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "transactions.json")

	manager := NewManager(logPath)

	// Try to undo non-existent transaction
	err := manager.Undo("nonexistent")
	assert.Error(t, err)
}

func TestRollbackTransaction(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "transactions.json")

	// Create source and target files
	sourceFile := filepath.Join(tmpDir, "source.txt")
	targetFile := filepath.Join(tmpDir, "target.txt")

	// Create source file
	err := os.WriteFile(sourceFile, []byte("content"), 0644)
	require.NoError(t, err)

	// Simulate a move operation
	err = os.Rename(sourceFile, targetFile)
	require.NoError(t, err)

	manager := NewManager(logPath)

	// Create a transaction
	tx := manager.Begin()
	op := &ExecutedOperation{
		Type:   OpMove,
		Source: sourceFile,
		Target: targetFile,
		Backup: sourceFile,
	}
	manager.AddOperation(tx, op)

	// Rollback the transaction
	err = manager.Rollback(tx)
	require.NoError(t, err)
	assert.Equal(t, StatusRolledback, tx.Status)
}

func TestMultipleOperationsInTransaction(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "transactions.json")

	manager := NewManager(logPath)

	// Create a transaction with multiple operations
	tx := manager.Begin()

	ops := []*ExecutedOperation{
		{Type: OpMove, Source: "/src1", Target: "/tgt1"},
		{Type: OpRename, Source: "/src2", Target: "/tgt2"},
		{Type: OpDelete, Source: "/src3", Target: "/tgt3"},
	}

	for _, op := range ops {
		manager.AddOperation(tx, op)
	}

	err := manager.Commit(tx)
	require.NoError(t, err)

	// Verify all operations are persisted
	history, err := manager.GetHistory(1)
	require.NoError(t, err)
	assert.Len(t, history[0].Operations, 3)
}
