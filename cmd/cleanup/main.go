package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/user/cleanup-cli/internal/analyzer"
	"github.com/user/cleanup-cli/internal/config"
	"github.com/user/cleanup-cli/internal/ollama"
	"github.com/user/cleanup-cli/internal/organizer"
	"github.com/user/cleanup-cli/internal/rules"
	"github.com/user/cleanup-cli/internal/shell"
	"github.com/user/cleanup-cli/internal/transaction"
)

var (
	// Global flags
	configPath        string
	dryRun            bool
	model             string
	targetDir         string
	excludeExtensions []string
	excludePatterns   []string
	excludeDirs       []string

	// Global managers
	configMgr    *config.Manager
	txnMgr       *transaction.Manager
	fileAnalyzer analyzer.Analyzer
	ruleEngine   rules.Engine
	fileOrganizer *organizer.Organizer
	ollamaClient ollama.Client
)

var (
	// Version is set during build time
	Version = "1.0.0"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "Intelligent file organization CLI tool",
	Long: `Cleanup is a smart file organization command-line tool that uses local Ollama models
to intelligently analyze, categorize, and organize your files.

Version: ` + Version + `

Use 'cleanup --help' to see available commands.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// If no subcommand provided, enter interactive mode
		return interactiveMode()
	},
}

// scanCmd represents the scan command
var scanCmd = &cobra.Command{
	Use:   "scan [path]",
	Short: "Scan a directory and analyze files",
	Long: `Scan a directory recursively and analyze all files.
If no path is provided, the current directory is used.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
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

		// Build scan options with exclusions
		scanOpts := buildScanOptions()

		// Scan directory
		fmt.Printf("Scanning directory: %s\n", absPath)
		if len(scanOpts.ExcludeExtensions) > 0 {
			fmt.Printf("Excluding extensions: %v\n", scanOpts.ExcludeExtensions)
		}
		if len(scanOpts.ExcludePatterns) > 0 {
			fmt.Printf("Excluding patterns: %v\n", scanOpts.ExcludePatterns)
		}
		if len(scanOpts.ExcludeDirs) > 0 {
			fmt.Printf("Excluding directories: %v\n", scanOpts.ExcludeDirs)
		}
		
		files, err := fileAnalyzer.AnalyzeDirectory(ctx, absPath, scanOpts)
		if err != nil {
			return fmt.Errorf("failed to scan directory: %w", err)
		}

		fmt.Printf("Found %d files\n", len(files))
		for _, file := range files {
			fmt.Printf("  - %s (%s, %d bytes)\n", file.Name, file.MimeType, file.Size)
		}

		return nil
	},
}

