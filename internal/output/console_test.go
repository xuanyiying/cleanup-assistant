package output

import (
	"bytes"
	"strings"
	"testing"

	"pgregory.net/rapid"
)

// Feature: enhanced-output-cleanup, Property 6: Message Type Styling
func TestMessageTypeStyling(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random message without format specifiers to avoid formatting issues
		message := rapid.StringMatching(`[^%]*`).Draw(t, "message")
		
		// Test with color enabled
		var buf bytes.Buffer
		console := NewConsole(&buf)
		console.SetColorEnabled(true)
		
		// Test Success message
		buf.Reset()
		console.Success(message)
		output := buf.String()
		
		// Should contain green color codes and checkmark or [OK]
		if !containsANSI(output) {
			t.Fatalf("Success() with color enabled should contain ANSI codes, got %q", output)
		}
		if !strings.Contains(output, message) {
			t.Fatalf("Success() output should contain the message %q, got %q", message, output)
		}
		// Should contain either ✓ or [OK]
		if !strings.Contains(output, "✓") && !strings.Contains(output, "[OK]") {
			t.Fatalf("Success() output should contain success symbol, got %q", output)
		}
		
		// Test Error message
		buf.Reset()
		console.Error(message)
		output = buf.String()
		
		// Should contain red color codes and error mark or [ERROR]
		if !containsANSI(output) {
			t.Fatalf("Error() with color enabled should contain ANSI codes, got %q", output)
		}
		if !strings.Contains(output, message) {
			t.Fatalf("Error() output should contain the message %q, got %q", message, output)
		}
		// Should contain either ✗ or [ERROR]
		if !strings.Contains(output, "✗") && !strings.Contains(output, "[ERROR]") {
			t.Fatalf("Error() output should contain error symbol, got %q", output)
		}
		
		// Test Warning message
		buf.Reset()
		console.Warning(message)
		output = buf.String()
		
		// Should contain yellow color codes and warning symbol or [WARN]
		if !containsANSI(output) {
			t.Fatalf("Warning() with color enabled should contain ANSI codes, got %q", output)
		}
		if !strings.Contains(output, message) {
			t.Fatalf("Warning() output should contain the message %q, got %q", message, output)
		}
		// Should contain either ⚠ or [WARN]
		if !strings.Contains(output, "⚠") && !strings.Contains(output, "[WARN]") {
			t.Fatalf("Warning() output should contain warning symbol, got %q", output)
		}
		
		// Test with color disabled
		console.SetColorEnabled(false)
		
		// Test Success message without color
		buf.Reset()
		console.Success(message)
		output = buf.String()
		
		// Should not contain ANSI codes
		if containsANSI(output) {
			t.Fatalf("Success() with color disabled should not contain ANSI codes, got %q", output)
		}
		if !strings.Contains(output, message) {
			t.Fatalf("Success() output should contain the message %q, got %q", message, output)
		}
		if !strings.Contains(output, "[OK]") {
			t.Fatalf("Success() without color should contain [OK], got %q", output)
		}
		
		// Test Error message without color
		buf.Reset()
		console.Error(message)
		output = buf.String()
		
		// Should not contain ANSI codes
		if containsANSI(output) {
			t.Fatalf("Error() with color disabled should not contain ANSI codes, got %q", output)
		}
		if !strings.Contains(output, message) {
			t.Fatalf("Error() output should contain the message %q, got %q", message, output)
		}
		if !strings.Contains(output, "[ERROR]") {
			t.Fatalf("Error() without color should contain [ERROR], got %q", output)
		}
		
		// Test Warning message without color
		buf.Reset()
		console.Warning(message)
		output = buf.String()
		
		// Should not contain ANSI codes
		if containsANSI(output) {
			t.Fatalf("Warning() with color disabled should not contain ANSI codes, got %q", output)
		}
		if !strings.Contains(output, message) {
			t.Fatalf("Warning() output should contain the message %q, got %q", message, output)
		}
		if !strings.Contains(output, "[WARN]") {
			t.Fatalf("Warning() without color should contain [WARN], got %q", output)
		}
	})
}

// Test Info message (no special styling)
func TestInfoMessage(t *testing.T) {
	var buf bytes.Buffer
	console := NewConsole(&buf)
	
	message := "test info message"
	console.Info(message)
	output := buf.String()
	
	if !strings.Contains(output, message) {
		t.Errorf("Info() output should contain the message %q, got %q", message, output)
	}
}

// Test Box formatting
func TestBoxFormatting(t *testing.T) {
	var buf bytes.Buffer
	console := NewConsole(&buf)
	
	title := "Test Title"
	content := []string{"Line 1", "Line 2"}
	
	console.Box(title, content)
	output := buf.String()
	
	if !strings.Contains(output, title) {
		t.Errorf("Box() output should contain title %q, got %q", title, output)
	}
	
	for _, line := range content {
		if !strings.Contains(output, line) {
			t.Errorf("Box() output should contain content line %q, got %q", line, output)
		}
	}
	
	// Should contain box drawing characters or ASCII equivalents
	hasBoxChars := strings.Contains(output, "┌") || strings.Contains(output, "+")
	if !hasBoxChars {
		t.Errorf("Box() output should contain box drawing characters, got %q", output)
	}
}

// Test Table formatting
func TestTableFormatting(t *testing.T) {
	var buf bytes.Buffer
	console := NewConsole(&buf)
	
	headers := []string{"Name", "Age"}
	rows := [][]string{
		{"Alice", "25"},
		{"Bob", "30"},
	}
	
	console.Table(headers, rows)
	output := buf.String()
	
	// Should contain headers
	for _, header := range headers {
		if !strings.Contains(output, header) {
			t.Errorf("Table() output should contain header %q, got %q", header, output)
		}
	}
	
	// Should contain row data
	for _, row := range rows {
		for _, cell := range row {
			if !strings.Contains(output, cell) {
				t.Errorf("Table() output should contain cell %q, got %q", cell, output)
			}
		}
	}
	
	// Should contain table drawing characters or ASCII equivalents
	hasTableChars := strings.Contains(output, "┌") || strings.Contains(output, "+")
	if !hasTableChars {
		t.Errorf("Table() output should contain table drawing characters, got %q", output)
	}
}