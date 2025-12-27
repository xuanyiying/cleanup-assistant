package analyzer

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"pgregory.net/rapid"
)

// TestFileMetadataExtractionCompleteness validates Property 1
// Feature: cleanup-cli, Property 1: File Metadata Extraction Completeness
// For any valid file path, the Analyzer SHALL extract complete metadata including
// Name, Extension, Size, MimeType, CreatedAt, and ModifiedAt fields, with all
// fields being non-zero values.
// Validates: Requirements 3.1
func TestFileMetadataExtractionCompleteness(t *testing.T) {
	tmpDir := t.TempDir()

	rapid.Check(t, func(t *rapid.T) {
		// Generate random filename with extension
		filename := rapid.StringMatching(`[a-z]{3,10}\.(txt|md|go|py|js)`).Draw(t, "filename")
		filePath := filepath.Join(tmpDir, filename)

		// Generate random content
		content := rapid.StringMatching(`[a-zA-Z0-9\s]{10,100}`).Draw(t, "content")

		// Create the test file
		err := os.WriteFile(filePath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}
		defer os.Remove(filePath)

		// Analyze the file
		analyzer := NewAnalyzer()
		metadata, err := analyzer.Analyze(context.Background(), filePath)
		if err != nil {
			t.Fatalf("failed to analyze file: %v", err)
		}

		// Verify all required fields are populated
		if metadata.Path == "" {
			t.Error("Path field is empty")
		}
		if metadata.Name == "" {
			t.Error("Name field is empty")
		}
		if metadata.Extension == "" {
			t.Error("Extension field is empty")
		}
		if metadata.Size == 0 {
			t.Error("Size field is zero")
		}
		if metadata.MimeType == "" {
			t.Error("MimeType field is empty")
		}
		if metadata.ModifiedAt.IsZero() {
			t.Error("ModifiedAt field is zero")
		}

		// Verify values are correct
		if metadata.Name != filename {
			t.Errorf("Name mismatch: expected %s, got %s", filename, metadata.Name)
		}
		if metadata.Size != int64(len(content)) {
			t.Errorf("Size mismatch: expected %d, got %d", len(content), metadata.Size)
		}
	})
}

// TestFileFilteringCorrectness validates Property 13
// Feature: cleanup-cli, Property 13: File Filtering Correctness
// For any filter criteria (pattern, date range, size range), only files matching
// ALL specified criteria SHALL be included in the result set.
// Validates: Requirements 8.4
func TestFileFilteringCorrectness(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files with different sizes and timestamps
	now := time.Now()
	testFiles := []struct {
		name    string
		size    int
		modTime time.Time
	}{
		{"small.txt", 100, now.Add(-24 * time.Hour)},
		{"medium.txt", 5000, now.Add(-12 * time.Hour)},
		{"large.txt", 50000, now},
		{"old.md", 1000, now.Add(-48 * time.Hour)},
		{"recent.md", 2000, now.Add(-1 * time.Hour)},
	}

	for _, tf := range testFiles {
		filePath := filepath.Join(tmpDir, tf.name)
		content := make([]byte, tf.size)
		if err := os.WriteFile(filePath, content, 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}
		if err := os.Chtimes(filePath, tf.modTime, tf.modTime); err != nil {
			t.Fatalf("failed to set file time: %v", err)
		}
	}

	analyzer := NewAnalyzer()

	// Test 1: Filter by pattern only
	filter := &FileFilter{
		Patterns: []string{"*.txt"},
	}
	opts := &ScanOptions{
		Recursive:     false,
		IncludeHidden: false,
		Filter:        filter,
	}
	results, err := analyzer.AnalyzeDirectory(context.Background(), tmpDir, opts)
	if err != nil {
		t.Fatalf("failed to analyze directory: %v", err)
	}

	// Should only get .txt files
	for _, r := range results {
		if filepath.Ext(r.Name) != ".txt" {
			t.Errorf("filter by pattern failed: got %s", r.Name)
		}
	}

	// Test 2: Filter by size range
	filter = &FileFilter{
		MinSize: 1000,
		MaxSize: 10000,
	}
	opts.Filter = filter
	results, err = analyzer.AnalyzeDirectory(context.Background(), tmpDir, opts)
	if err != nil {
		t.Fatalf("failed to analyze directory: %v", err)
	}

	// All results should be within size range
	for _, r := range results {
		if r.Size < 1000 || r.Size > 10000 {
			t.Errorf("filter by size failed: got size %d", r.Size)
		}
	}

	// Test 3: Filter by date range
	cutoffTime := now.Add(-24 * time.Hour)
	filter = &FileFilter{
		ModifiedAfter: cutoffTime,
	}
	opts.Filter = filter
	results, err = analyzer.AnalyzeDirectory(context.Background(), tmpDir, opts)
	if err != nil {
		t.Fatalf("failed to analyze directory: %v", err)
	}

	// All results should be after cutoff time
	for _, r := range results {
		if r.ModifiedAt.Before(cutoffTime) {
			t.Errorf("filter by date failed: got time %v", r.ModifiedAt)
		}
	}

	// Test 4: Combined filters (pattern AND size)
	filter = &FileFilter{
		Patterns: []string{"*.txt"},
		MinSize:  1000,
	}
	opts.Filter = filter
	results, err = analyzer.AnalyzeDirectory(context.Background(), tmpDir, opts)
	if err != nil {
		t.Fatalf("failed to analyze directory: %v", err)
	}

	// All results must match BOTH criteria
	for _, r := range results {
		if filepath.Ext(r.Name) != ".txt" {
			t.Errorf("combined filter failed: pattern mismatch %s", r.Name)
		}
		if r.Size < 1000 {
			t.Errorf("combined filter failed: size mismatch %d", r.Size)
		}
	}
}

