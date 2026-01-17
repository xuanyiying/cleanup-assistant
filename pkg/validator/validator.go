package validator

import (
	"fmt"
	"path/filepath"
	"strings"
)

// ValidateFilename checks if a filename is valid
func ValidateFilename(name string) error {
	if name == "" {
		return fmt.Errorf("filename cannot be empty")
	}

	// Check for illegal characters
	invalidChars := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|", "\x00"}
	for _, char := range invalidChars {
		if strings.Contains(name, char) {
			return fmt.Errorf("filename contains invalid character: %s", char)
		}
	}

	// Check for reserved names on Windows
	reservedNames := []string{
		"CON", "PRN", "AUX", "NUL",
		"COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9",
		"LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9",
	}
	nameUpper := strings.ToUpper(strings.TrimSuffix(name, filepath.Ext(name)))
	for _, reserved := range reservedNames {
		if nameUpper == reserved {
			return fmt.Errorf("filename is a reserved name: %s", name)
		}
	}

	// Check for names that are just dots
	if strings.Trim(name, ".") == "" {
		return fmt.Errorf("filename cannot consist only of dots")
	}

	// Check length (most filesystems support up to 255 bytes)
	if len(name) > 255 {
		return fmt.Errorf("filename is too long (max 255 characters)")
	}

	return nil
}

// ValidatePath checks if a path is valid and safe
func ValidatePath(path string) error {
	if path == "" {
		return fmt.Errorf("path cannot be empty")
	}

	// Clean the path
	cleanPath := filepath.Clean(path)

	// Check for path traversal attempts
	if strings.Contains(cleanPath, "..") {
		return fmt.Errorf("path contains parent directory references")
	}

	// Check for absolute path on Unix (starting with /)
	// or Windows (starting with drive letter)
	if !filepath.IsAbs(cleanPath) {
		// Relative paths are okay
		return nil
	}

	return nil
}

// SanitizeFilename removes or replaces invalid characters from a filename
func SanitizeFilename(name string) string {
	// First trim leading/trailing spaces and dots
	sanitized := strings.Trim(name, " .")

	// Replace invalid characters with underscore
	invalidChars := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|", "\x00"}
	for _, char := range invalidChars {
		sanitized = strings.ReplaceAll(sanitized, char, "_")
	}

	// Trim again after replacement
	sanitized = strings.Trim(sanitized, " ._")

	// If empty after sanitization, use a default name
	if sanitized == "" {
		return "unnamed"
	}

	// Truncate if too long
	if len(sanitized) > 255 {
		ext := filepath.Ext(sanitized)
		baseName := strings.TrimSuffix(sanitized, ext)
		maxBase := 255 - len(ext)
		if maxBase > 0 {
			sanitized = baseName[:maxBase] + ext
		} else {
			sanitized = sanitized[:255]
		}
	}

	return sanitized
}