// organizeCmd represents the organize command
var organizeCmd = &cobra.Command{
	Use:   "organize [path]",
	Short: "Organize files in a directory",
	Long: `Organize files in a directory based on configured rules.
If no path is provided, the current directory is used.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
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

		// Build scan options with exclusions
		scanOpts := buildScanOptions()

		// Scan directory
		fmt.Printf("Scanning directory: %s\n", absPath)
		if len(scanOpts.ExcludeExtensions) > 0 {
			fmt.Printf("Excluding extensions: %v\n", scanOpts.ExcludeExtensions)
		}
		if len(scanOpts.ExcludePatterns) > 0 {
			fmt.Printf("Excluding patterns: %v\n", scanOpts.ExcludePatterns)
		}
		if len(scanOpts.ExcludeDirs) > 0 {
			fmt.Printf("Excluding directories: %v\n", scanOpts.ExcludeDirs)
		}
		
		files, err := fileAnalyzer.AnalyzeDirectory(ctx, absPath, scanOpts)
		if err != nil {
			return fmt.Errorf("failed to scan directory: %w", err)
		}

		fmt.Printf("Found %d files\n", len(files))

		// Generate organization plan
		strategy := &organizer.OrganizeStrategy{
			UseAI:            true,
			CreateFolders:    true,
			ConflictStrategy: organizer.ConflictSuffix,
			DryRun:           dryRun,
			MaxConcurrency:   4,
		}

		plan, err := fileOrganizer.Organize(ctx, files, strategy)
		if err != nil {
			return fmt.Errorf("failed to generate organization plan: %w", err)
		}

		// Display plan summary
		fmt.Println("\n╔════════════════════════════════════════╗")
		fmt.Println("║       Organization Plan Summary        ║")
		fmt.Println("╚════════════════════════════════════════╝")
		fmt.Printf("  Total files:      %d\n", plan.Summary.TotalFiles)
		fmt.Printf("  Total operations: %d\n", plan.Summary.TotalOperations)
		fmt.Printf("  Moves:            %d\n", plan.Summary.MoveCount)
		fmt.Printf("  Renames:          %d\n", plan.Summary.RenameCount)
		fmt.Printf("  Skipped:          %d\n", plan.Summary.SkipCount)
		fmt.Printf("  Estimated size:   %.2f MB\n", float64(plan.Summary.EstimatedSize)/1024/1024)

		// Display detailed operations
		if len(plan.Operations) > 0 && !dryRun {
			fmt.Println("\n╔════════════════════════════════════════╗")
			fmt.Println("║         Planned Operations             ║")
			fmt.Println("╚════════════════════════════════════════╝")
			
			maxDisplay := 20
			displayCount := len(plan.Operations)
			if displayCount > maxDisplay {
				displayCount = maxDisplay
			}

			for i := 0; i < displayCount; i++ {
				op := plan.Operations[i]
				switch op.Type {
				case organizer.OpMove:
					fmt.Printf("  [%d] 📁 MOVE\n", i+1)
					fmt.Printf("      From: %s\n", op.Source)
					fmt.Printf("      To:   %s\n", op.Target)
					fmt.Printf("      Rule: %s\n", op.Reason)
				case organizer.OpRename:
					fmt.Printf("  [%d] ✏️  RENAME\n", i+1)
					fmt.Printf("      From: %s\n", filepath.Base(op.Source))
					fmt.Printf("      To:   %s\n", filepath.Base(op.Target))
					fmt.Printf("      Rule: %s\n", op.Reason)
				}
				fmt.Println()
			}

			if len(plan.Operations) > maxDisplay {
				fmt.Printf("  ... and %d more operations\n\n", len(plan.Operations)-maxDisplay)
			}
		}

		if dryRun {
			fmt.Println("\n[DRY-RUN MODE] No files were actually modified")
			return nil
		}

		// Execute plan
		fmt.Println("\nExecuting plan...")
		result, err := fileOrganizer.ExecutePlan(ctx, plan, strategy)
		if err != nil {
			return fmt.Errorf("failed to execute plan: %w", err)
		}

		// Display results
		fmt.Println("\n╔════════════════════════════════════════╗")
		fmt.Println("║          Execution Results             ║")
		fmt.Println("╚════════════════════════════════════════╝")
		fmt.Printf("  ✓ Successful:  %d\n", result.Successful)
		fmt.Printf("  ✗ Failed:      %d\n", result.Failed)
		fmt.Printf("  ⊘ Skipped:     %d\n", result.Skipped)

		if len(result.TransactionIDs) > 0 {
			fmt.Printf("\n  Transaction IDs (for undo):\n")
			for _, txnID := range result.TransactionIDs {
				fmt.Printf("    - %s\n", txnID)
			}
		}

		if len(result.Errors) > 0 {
			fmt.Println("\n  Errors encountered:")
			for file, err := range result.FailedFiles {
				fmt.Printf("    ✗ %s: %v\n", filepath.Base(file), err)
			}
		}

		if result.Successful > 0 {
			fmt.Println("\n  💡 Tip: Use 'cleanup undo' to revert changes if needed")
		}

		return nil
	},
}

// undoCmd represents the undo command
var undoCmd = &cobra.Command{
	Use:   "undo [transaction-id]",
	Short: "Undo a previous file operation",
	Long: `Undo a previous file operation by transaction ID.
If no transaction ID is provided, the last operation is undone.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var txnID string

		if len(args) > 0 {
			txnID = args[0]
		} else {
			// Get last transaction
			history, err := txnMgr.GetHistory(1)
			if err != nil {
				return fmt.Errorf("failed to get transaction history: %w", err)
			}

			if len(history) == 0 {
				return fmt.Errorf("no transactions to undo")
			}

			txnID = history[0].ID
		}

		fmt.Printf("Undoing transaction: %s\n", txnID)
		if err := txnMgr.Undo(txnID); err != nil {
			return fmt.Errorf("failed to undo transaction: %w", err)
		}

		fmt.Println("Transaction undone successfully")
		return nil
	},
}

