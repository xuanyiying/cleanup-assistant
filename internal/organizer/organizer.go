package organizer

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/xuanyiying/cleanup-cli/internal/ai"
	"github.com/xuanyiying/cleanup-cli/internal/analyzer"
	"github.com/xuanyiying/cleanup-cli/internal/config"
	"github.com/xuanyiying/cleanup-cli/internal/rules"
	"github.com/xuanyiying/cleanup-cli/internal/transaction"
	"github.com/xuanyiying/cleanup-cli/pkg/template"
)

// ConflictStrategy defines how to handle file name conflicts
type ConflictStrategy string

const (
	ConflictSkip      ConflictStrategy = "skip"
	ConflictSuffix    ConflictStrategy = "suffix"
	ConflictOverwrite ConflictStrategy = "overwrite"
	ConflictPrompt    ConflictStrategy = "prompt"
)

// OperationType represents the type of file operation
type OperationType string

const (
	OpMove   OperationType = "move"
	OpRename OperationType = "rename"
	OpDelete OperationType = "delete"
	OpMkdir  OperationType = "mkdir"
)

// RenameOptions specifies options for rename operations
type RenameOptions struct {
	DryRun            bool
	PreserveExtension bool
	ConflictStrategy  ConflictStrategy
}

// MoveOptions specifies options for move operations
type MoveOptions struct {
	DryRun           bool
	CreateTargetDir  bool
	ConflictStrategy ConflictStrategy
}

// OperationResult represents the result of a file operation
type OperationResult struct {
	Success       bool
	Source        string
	Target        string
	Error         error
	TransactionID string
}

// PlannedOperation represents a single operation in an organization plan
type PlannedOperation struct {
	Type   OperationType
	Source string
	Target string
	Reason string
}

// OrganizePlan represents a plan for organizing multiple files
type OrganizePlan struct {
	Operations []*PlannedOperation
	Summary    *PlanSummary
}

// PlanSummary provides statistics about an organization plan
type PlanSummary struct {
	TotalFiles      int
	TotalOperations int
	MoveCount       int
	RenameCount     int
	SkipCount       int
	EstimatedSize   int64
}

// BatchResult represents the result of batch processing
type BatchResult struct {
	Successful     int
	Failed         int
	Skipped        int
	Errors         []error
	TransactionIDs []string
	FailedFiles    map[string]error
}

// OrganizeStrategy represents the strategy for organizing files
type OrganizeStrategy struct {
	UseAI            bool
	CreateFolders    bool
	ConflictStrategy ConflictStrategy
	DryRun           bool
	MaxConcurrency   int
}

// Organizer handles file organization operations
type Organizer struct {
	txnManager   *transaction.Manager
	ruleEngine   rules.Engine
	analyzer     analyzer.Analyzer
	templateExp  *template.Expander
	ollamaClient interface {
		SuggestName(ctx context.Context, file *analyzer.FileMetadata) ([]string, error)
		SuggestCategory(ctx context.Context, file *analyzer.FileMetadata) ([]string, error)
	}
}

// NewOrganizer creates a new file organizer
func NewOrganizer(txnManager *transaction.Manager) *Organizer {
	return &Organizer{
		txnManager:  txnManager,
		ruleEngine:  rules.NewEngine(),
		analyzer:    analyzer.NewAnalyzer(),
		templateExp: template.NewExpander(make(map[string]string)),
	}
}

// NewOrganizerWithDeps creates a new file organizer with custom dependencies
func NewOrganizerWithDeps(txnManager *transaction.Manager, ruleEngine rules.Engine, analyzer analyzer.Analyzer) *Organizer {
	return &Organizer{
		txnManager:   txnManager,
		ruleEngine:   ruleEngine,
		analyzer:     analyzer,
		templateExp:  template.NewExpander(make(map[string]string)),
		ollamaClient: nil,
	}
}

