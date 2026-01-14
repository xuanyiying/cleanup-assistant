package cleaner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"
)

// TestPlatformSpecificJunkDetection tests Property 8: Platform-Specific Junk Detection
// Feature: enhanced-output-cleanup, Property 8: Platform-Specific Junk Detection
func TestPlatformSpecificJunkDetection(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Create a scanner
		scanner := NewJunkScanner()
		
		// Get default locations for current platform
		locations := scanner.GetDefaultLocations()
		
		// Verify that all locations are either for current platform or "all"
		currentPlatform := runtime.GOOS
		for _, location := range locations {
			assert.True(t, 
				location.Platform == currentPlatform || location.Platform == "all",
				"Location %s has platform %s but current platform is %s", 
				location.Path, location.Platform, currentPlatform)
		}
		
		// Verify platform-specific locations exist
		switch currentPlatform {
		case "darwin":
			// Should have macOS-specific locations
			foundMacOSLocation := false
			for _, location := range locations {
				if location.Platform == "darwin" {
					foundMacOSLocation = true
					break
				}
			}
			assert.True(t, foundMacOSLocation, "Should have macOS-specific locations")
			
		case "windows":
			// Should have Windows-specific locations
			foundWindowsLocation := false
			for _, location := range locations {
				if location.Platform == "windows" {
					foundWindowsLocation = true
					break
				}
			}
			assert.True(t, foundWindowsLocation, "Should have Windows-specific locations")
		}
		
		// Verify each category is represented
		categories := make(map[JunkCategory]bool)
		for _, location := range locations {
			categories[location.Category] = true
		}
		
		// Should have at least cache and temp categories
		assert.True(t, categories[CategoryCache], "Should have cache category")
		assert.True(t, categories[CategoryTemp], "Should have temp category")
	})
}

// TestJunkSizeCalculation tests Property 9: Junk Size Calculation
// Feature: enhanced-output-cleanup, Property 9: Junk Size Calculation
func TestJunkSizeCalculation(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		// Create temporary directory structure
		tempDir := t.TempDir()
		
		// Generate random file sizes
		numFiles := rapid.IntRange(1, 10).Draw(rt, "numFiles")
		var expectedTotalSize int64
		
		// Create files with known sizes and unique names
		for i := 0; i < numFiles; i++ {
			fileSize := rapid.Int64Range(0, 1024*1024).Draw(rt, "fileSize") // Up to 1MB
			// Use index to ensure unique filenames
			fileName := filepath.Join(tempDir, fmt.Sprintf("file_%d.tmp", i))
			
			// Create file with specific size
			file, err := os.Create(fileName)
			require.NoError(t, err)
			
			// Write data to reach desired size
			if fileSize > 0 {
				data := make([]byte, fileSize)
				_, err = file.Write(data)
				require.NoError(t, err)
			}
			file.Close()
			
			expectedTotalSize += fileSize
		}
		
		// Create scanner with custom location
		scanner := NewJunkScanner()
		scanner.locations = []*JunkLocation{
			{
				Path:        tempDir,
				Category:    CategoryTemp,
				Description: "Test temp files",
				Platform:    "all",
			},
		}
		
		// Scan for junk files
		ctx := context.Background()
		result, err := scanner.Scan(ctx)
		require.NoError(t, err)
		
		// Verify total size calculation
		var actualTotalSize int64
		for _, file := range result.Files {
			actualTotalSize += file.Size
		}
		
		assert.Equal(t, expectedTotalSize, actualTotalSize, 
			"Total size should equal sum of individual file sizes")
		assert.Equal(t, expectedTotalSize, result.TotalSize,
			"Result.TotalSize should equal sum of individual file sizes")
	})
}

// TestJunkCategorization tests Property 10: Junk Categorization
// Feature: enhanced-output-cleanup, Property 10: Junk Categorization
func TestJunkCategorization(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		// Create temporary directory structure
		tempDir := t.TempDir()
		
		// Generate random category
		categories := []JunkCategory{CategoryCache, CategoryLogs, CategoryTemp, CategoryTrash}
		selectedCategory := rapid.SampledFrom(categories).Draw(rt, "category")
		
		// Create some files
		numFiles := rapid.IntRange(1, 5).Draw(rt, "numFiles")
		for i := 0; i < numFiles; i++ {
			fileName := filepath.Join(tempDir, rapid.StringMatching(`[a-z]+\.tmp`).Draw(rt, "fileName"))
			file, err := os.Create(fileName)
			require.NoError(t, err)
			file.Close()
		}
		
		// Create scanner with custom location
		scanner := NewJunkScanner()
		scanner.locations = []*JunkLocation{
			{
				Path:        tempDir,
				Category:    selectedCategory,
				Description: "Test files",
				Platform:    "all",
			},
		}
		
		// Scan for junk files
		ctx := context.Background()
		result, err := scanner.Scan(ctx)
		require.NoError(t, err)
		
		// Verify all files are assigned to the correct category
		for _, file := range result.Files {
			assert.Equal(t, selectedCategory, file.Category,
				"File %s should be categorized as %s", file.Path, selectedCategory)
		}
		
		// Verify category map contains files
		if len(result.Files) > 0 {
			assert.Contains(t, result.ByCategory, selectedCategory,
				"ByCategory map should contain the selected category")
			assert.Equal(t, len(result.Files), len(result.ByCategory[selectedCategory]),
				"ByCategory should contain all files for the category")
		}
	})
}