// historyCmd represents the history command
var historyCmd = &cobra.Command{
	Use:   "history [limit]",
	Short: "Show transaction history",
	Long:  `Show recent file operation transactions.`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		limit := 10
		if len(args) > 0 {
			fmt.Sscanf(args[0], "%d", &limit)
		}

		history, err := txnMgr.GetHistory(limit)
		if err != nil {
			return fmt.Errorf("failed to get transaction history: %w", err)
		}

		if len(history) == 0 {
			fmt.Println("No transactions found")
			return nil
		}

		fmt.Printf("Recent transactions (showing %d):\n", len(history))
		for _, txn := range history {
			fmt.Printf("  ID: %s\n", txn.ID)
			fmt.Printf("    Time: %s\n", txn.Timestamp.Format("2006-01-02 15:04:05"))
			fmt.Printf("    Status: %s\n", txn.Status)
			fmt.Printf("    Operations: %d\n", len(txn.Operations))
		}

		return nil
	},
}

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Long:  `Display the version of Cleanup CLI.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Cleanup CLI v%s\n", Version)
		fmt.Println("Intelligent file organization tool powered by Ollama")
		fmt.Println("https://github.com/user/cleanup-cli")
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// buildScanOptions builds scan options from command-line flags and config
func buildScanOptions() *analyzer.ScanOptions {
	opts := &analyzer.ScanOptions{
		Recursive:     true,
		IncludeHidden: false,
	}

	// Merge exclusions from command-line flags
	if len(excludeExtensions) > 0 {
		opts.ExcludeExtensions = append(opts.ExcludeExtensions, excludeExtensions...)
	}
	if len(excludePatterns) > 0 {
		opts.ExcludePatterns = append(opts.ExcludePatterns, excludePatterns...)
	}
	if len(excludeDirs) > 0 {
		opts.ExcludeDirs = append(opts.ExcludeDirs, excludeDirs...)
	}

	// Merge exclusions from config file
	cfg, err := configMgr.Load()
	if err == nil && cfg != nil && cfg.Exclude != nil {
		if len(cfg.Exclude.Extensions) > 0 {
			opts.ExcludeExtensions = append(opts.ExcludeExtensions, cfg.Exclude.Extensions...)
		}
		if len(cfg.Exclude.Patterns) > 0 {
			opts.ExcludePatterns = append(opts.ExcludePatterns, cfg.Exclude.Patterns...)
		}
		if len(cfg.Exclude.Dirs) > 0 {
			opts.ExcludeDirs = append(opts.ExcludeDirs, cfg.Exclude.Dirs...)
		}
	}

	return opts
}

func init() {
	// Initialize global managers
	homeDir, _ := os.UserHomeDir()
	defaultConfigPath := filepath.Join(homeDir, ".cleanuprc.yaml")

	configMgr = config.NewManager(defaultConfigPath)
	txnMgr = transaction.NewManager(filepath.Join(homeDir, ".cleanup", "transactions.json"))
	fileAnalyzer = analyzer.NewAnalyzer()
	ruleEngine = rules.NewEngine()
	fileOrganizer = organizer.NewOrganizerWithDeps(txnMgr, ruleEngine, fileAnalyzer)

	// Load configuration
	cfg, err := configMgr.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to load config: %v\n", err)
	}

	// Initialize Ollama client
	if cfg != nil {
		ollamaClient = ollama.NewClient(&ollama.Config{
			BaseURL: cfg.Ollama.BaseURL,
			Model:   cfg.Ollama.Model,
			Timeout: cfg.Ollama.Timeout,
		})
		// Load rules into rule engine
		if len(cfg.Rules) > 0 {
			if err := ruleEngine.LoadRules(cfg.Rules); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to load rules: %v\n", err)
			}
		}
	} else {
		ollamaClient = ollama.NewClient(nil)
	}

	// Set Ollama client in organizer for AI features
	fileOrganizer.SetOllamaClient(ollamaClient)

	// Add persistent flags
	rootCmd.PersistentFlags().StringVar(&configPath, "config", defaultConfigPath, "Path to configuration file")
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "Preview changes without modifying files")
	rootCmd.PersistentFlags().StringVar(&model, "model", "llama3.2", "Ollama model to use for analysis")
	rootCmd.PersistentFlags().StringSliceVar(&excludeExtensions, "exclude-ext", []string{}, "File extensions to exclude (e.g., log,tmp)")
	rootCmd.PersistentFlags().StringSliceVar(&excludePatterns, "exclude-pattern", []string{}, "File name patterns to exclude (e.g., *.bak,temp*)")
	rootCmd.PersistentFlags().StringSliceVar(&excludeDirs, "exclude-dir", []string{}, "Directory names to exclude (e.g., .git,node_modules)")

	// Add subcommands
	rootCmd.AddCommand(scanCmd)
	rootCmd.AddCommand(organizeCmd)
	rootCmd.AddCommand(undoCmd)
	rootCmd.AddCommand(historyCmd)
	rootCmd.AddCommand(versionCmd)
}

// interactiveMode enters the interactive shell
func interactiveMode() error {
	// Verify Ollama is available
	ctx := context.Background()
	if err := ollamaClient.CheckHealth(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Error: Ollama service is not available\n")
		fmt.Fprintf(os.Stderr, "Please ensure Ollama is running at http://localhost:11434\n")
		fmt.Fprintf(os.Stderr, "Visit https://ollama.ai for installation instructions\n")
		return err
	}

	// Create and start interactive shell
	interactiveShell := shell.NewInteractiveShell(configMgr, txnMgr, fileAnalyzer, ruleEngine, fileOrganizer, ollamaClient)
	return interactiveShell.Start()
}

func main() {
	Execute()
}
