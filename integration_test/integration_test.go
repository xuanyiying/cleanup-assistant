package integration_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/xuanyiying/cleanup-cli/internal/analyzer"
	"github.com/xuanyiying/cleanup-cli/internal/config"
	"github.com/xuanyiying/cleanup-cli/internal/organizer"
	"github.com/xuanyiying/cleanup-cli/internal/rules"
	"github.com/xuanyiying/cleanup-cli/internal/transaction"
)

// TestCompleteOrganizationWorkflow tests the complete scan-analyze-organize flow
func TestCompleteOrganizationWorkflow(t *testing.T) {
	tmpDir := t.TempDir()
	testDir := filepath.Join(tmpDir, "test_files")
	if err := os.MkdirAll(testDir, 0755); err != nil {
		t.Fatalf("failed to create test directory: %v", err)
	}

	// Create test files
	testFiles := map[string]string{
		"document1.pdf": "PDF content",
		"image1.jpg":    "JPEG content",
		"image2.png":    "PNG content",
		"readme.txt":    "This is a readme file",
		"data.json":     `{"key": "value"}`,
	}

	for filename, content := range testFiles {
		filePath := filepath.Join(testDir, filename)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("failed to create test file %s: %v", filename, err)
		}
	}

	// Initialize components
	txnMgr := transaction.NewManager(filepath.Join(tmpDir, "transactions.json"))
	fileAnalyzer := analyzer.NewAnalyzer()
	ruleEngine := rules.NewEngine()
	fileOrganizer := organizer.NewOrganizerWithDeps(txnMgr, ruleEngine, fileAnalyzer)

	ctx := context.Background()

	// Step 1: Scan directory
	files, err := fileAnalyzer.AnalyzeDirectory(ctx, testDir, &analyzer.ScanOptions{
		Recursive:     true,
		IncludeHidden: false,
	})
	if err != nil {
		t.Fatalf("failed to scan directory: %v", err)
	}

	if len(files) != len(testFiles) {
		t.Errorf("expected %d files, got %d", len(testFiles), len(files))
	}

	// Verify all files were analyzed
	for _, file := range files {
		if file.Name == "" {
			t.Error("file name is empty")
		}
		if file.Extension == "" {
			t.Error("file extension is empty")
		}
		if file.Size == 0 {
			t.Error("file size is zero")
		}
		if file.MimeType == "" {
			t.Error("file mime type is empty")
		}
	}

	// Step 2: Set up rules
	testRules := []*config.Rule{
		{
			Name:     "pdf-files",
			Priority: 10,
			Condition: &config.RuleCondition{
				Type:     "extension",
				Value:    "pdf",
				Operator: "match",
			},
			Action: &config.RuleAction{
				Type:   "move",
				Target: "Documents",
			},
		},
		{
			Name:     "image-files",
			Priority: 20,
			Condition: &config.RuleCondition{
				Type:     "extension",
				Value:    "jpg,png,gif",
				Operator: "match",
			},
			Action: &config.RuleAction{
				Type:   "move",
				Target: "Pictures",
			},
		},
	}

	if err := ruleEngine.LoadRules(testRules); err != nil {
		t.Fatalf("failed to load rules: %v", err)
	}

	// Step 3: Generate organization plan
	strategy := &organizer.OrganizeStrategy{
		UseAI:            false,
		CreateFolders:    true,
		ConflictStrategy: organizer.ConflictSuffix,
		DryRun:           false,
		MaxConcurrency:   4,
	}

	plan, err := fileOrganizer.Organize(ctx, files, strategy)
	if err != nil {
		t.Fatalf("failed to generate organization plan: %v", err)
	}

	if plan == nil {
		t.Fatal("organization plan is nil")
	}

	// Verify plan has operations
	if len(plan.Operations) == 0 {
		t.Error("no operations in plan")
	}

	// Display plan details
	t.Logf("Organization Plan:")
	for i, op := range plan.Operations {
		t.Logf("  [%d] %s: %s -> %s (reason: %s)", i+1, op.Type, op.Source, op.Target, op.Reason)
	}

	// Step 4: Execute plan
	result, err := fileOrganizer.ExecutePlan(ctx, plan, strategy)
	if err != nil {
		t.Fatalf("failed to execute plan: %v", err)
	}

	if result == nil {
		t.Fatal("execution result is nil")
	}

	// Verify execution results
	if result.Successful == 0 && result.Failed == 0 {
		t.Error("no operations were executed")
	}

	// Verify at least some operations succeeded
	if result.Successful < len(plan.Operations)-2 {
		t.Errorf("expected at least %d successful operations, got %d", len(plan.Operations)-2, result.Successful)
	}

	// Step 5: Test undo functionality
	if len(result.TransactionIDs) > 0 {
		txnID := result.TransactionIDs[0]
		if err := txnMgr.Undo(txnID); err != nil {
			t.Fatalf("failed to undo transaction: %v", err)
		}
	}
}