// SetOllamaClient sets the Ollama client for AI-powered features
func (o *Organizer) SetOllamaClient(client ai.Client) {
	o.ollamaClient = client
}

// Rename renames a file with conflict resolution
func (o *Organizer) Rename(ctx context.Context, source, newName string, opts *RenameOptions) (*OperationResult, error) {
	if opts == nil {
		opts = &RenameOptions{
			DryRun:            false,
			PreserveExtension: true,
			ConflictStrategy:  ConflictSkip,
		}
	}

	// Validate source file exists
	_, err := os.Stat(source)
	if err != nil {
		return &OperationResult{
			Success: false,
			Source:  source,
			Error:   fmt.Errorf("source file not found: %w", err),
		}, nil
	}

	// Get directory and construct target path
	sourceDir := filepath.Dir(source)
	var targetName string

	if opts.PreserveExtension {
		// Extract extension from original file
		ext := filepath.Ext(source)
		// Remove extension from newName if it already has one
		baseName := strings.TrimSuffix(newName, filepath.Ext(newName))
		targetName = baseName + ext
	} else {
		targetName = newName
	}

	targetPath := filepath.Join(sourceDir, targetName)

	// Handle conflicts
	finalTarget, err := o.resolveConflict(targetPath, opts.ConflictStrategy)
	if err != nil {
		return &OperationResult{
			Success: false,
			Source:  source,
			Target:  targetPath,
			Error:   err,
		}, nil
	}

	// If conflict strategy is skip and file exists, return success without doing anything
	if finalTarget == "" {
		return &OperationResult{
			Success: true,
			Source:  source,
			Target:  targetPath,
		}, nil
	}

	// Dry-run mode: don't actually rename
	if opts.DryRun {
		return &OperationResult{
			Success: true,
			Source:  source,
			Target:  finalTarget,
		}, nil
	}

	// Start transaction
	tx := o.txnManager.Begin()

	// Create backup of source if it exists at target
	var backupPath string
	if _, err := os.Stat(finalTarget); err == nil {
		backupPath = finalTarget + ".backup"
		if err := os.Rename(finalTarget, backupPath); err != nil {
			return &OperationResult{
				Success: false,
				Source:  source,
				Target:  finalTarget,
				Error:   fmt.Errorf("failed to create backup: %w", err),
			}, nil
		}
	}

	// Perform rename
	if err := os.Rename(source, finalTarget); err != nil {
		// Restore backup if it was created
		if backupPath != "" {
			os.Rename(backupPath, finalTarget)
		}
		return &OperationResult{
			Success: false,
			Source:  source,
			Target:  finalTarget,
			Error:   fmt.Errorf("failed to rename file: %w", err),
		}, nil
	}

	// Record operation in transaction
	op := &transaction.ExecutedOperation{
		Type:   transaction.OpRename,
		Source: source,
		Target: finalTarget,
		Backup: backupPath,
	}
	o.txnManager.AddOperation(tx, op)

	// Commit transaction
	if err := o.txnManager.Commit(tx); err != nil {
		// Try to rollback the rename
		os.Rename(finalTarget, source)
		if backupPath != "" {
			os.Rename(backupPath, finalTarget)
		}
		return &OperationResult{
			Success: false,
			Source:  source,
			Target:  finalTarget,
			Error:   fmt.Errorf("failed to commit transaction: %w", err),
		}, nil
	}

	// Clean up backup if operation succeeded
	if backupPath != "" {
		os.Remove(backupPath)
	}

	return &OperationResult{
		Success:       true,
		Source:        source,
		Target:        finalTarget,
		TransactionID: tx.ID,
	}, nil
}

