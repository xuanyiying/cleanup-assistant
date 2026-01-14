package shell

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/xuanyiying/cleanup-cli/internal/analyzer"
	"github.com/xuanyiying/cleanup-cli/internal/config"
	"github.com/xuanyiying/cleanup-cli/internal/ai"
	"github.com/xuanyiying/cleanup-cli/internal/organizer"
	"github.com/xuanyiying/cleanup-cli/internal/rules"
	"github.com/xuanyiying/cleanup-cli/internal/transaction"
)

// InteractiveShell represents the interactive command-line interface
type InteractiveShell struct {
	configMgr    *config.Manager
	txnMgr       *transaction.Manager
	analyzer     analyzer.Analyzer
	ruleEngine   rules.Engine
	organizer    *organizer.Organizer
	ollamaClient ai.Client
	targetDir    string
	program      *tea.Program
}

// NewInteractiveShell creates a new interactive shell
func NewInteractiveShell(
	configMgr *config.Manager,
	txnMgr *transaction.Manager,
	analyzer analyzer.Analyzer,
	ruleEngine rules.Engine,
	organizer *organizer.Organizer,
	ollamaClient ai.Client,
) *InteractiveShell {
	return &InteractiveShell{
		configMgr:    configMgr,
		txnMgr:       txnMgr,
		analyzer:     analyzer,
		ruleEngine:   ruleEngine,
		organizer:    organizer,
		ollamaClient: ollamaClient,
		targetDir:    ".",
	}
}

// Start begins the interactive shell session
func (s *InteractiveShell) Start() error {
	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}
	s.targetDir = cwd

	// Display welcome message
	fmt.Println("╔════════════════════════════════════════╗")
	fmt.Println("║     Cleanup CLI - Interactive Mode     ║")
	fmt.Println("╚════════════════════════════════════════╝")
	fmt.Printf("\nTarget directory: %s\n", s.targetDir)
	fmt.Println("\nAvailable commands:")
	fmt.Println("  scan [path]        - Scan and analyze files")
	fmt.Println("  organize [path]    - Organize files based on rules")
	fmt.Println("  undo [txn-id]      - Undo a previous operation")
	fmt.Println("  history [limit]    - Show transaction history")
	fmt.Println("  help               - Show this help message")
	fmt.Println("  exit, quit         - Exit the interactive shell")
	fmt.Println("\nOr describe what you want to do in natural language.")
	fmt.Println("Type 'help' for more information.")

	// Start interactive loop
	return s.interactiveLoop()
}

// interactiveLoop handles the main interactive command loop
func (s *InteractiveShell) interactiveLoop() error {
	reader := bufio.NewReader(os.Stdin)

	for {
		// Display prompt
		fmt.Print("cleanup> ")

		// Read user input
		input, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read input: %w", err)
		}

		// Trim whitespace
		input = strings.TrimSpace(input)

		// Skip empty input
		if input == "" {
			continue
		}

		// Handle exit commands
		if input == "exit" || input == "quit" {
			fmt.Println("Goodbye!")
			return nil
		}

		// Handle help command
		if input == "help" {
			s.showHelp()
			continue
		}

		// Parse and execute command
		if err := s.executeCommand(input); err != nil {
			fmt.Printf("Error: %v\n", err)
		}
	}
}

// executeCommand parses and executes a user command
func (s *InteractiveShell) executeCommand(input string) error {
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return nil
	}

	command := parts[0]
	args := parts[1:]

	switch command {
	case "scan":
		return s.handleScan(args)
	case "organize":
		return s.handleOrganize(args)
	case "undo":
		return s.handleUndo(args)
	case "history":
		return s.handleHistory(args)
	default:
		// Try to parse as natural language command
		return s.handleNaturalLanguage(input)
	}
}

// handleScan handles the scan command
func (s *InteractiveShell) handleScan(args []string) error {
	path := s.targetDir
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

	// Show spinner
	spinner := s.showSpinner("Scanning directory...")
	defer spinner.Stop()

	// Scan directory
	files, err := s.analyzer.AnalyzeDirectory(ctx, absPath, &analyzer.ScanOptions{
		Recursive:     true,
		IncludeHidden: false,
	})
	if err != nil {
		spinner.Fail(fmt.Sprintf("Failed to scan: %v", err))
		return err
	}

	spinner.Succeed(fmt.Sprintf("Found %d files", len(files)))

	// Display file summary
	fmt.Println("\nFile Summary:")
	fmt.Printf("  Total files: %d\n", len(files))

	// Group by type
	typeCount := make(map[string]int)
	totalSize := int64(0)
	for _, file := range files {
		typeCount[file.MimeType]++
		totalSize += file.Size
	}

	fmt.Printf("  Total size: %.2f MB\n", float64(totalSize)/1024/1024)
	fmt.Println("\n  By type:")
	for mimeType, count := range typeCount {
		fmt.Printf("    - %s: %d files\n", mimeType, count)
	}

	return nil
}

