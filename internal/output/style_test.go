package output

import (
	"strings"
	"testing"

	"pgregory.net/rapid"
)

// Feature: enhanced-output-cleanup, Property 7: Color Fallback
func TestColorFallback(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random text
		text := rapid.String().Draw(t, "text")
		
		// Create styler with color disabled
		styler := NewStyler(false)
		
		// Test all color methods
		redResult := styler.Red(text)
		greenResult := styler.Green(text)
		yellowResult := styler.Yellow(text)
		blueResult := styler.Blue(text)
		boldResult := styler.Bold(text)
		dimResult := styler.Dim(text)
		
		// Generate random style
		style := Style{
			FgColor:   Color(rapid.IntRange(0, 8).Draw(t, "fgColor")),
			BgColor:   Color(rapid.IntRange(0, 8).Draw(t, "bgColor")),
			Bold:      rapid.Bool().Draw(t, "bold"),
			Italic:    rapid.Bool().Draw(t, "italic"),
			Underline: rapid.Bool().Draw(t, "underline"),
		}
		applyResult := styler.Apply(text, style)
		
		// When color is disabled, output should contain no ANSI escape sequences
		if containsANSI(redResult) {
			t.Fatalf("Red() with disabled color contains ANSI codes: %q", redResult)
		}
		if containsANSI(greenResult) {
			t.Fatalf("Green() with disabled color contains ANSI codes: %q", greenResult)
		}
		if containsANSI(yellowResult) {
			t.Fatalf("Yellow() with disabled color contains ANSI codes: %q", yellowResult)
		}
		if containsANSI(blueResult) {
			t.Fatalf("Blue() with disabled color contains ANSI codes: %q", blueResult)
		}
		if containsANSI(boldResult) {
			t.Fatalf("Bold() with disabled color contains ANSI codes: %q", boldResult)
		}
		if containsANSI(dimResult) {
			t.Fatalf("Dim() with disabled color contains ANSI codes: %q", dimResult)
		}
		if containsANSI(applyResult) {
			t.Fatalf("Apply() with disabled color contains ANSI codes: %q", applyResult)
		}
		
		// All results should equal the original text
		if redResult != text {
			t.Fatalf("Red() with disabled color should return original text, got %q, want %q", redResult, text)
		}
		if greenResult != text {
			t.Fatalf("Green() with disabled color should return original text, got %q, want %q", greenResult, text)
		}
		if yellowResult != text {
			t.Fatalf("Yellow() with disabled color should return original text, got %q, want %q", yellowResult, text)
		}
		if blueResult != text {
			t.Fatalf("Blue() with disabled color should return original text, got %q, want %q", blueResult, text)
		}
		if boldResult != text {
			t.Fatalf("Bold() with disabled color should return original text, got %q, want %q", boldResult, text)
		}
		if dimResult != text {
			t.Fatalf("Dim() with disabled color should return original text, got %q, want %q", dimResult, text)
		}
		if applyResult != text {
			t.Fatalf("Apply() with disabled color should return original text, got %q, want %q", applyResult, text)
		}
	})
}

// containsANSI checks if a string contains ANSI escape sequences
func containsANSI(s string) bool {
	return strings.Contains(s, "\x1b[")
}

// Test that enabled styler produces ANSI codes
func TestColorEnabled(t *testing.T) {
	styler := NewStyler(true)
	
	result := styler.Red("test")
	if !containsANSI(result) {
		t.Errorf("Red() with enabled color should contain ANSI codes, got %q", result)
	}
	
	result = styler.Green("test")
	if !containsANSI(result) {
		t.Errorf("Green() with enabled color should contain ANSI codes, got %q", result)
	}
	
	result = styler.Yellow("test")
	if !containsANSI(result) {
		t.Errorf("Yellow() with enabled color should contain ANSI codes, got %q", result)
	}
	
	result = styler.Blue("test")
	if !containsANSI(result) {
		t.Errorf("Blue() with enabled color should contain ANSI codes, got %q", result)
	}
	
	result = styler.Bold("test")
	if !containsANSI(result) {
		t.Errorf("Bold() with enabled color should contain ANSI codes, got %q", result)
	}
	
	result = styler.Dim("test")
	if !containsANSI(result) {
		t.Errorf("Dim() with enabled color should contain ANSI codes, got %q", result)
	}
}