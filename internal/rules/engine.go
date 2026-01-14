package rules

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/xuanyiying/cleanup-cli/internal/analyzer"
	"github.com/xuanyiying/cleanup-cli/internal/config"
)

// Engine defines the interface for rule matching and application
type Engine interface {
	LoadRules(rules []*config.Rule) error
	Match(file *analyzer.FileMetadata) []*config.Rule
	Apply(file *analyzer.FileMetadata, rules []*config.Rule) []*config.RuleAction
}

// RuleEngine implements the Engine interface
type RuleEngine struct {
	rules []*config.Rule
}

// NewEngine creates a new rule engine
func NewEngine() *RuleEngine {
	return &RuleEngine{
		rules: make([]*config.Rule, 0),
	}
}

// LoadRules loads rules into the engine
func (re *RuleEngine) LoadRules(rules []*config.Rule) error {
	if rules == nil {
		rules = make([]*config.Rule, 0)
	}
	re.rules = rules
	return nil
}

// Match returns all rules that match the given file, sorted by priority (descending)
func (re *RuleEngine) Match(file *analyzer.FileMetadata) []*config.Rule {
	var matchedRules []*config.Rule

	for _, rule := range re.rules {
		if re.matchesCondition(file, rule.Condition) {
			matchedRules = append(matchedRules, rule)
		}
	}

	// Sort by priority descending (highest priority first)
	for i := 0; i < len(matchedRules); i++ {
		for j := i + 1; j < len(matchedRules); j++ {
			if matchedRules[j].Priority > matchedRules[i].Priority {
				matchedRules[i], matchedRules[j] = matchedRules[j], matchedRules[i]
			}
		}
	}

	return matchedRules
}

// Apply returns the actions for the highest priority matching rule
func (re *RuleEngine) Apply(file *analyzer.FileMetadata, rules []*config.Rule) []*config.RuleAction {
	if len(rules) == 0 {
		return nil
	}

	// Return action from the highest priority rule (first in sorted list)
	if rules[0].Action != nil {
		return []*config.RuleAction{rules[0].Action}
	}

	return nil
}

// matchesCondition checks if a file matches a rule condition
func (re *RuleEngine) matchesCondition(file *analyzer.FileMetadata, condition *config.RuleCondition) bool {
	if condition == nil {
		return false
	}

	switch condition.Type {
	case "extension":
		return re.matchExtension(file, condition)
	case "pattern":
		return re.matchPattern(file, condition)
	case "size":
		return re.matchSize(file, condition)
	case "date":
		return re.matchDate(file, condition)
	case "composite":
		return re.matchComposite(file, condition)
	default:
		return false
	}
}

