package visualizer

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xuanyiying/cleanup-cli/internal/output"
	"pgregory.net/rapid"
)

// TestTreeRenderingStructureCorrectness validates Property 1: Tree Rendering Structure Correctness
// For any valid directory structure, the rendered tree output SHALL contain proper branch characters 
// (├── └── │) with correct indentation levels matching the directory depth.
// Feature: enhanced-output-cleanup, Property 1: Tree Rendering Structure Correctness
// Validates: Requirements 1.1, 1.2
func TestTreeRenderingStructureCorrectness(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Create temporary directory for testing
		tmpDir, err := os.MkdirTemp("", "tree-test-*")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		// Generate random directory structure
		maxDepth := rapid.IntRange(1, 4).Draw(t, "maxDepth")
		filesPerDir := rapid.IntRange(1, 5).Draw(t, "filesPerDir")
		
		// Create nested directory structure
		createRandomStructure(t, tmpDir, maxDepth, filesPerDir, 0)

		// Create tree visualizer with Unicode enabled
		console := output.NewConsole(&bytes.Buffer{})
		options := &TreeOptions{
			MaxDepth:   0, // No depth limit for this test
			ShowSize:   false,
			ShowHidden: false,
			UseColor:   false, // Disable color for easier testing
			UseUnicode: true,
			IndentSize: 3,
		}
		visualizer := NewTreeVisualizer(console, options)

		// Build and render tree
		tree, err := visualizer.BuildTree(tmpDir)
		if err != nil {
			t.Fatalf("failed to build tree: %v", err)
		}

		rendered := visualizer.Render(tree)
		lines := strings.Split(strings.TrimSpace(rendered), "\n")

		// PROPERTY: Verify proper branch characters are used
		for i, line := range lines {
			if i == 0 {
				// Root node should not have branch characters
				continue
			}

			// Count leading spaces to determine depth
			trimmed := strings.TrimLeft(line, " ")
			if trimmed == "" {
				continue
			}

			// Check for proper branch characters
			hasBranchChars := strings.Contains(line, BranchTee) || 
							 strings.Contains(line, BranchCorner) || 
							 strings.Contains(line, BranchVertical)

			if !hasBranchChars && !strings.HasPrefix(trimmed, filepath.Base(tmpDir)) {
				t.Fatalf("line %d missing proper branch characters: %q", i, line)
			}

			// Verify indentation consistency
			if strings.Contains(line, BranchTee) || strings.Contains(line, BranchCorner) {
				// This is a direct child line, verify it has proper structure
				parts := strings.Split(line, BranchTee)
				if len(parts) == 1 {
					parts = strings.Split(line, BranchCorner)
				}
				
				if len(parts) >= 2 {
					prefix := parts[0]
					// Prefix should only contain spaces and vertical bars
					for _, char := range prefix {
						if char != ' ' && char != '│' {
							t.Fatalf("invalid prefix character in line %d: %q", i, line)
						}
					}
				}
			}
		}

		// PROPERTY: Verify depth consistency
		verifyDepthConsistency(tree, 0)
	})
}

// createRandomStructure creates a random directory structure for testing
func createRandomStructure(t *rapid.T, basePath string, maxDepth, filesPerDir, currentDepth int) {
	if currentDepth >= maxDepth {
		return
	}

	// Create some files in current directory
	numFiles := rapid.IntRange(0, filesPerDir).Draw(t, "numFiles")
	for i := 0; i < numFiles; i++ {
		fileName := rapid.StringMatching(`[a-z]{3,8}\.txt`).Draw(t, "fileName")
		filePath := filepath.Join(basePath, fileName)
		content := rapid.StringMatching(`[a-zA-Z0-9 ]{10,50}`).Draw(t, "fileContent")
		
		err := os.WriteFile(filePath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("failed to create file %s: %v", filePath, err)
		}
	}

	// Create some subdirectories
	numDirs := rapid.IntRange(0, 3).Draw(t, "numDirs")
	for i := 0; i < numDirs; i++ {
		dirName := rapid.StringMatching(`[a-z]{3,8}`).Draw(t, "dirName")
		dirPath := filepath.Join(basePath, dirName)
		
		err := os.Mkdir(dirPath, 0755)
		if err != nil {
			t.Fatalf("failed to create directory %s: %v", dirPath, err)
		}

		// Recursively create structure in subdirectory
		createRandomStructure(t, dirPath, maxDepth, filesPerDir, currentDepth+1)
	}
}