// Move moves a file to a target directory with conflict resolution
func (o *Organizer) Move(ctx context.Context, source, targetDir string, opts *MoveOptions) (*OperationResult, error) {
	if opts == nil {
		opts = &MoveOptions{
			DryRun:           false,
			CreateTargetDir:  true,
			ConflictStrategy: ConflictSkip,
		}
	}

	// Validate source file exists
	_, err := os.Stat(source)
	if err != nil {
		return &OperationResult{
			Success: false,
			Source:  source,
			Error:   fmt.Errorf("source file not found: %w", err),
		}, nil
	}

	// Create target directory if needed
	if opts.CreateTargetDir {
		if err := os.MkdirAll(targetDir, 0755); err != nil {
			return &OperationResult{
				Success: false,
				Source:  source,
				Target:  targetDir,
				Error:   fmt.Errorf("failed to create target directory: %w", err),
			}, nil
		}
	} else {
		// Verify target directory exists
		if _, err := os.Stat(targetDir); err != nil {
			return &OperationResult{
				Success: false,
				Source:  source,
				Target:  targetDir,
				Error:   fmt.Errorf("target directory does not exist: %w", err),
			}, nil
		}
	}

	// Construct target path
	fileName := filepath.Base(source)
	targetPath := filepath.Join(targetDir, fileName)

	// Handle conflicts
	finalTarget, err := o.resolveConflict(targetPath, opts.ConflictStrategy)
	if err != nil {
		return &OperationResult{
			Success: false,
			Source:  source,
			Target:  targetPath,
			Error:   err,
		}, nil
	}

	// If conflict strategy is skip and file exists, return success without doing anything
	if finalTarget == "" {
		return &OperationResult{
			Success: true,
			Source:  source,
			Target:  targetPath,
		}, nil
	}

	// Dry-run mode: don't actually move
	if opts.DryRun {
		return &OperationResult{
			Success: true,
			Source:  source,
			Target:  finalTarget,
		}, nil
	}

	// Start transaction
	tx := o.txnManager.Begin()

	// Create backup of target if it exists
	var backupPath string
	if _, err := os.Stat(finalTarget); err == nil {
		backupPath = finalTarget + ".backup"
		if err := os.Rename(finalTarget, backupPath); err != nil {
			return &OperationResult{
				Success: false,
				Source:  source,
				Target:  finalTarget,
				Error:   fmt.Errorf("failed to create backup: %w", err),
			}, nil
		}
	}

	// Perform move
	if err := os.Rename(source, finalTarget); err != nil {
		// Restore backup if it was created
		if backupPath != "" {
			os.Rename(backupPath, finalTarget)
		}
		return &OperationResult{
			Success: false,
			Source:  source,
			Target:  finalTarget,
			Error:   fmt.Errorf("failed to move file: %w", err),
		}, nil
	}

	// Record operation in transaction
	op := &transaction.ExecutedOperation{
		Type:   transaction.OpMove,
		Source: source,
		Target: finalTarget,
		Backup: backupPath,
	}
	o.txnManager.AddOperation(tx, op)

	// Commit transaction
	if err := o.txnManager.Commit(tx); err != nil {
		// Try to rollback the move
		os.Rename(finalTarget, source)
		if backupPath != "" {
			os.Rename(backupPath, finalTarget)
		}
		return &OperationResult{
			Success: false,
			Source:  source,
			Target:  finalTarget,
			Error:   fmt.Errorf("failed to commit transaction: %w", err),
		}, nil
	}

	// Clean up backup if operation succeeded
	if backupPath != "" {
		os.Remove(backupPath)
	}

	return &OperationResult{
		Success:       true,
		Source:        source,
		Target:        finalTarget,
		TransactionID: tx.ID,
	}, nil
}

// resolveConflict handles file name conflicts based on strategy
// Returns the final target path, or empty string if skipped, or error if failed
func (o *Organizer) resolveConflict(targetPath string, strategy ConflictStrategy) (string, error) {
	// Check if target exists
	if _, err := os.Stat(targetPath); err != nil {
		// File doesn't exist, no conflict
		return targetPath, nil
	}

	// File exists, handle based on strategy
	switch strategy {
	case ConflictSkip:
		// Return empty string to indicate skip
		return "", nil

	case ConflictSuffix:
		// Append a unique suffix
		return o.generateUniquePath(targetPath), nil

	case ConflictOverwrite:
		// Return the same path (will overwrite)
		return targetPath, nil

	case ConflictPrompt:
		// For now, treat as suffix (prompt would be handled at CLI level)
		return o.generateUniquePath(targetPath), nil

	default:
		return "", fmt.Errorf("unknown conflict strategy: %s", strategy)
	}
}