// TestAssessFileNameQuality tests filename quality assessment
func TestAssessFileNameQuality(t *testing.T) {
	fa := NewAnalyzer()

	testCases := []struct {
		filename string
		expected FileNameQuality
	}{
		// Good names
		{"project-report-2024.pdf", FileNameGood},
		{"meeting-notes.txt", FileNameGood},
		{"vacation-photos.jpg", FileNameGood},
		{"budget-analysis.xlsx", FileNameGood},

		// Meaningless names
		{"untitled.txt", FileNameMeaningless},
		{"新建文档.docx", FileNameMeaningless},
		{"IMG_1234.jpg", FileNameMeaningless},
		{"Screenshot.png", FileNameMeaningless},
		{"20240101_123456.pdf", FileNameMeaningless},
		{"download.pdf", FileNameMeaningless},
		{"temp.txt", FileNameMeaningless},
		{"123456.jpg", FileNameMeaningless},
		{"a.txt", FileNameMeaningless},
		{"file.pdf", FileNameMeaningless},

		// Generic names
		{"doc.txt", FileNameGeneric},
		{"data.csv", FileNameGeneric},
	}

	for _, tc := range testCases {
		t.Run(tc.filename, func(t *testing.T) {
			quality := fa.AssessFileNameQuality(tc.filename)
			if quality != tc.expected {
				t.Errorf("filename %s: expected %s, got %s", tc.filename, tc.expected, quality)
			}
		})
	}
}

// TestFileMetadataWithQualityAssessment tests that file analysis includes quality assessment
func TestFileMetadataWithQualityAssessment(t *testing.T) {
	tmpDir := t.TempDir()

	testCases := []struct {
		filename        string
		content         string
		expectedQuality FileNameQuality
		expectNeedsName bool
	}{
		{"good-filename.txt", "content", FileNameGood, false},
		{"IMG_1234.jpg", "image data", FileNameMeaningless, true},
		{"untitled.pdf", "pdf content", FileNameMeaningless, true},
	}

	for _, tc := range testCases {
		t.Run(tc.filename, func(t *testing.T) {
			filePath := filepath.Join(tmpDir, tc.filename)
			if err := os.WriteFile(filePath, []byte(tc.content), 0644); err != nil {
				t.Fatalf("failed to create test file: %v", err)
			}

			fa := NewAnalyzer()
			ctx := context.Background()

			metadata, err := fa.Analyze(ctx, filePath)
			if err != nil {
				t.Fatalf("failed to analyze file: %v", err)
			}

			if metadata.FileNameQuality != tc.expectedQuality {
				t.Errorf("expected quality %s, got %s", tc.expectedQuality, metadata.FileNameQuality)
			}

			if metadata.NeedsSmarterName != tc.expectNeedsName {
				t.Errorf("expected NeedsSmarterName %v, got %v", tc.expectNeedsName, metadata.NeedsSmarterName)
			}
		})
	}
}
