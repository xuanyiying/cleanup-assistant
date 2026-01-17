package validator

import (
	"strings"
	"testing"
)

func TestValidateFilename(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		wantErr  bool
	}{
		{"valid filename", "document.txt", false},
		{"valid with spaces", "my document.txt", false},
		{"valid with dash", "my-document.txt", false},
		{"valid with underscore", "my_document.txt", false},
		{"empty filename", "", true},
		{"slash", "my/document.txt", true},
		{"backslash", "my\\document.txt", true},
		{"colon", "my:document.txt", true},
		{"asterisk", "my*document.txt", true},
		{"question mark", "my?document.txt", true},
		{"quote", "my\"document.txt", true},
		{"less than", "my<document.txt", true},
		{"greater than", "my>document.txt", true},
		{"pipe", "my|document.txt", true},
		{"null byte", "my\x00document.txt", true},
		{"reserved name CON", "CON.txt", true},
		{"reserved name PRN", "PRN", true},
		{"reserved name COM1", "COM1.doc", true},
		{"only dots", "...", true},
		{"too long", strings.Repeat("a", 256) + ".txt", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFilename(tt.filename)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateFilename() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidatePath(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{"valid relative path", "documents/file.txt", false},
		{"valid absolute path", "/home/user/documents", false},
		{"empty path", "", true},
		{"parent reference", "../etc/passwd", true},
		{"hidden parent reference", "documents/../../etc", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePath() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     string
	}{
		{"valid filename", "document.txt", "document.txt"},
		{"with slash", "my/document.txt", "my_document.txt"},
		{"with backslash", "my\\document.txt", "my_document.txt"},
		{"with multiple invalid", "my:doc*ument?.txt", "my_doc_ument_.txt"},
		{"leading spaces", "  document.txt", "document.txt"},
		{"trailing dots", "document.txt...", "document.txt"},
		{"empty after sanitization", "///", "unnamed"},
		{"too long", strings.Repeat("a", 300) + ".txt", strings.Repeat("a", 251) + ".txt"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeFilename(tt.filename)
			if got != tt.want {
				t.Errorf("SanitizeFilename() = %v, want %v", got, tt.want)
			}
		})
	}
}