// generateUniquePath generates a unique file path by appending a suffix
func (o *Organizer) generateUniquePath(targetPath string) string {
	dir := filepath.Dir(targetPath)
	fileName := filepath.Base(targetPath)
	ext := filepath.Ext(fileName)
	baseName := strings.TrimSuffix(fileName, ext)

	// Try appending numbers until we find a unique name
	for i := 1; i <= 1000; i++ {
		newName := fmt.Sprintf("%s_%d%s", baseName, i, ext)
		newPath := filepath.Join(dir, newName)
		if _, err := os.Stat(newPath); err != nil {
			// File doesn't exist, this is our unique path
			return newPath
		}
	}

	// Fallback: use timestamp
	return filepath.Join(dir, fmt.Sprintf("%s_%d%s", baseName, os.Getpid(), ext))
}

// Delete moves a file to trash instead of permanently deleting it
func (o *Organizer) Delete(ctx context.Context, source, trashDir string) (*OperationResult, error) {
	// Ensure trash directory exists
	if err := os.MkdirAll(trashDir, 0755); err != nil {
		return &OperationResult{
			Success: false,
			Source:  source,
			Error:   fmt.Errorf("failed to create trash directory: %w", err),
		}, nil
	}

	// Move to trash
	fileName := filepath.Base(source)
	trashPath := filepath.Join(trashDir, fileName)

	// Handle conflicts in trash
	finalTrashPath, err := o.resolveConflict(trashPath, ConflictSuffix)
	if err != nil {
		return &OperationResult{
			Success: false,
			Source:  source,
			Error:   err,
		}, nil
	}

	// Start transaction
	tx := o.txnManager.Begin()

	// Perform move to trash
	if err := os.Rename(source, finalTrashPath); err != nil {
		return &OperationResult{
			Success: false,
			Source:  source,
			Target:  finalTrashPath,
			Error:   fmt.Errorf("failed to move to trash: %w", err),
		}, nil
	}

	// Record operation in transaction
	op := &transaction.ExecutedOperation{
		Type:   transaction.OpDelete,
		Source: source,
		Target: finalTrashPath,
		Backup: finalTrashPath,
	}
	o.txnManager.AddOperation(tx, op)

	// Commit transaction
	if err := o.txnManager.Commit(tx); err != nil {
		// Try to rollback
		os.Rename(finalTrashPath, source)
		return &OperationResult{
			Success: false,
			Source:  source,
			Target:  finalTrashPath,
			Error:   fmt.Errorf("failed to commit transaction: %w", err),
		}, nil
	}

	return &OperationResult{
		Success:       true,
		Source:        source,
		Target:        finalTrashPath,
		TransactionID: tx.ID,
	}, nil
}

// Ensure Organizer implements the interface
var _ Organizer