// matchExtension checks if file extension matches the condition
func (re *RuleEngine) matchExtension(file *analyzer.FileMetadata, condition *config.RuleCondition) bool {
	if condition.Value == nil {
		return false
	}

	valueStr, ok := condition.Value.(string)
	if !ok {
		return false
	}

	// Parse comma-separated extensions
	extensions := strings.Split(valueStr, ",")
	for i := range extensions {
		extensions[i] = strings.TrimSpace(extensions[i])
	}

	fileExt := strings.ToLower(file.Extension)

	switch condition.Operator {
	case "match", "eq":
		for _, ext := range extensions {
			if strings.EqualFold(fileExt, ext) {
				return true
			}
		}
		return false
	case "ne":
		for _, ext := range extensions {
			if strings.EqualFold(fileExt, ext) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

// matchPattern checks if filename matches glob or regex pattern
func (re *RuleEngine) matchPattern(file *analyzer.FileMetadata, condition *config.RuleCondition) bool {
	if condition.Value == nil {
		return false
	}

	patternStr, ok := condition.Value.(string)
	if !ok {
		return false
	}

	filename := filepath.Base(file.Path)

	switch condition.Operator {
	case "glob":
		match, err := filepath.Match(patternStr, filename)
		return err == nil && match
	case "regex":
		re, err := regexp.Compile(patternStr)
		if err != nil {
			return false
		}
		return re.MatchString(filename)
	case "match":
		// Default to glob matching
		match, err := filepath.Match(patternStr, filename)
		return err == nil && match
	default:
		return false
	}
}

// matchSize checks if file size matches the condition
func (re *RuleEngine) matchSize(file *analyzer.FileMetadata, condition *config.RuleCondition) bool {
	if condition.Value == nil {
		return false
	}

	// Value can be int64 or string like "1MB", "100KB"
	var sizeBytes int64

	switch v := condition.Value.(type) {
	case float64:
		sizeBytes = int64(v)
	case int:
		sizeBytes = int64(v)
	case int64:
		sizeBytes = v
	case string:
		var err error
		sizeBytes, err = parseSizeString(v)
		if err != nil {
			return false
		}
	default:
		return false
	}

	switch condition.Operator {
	case "eq":
		return file.Size == sizeBytes
	case "ne":
		return file.Size != sizeBytes
	case "gt":
		return file.Size > sizeBytes
	case "lt":
		return file.Size < sizeBytes
	case "gte":
		return file.Size >= sizeBytes
	case "lte":
		return file.Size <= sizeBytes
	default:
		return false
	}
}

// matchDate checks if file modification date matches the condition
func (re *RuleEngine) matchDate(file *analyzer.FileMetadata, condition *config.RuleCondition) bool {
	if condition.Value == nil {
		return false
	}

	var targetTime time.Time

	switch v := condition.Value.(type) {
	case string:
		// Try to parse as RFC3339 or other common formats
		var err error
		targetTime, err = time.Parse(time.RFC3339, v)
		if err != nil {
			// Try other formats
			targetTime, err = time.Parse("2006-01-02", v)
			if err != nil {
				return false
			}
		}
	case time.Time:
		targetTime = v
	default:
		return false
	}

	switch condition.Operator {
	case "before":
		return file.ModifiedAt.Before(targetTime)
	case "after":
		return file.ModifiedAt.After(targetTime)
	case "eq":
		// Compare dates only (ignore time)
		return file.ModifiedAt.Format("2006-01-02") == targetTime.Format("2006-01-02")
	default:
		return false
	}
}

// matchComposite checks composite conditions (and, or)
func (re *RuleEngine) matchComposite(file *analyzer.FileMetadata, condition *config.RuleCondition) bool {
	if condition.Value == nil {
		return false
	}

	// Value should be a slice of conditions
	conditions, ok := condition.Value.([]interface{})
	if !ok {
		return false
	}

	switch condition.Operator {
	case "and":
		// All conditions must match
		for _, c := range conditions {
			subCondition, ok := c.(*config.RuleCondition)
			if !ok {
				// Try to convert from map
				if m, ok := c.(map[string]interface{}); ok {
					subCondition = re.mapToCondition(m)
				} else {
					return false
				}
			}
			if !re.matchesCondition(file, subCondition) {
				return false
			}
		}
		return true
	case "or":
		// At least one condition must match
		for _, c := range conditions {
			subCondition, ok := c.(*config.RuleCondition)
			if !ok {
				// Try to convert from map
				if m, ok := c.(map[string]interface{}); ok {
					subCondition = re.mapToCondition(m)
				} else {
					continue
				}
			}
			if re.matchesCondition(file, subCondition) {
				return true
			}
		}
		return false
	default:
		return false
	}
}

// mapToCondition converts a map to a RuleCondition
func (re *RuleEngine) mapToCondition(m map[string]interface{}) *config.RuleCondition {
	condition := &config.RuleCondition{}

	if t, ok := m["type"].(string); ok {
		condition.Type = t
	}
	if v, ok := m["value"]; ok {
		condition.Value = v
	}
	if op, ok := m["operator"].(string); ok {
		condition.Operator = op
	}

	return condition
}

// parseSizeString parses size strings like "1MB", "100KB", "1GB"
func parseSizeString(s string) (int64, error) {
	s = strings.TrimSpace(s)
	sUpper := strings.ToUpper(s)

	// Check for suffixes in order of length (longest first to avoid partial matches)
	suffixes := []struct {
		suffix     string
		multiplier int64
	}{
		{"TB", 1024 * 1024 * 1024 * 1024},
		{"GB", 1024 * 1024 * 1024},
		{"MB", 1024 * 1024},
		{"KB", 1024},
		{"B", 1},
	}

	for _, sf := range suffixes {
		if strings.HasSuffix(sUpper, sf.suffix) {
			numStr := strings.TrimSuffix(sUpper, sf.suffix)
			numStr = strings.TrimSpace(numStr)

			var num float64
			_, err := fmt.Sscanf(numStr, "%f", &num)
			if err != nil {
				return 0, err
			}

			return int64(num * float64(sf.multiplier)), nil
		}
	}

	// Try parsing as plain number
	var num int64
	_, err := fmt.Sscanf(s, "%d", &num)
	if err != nil {
		return 0, err
	}

	return num, nil
}
