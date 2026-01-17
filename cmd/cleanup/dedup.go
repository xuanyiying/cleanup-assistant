package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/xuanyiying/cleanup-cli/internal/dedup"
	"github.com/xuanyiying/cleanup-cli/internal/output"
)

var (
	dedupMinSize     int64
	dedupMaxSize     int64
	dedupKeepStrategy string
	dedupAutoRemove  bool
)

// dedupCmd represents the dedup command
var dedupCmd = &cobra.Command{
	Use:     "dedup [path]",
	Aliases: []string{"dup", "duplicate"},
	Short:   "Find and remove duplicate files",
	Long: `Find duplicate files in a directory based on content hash.
Supports different strategies for choosing which file to keep.

Examples:
  cleanup dedup ~/Downloads
  cleanup dedup ~/Documents --keep newest
  cleanup dedup . --min-size 1048576  # Only files >= 1MB
  cleanup dedup . --dry-run           # Preview without removing`,
	Args: cobra.MaximumNArgs(1),
	RunE: runDedup,
}

func init() {
	dedupCmd.Flags().Int64Var(&dedupMinSize, "min-size", 1024, "Minimum file size in bytes")
	dedupCmd.Flags().Int64Var(&dedupMaxSize, "max-size", 100*1024*1024, "Maximum file size in bytes (0 = no limit)")
	dedupCmd.Flags().StringVar(&dedupKeepStrategy, "keep", "newest", "Which file to keep: newest, oldest, first")
	dedupCmd.Flags().BoolVar(&dedupAutoRemove, "auto", false, "Automatically remove duplicates without confirmation")
	dedupCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview changes without executing")

	rootCmd.AddCommand(dedupCmd)
}

func runDedup(cmd *cobra.Command, args []string) error {
	path := "."
	if len(args) > 0 {
		path = args[0]
	}

	// Resolve absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}

	// Verify directory exists
	if _, err := os.Stat(absPath); err != nil {
		return fmt.Errorf("directory not found: %w", err)
	}

	ctx := context.Background()
	console := output.NewConsole(os.Stdout)

	// Create deduplicator
	deduplicator := dedup.NewDeduplicator()
	deduplicator.MinSize = dedupMinSize
	deduplicator.MaxSize = dedupMaxSize

	// Find duplicates
	console.Info(fmt.Sprintf("Scanning for duplicates in: %s", absPath))
	console.Info(fmt.Sprintf("Min size: %d bytes, Max size: %d bytes",
		dedupMinSize, dedupMaxSize))

	groups, err := deduplicator.FindDuplicates(ctx, absPath)
	if err != nil {
		return fmt.Errorf("failed to find duplicates: %w", err)
	}

	if len(groups) == 0 {
		console.Success("No duplicate files found!")
		return nil
	}

	// Display statistics
	stats := dedup.GetStats(groups)
	console.Info(fmt.Sprintf("\nFound %d duplicate groups:", stats.TotalGroups))
	console.Info(fmt.Sprintf("  Total files: %d", stats.TotalFiles))
	console.Info(fmt.Sprintf("  Duplicate files: %d", stats.TotalDuplicates))
	console.Info(fmt.Sprintf("  Wasted space: %d bytes", stats.WastedSpace))
	console.Info(fmt.Sprintf("  Largest duplicate: %d bytes", stats.LargestDuplicate))

	// Display duplicate groups
	fmt.Println("\nDuplicate Groups:")
	fmt.Println("==========================================")

	for i, group := range groups {
		if i >= 10 && !dedupAutoRemove { // Limit display to 10 groups in interactive mode
			console.Info(fmt.Sprintf("\n... and %d more groups", len(groups)-10))
			break
		}

		fmt.Printf("\nGroup %d (Hash: %s, Size: %d bytes):\n",
			i+1, group.Hash[:16]+"...", group.Size)

		for j, file := range group.Files {
			marker := " "
			if j == 0 {
				marker = "✓" // File to keep
			} else {
				marker = "✗" // File to remove
			}

			relPath, _ := filepath.Rel(absPath, file.Path)
			fmt.Printf("  %s %s (modified: %s)\n",
				marker, relPath, file.ModTime.Format("2006-01-02 15:04:05"))
		}
	}

	// Create removal plan
	plan := deduplicator.CreateRemovalPlan(groups, dedupKeepStrategy)

	fmt.Println("\n==========================================")
	console.Warning(fmt.Sprintf("Will remove %d files, saving %d bytes",
		len(plan.ToRemove), plan.SpaceSaved))

	// Confirm or auto-remove
	if !dedupAutoRemove && !dryRun {
		fmt.Print("\nProceed with removal? (y/n): ")
		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			console.Info("Operation cancelled")
			return nil
		}
	}

	// Execute removal
	if dryRun {
		console.Info("\nDry run mode - no files will be removed")
		for _, file := range plan.ToRemove {
			relPath, _ := filepath.Rel(absPath, file.Path)
			fmt.Printf("  Would remove: %s\n", relPath)
		}
	} else {
		console.Info("\nRemoving duplicate files...")
		if err := deduplicator.ExecuteRemovalPlan(ctx, plan, false); err != nil {
			return fmt.Errorf("failed to remove duplicates: %w", err)
		}
		console.Success(fmt.Sprintf("Successfully removed %d duplicate files, saved %d bytes",
			len(plan.ToRemove), plan.SpaceSaved))
	}

	return nil
}
