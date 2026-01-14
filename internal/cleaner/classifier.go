package cleaner

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

// FileImportance represents the importance level of a file
type FileImportance int

const (
	ImportanceNormal FileImportance = iota
	ImportanceImportant
	ImportanceCritical
	ImportanceUncertain
)

// ImportantPattern represents a pattern for important files
type ImportantPattern struct {
	Pattern     string         // Glob or regex pattern
	Type        string         // "name", "path", "extension"
	Importance  FileImportance
	Description string
}

// ClassificationResult represents the result of classifying a file
type ClassificationResult struct {
	Path       string
	Importance FileImportance
	Reason     string
	Patterns   []string // Matched patterns
}

// FileClassifier classifies files by importance
type FileClassifier struct {
	patterns      []*ImportantPattern
	sizeThreshold int64 // Files larger than this are considered important
	recentDays    int   // Files modified within this many days are considered important
}

// NewFileClassifier creates a new file classifier
func NewFileClassifier() *FileClassifier {
	fc := &FileClassifier{
		patterns:      []*ImportantPattern{},
		sizeThreshold: 100 * 1024 * 1024, // 100MB default
		recentDays:    7,                  // 7 days default
	}
	
	// Add default patterns
	for _, pattern := range fc.GetDefaultPatterns() {
		fc.AddPattern(pattern)
	}
	
	return fc
}

// AddPattern adds a custom important file pattern
func (c *FileClassifier) AddPattern(pattern *ImportantPattern) {
	c.patterns = append(c.patterns, pattern)
}

// Classify classifies a file's importance
func (c *FileClassifier) Classify(path string, info os.FileInfo) *ClassificationResult {
	result := &ClassificationResult{
		Path:       path,
		Importance: ImportanceNormal,
		Reason:     "No matching patterns",
		Patterns:   []string{},
	}

	// Check against patterns
	for _, pattern := range c.patterns {
		if c.matchesPattern(path, pattern) {
			result.Importance = pattern.Importance
			result.Reason = pattern.Description
			result.Patterns = append(result.Patterns, pattern.Pattern)
			
			// Critical files take precedence
			if pattern.Importance == ImportanceCritical {
				break
			}
		}
	}

	// Check size threshold (only if not already critical)
	if result.Importance != ImportanceCritical && info.Size() > c.sizeThreshold {
		if result.Importance < ImportanceImportant {
			result.Importance = ImportanceImportant
			result.Reason = "Large file size"
		}
	}

	// Check recent modification (only if not already important/critical)
	if result.Importance == ImportanceNormal {
		recentThreshold := time.Now().AddDate(0, 0, -c.recentDays)
		if info.ModTime().After(recentThreshold) {
			result.Importance = ImportanceUncertain
			result.Reason = "Recently modified file"
		}
	}

	return result
}

// IsImportant returns true if the file is important
func (c *FileClassifier) IsImportant(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	
	result := c.Classify(path, info)
	return result.Importance == ImportanceImportant || result.Importance == ImportanceCritical
}

// IsUncertain returns true if the file's safety is uncertain
func (c *FileClassifier) IsUncertain(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	
	result := c.Classify(path, info)
	return result.Importance == ImportanceUncertain
}

// GetDefaultPatterns returns default important file patterns
func (c *FileClassifier) GetDefaultPatterns() []*ImportantPattern {
	return []*ImportantPattern{
		// Security files
		{Pattern: "*.key", Type: "extension", Importance: ImportanceCritical, Description: "Private key file"},
		{Pattern: "*.pem", Type: "extension", Importance: ImportanceCritical, Description: "Certificate file"},
		{Pattern: "*.p12", Type: "extension", Importance: ImportanceCritical, Description: "PKCS12 certificate"},
		{Pattern: "id_rsa*", Type: "name", Importance: ImportanceCritical, Description: "SSH private key"},
		{Pattern: "id_ed25519*", Type: "name", Importance: ImportanceCritical, Description: "SSH private key"},

		// Configuration files
		{Pattern: ".env*", Type: "name", Importance: ImportanceImportant, Description: "Environment config"},
		{Pattern: "*.credentials", Type: "extension", Importance: ImportanceCritical, Description: "Credentials file"},
		{Pattern: "config.yaml", Type: "name", Importance: ImportanceImportant, Description: "Configuration file"},
		{Pattern: "secrets.*", Type: "name", Importance: ImportanceCritical, Description: "Secrets file"},

		// Important directories
		{Pattern: "*/Documents/*", Type: "path", Importance: ImportanceImportant, Description: "Documents folder"},
		{Pattern: "*/Desktop/*", Type: "path", Importance: ImportanceImportant, Description: "Desktop folder"},
		{Pattern: "*backup*", Type: "path", Importance: ImportanceImportant, Description: "Backup file"},
	}
}

// matchesPattern checks if a path matches a given pattern
func (c *FileClassifier) matchesPattern(path string, pattern *ImportantPattern) bool {
	switch pattern.Type {
	case "extension":
		// Remove the * from pattern like "*.key" to get ".key"
		ext := strings.TrimPrefix(pattern.Pattern, "*")
		return strings.HasSuffix(strings.ToLower(path), strings.ToLower(ext))
		
	case "name":
		filename := filepath.Base(path)
		// Handle patterns like "id_rsa*" or ".env*"
		if strings.HasSuffix(pattern.Pattern, "*") {
			prefix := strings.TrimSuffix(pattern.Pattern, "*")
			return strings.HasPrefix(strings.ToLower(filename), strings.ToLower(prefix))
		}
		// Exact match for patterns like "config.yaml"
		return strings.EqualFold(filename, pattern.Pattern)
		
	case "path":
		// Handle path patterns like "*/Documents/*" or "*backup*"
		normalizedPath := filepath.ToSlash(strings.ToLower(path))
		normalizedPattern := strings.ToLower(pattern.Pattern)
		
		// Simple wildcard matching
		if strings.HasPrefix(normalizedPattern, "*") && strings.HasSuffix(normalizedPattern, "*") {
			// Pattern like "*backup*"
			middle := strings.Trim(normalizedPattern, "*")
			return strings.Contains(normalizedPath, middle)
		} else if strings.HasPrefix(normalizedPattern, "*") {
			// Pattern like "*/Documents/*"
			suffix := strings.TrimPrefix(normalizedPattern, "*")
			return strings.Contains(normalizedPath, suffix)
		} else if strings.HasSuffix(normalizedPattern, "*") {
			// Pattern like "Documents/*"
			prefix := strings.TrimSuffix(normalizedPattern, "*")
			return strings.Contains(normalizedPath, prefix)
		}
		// Exact path match
		return strings.Contains(normalizedPath, normalizedPattern)
		
	default:
		return false
	}
}