// verifyDepthConsistency verifies that tree node depths are consistent
func verifyDepthConsistency(node *TreeNode, expectedDepth int) {
	if node.Depth != expectedDepth {
		panic(fmt.Sprintf("node %s has incorrect depth: expected %d, got %d", node.Name, expectedDepth, node.Depth))
	}

	for _, child := range node.Children {
		verifyDepthConsistency(child, expectedDepth+1)
	}
}

// TestTreeDepthLimiting validates Property 2: Tree Depth Limiting
// For any directory structure with depth greater than maxDepth, the rendered tree SHALL only 
// display nodes up to maxDepth levels, with no nodes beyond that depth appearing in the output.
// Feature: enhanced-output-cleanup, Property 2: Tree Depth Limiting
// Validates: Requirements 1.5
func TestTreeDepthLimiting(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Create temporary directory for testing
		tmpDir, err := os.MkdirTemp("", "tree-depth-test-*")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		// Generate random parameters
		maxDepth := rapid.IntRange(1, 3).Draw(t, "maxDepth")
		actualDepth := rapid.IntRange(maxDepth+1, maxDepth+3).Draw(t, "actualDepth")
		
		// Create deep directory structure that exceeds maxDepth
		createDeepStructure(t, tmpDir, actualDepth)

		// Create tree visualizer with depth limit
		console := output.NewConsole(&bytes.Buffer{})
		options := &TreeOptions{
			MaxDepth:   maxDepth,
			ShowSize:   false,
			ShowHidden: false,
			UseColor:   false,
			UseUnicode: true,
			IndentSize: 3,
		}
		visualizer := NewTreeVisualizer(console, options)

		// Build and render tree
		tree, err := visualizer.BuildTree(tmpDir)
		if err != nil {
			t.Fatalf("failed to build tree: %v", err)
		}

		// PROPERTY: Verify no nodes exceed maxDepth
		verifyMaxDepth(tree, maxDepth)

		// Also verify in rendered output
		rendered := visualizer.Render(tree)
		lines := strings.Split(strings.TrimSpace(rendered), "\n")

		// Count maximum indentation level in output
		maxIndentLevel := 0
		for _, line := range lines {
			if strings.TrimSpace(line) == "" {
				continue
			}
			
			indentLevel := countIndentLevel(line)
			if indentLevel > maxIndentLevel {
				maxIndentLevel = indentLevel
			}
		}

		// PROPERTY: Maximum indent level should not exceed maxDepth
		if maxIndentLevel > maxDepth {
			t.Fatalf("rendered tree exceeds maxDepth: maxDepth=%d, maxIndentLevel=%d", maxDepth, maxIndentLevel)
		}
	})
}

// createDeepStructure creates a deep directory structure for testing depth limits
func createDeepStructure(t *rapid.T, basePath string, depth int) {
	currentPath := basePath
	
	for i := 0; i < depth; i++ {
		// Create a file at current level
		fileName := rapid.StringMatching(`file[0-9]\.txt`).Draw(t, "fileName")
		filePath := filepath.Join(currentPath, fileName)
		content := rapid.StringMatching(`content[0-9]+`).Draw(t, "content")
		
		err := os.WriteFile(filePath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("failed to create file %s: %v", filePath, err)
		}

		// Create next level directory if not at max depth
		if i < depth-1 {
			dirName := rapid.StringMatching(`dir[0-9]`).Draw(t, "dirName")
			nextPath := filepath.Join(currentPath, dirName)
			
			err := os.Mkdir(nextPath, 0755)
			if err != nil {
				t.Fatalf("failed to create directory %s: %v", nextPath, err)
			}
			
			currentPath = nextPath
		}
	}
}

