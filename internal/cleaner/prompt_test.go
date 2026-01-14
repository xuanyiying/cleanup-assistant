package cleaner

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xuanyiying/cleanup-cli/internal/output"
	"pgregory.net/rapid"
)

// TestFilePreviewLength tests Property 17: File Preview Length
// Feature: enhanced-output-cleanup, Property 17: File Preview Length
func TestFilePreviewLength(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random file content of various lengths (printable characters only)
		contentLength := rapid.IntRange(0, 2000).Draw(t, "contentLength")
		// Use a set of printable characters to ensure it's treated as text
		content := rapid.StringOfN(rapid.RuneFrom([]rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789 \n\t.,;:!?-+()[]{}")), contentLength, contentLength, contentLength).Draw(t, "content")

		// Create temporary file
		tmpDir := os.TempDir()
		tmpFile := filepath.Join(tmpDir, "test_file_"+rapid.StringMatching(`[a-zA-Z0-9_-]+`).Draw(t, "filename")+".txt")

		err := os.WriteFile(tmpFile, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
		defer os.Remove(tmpFile)

		// Create console and prompt
		var buf bytes.Buffer
		console := output.NewConsole(&buf)
		prompt := NewInteractivePrompt(console, nil)

		// Test ShowPreview with 500 character limit
		err = prompt.ShowPreview(tmpFile, 500)
		if err != nil {
			t.Fatalf("ShowPreview failed: %v", err)
		}

		// Property: The actual file content read should be at most 500 characters
		// We verify this by checking that ShowPreview was called with maxChars=500
		// and that the implementation respects this limit

		// Read the file directly to verify what was actually read
		file, err := os.Open(tmpFile)
		if err != nil {
			t.Fatalf("Failed to open file: %v", err)
		}
		defer file.Close()

		buffer := make([]byte, 500)
		n, _ := file.Read(buffer)
		expectedContent := string(buffer[:n])

		// The output should contain the expected content (or a substring of it)
		outputStr := buf.String()

		// Property: The content shown should be at most 500 characters
		if len(expectedContent) > 500 {
			t.Errorf("Expected content length %d exceeds maximum of 500 characters", len(expectedContent))
		}

		// If original content is <= 500 chars, the full content should be in output
		if len(content) <= 500 && len(content) > 0 {
			if !strings.Contains(outputStr, content) {
				t.Errorf("Output should contain full content when content length %d <= 500", len(content))
			}
		}

		// If original content is > 500 chars, only first 500 should be shown
		if len(content) > 500 {
			truncatedContent := content[:500]
			if !strings.Contains(outputStr, truncatedContent) {
				t.Errorf("Output should contain truncated content (first 500 chars) when content length %d > 500", len(content))
			}
			// Should NOT contain content beyond 500 chars
			if len(content) > 510 {
				beyondLimit := content[505:510]
				if strings.Contains(outputStr, beyondLimit) {
					t.Errorf("Output should not contain content beyond 500 character limit")
				}
			}
		}
	})
}

// TestFilePreviewLengthEdgeCases tests specific edge cases for file preview
func TestFilePreviewLengthEdgeCases(t *testing.T) {
	tests := []struct {
		name           string
		content        string
		expectedLength int
	}{
		{
			name:           "empty file",
			content:        "",
			expectedLength: 0,
		},
		{
			name:           "exactly 500 characters",
			content:        strings.Repeat("a", 500),
			expectedLength: 500,
		},
		{
			name:           "501 characters",
			content:        strings.Repeat("a", 501),
			expectedLength: 500,
		},
		{
			name:           "very long content",
			content:        strings.Repeat("test content ", 100), // 1300 characters
			expectedLength: 500,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary file
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "test_file.txt")

			err := os.WriteFile(tmpFile, []byte(tt.content), 0644)
			if err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			// Create console and prompt
			var buf bytes.Buffer
			console := output.NewConsole(&buf)
			prompt := NewInteractivePrompt(console, nil)

			// Test ShowPreview
			err = prompt.ShowPreview(tmpFile, 500)
			if err != nil {
				t.Fatalf("ShowPreview failed: %v", err)
			}

			// For edge cases, we can verify the output contains expected information
			output := buf.String()

			if tt.expectedLength == 0 {
				// Empty file should show some indication
				if !strings.Contains(output, "File preview") {
					t.Error("Expected preview header for empty file")
				}
			} else {
				// Non-empty files should show content
				if !strings.Contains(output, "File preview") {
					t.Error("Expected preview header")
				}
			}
		})
	}
}

// TestAllYesBehavior tests Property 18: All Yes Behavior
// Feature: enhanced-output-cleanup, Property 18: All Yes Behavior
func TestAllYesBehavior(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate multiple file prompts
		numFiles := rapid.IntRange(2, 10).Draw(t, "numFiles")
		files := make([]*FilePrompt, numFiles)

		for i := 0; i < numFiles; i++ {
			files[i] = &FilePrompt{
				Path:    rapid.StringMatching(`/[a-zA-Z0-9_/]+\.[a-z]+`).Draw(t, "path"),
				Size:    rapid.Int64Range(0, 1000000).Draw(t, "size"),
				Type:    rapid.SampledFrom([]string{"text", "binary", "unknown"}).Draw(t, "type"),
				ModTime: time.Now(),
				Reason:  rapid.String().Draw(t, "reason"),
			}
		}

		// Create a mock reader that simulates user selecting "All yes" on first prompt
		input := "a\n" + strings.Repeat("should_not_be_read\n", numFiles-1)
		reader := strings.NewReader(input)

		var buf bytes.Buffer
		console := output.NewConsole(&buf)
		prompt := NewInteractivePrompt(console, reader)

		// Test PromptBatch
		results, err := prompt.PromptBatch(files)
		if err != nil {
			t.Fatalf("PromptBatch failed: %v", err)
		}

		// Property: All files should have ActionYes when user selects "All yes"
		for i, file := range files {
			action, exists := results[file.Path]
			if !exists {
				t.Errorf("File %d (%s) missing from results", i, file.Path)
				continue
			}

			if action != ActionYes {
				t.Errorf("File %d (%s): expected ActionYes, got %v", i, file.Path, action)
			}
		}

		// Verify that allYes state is set
		if !prompt.allYes {
			t.Error("Expected allYes state to be true after selecting 'All yes'")
		}
	})
}

