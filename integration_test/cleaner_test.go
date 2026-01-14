package integration_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xuanyiying/cleanup-cli/internal/cleaner"
	"github.com/xuanyiying/cleanup-cli/internal/transaction"
)

func TestCleanerEndToEnd(t *testing.T) {
	// Create temp dir for test
	tempDir := t.TempDir()
	junkDir := filepath.Join(tempDir, "junk")
	trashDir := filepath.Join(tempDir, "trash")
	txnLog := filepath.Join(tempDir, "txn.json")

	err := os.MkdirAll(junkDir, 0755)
	require.NoError(t, err)

	// Create some junk files
	junkFiles := []string{"test.tmp", "cache.log", "temp.bak"}
	for _, name := range junkFiles {
		err := os.WriteFile(filepath.Join(junkDir, name), []byte("junk content"), 0644)
		require.NoError(t, err)
	}

	// Setup components
	txnManager := transaction.NewManager(txnLog)
	systemCleaner := cleaner.NewSystemCleaner(txnManager)

	// Configure system cleaner to look at our junk dir
	systemCleaner.ClearLocations()
	systemCleaner.Configure([]string{junkDir}, nil)

	ctx := context.Background()

	// Step 1: Scan (Preview)
	opts := &cleaner.CleanOptions{
		DryRun: true,
	}
	scanResult, err := systemCleaner.Preview(ctx, opts)
	require.NoError(t, err)

	// Note: The scanner might find .DS_Store or other system files if they exist,
	// but in a fresh temp dir usually only what we created.
	// Also, scanner recursively scans.
	// Since we use Configure with custom location, it adds it as 'temp' category.

	// Verify we found at least our files
	foundCount := 0
	for _, file := range scanResult.Files {
		for _, name := range junkFiles {
			if filepath.Base(file.Path) == name {
				foundCount++
				break
			}
		}
	}
	assert.Equal(t, len(junkFiles), foundCount, "Should find all created junk files")

	// Step 2: Clean (Move to trash)
	opts.DryRun = false
	opts.TrashPath = trashDir
	opts.Force = false
	opts.Interactive = false // Disable interactive prompt for test

	cleanResult, err := systemCleaner.Clean(ctx, opts)
	require.NoError(t, err)

	assert.Equal(t, len(junkFiles), len(cleanResult.Cleaned), "Should clean all files")

	// Verify files moved to trash
	for _, name := range junkFiles {
		// File should be gone from junkDir
		_, err := os.Stat(filepath.Join(junkDir, name))
		assert.True(t, os.IsNotExist(err), "File should be removed from source")

		// File should exist in trashDir
		// Note: Trash filename might be different if collision, but here unique.
		_, err = os.Stat(filepath.Join(trashDir, name))
		assert.NoError(t, err, "File should be in trash")
	}

	// Step 3: Undo (Rollback)
	// Get last transaction
	history, err := txnManager.GetHistory(1)
	require.NoError(t, err)
	require.NotEmpty(t, history)

	lastTxn := history[0]
	err = txnManager.Undo(lastTxn.ID)
	require.NoError(t, err)

	// Verify files restored
	for _, name := range junkFiles {
		_, err := os.Stat(filepath.Join(junkDir, name))
		assert.NoError(t, err, "File should be restored to source")
	}
}