// verifyMaxDepth verifies that no node in the tree exceeds the maximum depth
func verifyMaxDepth(node *TreeNode, maxDepth int) {
	if node.Depth > maxDepth {
		panic(fmt.Sprintf("node %s exceeds maxDepth: depth=%d, maxDepth=%d", node.Name, node.Depth, maxDepth))
	}

	for _, child := range node.Children {
		verifyMaxDepth(child, maxDepth)
	}
}

// countIndentLevel counts the indentation level of a line based on branch characters
func countIndentLevel(line string) int {
	level := 0
	
	// Count occurrences of vertical bars and branch characters to determine depth
	for i, char := range line {
		if char == '│' || char == '├' || char == '└' {
			// This indicates we're at a certain depth level
			// Count how many branch/vertical characters we've seen
			prefix := line[:i+1]
			level = strings.Count(prefix, "│") + strings.Count(prefix, "├") + strings.Count(prefix, "└")
			break
		}
	}
	
	return level
}
// TestUnicodeFallback validates Property 22: Unicode Fallback
// For any tree rendering when Unicode is disabled, the output SHALL use ASCII fallback 
// characters (|, --, +, `) instead of Unicode box-drawing characters.
// Feature: enhanced-output-cleanup, Property 22: Unicode Fallback
// Validates: Requirements 8.6
func TestUnicodeFallback(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Create temporary directory for testing
		tmpDir, err := os.MkdirTemp("", "tree-unicode-test-*")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		// Generate random directory structure
		numFiles := rapid.IntRange(1, 3).Draw(t, "numFiles")
		numDirs := rapid.IntRange(1, 2).Draw(t, "numDirs")
		
		// Create files and directories
		for i := 0; i < numFiles; i++ {
			fileName := rapid.StringMatching(`[a-z]{3,6}\.txt`).Draw(t, "fileName")
			filePath := filepath.Join(tmpDir, fileName)
			content := rapid.StringMatching(`[a-zA-Z0-9 ]{5,20}`).Draw(t, "content")
			
			err := os.WriteFile(filePath, []byte(content), 0644)
			if err != nil {
				t.Fatalf("failed to create file %s: %v", filePath, err)
			}
		}

		for i := 0; i < numDirs; i++ {
			dirName := rapid.StringMatching(`[a-z]{3,6}`).Draw(t, "dirName")
			dirPath := filepath.Join(tmpDir, dirName)
			
			err := os.Mkdir(dirPath, 0755)
			if err != nil {
				t.Fatalf("failed to create directory %s: %v", dirPath, err)
			}

			// Add one file to the subdirectory
			subFileName := rapid.StringMatching(`sub[a-z]{2,4}\.txt`).Draw(t, "subFileName")
			subFilePath := filepath.Join(dirPath, subFileName)
			subContent := rapid.StringMatching(`[a-zA-Z0-9 ]{5,15}`).Draw(t, "subContent")
			
			err = os.WriteFile(subFilePath, []byte(subContent), 0644)
			if err != nil {
				t.Fatalf("failed to create sub file %s: %v", subFilePath, err)
			}
		}

		// Test with Unicode enabled
		console := output.NewConsole(&bytes.Buffer{})
		unicodeOptions := &TreeOptions{
			MaxDepth:   0,
			ShowSize:   false,
			ShowHidden: false,
			UseColor:   false,
			UseUnicode: true,
			IndentSize: 3,
		}
		unicodeVisualizer := NewTreeVisualizer(console, unicodeOptions)

		tree, err := unicodeVisualizer.BuildTree(tmpDir)
		if err != nil {
			t.Fatalf("failed to build tree: %v", err)
		}

		unicodeRendered := unicodeVisualizer.Render(tree)

		// Test with Unicode disabled (ASCII fallback)
		asciiOptions := &TreeOptions{
			MaxDepth:   0,
			ShowSize:   false,
			ShowHidden: false,
			UseColor:   false,
			UseUnicode: false,
			IndentSize: 3,
		}
		asciiVisualizer := NewTreeVisualizer(console, asciiOptions)

		asciiRendered := asciiVisualizer.Render(tree)

		// PROPERTY: Unicode version should contain Unicode box-drawing characters
		unicodeChars := []string{BranchVertical, BranchTee, BranchCorner, BranchHorizontal}
		hasUnicodeChars := false
		for _, char := range unicodeChars {
			if strings.Contains(unicodeRendered, char) {
				hasUnicodeChars = true
				break
			}
		}

		if !hasUnicodeChars && len(tree.Children) > 0 {
			t.Fatalf("Unicode rendering should contain Unicode box-drawing characters when tree has children")
		}

		// PROPERTY: ASCII version should NOT contain Unicode box-drawing characters
		for _, char := range unicodeChars {
			if strings.Contains(asciiRendered, char) {
				t.Fatalf("ASCII rendering contains Unicode character %q: %s", char, asciiRendered)
			}
		}

		// PROPERTY: ASCII version should contain ASCII fallback characters
		asciiChars := []string{BranchVerticalASCII, BranchTeeASCII, BranchCornerASCII, BranchHorizontalASCII}
		hasAsciiChars := false
		for _, char := range asciiChars {
			if strings.Contains(asciiRendered, char) {
				hasAsciiChars = true
				break
			}
		}

		if !hasAsciiChars && len(tree.Children) > 0 {
			t.Fatalf("ASCII rendering should contain ASCII fallback characters when tree has children")
		}

		// PROPERTY: Both renderings should have the same number of lines (same structure)
		unicodeLines := strings.Split(strings.TrimSpace(unicodeRendered), "\n")
		asciiLines := strings.Split(strings.TrimSpace(asciiRendered), "\n")

		if len(unicodeLines) != len(asciiLines) {
			t.Fatalf("Unicode and ASCII renderings have different number of lines: unicode=%d, ascii=%d", 
				len(unicodeLines), len(asciiLines))
		}

		// PROPERTY: Both renderings should contain the same file/directory names
		unicodeNames := extractFileNames(unicodeRendered)
		asciiNames := extractFileNames(asciiRendered)

		if len(unicodeNames) != len(asciiNames) {
			t.Fatalf("Unicode and ASCII renderings have different number of files: unicode=%d, ascii=%d", 
				len(unicodeNames), len(asciiNames))
		}

		// Verify all names are present in both renderings
		for _, name := range unicodeNames {
			found := false
			for _, asciiName := range asciiNames {
				if name == asciiName {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("file name %q found in Unicode rendering but not in ASCII rendering", name)
			}
		}
	})
}

