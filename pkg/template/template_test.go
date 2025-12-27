package template

import (
	"fmt"
	"regexp"
	"testing"
	"time"

	"pgregory.net/rapid"
)

// TestTemplateExpansionCorrectness validates Property 6
// Feature: cleanup-cli, Property 6: Template Expansion Correctness
// For any valid template string with placeholders (e.g., {year}, {month}, {category}),
// the expanded path SHALL contain the actual values substituted for all placeholders,
// with no unexpanded placeholders remaining.
// Validates: Requirements 5.4
func TestTemplateExpansionCorrectness(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random placeholder values
		year := rapid.IntRange(2000, 2100).Draw(t, "year")
		month := rapid.IntRange(1, 12).Draw(t, "month")
		day := rapid.IntRange(1, 28).Draw(t, "day")
		category := rapid.StringMatching(`[a-z]{3,10}`).Draw(t, "category")
		ext := rapid.StringMatching(`[a-z]{2,4}`).Draw(t, "ext")

		// Create placeholders map
		placeholders := map[string]string{
			"year":     fmt.Sprintf("%04d", year),
			"month":    fmt.Sprintf("%02d", month),
			"day":      fmt.Sprintf("%02d", day),
			"category": category,
			"ext":      ext,
		}

		expander := NewExpander(placeholders)

		// Test 1: Single placeholder expansion
		template := "{category}/{year}"
		result, err := expander.ExpandPath(template)
		if err != nil {
			t.Fatalf("failed to expand template: %v", err)
		}

		// Verify no unexpanded placeholders remain
		if regexp.MustCompile(`\{[^}]+\}`).MatchString(result) {
			t.Errorf("unexpanded placeholders remain: %s", result)
		}

		// Verify actual values are present
		if !regexp.MustCompile(category).MatchString(result) {
			t.Errorf("category value not found in result: %s", result)
		}
		if !regexp.MustCompile(fmt.Sprintf("%04d", year)).MatchString(result) {
			t.Errorf("year value not found in result: %s", result)
		}

		// Test 2: Multiple placeholders expansion
		template = "{year}/{month}/{day}/{category}.{ext}"
		result, err = expander.ExpandPath(template)
		if err != nil {
			t.Fatalf("failed to expand template: %v", err)
		}

		// Verify no unexpanded placeholders remain
		if regexp.MustCompile(`\{[^}]+\}`).MatchString(result) {
			t.Errorf("unexpanded placeholders remain: %s", result)
		}

		// Verify all values are present
		expectedValues := []string{
			fmt.Sprintf("%04d", year),
			fmt.Sprintf("%02d", month),
			fmt.Sprintf("%02d", day),
			category,
			ext,
		}
		for _, val := range expectedValues {
			if !regexp.MustCompile(regexp.QuoteMeta(val)).MatchString(result) {
				t.Errorf("expected value %s not found in result: %s", val, result)
			}
		}

		// Test 3: Path with directory separators
		template = "Documents/{year}/{month}/{category}"
		result, err = expander.ExpandPath(template)
		if err != nil {
			t.Fatalf("failed to expand template: %v", err)
		}

		// Verify no unexpanded placeholders remain
		if regexp.MustCompile(`\{[^}]+\}`).MatchString(result) {
			t.Errorf("unexpanded placeholders remain: %s", result)
		}

		// Verify structure is preserved
		if !regexp.MustCompile(`^Documents/`).MatchString(result) {
			t.Errorf("path structure not preserved: %s", result)
		}
	})
}

// TestTemplateExpansionWithFileMetadata validates template expansion with time-based placeholders
// For any valid template and modification time, the expanded path SHALL contain
// correctly formatted year, month, and day values extracted from the time.
func TestTemplateExpansionWithFileMetadata(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random time
		year := rapid.IntRange(2000, 2100).Draw(t, "year")
		month := rapid.IntRange(1, 12).Draw(t, "month")
		day := rapid.IntRange(1, 28).Draw(t, "day")
		modTime := time.Date(year, time.Month(month), day, 12, 0, 0, 0, time.UTC)

		// Generate other placeholders
		category := rapid.StringMatching(`[a-z]{3,10}`).Draw(t, "category")
		ext := rapid.StringMatching(`[a-z]{2,4}`).Draw(t, "ext")

		placeholders := map[string]string{
			"category": category,
			"ext":      ext,
		}

		expander := NewExpander(placeholders)

		// Expand template with file metadata
		template := "{year}/{month}/{day}/{category}.{ext}"
		result, err := expander.ExpandPathWithFileMetadata(template, modTime)
		if err != nil {
			t.Fatalf("failed to expand template with metadata: %v", err)
		}

		// Verify no unexpanded placeholders remain
		if regexp.MustCompile(`\{[^}]+\}`).MatchString(result) {
			t.Errorf("unexpanded placeholders remain: %s", result)
		}

		// Verify time values are correctly formatted
		expectedYear := fmt.Sprintf("%04d", year)
		expectedMonth := fmt.Sprintf("%02d", month)
		expectedDay := fmt.Sprintf("%02d", day)

		if !regexp.MustCompile(regexp.QuoteMeta(expectedYear)).MatchString(result) {
			t.Errorf("year not found in result: expected %s in %s", expectedYear, result)
		}
		if !regexp.MustCompile(regexp.QuoteMeta(expectedMonth)).MatchString(result) {
			t.Errorf("month not found in result: expected %s in %s", expectedMonth, result)
		}
		if !regexp.MustCompile(regexp.QuoteMeta(expectedDay)).MatchString(result) {
			t.Errorf("day not found in result: expected %s in %s", expectedDay, result)
		}
	})
}

// TestTemplateValidation validates that template validation correctly identifies unknown placeholders
func TestTemplateValidation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate valid placeholders
		category := rapid.StringMatching(`[a-z]{3,10}`).Draw(t, "category")
		ext := rapid.StringMatching(`[a-z]{2,4}`).Draw(t, "ext")

		placeholders := map[string]string{
			"category": category,
			"ext":      ext,
		}

		expander := NewExpander(placeholders)

		// Test 1: Valid template should pass validation
		validTemplate := "{category}/{ext}"
		err := expander.ValidateTemplate(validTemplate)
		if err != nil {
			t.Errorf("valid template failed validation: %v", err)
		}

		// Test 2: Invalid template should fail validation
		invalidTemplate := "{category}/{unknown}"
		err = expander.ValidateTemplate(invalidTemplate)
		if err == nil {
			t.Error("invalid template passed validation")
		}
	})
}

// TestTemplateExpansionErrorHandling validates error handling for invalid templates
func TestTemplateExpansionErrorHandling(t *testing.T) {
	placeholders := map[string]string{
		"category": "documents",
		"ext":      "pdf",
	}
	expander := NewExpander(placeholders)

	// Test 1: Unknown placeholder should return error
	template := "{category}/{unknown}"
	_, err := expander.ExpandPath(template)
	if err == nil {
		t.Error("expected error for unknown placeholder")
	}

	// Test 2: Empty template should return error
	_, err = expander.ExpandPath("")
	if err == nil {
		t.Error("expected error for empty template")
	}
}