// Organize generates an execution plan for organizing files based on rules
func (o *Organizer) Organize(ctx context.Context, files []*analyzer.FileMetadata, strategy *OrganizeStrategy) (*OrganizePlan, error) {
	if strategy == nil {
		strategy = &OrganizeStrategy{
			UseAI:            true,
			CreateFolders:    true,
			ConflictStrategy: ConflictSuffix,
			DryRun:           false,
			MaxConcurrency:   4,
		}
	}

	plan := &OrganizePlan{
		Operations: make([]*PlannedOperation, 0),
		Summary: &PlanSummary{
			TotalFiles:      len(files),
			TotalOperations: 0,
			MoveCount:       0,
			RenameCount:     0,
			SkipCount:       0,
			EstimatedSize:   0,
		},
	}

	for _, file := range files {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// Step 0: Analyze document scenario if needed
		if strategy.UseAI && file.NeedsScenarioAnalysis && o.ollamaClient != nil {
			fmt.Printf("  📊 Analyzing document scenario: %s\n", file.Name)

			categories, err := o.ollamaClient.SuggestCategory(ctx, file)
			if err == nil && len(categories) > 0 && categories[0] != "" {
				file.ScenarioCategory = categories[0]
				fmt.Printf("     → Category: %s\n", file.ScenarioCategory)
			} else if err != nil {
				fmt.Printf("     ⚠ Failed to analyze scenario: %v\n", err)
			}
		}

		// Step 1: Check if file needs smarter name
		var renamedPath string
		if strategy.UseAI && file.NeedsSmarterName && o.ollamaClient != nil {
			fmt.Printf("  🤖 Analyzing: %s (filename quality: %s)\n", file.Name, file.FileNameQuality)

			// Use AI to suggest a better name
			suggestions, err := o.ollamaClient.SuggestName(ctx, file)
			if err == nil && len(suggestions) > 0 && suggestions[0] != "" {
				file.SuggestedName = suggestions[0]

				fmt.Printf("     → Suggested name: %s.%s\n", file.SuggestedName, file.Extension)

				// Add rename operation
				newName := file.SuggestedName + "." + file.Extension
				renamedPath = filepath.Join(filepath.Dir(file.Path), newName)
				op := &PlannedOperation{
					Type:   OpRename,
					Source: file.Path,
					Target: renamedPath,
					Reason: "AI-suggested meaningful name",
				}
				plan.Operations = append(plan.Operations, op)
				plan.Summary.RenameCount++
				plan.Summary.TotalOperations++

				// Update file metadata for subsequent operations
				file.Name = newName
				file.Path = renamedPath
			} else if err != nil {
				fmt.Printf("     ⚠ Failed to generate name: %v\n", err)
			}
		}

		// Step 2: Match rules for this file
		matchedRules := o.ruleEngine.Match(file)
		if len(matchedRules) == 0 {
			// No rules matched, skip this file
			plan.Summary.SkipCount++
			continue
		}

		// Get action from highest priority rule
		actions := o.ruleEngine.Apply(file, matchedRules)
		if len(actions) == 0 {
			plan.Summary.SkipCount++
			continue
		}

		action := actions[0]

		// Expand template to get target path
		targetPath, err := o.expandActionTemplate(action, file)
		if err != nil {
			// Skip files with template expansion errors
			plan.Summary.SkipCount++
			continue
		}

		// Determine operation type based on action
		var op *PlannedOperation
		if action.Type == "move" {
			// Use the renamed path as source if file was renamed
			sourcePath := file.Path

			// Construct full target path with filename
			targetDir := targetPath
			targetFullPath := filepath.Join(targetDir, file.Name)

			op = &PlannedOperation{
				Type:   OpMove,
				Source: sourcePath,
				Target: targetFullPath,
				Reason: matchedRules[0].Name,
			}
			plan.Summary.MoveCount++
		} else if action.Type == "rename" {
			op = &PlannedOperation{
				Type:   OpRename,
				Source: file.Path,
				Target: filepath.Join(filepath.Dir(file.Path), targetPath),
				Reason: matchedRules[0].Name,
			}
			plan.Summary.RenameCount++
		} else {
			plan.Summary.SkipCount++
			continue
		}

		plan.Operations = append(plan.Operations, op)
		plan.Summary.TotalOperations++
		plan.Summary.EstimatedSize += file.Size
	}

	return plan, nil
}