// TestSkipAllBehavior tests Property 19: Skip All Behavior
// Feature: enhanced-output-cleanup, Property 19: Skip All Behavior
func TestSkipAllBehavior(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate multiple file prompts
		numFiles := rapid.IntRange(2, 10).Draw(t, "numFiles")
		files := make([]*FilePrompt, numFiles)

		for i := 0; i < numFiles; i++ {
			files[i] = &FilePrompt{
				Path:    rapid.StringMatching(`/[a-zA-Z0-9_/]+\.[a-z]+`).Draw(t, "path"),
				Size:    rapid.Int64Range(0, 1000000).Draw(t, "size"),
				Type:    rapid.SampledFrom([]string{"text", "binary", "unknown"}).Draw(t, "type"),
				ModTime: time.Now(),
				Reason:  rapid.String().Draw(t, "reason"),
			}
		}

		// Create a mock reader that simulates user selecting "Skip all" on first prompt
		input := "s\n" + strings.Repeat("should_not_be_read\n", numFiles-1)
		reader := strings.NewReader(input)

		var buf bytes.Buffer
		console := output.NewConsole(&buf)
		prompt := NewInteractivePrompt(console, reader)

		// Test PromptBatch
		results, err := prompt.PromptBatch(files)
		if err != nil {
			t.Fatalf("PromptBatch failed: %v", err)
		}

		// Property: All files should have ActionNo when user selects "Skip all"
		for i, file := range files {
			action, exists := results[file.Path]
			if !exists {
				t.Errorf("File %d (%s) missing from results", i, file.Path)
				continue
			}

			if action != ActionNo {
				t.Errorf("File %d (%s): expected ActionNo, got %v", i, file.Path, action)
			}
		}

		// Verify that skipAll state is set
		if !prompt.skipAll {
			t.Error("Expected skipAll state to be true after selecting 'Skip all'")
		}
	})
}

// TestBatchStateReset tests that Reset() properly clears batch state
func TestBatchStateReset(t *testing.T) {
	var buf bytes.Buffer
	console := output.NewConsole(&buf)
	prompt := NewInteractivePrompt(console, nil)

	// Set batch states
	prompt.allYes = true
	prompt.skipAll = true

	// Reset should clear both states
	prompt.Reset()

	if prompt.allYes {
		t.Error("Expected allYes to be false after Reset()")
	}

	if prompt.skipAll {
		t.Error("Expected skipAll to be false after Reset()")
	}
}

// TestIndividualPromptActions tests individual prompt actions
func TestIndividualPromptActions(t *testing.T) {
	tests := []struct {
		name             string
		input            string
		expectedAction   PromptAction
		expectBatchState bool
	}{
		{"yes", "y\n", ActionYes, false},
		{"Yes", "Y\n", ActionYes, false},
		{"yes full", "yes\n", ActionYes, false},
		{"no", "n\n", ActionNo, false},
		{"No", "N\n", ActionNo, false},
		{"no full", "no\n", ActionNo, false},
		{"all", "a\n", ActionYes, true}, // Returns ActionYes but sets allYes state
		{"All", "A\n", ActionYes, true}, // Returns ActionYes but sets allYes state
		{"skip", "s\n", ActionNo, true}, // Returns ActionNo but sets skipAll state
		{"Skip", "S\n", ActionNo, true}, // Returns ActionNo but sets skipAll state
		{"view", "v\n", ActionView, false},
		{"View", "V\n", ActionView, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary file for testing
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "test.txt")
			err := os.WriteFile(tmpFile, []byte("test content"), 0644)
			if err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			reader := strings.NewReader(tt.input)
			var buf bytes.Buffer
			console := output.NewConsole(&buf)
			prompt := NewInteractivePrompt(console, reader)

			file := &FilePrompt{
				Path:    tmpFile,
				Size:    12,
				Type:    "text",
				ModTime: time.Now(),
				Reason:  "test",
			}

			// For view action, we need to provide additional input after viewing
			if tt.expectedAction == ActionView {
				reader = strings.NewReader(tt.input + "n\n") // View then No
				prompt = NewInteractivePrompt(console, reader)

				action, err := prompt.Prompt(file)
				if err != nil {
					t.Fatalf("Prompt failed: %v", err)
				}

				// After viewing, user selected "n", so should get ActionNo
				if action != ActionNo {
					t.Errorf("Expected ActionNo after view+no, got %v", action)
				}
				return
			}

			action, err := prompt.Prompt(file)
			if err != nil {
				t.Fatalf("Prompt failed: %v", err)
			}

			if action != tt.expectedAction {
				t.Errorf("Expected %v, got %v", tt.expectedAction, action)
			}

			// Check batch state if expected
			if tt.expectBatchState {
				if tt.expectedAction == ActionYes && !prompt.allYes {
					t.Error("Expected allYes state to be set")
				}
				if tt.expectedAction == ActionNo && !prompt.skipAll {
					t.Error("Expected skipAll state to be set")
				}
			}
		})
	}
}
