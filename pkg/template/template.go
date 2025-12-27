package template

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// Expander handles template expansion with placeholder substitution
type Expander struct {
	placeholders map[string]string
}

// NewExpander creates a new template expander with the given placeholders
func NewExpander(placeholders map[string]string) *Expander {
	return &Expander{
		placeholders: placeholders,
	}
}

// ExpandPath expands a template string by replacing placeholders with their values
// Supported placeholders: {year}, {month}, {day}, {category}, {ext}
// Returns the expanded path with all placeholders replaced
func (e *Expander) ExpandPath(template string) (string, error) {
	if template == "" {
		return "", fmt.Errorf("template cannot be empty")
	}

	result := template

	// Find all placeholders in the template
	placeholderRegex := regexp.MustCompile(`\{([^}]+)\}`)
	matches := placeholderRegex.FindAllStringSubmatch(result, -1)

	// Replace each placeholder with its value
	for _, match := range matches {
		placeholder := match[0]      // e.g., "{year}"
		key := match[1]              // e.g., "year"
		value, exists := e.placeholders[key]

		if !exists {
			return "", fmt.Errorf("unknown placeholder: %s", placeholder)
		}

		result = strings.ReplaceAll(result, placeholder, value)
	}

	// Verify no unexpanded placeholders remain
	if placeholderRegex.MatchString(result) {
		return "", fmt.Errorf("unexpanded placeholders remain in result")
	}

	return result, nil
}

// ExpandPathWithFileMetadata expands a template using file metadata
// Automatically extracts year, month, day from the provided time
// Requires: category and ext to be provided in the placeholders map
func (e *Expander) ExpandPathWithFileMetadata(template string, modTime time.Time) (string, error) {
	// Create a copy of placeholders and add time-based values
	expandedPlaceholders := make(map[string]string)
	for k, v := range e.placeholders {
		expandedPlaceholders[k] = v
	}

	// Add time-based placeholders if not already present
	if _, exists := expandedPlaceholders["year"]; !exists {
		expandedPlaceholders["year"] = fmt.Sprintf("%04d", modTime.Year())
	}
	if _, exists := expandedPlaceholders["month"]; !exists {
		expandedPlaceholders["month"] = fmt.Sprintf("%02d", modTime.Month())
	}
	if _, exists := expandedPlaceholders["day"]; !exists {
		expandedPlaceholders["day"] = fmt.Sprintf("%02d", modTime.Day())
	}

	// Create a new expander with the expanded placeholders
	tempExpander := NewExpander(expandedPlaceholders)
	return tempExpander.ExpandPath(template)
}

// ValidateTemplate checks if a template string contains only valid placeholders
// Returns an error if any unknown placeholders are found
func (e *Expander) ValidateTemplate(template string) error {
	placeholderRegex := regexp.MustCompile(`\{([^}]+)\}`)
	matches := placeholderRegex.FindAllStringSubmatch(template, -1)

	for _, match := range matches {
		key := match[1]
		if _, exists := e.placeholders[key]; !exists {
			return fmt.Errorf("unknown placeholder: {%s}", key)
		}
	}

	return nil
}
