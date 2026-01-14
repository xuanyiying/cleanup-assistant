package cleaner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/xuanyiying/cleanup-cli/internal/output"
	"github.com/xuanyiying/cleanup-cli/internal/transaction"
)

// CleanOptions configures cleanup behavior
type CleanOptions struct {
	DryRun      bool           // Preview only, don't delete
	Force       bool           // Permanently delete instead of trash
	Categories  []JunkCategory // Categories to clean (empty = all)
	Interactive bool           // Prompt for uncertain files
	TrashPath   string         // Custom trash directory
}

// CleanResult represents the result of a cleanup operation
type CleanResult struct {
	Cleaned    []*JunkFile
	Skipped    []*JunkFile
	Failed     []*JunkFile
	SpaceFreed int64
	Errors     []error
}

// SystemCleaner handles system junk cleanup
type SystemCleaner struct {
	scanner    *JunkScanner
	classifier *FileClassifier
	prompt     *InteractivePrompt
	console    *output.Console
	txnManager *transaction.Manager
}

// NewSystemCleaner creates a new system cleaner
func NewSystemCleaner(txnManager *transaction.Manager) *SystemCleaner {
	console := output.NewConsole(os.Stdout)

	return &SystemCleaner{
		scanner:    NewJunkScanner(),
		classifier: NewFileClassifier(),
		prompt:     NewInteractivePrompt(console, os.Stdin),
		console:    console,
		txnManager: txnManager,
	}
}

// ClearLocations clears all junk locations
func (c *SystemCleaner) ClearLocations() {
	c.scanner.ClearLocations()
}

// Configure configures the cleaner with custom settings
func (c *SystemCleaner) Configure(junkLocations []string, importantPatterns []string) {
	// Add custom junk locations
	for _, loc := range junkLocations {
		c.scanner.AddLocation(&JunkLocation{
			Path:        loc,
			Category:    CategoryTemp, // Default to temp for custom locations
			Description: "Custom junk location",
			Platform:    "all",
		})
	}

	// Add custom important patterns
	for _, pattern := range importantPatterns {
		pType := "name"
		if strings.HasPrefix(pattern, "*.") {
			pType = "extension"
		} else if strings.Contains(pattern, "/") || strings.Contains(pattern, "\\") {
			pType = "path"
		}

		c.classifier.AddPattern(&ImportantPattern{
			Pattern:     pattern,
			Type:        pType,
			Importance:  ImportanceImportant,
			Description: "Custom important pattern",
		})
	}
}

// Preview shows what would be cleaned without actually cleaning
func (c *SystemCleaner) Preview(ctx context.Context, opts *CleanOptions) (*ScanResult, error) {
	if opts == nil {
		opts = &CleanOptions{}
	}

	// Scan for junk files
	var result *ScanResult
	var err error

	if len(opts.Categories) > 0 {
		// Scan specific categories
		result = &ScanResult{
			Files:      []*JunkFile{},
			TotalSize:  0,
			ByCategory: make(map[JunkCategory][]*JunkFile),
			Skipped:    []string{},
			Errors:     []error{},
		}

		for _, category := range opts.Categories {
			categoryResult, err := c.scanner.ScanCategory(ctx, category)
			if err != nil {
				return nil, fmt.Errorf("failed to scan category %s: %w", category, err)
			}

			// Merge results
			result.Files = append(result.Files, categoryResult.Files...)
			result.TotalSize += categoryResult.TotalSize
			for cat, files := range categoryResult.ByCategory {
				result.ByCategory[cat] = append(result.ByCategory[cat], files...)
			}
			result.Skipped = append(result.Skipped, categoryResult.Skipped...)
			result.Errors = append(result.Errors, categoryResult.Errors...)
		}
	} else {
		// Scan all categories
		result, err = c.scanner.Scan(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to scan for junk files: %w", err)
		}
	}

	return result, nil
}