// extractFileNames extracts file and directory names from rendered tree output
func extractFileNames(rendered string) []string {
	var names []string
	lines := strings.Split(rendered, "\n")
	
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		// Remove branch characters and extract the name
		cleaned := line
		
		// Remove Unicode branch characters
		cleaned = strings.ReplaceAll(cleaned, BranchVertical, "")
		cleaned = strings.ReplaceAll(cleaned, BranchTee, "")
		cleaned = strings.ReplaceAll(cleaned, BranchCorner, "")
		cleaned = strings.ReplaceAll(cleaned, BranchHorizontal, "")
		
		// Remove ASCII branch characters
		cleaned = strings.ReplaceAll(cleaned, BranchVerticalASCII, "")
		cleaned = strings.ReplaceAll(cleaned, BranchTeeASCII, "")
		cleaned = strings.ReplaceAll(cleaned, BranchCornerASCII, "")
		cleaned = strings.ReplaceAll(cleaned, BranchHorizontalASCII, "")
		
		// Remove extra spaces and extract name
		cleaned = strings.TrimSpace(cleaned)
		
		if cleaned != "" {
			// Remove any size information in parentheses
			if idx := strings.Index(cleaned, "("); idx != -1 {
				cleaned = strings.TrimSpace(cleaned[:idx])
			}
			
			names = append(names, cleaned)
		}
	}
	
	return names
}