// TestDryRunMode tests that dry-run mode doesn't modify files
func TestDryRunMode(t *testing.T) {
	tmpDir := t.TempDir()
	testDir := filepath.Join(tmpDir, "test_files")
	if err := os.MkdirAll(testDir, 0755); err != nil {
		t.Fatalf("failed to create test directory: %v", err)
	}

	// Create test files
	testFile := filepath.Join(testDir, "test.pdf")
	if err := os.WriteFile(testFile, []byte("PDF content"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Initialize components
	txnMgr := transaction.NewManager(filepath.Join(tmpDir, "transactions.json"))
	fileAnalyzer := analyzer.NewAnalyzer()
	ruleEngine := rules.NewEngine()
	fileOrganizer := organizer.NewOrganizerWithDeps(txnMgr, ruleEngine, fileAnalyzer)

	ctx := context.Background()

	// Scan directory
	files, err := fileAnalyzer.AnalyzeDirectory(ctx, testDir, &analyzer.ScanOptions{
		Recursive:     true,
		IncludeHidden: false,
	})
	if err != nil {
		t.Fatalf("failed to scan directory: %v", err)
	}

	// Set up rules
	testRules := []*config.Rule{
		{
			Name:     "pdf-files",
			Priority: 10,
			Condition: &config.RuleCondition{
				Type:     "extension",
				Value:    "pdf",
				Operator: "match",
			},
			Action: &config.RuleAction{
				Type:   "move",
				Target: "Documents",
			},
		},
	}

	if err := ruleEngine.LoadRules(testRules); err != nil {
		t.Fatalf("failed to load rules: %v", err)
	}

	// Generate plan
	strategy := &organizer.OrganizeStrategy{
		UseAI:            false,
		CreateFolders:    true,
		ConflictStrategy: organizer.ConflictSuffix,
		DryRun:           true, // Enable dry-run mode
		MaxConcurrency:   4,
	}

	plan, err := fileOrganizer.Organize(ctx, files, strategy)
	if err != nil {
		t.Fatalf("failed to generate organization plan: %v", err)
	}

	// Execute plan in dry-run mode
	result, err := fileOrganizer.ExecutePlan(ctx, plan, strategy)
	if err != nil {
		t.Fatalf("failed to execute plan: %v", err)
	}

	// Verify file was not moved
	if _, err := os.Stat(testFile); err != nil {
		t.Error("file was moved in dry-run mode")
	}

	// Verify Documents directory was not created
	documentsDir := filepath.Join(testDir, "Documents")
	if _, err := os.Stat(documentsDir); err == nil {
		t.Error("Documents directory was created in dry-run mode")
	}

	// Verify result shows successful operations (but no actual changes)
	if result.Successful == 0 {
		t.Error("dry-run should report successful operations")
	}
}

// TestConflictResolution tests conflict handling strategies
func TestConflictResolution(t *testing.T) {
	tmpDir := t.TempDir()
	testDir := filepath.Join(tmpDir, "test_files")
	if err := os.MkdirAll(testDir, 0755); err != nil {
		t.Fatalf("failed to create test directory: %v", err)
	}

	// Create test files
	testFile := filepath.Join(testDir, "document.pdf")
	if err := os.WriteFile(testFile, []byte("PDF content"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Create target directory with existing file
	targetDir := filepath.Join(testDir, "Documents")
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatalf("failed to create target directory: %v", err)
	}

	existingFile := filepath.Join(targetDir, "document.pdf")
	if err := os.WriteFile(existingFile, []byte("existing content"), 0644); err != nil {
		t.Fatalf("failed to create existing file: %v", err)
	}

	// Initialize components
	txnMgr := transaction.NewManager(filepath.Join(tmpDir, "transactions.json"))
	fileAnalyzer := analyzer.NewAnalyzer()
	ruleEngine := rules.NewEngine()
	fileOrganizer := organizer.NewOrganizerWithDeps(txnMgr, ruleEngine, fileAnalyzer)

	ctx := context.Background()

	// Test suffix strategy
	opts := &organizer.MoveOptions{
		DryRun:           false,
		CreateTargetDir:  true,
		ConflictStrategy: organizer.ConflictSuffix,
	}

	result, err := fileOrganizer.Move(ctx, testFile, targetDir, opts)
	if err != nil {
		t.Fatalf("failed to move file: %v", err)
	}

	if !result.Success {
		t.Error("move operation failed")
	}

	// Verify file was moved with suffix
	if _, err := os.Stat(result.Target); err != nil {
		t.Errorf("moved file not found at: %s", result.Target)
	}

	// Verify original file no longer exists
	if _, err := os.Stat(testFile); err == nil {
		t.Error("original file still exists after move")
	}

	// Verify existing file is unchanged
	if _, err := os.Stat(existingFile); err != nil {
		t.Error("existing file was modified")
	}
}

// TestBatchErrorResilience tests that batch operations continue on errors
func TestBatchErrorResilience(t *testing.T) {
	tmpDir := t.TempDir()
	testDir := filepath.Join(tmpDir, "test_files")
	if err := os.MkdirAll(testDir, 0755); err != nil {
		t.Fatalf("failed to create test directory: %v", err)
	}

	// Create test files
	testFiles := []string{"file1.pdf", "file2.pdf", "file3.pdf"}
	for _, filename := range testFiles {
		filePath := filepath.Join(testDir, filename)
		if err := os.WriteFile(filePath, []byte("PDF content"), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}
	}

	// Initialize components
	txnMgr := transaction.NewManager(filepath.Join(tmpDir, "transactions.json"))
	fileAnalyzer := analyzer.NewAnalyzer()
	ruleEngine := rules.NewEngine()
	fileOrganizer := organizer.NewOrganizerWithDeps(txnMgr, ruleEngine, fileAnalyzer)

	ctx := context.Background()

	// Scan directory
	files, err := fileAnalyzer.AnalyzeDirectory(ctx, testDir, &analyzer.ScanOptions{
		Recursive:     true,
		IncludeHidden: false,
	})
	if err != nil {
		t.Fatalf("failed to scan directory: %v", err)
	}

	// Set up rules
	testRules := []*config.Rule{
		{
			Name:     "pdf-files",
			Priority: 10,
			Condition: &config.RuleCondition{
				Type:     "extension",
				Value:    "pdf",
				Operator: "match",
			},
			Action: &config.RuleAction{
				Type:   "move",
				Target: "Documents",
			},
		},
	}

	if err := ruleEngine.LoadRules(testRules); err != nil {
		t.Fatalf("failed to load rules: %v", err)
	}

	// Generate plan
	strategy := &organizer.OrganizeStrategy{
		UseAI:            false,
		CreateFolders:    true,
		ConflictStrategy: organizer.ConflictSuffix,
		DryRun:           false,
		MaxConcurrency:   2, // Use limited concurrency to test resilience
	}

	plan, err := fileOrganizer.Organize(ctx, files, strategy)
	if err != nil {
		t.Fatalf("failed to generate organization plan: %v", err)
	}

	// Execute plan
	result, err := fileOrganizer.ExecutePlan(ctx, plan, strategy)
	if err != nil {
		t.Fatalf("failed to execute plan: %v", err)
	}

	// Verify all files were processed
	totalProcessed := result.Successful + result.Failed + result.Skipped
	if totalProcessed != len(testFiles) {
		t.Errorf("expected %d files processed, got %d", len(testFiles), totalProcessed)
	}

	// Verify at least some files were successfully moved
	if result.Successful == 0 {
		t.Error("no files were successfully moved")
	}
}

// TestTransactionPersistence tests that transactions are persisted and can be recovered
func TestTransactionPersistence(t *testing.T) {
	tmpDir := t.TempDir()
	testDir := filepath.Join(tmpDir, "test_files")
	logPath := filepath.Join(tmpDir, "transactions.json")

	if err := os.MkdirAll(testDir, 0755); err != nil {
		t.Fatalf("failed to create test directory: %v", err)
	}

	// Create test file
	testFile := filepath.Join(testDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Create transaction manager and perform operation
	txnMgr := transaction.NewManager(logPath)
	tx := txnMgr.Begin()

	op := &transaction.ExecutedOperation{
		Type:   transaction.OpRename,
		Source: testFile,
		Target: filepath.Join(testDir, "renamed.txt"),
		Backup: "",
	}
	txnMgr.AddOperation(tx, op)

	if err := txnMgr.Commit(tx); err != nil {
		t.Fatalf("failed to commit transaction: %v", err)
	}

	// Create new transaction manager instance (simulating restart)
	txnMgr2 := transaction.NewManager(logPath)

	// Retrieve transaction history
	history, err := txnMgr2.GetHistory(10)
	if err != nil {
		t.Fatalf("failed to get transaction history: %v", err)
	}

	if len(history) == 0 {
		t.Error("transaction history is empty")
	}

	// Verify transaction was persisted
	found := false
	for _, h := range history {
		if h.ID == tx.ID {
			found = true
			if h.Status != transaction.StatusCommitted {
				t.Errorf("expected status %s, got %s", transaction.StatusCommitted, h.Status)
			}
			if len(h.Operations) != 1 {
				t.Errorf("expected 1 operation, got %d", len(h.Operations))
			}
			break
		}
	}

	if !found {
		t.Error("transaction not found in history")
	}
}

// TestFileMetadataExtraction tests that file metadata is correctly extracted
func TestFileMetadataExtraction(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files with different types
	testCases := []struct {
		filename string
		content  []byte
		expected string
	}{
		{"test.txt", []byte("text content"), "text/plain"},
		{"test.json", []byte(`{"key": "value"}`), "text/plain"},
		{"test.pdf", []byte("%PDF-1.4"), "application/pdf"},
	}

	for _, tc := range testCases {
		filePath := filepath.Join(tmpDir, tc.filename)
		if err := os.WriteFile(filePath, tc.content, 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		fileAnalyzer := analyzer.NewAnalyzer()
		ctx := context.Background()

		metadata, err := fileAnalyzer.Analyze(ctx, filePath)
		if err != nil {
			t.Fatalf("failed to analyze file: %v", err)
		}

		// Verify metadata
		if metadata.Name != tc.filename {
			t.Errorf("expected name %s, got %s", tc.filename, metadata.Name)
		}

		if metadata.Size == 0 {
			t.Error("file size is zero")
		}

		if metadata.ModifiedAt.IsZero() {
			t.Error("modified time is zero")
		}

		if metadata.Extension == "" {
			t.Error("extension is empty")
		}
	}
}

// TestRuleMatching tests that rules are correctly matched and applied
func TestRuleMatching(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.pdf")
	if err := os.WriteFile(testFile, []byte("PDF content"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	fileAnalyzer := analyzer.NewAnalyzer()
	ctx := context.Background()

	metadata, err := fileAnalyzer.Analyze(ctx, testFile)
	if err != nil {
		t.Fatalf("failed to analyze file: %v", err)
	}

	// Create rule engine with test rules
	ruleEngine := rules.NewEngine()
	testRules := []*config.Rule{
		{
			Name:     "pdf-files",
			Priority: 10,
			Condition: &config.RuleCondition{
				Type:     "extension",
				Value:    "pdf",
				Operator: "match",
			},
			Action: &config.RuleAction{
				Type:   "move",
				Target: "Documents",
			},
		},
	}

	if err := ruleEngine.LoadRules(testRules); err != nil {
		t.Fatalf("failed to load rules: %v", err)
	}

	// Test rule matching
	matchedRules := ruleEngine.Match(metadata)
	if len(matchedRules) == 0 {
		t.Error("no rules matched")
	}

	// Test rule application
	actions := ruleEngine.Apply(metadata, matchedRules)
	if len(actions) == 0 {
		t.Error("no actions returned")
	}

	if actions[0].Target != "Documents" {
		t.Errorf("expected target Documents, got %s", actions[0].Target)
	}
}

// TestTemplateExpansion tests that path templates are correctly expanded
func TestTemplateExpansion(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.pdf")
	if err := os.WriteFile(testFile, []byte("PDF content"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	fileAnalyzer := analyzer.NewAnalyzer()
	ctx := context.Background()

	metadata, err := fileAnalyzer.Analyze(ctx, testFile)
	if err != nil {
		t.Fatalf("failed to analyze file: %v", err)
	}

	// Test template expansion through the organization workflow
	txnMgr := transaction.NewManager(filepath.Join(tmpDir, "transactions.json"))
	ruleEngine := rules.NewEngine()
	fileOrganizer := organizer.NewOrganizerWithDeps(txnMgr, ruleEngine, fileAnalyzer)

	// Create a rule with template
	testRules := []*config.Rule{
		{
			Name:     "date-organized",
			Priority: 10,
			Condition: &config.RuleCondition{
				Type:     "extension",
				Value:    "pdf",
				Operator: "match",
			},
			Action: &config.RuleAction{
				Type:   "move",
				Target: "{year}/{month}/Documents",
			},
		},
	}

	if err := ruleEngine.LoadRules(testRules); err != nil {
		t.Fatalf("failed to load rules: %v", err)
	}

	// Generate organization plan which will expand templates
	strategy := &organizer.OrganizeStrategy{
		UseAI:            false,
		CreateFolders:    true,
		ConflictStrategy: organizer.ConflictSuffix,
		DryRun:           true,
		MaxConcurrency:   4,
	}

	plan, err := fileOrganizer.Organize(ctx, []*analyzer.FileMetadata{metadata}, strategy)
	if err != nil {
		t.Fatalf("failed to generate organization plan: %v", err)
	}

	if len(plan.Operations) == 0 {
		t.Error("no operations in plan")
		return
	}

	// Verify template was expanded in the plan
	targetPath := plan.Operations[0].Target
	if targetPath == "" {
		t.Error("target path is empty")
	}

	// Verify no placeholders remain
	if contains(targetPath, "{") || contains(targetPath, "}") {
		t.Errorf("unexpanded placeholders in path: %s", targetPath)
	}

	// Verify year and month are present
	now := time.Now()
	expectedYear := now.Format("2006")
	expectedMonth := now.Format("01")

	if !contains(targetPath, expectedYear) {
		t.Errorf("expected year %s in path: %s", expectedYear, targetPath)
	}

	if !contains(targetPath, expectedMonth) {
		t.Errorf("expected month %s in path: %s", expectedMonth, targetPath)
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