// TestImportantFileExclusion tests Property 16: Important File Exclusion
// Feature: enhanced-output-cleanup, Property 16: Important File Exclusion
func TestImportantFileExclusion(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		// Create temporary directory structure
		tempDir := t.TempDir()
		
		// Create important file patterns
		importantPatterns := []string{"test.key", "config.pem", ".env", "id_rsa"}
		selectedPattern := rapid.SampledFrom(importantPatterns).Draw(rt, "pattern")
		
		// Create important file
		importantFile := filepath.Join(tempDir, selectedPattern)
		file, err := os.Create(importantFile)
		require.NoError(t, err)
		file.Close()
		
		// Create some normal junk files
		numJunkFiles := rapid.IntRange(1, 3).Draw(rt, "numJunkFiles")
		for i := 0; i < numJunkFiles; i++ {
			// Use index to ensure unique filenames
			fileName := filepath.Join(tempDir, fmt.Sprintf("junk%d_%s.tmp", i, rapid.StringMatching(`[0-9]+`).Draw(rt, "suffix")))
			file, err := os.Create(fileName)
			require.NoError(t, err)
			file.Close()
		}
		
		// Create scanner with custom location
		scanner := NewJunkScanner()
		scanner.locations = []*JunkLocation{
			{
				Path:        tempDir,
				Category:    CategoryTemp,
				Description: "Test files",
				Platform:    "all",
			},
		}
		
		// Scan for junk files
		ctx := context.Background()
		result, err := scanner.Scan(ctx)
		require.NoError(t, err)
		
		// Verify important file is excluded from cleanup list
		for _, file := range result.Files {
			assert.False(t, scanner.classifier.IsImportant(file.Path),
				"Important file %s should be excluded from cleanup list", file.Path)
		}
		
		// Verify we still found the junk files
		assert.Equal(t, numJunkFiles, len(result.Files),
			"Should find all junk files but exclude important files")
	})
}

// Unit tests for specific functionality

func TestNewJunkScanner(t *testing.T) {
	scanner := NewJunkScanner()
	
	assert.NotNil(t, scanner)
	assert.NotNil(t, scanner.classifier)
	assert.Equal(t, runtime.GOOS, scanner.platform)
	assert.NotEmpty(t, scanner.locations, "Should have default locations")
}

func TestAddLocation(t *testing.T) {
	scanner := NewJunkScanner()
	initialCount := len(scanner.locations)
	
	customLocation := &JunkLocation{
		Path:        "/custom/path",
		Category:    CategoryCache,
		Description: "Custom location",
		Platform:    "all",
	}
	
	scanner.AddLocation(customLocation)
	
	assert.Equal(t, initialCount+1, len(scanner.locations))
	assert.Contains(t, scanner.locations, customLocation)
}

func TestExpandPath(t *testing.T) {
	scanner := NewJunkScanner()
	
	// Test home directory expansion
	homeDir, _ := os.UserHomeDir()
	expanded := scanner.expandPath("~/test")
	expected := filepath.Join(homeDir, "test")
	assert.Equal(t, expected, expanded)
	
	// Test environment variable expansion
	os.Setenv("TEST_VAR", "test_value")
	expanded = scanner.expandPath("$TEST_VAR/path")
	assert.Contains(t, expanded, "test_value")
}

func TestScanNonExistentPath(t *testing.T) {
	scanner := NewJunkScanner()
	scanner.locations = []*JunkLocation{
		{
			Path:        "/non/existent/path",
			Category:    CategoryTemp,
			Description: "Non-existent path",
			Platform:    "all",
		},
	}
	
	ctx := context.Background()
	result, err := scanner.Scan(ctx)
	
	assert.NoError(t, err)
	assert.Empty(t, result.Files, "Should not find files in non-existent path")
	assert.Equal(t, int64(0), result.TotalSize)
}

func TestScanCategoryFilter(t *testing.T) {
	// Create temporary directory with files
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.tmp")
	file, err := os.Create(testFile)
	require.NoError(t, err)
	file.Close()
	
	scanner := NewJunkScanner()
	scanner.locations = []*JunkLocation{
		{
			Path:        tempDir,
			Category:    CategoryCache,
			Description: "Cache files",
			Platform:    "all",
		},
		{
			Path:        tempDir,
			Category:    CategoryTemp,
			Description: "Temp files",
			Platform:    "all",
		},
	}
	
	ctx := context.Background()
	
	// Scan only cache category
	result, err := scanner.ScanCategory(ctx, CategoryCache)
	require.NoError(t, err)
	
	// All files should be categorized as cache
	for _, file := range result.Files {
		assert.Equal(t, CategoryCache, file.Category)
	}
}

func TestContextCancellation(t *testing.T) {
	scanner := NewJunkScanner()
	
	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	
	result, err := scanner.Scan(ctx)
	
	assert.Error(t, err)
	assert.Equal(t, context.Canceled, err)
	assert.NotNil(t, result)
}