// handleOrganize handles the organize command
func (s *InteractiveShell) handleOrganize(args []string) error {
	path := s.targetDir
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

	// Show spinner for scanning
	spinner := s.showSpinner("Scanning directory...")
	files, err := s.analyzer.AnalyzeDirectory(ctx, absPath, &analyzer.ScanOptions{
		Recursive:     true,
		IncludeHidden: false,
	})
	if err != nil {
		spinner.Fail(fmt.Sprintf("Failed to scan: %v", err))
		return err
	}
	spinner.Succeed(fmt.Sprintf("Found %d files", len(files)))

	// Generate organization plan
	spinner = s.showSpinner("Generating organization plan...")
	strategy := &organizer.OrganizeStrategy{
		UseAI:            true,
		CreateFolders:    true,
		ConflictStrategy: organizer.ConflictSuffix,
		DryRun:           false,
		MaxConcurrency:   4,
	}

	plan, err := s.organizer.Organize(ctx, files, strategy)
	if err != nil {
		spinner.Fail(fmt.Sprintf("Failed to generate plan: %v", err))
		return err
	}
	spinner.Succeed("Organization plan generated")

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
	if len(plan.Operations) > 0 {
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

	// Ask for confirmation
	fmt.Print("\nProceed with organization? (yes/no): ")
	reader := bufio.NewReader(os.Stdin)
	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(strings.ToLower(response))

	if response != "yes" && response != "y" {
		fmt.Println("Organization cancelled")
		return nil
	}

	// Execute plan
	spinner = s.showSpinner("Executing organization plan...")
	result, err := s.organizer.ExecutePlan(ctx, plan, strategy)
	if err != nil {
		spinner.Fail(fmt.Sprintf("Failed to execute plan: %v", err))
		return err
	}
	spinner.Succeed("Organization completed")

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
		fmt.Println("\n  💡 Tip: Use 'undo' command to revert changes if needed")
	}

	return nil
}

// handleUndo handles the undo command
func (s *InteractiveShell) handleUndo(args []string) error {
	var txnID string

	if len(args) > 0 {
		txnID = args[0]
	} else {
		// Get last transaction
		history, err := s.txnMgr.GetHistory(1)
		if err != nil {
			return fmt.Errorf("failed to get transaction history: %w", err)
		}

		if len(history) == 0 {
			return fmt.Errorf("no transactions to undo")
		}

		txnID = history[0].ID
	}

	spinner := s.showSpinner(fmt.Sprintf("Undoing transaction %s...", txnID))
	if err := s.txnMgr.Undo(txnID); err != nil {
		spinner.Fail(fmt.Sprintf("Failed to undo: %v", err))
		return err
	}

	spinner.Succeed("Transaction undone successfully")
	return nil
}

// handleHistory handles the history command
func (s *InteractiveShell) handleHistory(args []string) error {
	limit := 10
	if len(args) > 0 {
		fmt.Sscanf(args[0], "%d", &limit)
	}

	history, err := s.txnMgr.GetHistory(limit)
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
}

// handleNaturalLanguage attempts to parse natural language commands
func (s *InteractiveShell) handleNaturalLanguage(input string) error {
	ctx := context.Background()

	// Show spinner while analyzing intent
	spinner := s.showSpinner("Analyzing your request...")

	// Use Ollama to parse intent
	prompt := fmt.Sprintf(`You are a file organization assistant. Parse the following user request and determine what action to take.
Respond with ONLY one of these commands:
- scan [path]
- organize [path]
- undo [txn-id]
- history [limit]
- help

User request: %s

Respond with the command only, no explanation.`, input)

	result, err := s.ollamaClient.Analyze(ctx, prompt, "")
	if err != nil {
		spinner.Fail(fmt.Sprintf("Failed to analyze request: %v", err))
		return err
	}

	if !result.Success {
		spinner.Fail("Failed to analyze request")
		return fmt.Errorf("analysis did not complete successfully")
	}

	spinner.Succeed("Request analyzed")

	// Parse the response
	command := strings.TrimSpace(result.Content)
	command = strings.ToLower(command)

	// Execute the parsed command
	fmt.Printf("Executing: %s\n", command)
	return s.executeCommand(command)
}

// showHelp displays help information
func (s *InteractiveShell) showHelp() {
	fmt.Println("\nAvailable Commands:")
	fmt.Println("  scan [path]        - Scan and analyze files in a directory")
	fmt.Println("  organize [path]    - Organize files based on configured rules")
	fmt.Println("  undo [txn-id]      - Undo a previous file operation")
	fmt.Println("  history [limit]    - Show recent transaction history")
	fmt.Println("  help               - Show this help message")
	fmt.Println("  exit, quit         - Exit the interactive shell")
	fmt.Println("\nExamples:")
	fmt.Println("  cleanup> scan")
	fmt.Println("  cleanup> organize ~/Downloads")
	fmt.Println("  cleanup> undo")
	fmt.Println("  cleanup> organize my files")
	fmt.Println()
}

// showSpinner displays a loading spinner
func (s *InteractiveShell) showSpinner(message string) *Spinner {
	return NewSpinner(message)
}

// Spinner represents a loading spinner
type Spinner struct {
	message string
	done    chan bool
	ticker  *time.Ticker
	frame   int
}

// NewSpinner creates a new spinner
func NewSpinner(message string) *Spinner {
	spinner := &Spinner{
		message: message,
		done:    make(chan bool),
		ticker:  time.NewTicker(100 * time.Millisecond),
		frame:   0,
	}

	// Start spinner animation
	go func() {
		frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		for {
			select {
			case <-spinner.done:
				return
			case <-spinner.ticker.C:
				fmt.Printf("\r%s %s", frames[spinner.frame%len(frames)], spinner.message)
				spinner.frame++
			}
		}
	}()

	return spinner
}

// Succeed marks the spinner as successful
func (s *Spinner) Succeed(message string) {
	s.Stop()
	fmt.Printf("\r✓ %s\n", message)
}

// Fail marks the spinner as failed
func (s *Spinner) Fail(message string) {
	s.Stop()
	fmt.Printf("\r✗ %s\n", message)
}

// Stop stops the spinner
func (s *Spinner) Stop() {
	s.ticker.Stop()
	s.done <- true
	fmt.Print("\r")
}

// Update updates the spinner message
func (s *Spinner) Update(message string) {
	s.message = message
}