// ExecutePlan executes an organization plan with error resilience
func (o *Organizer) ExecutePlan(ctx context.Context, plan *OrganizePlan, strategy *OrganizeStrategy) (*BatchResult, error) {
	if plan == nil {
		return nil, fmt.Errorf("plan is nil")
	}

	if strategy == nil {
		strategy = &OrganizeStrategy{
			UseAI:            true,
			CreateFolders:    true,
			ConflictStrategy: ConflictSuffix,
			DryRun:           false,
			MaxConcurrency:   4,
		}
	}

	result := &BatchResult{
		Successful:     0,
		Failed:         0,
		Skipped:        0,
		Errors:         make([]error, 0),
		TransactionIDs: make([]string, 0),
		FailedFiles:    make(map[string]error),
	}

	// If dry-run mode, just return the plan without executing
	if strategy.DryRun {
		result.Successful = len(plan.Operations)
		return result, nil
	}

	// Use semaphore to limit concurrency
	maxConcurrency := strategy.MaxConcurrency
	if maxConcurrency <= 0 {
		maxConcurrency = 4
	}

	semaphore := make(chan struct{}, maxConcurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex

	// Progress tracking
	totalOps := len(plan.Operations)
	completed := 0

	for idx, op := range plan.Operations {
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		default:
		}

		wg.Add(1)
		semaphore <- struct{}{}

		go func(operation *PlannedOperation, opIndex int) {
			defer wg.Done()
			defer func() { <-semaphore }()

			var opResult *OperationResult
			var err error

			switch operation.Type {
			case OpMove:
				opResult, err = o.Move(ctx, operation.Source, filepath.Dir(operation.Target), &MoveOptions{
					DryRun:           false,
					CreateTargetDir:  true,
					ConflictStrategy: strategy.ConflictStrategy,
				})

			case OpRename:
				newName := filepath.Base(operation.Target)
				opResult, err = o.Rename(ctx, operation.Source, newName, &RenameOptions{
					DryRun:            false,
					PreserveExtension: true,
					ConflictStrategy:  strategy.ConflictStrategy,
				})

			default:
				err = fmt.Errorf("unknown operation type: %s", operation.Type)
			}

			mu.Lock()
			defer mu.Unlock()

			completed++

			// Print progress
			if opResult != nil && opResult.Success {
				fmt.Printf("  [%d/%d] ✓ %s: %s\n", completed, totalOps, operation.Type, filepath.Base(operation.Source))
			} else if err != nil {
				fmt.Printf("  [%d/%d] ✗ %s: %s (error: %v)\n", completed, totalOps, operation.Type, filepath.Base(operation.Source), err)
			}

			if err != nil {
				result.Failed++
				result.Errors = append(result.Errors, err)
				result.FailedFiles[operation.Source] = err
			} else if opResult != nil {
				if opResult.Success {
					result.Successful++
					if opResult.TransactionID != "" {
						result.TransactionIDs = append(result.TransactionIDs, opResult.TransactionID)
					}
				} else {
					result.Failed++
					if opResult.Error != nil {
						result.Errors = append(result.Errors, opResult.Error)
						result.FailedFiles[operation.Source] = opResult.Error
					}
				}
			}
		}(op, idx)
	}

	wg.Wait()
	return result, nil
}

// expandActionTemplate expands a rule action template using file metadata
func (o *Organizer) expandActionTemplate(action *config.RuleAction, file *analyzer.FileMetadata) (string, error) {
	if action == nil || action.Target == "" {
		return "", fmt.Errorf("action or target is empty")
	}

	// Create placeholders from file metadata
	placeholders := map[string]string{
		"ext":      file.Extension,
		"category": file.ScenarioCategory,
		"year":     fmt.Sprintf("%04d", file.ModifiedAt.Year()),
		"month":    fmt.Sprintf("%02d", file.ModifiedAt.Month()),
		"day":      fmt.Sprintf("%02d", file.ModifiedAt.Day()),
	}

	// If no scenario category, use "uncategorized"
	if placeholders["category"] == "" {
		placeholders["category"] = "uncategorized"
	}

	expander := template.NewExpander(placeholders)
	return expander.ExpandPath(action.Target)
}