// Clean performs the cleanup operation
func (c *SystemCleaner) Clean(ctx context.Context, opts *CleanOptions) (*CleanResult, error) {
	if opts == nil {
		opts = &CleanOptions{}
	}

	// First, preview what will be cleaned
	scanResult, err := c.Preview(ctx, opts)
	if err != nil {
		return nil, err
	}

	result := &CleanResult{
		Cleaned:    []*JunkFile{},
		Skipped:    []*JunkFile{},
		Failed:     []*JunkFile{},
		SpaceFreed: 0,
		Errors:     []error{},
	}

	// If dry run, just return the scan result
	if opts.DryRun {
		c.console.Info("Dry run mode - no files will be deleted")
		c.displayPreview(scanResult)
		return result, nil
	}

	// Start a transaction for rollback support
	tx := c.txnManager.Begin()

	// Process each file
	for _, file := range scanResult.Files {
		// Check if context is cancelled
		select {
		case <-ctx.Done():
			// Rollback transaction on cancellation
			if err := c.txnManager.Rollback(tx); err != nil {
				c.console.Error("Failed to rollback transaction: %v", err)
			}
			return result, ctx.Err()
		default:
		}

		// Check if file is uncertain and interactive mode is enabled
		if opts.Interactive && c.classifier.IsUncertain(file.Path) {
			filePrompt := &FilePrompt{
				Path:    file.Path,
				Size:    file.Size,
				Type:    string(file.Category),
				ModTime: file.ModTime,
				Reason:  "File classification is uncertain",
			}

			action, err := c.prompt.Prompt(filePrompt)
			if err != nil {
				result.Errors = append(result.Errors, fmt.Errorf("prompt error for %s: %w", file.Path, err))
				result.Failed = append(result.Failed, file)
				continue
			}

			if action == ActionNo || action == ActionSkipAll {
				result.Skipped = append(result.Skipped, file)
				continue
			}
		}

		// Clean the file
		if err := c.cleanFile(file, opts, tx); err != nil {
			result.Errors = append(result.Errors, err)
			result.Failed = append(result.Failed, file)
		} else {
			result.Cleaned = append(result.Cleaned, file)
			result.SpaceFreed += file.Size
		}
	}

	// Commit transaction
	if err := c.txnManager.Commit(tx); err != nil {
		c.console.Error("Failed to commit transaction: %v", err)
		// Try to rollback
		if rbErr := c.txnManager.Rollback(tx); rbErr != nil {
			c.console.Error("Failed to rollback transaction: %v", rbErr)
		}
		return result, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Display results
	c.displayResults(result)

	return result, nil
}

// CleanCategory cleans only a specific category
func (c *SystemCleaner) CleanCategory(ctx context.Context, category JunkCategory, opts *CleanOptions) (*CleanResult, error) {
	if opts == nil {
		opts = &CleanOptions{}
	}

	// Set category filter
	opts.Categories = []JunkCategory{category}

	return c.Clean(ctx, opts)
}

// cleanFile cleans a single file (move to trash or permanently delete)
func (c *SystemCleaner) cleanFile(file *JunkFile, opts *CleanOptions, tx *transaction.Transaction) error {
	if opts.Force {
		// Permanently delete
		if err := os.Remove(file.Path); err != nil {
			return fmt.Errorf("failed to delete %s: %w", file.Path, err)
		}

		// Record operation for transaction
		op := &transaction.ExecutedOperation{
			Type:   transaction.OpDelete,
			Source: file.Path,
			Target: "",
			Backup: "", // No backup for permanent delete
		}
		c.txnManager.AddOperation(tx, op)

	} else {
		// Move to trash
		trashPath := opts.TrashPath
		if trashPath == "" {
			trashPath = c.getDefaultTrashPath()
		}

		// Ensure trash directory exists
		if err := os.MkdirAll(trashPath, 0755); err != nil {
			return fmt.Errorf("failed to create trash directory: %w", err)
		}

		// Generate unique trash filename
		trashFile := filepath.Join(trashPath, filepath.Base(file.Path))
		counter := 1
		for {
			if _, err := os.Stat(trashFile); os.IsNotExist(err) {
				break
			}
			trashFile = filepath.Join(trashPath, fmt.Sprintf("%s.%d", filepath.Base(file.Path), counter))
			counter++
		}

		// Move file to trash
		if err := os.Rename(file.Path, trashFile); err != nil {
			return fmt.Errorf("failed to move %s to trash: %w", file.Path, err)
		}

		// Record operation for transaction
		op := &transaction.ExecutedOperation{
			Type:   transaction.OpMove,
			Source: file.Path,
			Target: trashFile,
			Backup: trashFile, // Backup is the trash location
		}
		c.txnManager.AddOperation(tx, op)
	}

	return nil
}

// getDefaultTrashPath returns the default trash path for the current platform
func (c *SystemCleaner) getDefaultTrashPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ".cleanup_trash"
	}

	return filepath.Join(homeDir, ".cleanup", "trash")
}

// displayPreview displays a preview of files to be cleaned
func (c *SystemCleaner) displayPreview(result *ScanResult) {
	c.console.Info("\nFiles to be cleaned:")
	c.console.Info("====================")

	for category, files := range result.ByCategory {
		c.console.Info("\n%s (%d files):", category, len(files))
		for _, file := range files {
			c.console.Info("  - %s (%s)", file.Path, formatFileSize(file.Size))
		}
	}

	c.console.Info("\nTotal: %d files, %s", len(result.Files), formatFileSize(result.TotalSize))

	if len(result.Skipped) > 0 {
		c.console.Warning("\nSkipped %d paths due to permissions", len(result.Skipped))
	}

	if len(result.Errors) > 0 {
		c.console.Error("\nEncountered %d errors during scan", len(result.Errors))
	}
}

// displayResults displays the results of a cleanup operation
func (c *SystemCleaner) displayResults(result *CleanResult) {
	c.console.Success("\nCleanup completed!")
	c.console.Info("==================")
	c.console.Info("Cleaned: %d files", len(result.Cleaned))
	c.console.Info("Skipped: %d files", len(result.Skipped))
	c.console.Info("Failed: %d files", len(result.Failed))
	c.console.Success("Space freed: %s", formatFileSize(result.SpaceFreed))

	if len(result.Errors) > 0 {
		c.console.Warning("\nErrors encountered:")
		for _, err := range result.Errors {
			c.console.Error("  - %v", err)
		}
	}
}
