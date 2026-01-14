package rules

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/xuanyiying/cleanup-cli/internal/analyzer"
	"github.com/xuanyiying/cleanup-cli/internal/config"
	"pgregory.net/rapid"
)

// TestRuleMatchingAndPriority tests that rules are matched and applied in priority order
// Feature: cleanup-cli, Property 7: Rule Matching and Priority
// Validates: Requirements 6.2, 6.3, 6.4
func TestRuleMatchingAndPriority(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random file metadata
		file := &analyzer.FileMetadata{
			Path:       "/test/file.txt",
			Name:       "file.txt",
			Extension:  "txt",
			Size:       1024,
			MimeType:   "text/plain",
			ModifiedAt: time.Now(),
		}

		// Generate random priorities for rules
		priority1 := rapid.IntRange(1, 100).Draw(t, "priority1")
		priority2 := rapid.IntRange(1, 100).Draw(t, "priority2")
		priority3 := rapid.IntRange(1, 100).Draw(t, "priority3")

		// Create rules with different priorities
		rule1 := &config.Rule{
			Name:     "rule1",
			Priority: priority1,
			Condition: &config.RuleCondition{
				Type:     "extension",
				Value:    "txt",
				Operator: "match",
			},
			Action: &config.RuleAction{
				Type:   "move",
				Target: "Documents",
			},
		}

		rule2 := &config.Rule{
			Name:     "rule2",
			Priority: priority2,
			Condition: &config.RuleCondition{
				Type:     "extension",
				Value:    "txt",
				Operator: "match",
			},
			Action: &config.RuleAction{
				Type:   "move",
				Target: "Archive",
			},
		}

		rule3 := &config.Rule{
			Name:     "rule3",
			Priority: priority3,
			Condition: &config.RuleCondition{
				Type:     "extension",
				Value:    "txt",
				Operator: "match",
			},
			Action: &config.RuleAction{
				Type:   "move",
				Target: "Backup",
			},
		}

		engine := NewEngine()
		engine.LoadRules([]*config.Rule{rule1, rule2, rule3})

		// Match the file
		matchedRules := engine.Match(file)

		// All three rules should match
		assert.Equal(t, 3, len(matchedRules), "all three rules should match")

		// Rules should be sorted by priority descending
		for i := 0; i < len(matchedRules)-1; i++ {
			assert.GreaterOrEqual(t, matchedRules[i].Priority, matchedRules[i+1].Priority,
				"rules should be sorted by priority descending")
		}

		// Apply should return action from highest priority rule
		actions := engine.Apply(file, matchedRules)
		assert.Equal(t, 1, len(actions), "should return exactly one action")

		// The action should be from the highest priority rule
		highestPriority := matchedRules[0].Priority
		expectedTarget := matchedRules[0].Action.Target
		assert.Equal(t, expectedTarget, actions[0].Target,
			"action should be from highest priority rule")

		// Verify the highest priority rule is indeed the one with max priority
		maxPriority := priority1
		if priority2 > maxPriority {
			maxPriority = priority2
		}
		if priority3 > maxPriority {
			maxPriority = priority3
		}
		assert.Equal(t, maxPriority, highestPriority,
			"highest priority in matched rules should equal max priority")
	})
}

// TestExtensionMatching tests extension-based rule matching
func TestExtensionMatching(t *testing.T) {
	tests := []struct {
		name      string
		extension string
		value     string
		operator  string
		expected  bool
	}{
		{
			name:      "single extension match",
			extension: "txt",
			value:     "txt",
			operator:  "match",
			expected:  true,
		},
		{
			name:      "multiple extensions match",
			extension: "pdf",
			value:     "txt,pdf,doc",
			operator:  "match",
			expected:  true,
		},
		{
			name:      "extension no match",
			extension: "jpg",
			value:     "txt,pdf,doc",
			operator:  "match",
			expected:  false,
		},
		{
			name:      "extension not equal",
			extension: "jpg",
			value:     "txt",
			operator:  "ne",
			expected:  true,
		},
		{
			name:      "case insensitive match",
			extension: "TXT",
			value:     "txt",
			operator:  "match",
			expected:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file := &analyzer.FileMetadata{
				Path:      "/test/file",
				Extension: tt.extension,
			}

			condition := &config.RuleCondition{
				Type:     "extension",
				Value:    tt.value,
				Operator: tt.operator,
			}

			engine := NewEngine()
			result := engine.matchExtension(file, condition)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestPatternMatching tests glob and regex pattern matching
func TestPatternMatching(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		pattern  string
		operator string
		expected bool
	}{
		{
			name:     "glob pattern match",
			filename: "document.txt",
			pattern:  "*.txt",
			operator: "glob",
			expected: true,
		},
		{
			name:     "glob pattern no match",
			filename: "image.jpg",
			pattern:  "*.txt",
			operator: "glob",
			expected: false,
		},
		{
			name:     "regex pattern match",
			filename: "file123.txt",
			pattern:  `file\d+\.txt`,
			operator: "regex",
			expected: true,
		},
		{
			name:     "regex pattern no match",
			filename: "file.txt",
			pattern:  `file\d+\.txt`,
			operator: "regex",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file := &analyzer.FileMetadata{
				Path: "/test/" + tt.filename,
			}

			condition := &config.RuleCondition{
				Type:     "pattern",
				Value:    tt.pattern,
				Operator: tt.operator,
			}

			engine := NewEngine()
			result := engine.matchPattern(file, condition)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestSizeMatching tests file size-based rule matching
func TestSizeMatching(t *testing.T) {
	tests := []struct {
		name     string
		fileSize int64
		value    interface{}
		operator string
		expected bool
	}{
		{
			name:     "size equal",
			fileSize: 1024,
			value:    int64(1024),
			operator: "eq",
			expected: true,
		},
		{
			name:     "size greater than",
			fileSize: 2048,
			value:    int64(1024),
			operator: "gt",
			expected: true,
		},
		{
			name:     "size less than",
			fileSize: 512,
			value:    int64(1024),
			operator: "lt",
			expected: true,
		},
		{
			name:     "size string MB",
			fileSize: 1024 * 1024,
			value:    "1MB",
			operator: "eq",
			expected: true,
		},
		{
			name:     "size string KB",
			fileSize: 512 * 1024,
			value:    "512KB",
			operator: "eq",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file := &analyzer.FileMetadata{
				Path: "/test/file",
				Size: tt.fileSize,
			}

			condition := &config.RuleCondition{
				Type:     "size",
				Value:    tt.value,
				Operator: tt.operator,
			}

			engine := NewEngine()
			result := engine.matchSize(file, condition)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestDateMatching tests file date-based rule matching
func TestDateMatching(t *testing.T) {
	now := time.Now()
	yesterday := now.AddDate(0, 0, -1)
	tomorrow := now.AddDate(0, 0, 1)

	tests := []struct {
		name     string
		modTime  time.Time
		value    string
		operator string
		expected bool
	}{
		{
			name:     "date before",
			modTime:  yesterday,
			value:    now.Format(time.RFC3339),
			operator: "before",
			expected: true,
		},
		{
			name:     "date after",
			modTime:  tomorrow,
			value:    now.Format(time.RFC3339),
			operator: "after",
			expected: true,
		},
		{
			name:     "date equal",
			modTime:  now,
			value:    now.Format("2006-01-02"),
			operator: "eq",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file := &analyzer.FileMetadata{
				Path:       "/test/file",
				ModifiedAt: tt.modTime,
			}

			condition := &config.RuleCondition{
				Type:     "date",
				Value:    tt.value,
				Operator: tt.operator,
			}

			engine := NewEngine()
			result := engine.matchDate(file, condition)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestCompositeConditions tests AND/OR composite conditions
func TestCompositeConditions(t *testing.T) {
	file := &analyzer.FileMetadata{
		Path:      "/test/document.txt",
		Extension: "txt",
		Size:      1024,
	}

	t.Run("composite AND all match", func(t *testing.T) {
		conditions := []interface{}{
			&config.RuleCondition{
				Type:     "extension",
				Value:    "txt",
				Operator: "match",
			},
			&config.RuleCondition{
				Type:     "size",
				Value:    int64(1024),
				Operator: "eq",
			},
		}

		condition := &config.RuleCondition{
			Type:     "composite",
			Value:    conditions,
			Operator: "and",
		}

		engine := NewEngine()
		result := engine.matchComposite(file, condition)
		assert.True(t, result)
	})

	t.Run("composite AND one fails", func(t *testing.T) {
		conditions := []interface{}{
			&config.RuleCondition{
				Type:     "extension",
				Value:    "txt",
				Operator: "match",
			},
			&config.RuleCondition{
				Type:     "size",
				Value:    int64(2048),
				Operator: "eq",
			},
		}

		condition := &config.RuleCondition{
			Type:     "composite",
			Value:    conditions,
			Operator: "and",
		}

		engine := NewEngine()
		result := engine.matchComposite(file, condition)
		assert.False(t, result)
	})

	t.Run("composite OR one matches", func(t *testing.T) {
		conditions := []interface{}{
			&config.RuleCondition{
				Type:     "extension",
				Value:    "jpg",
				Operator: "match",
			},
			&config.RuleCondition{
				Type:     "extension",
				Value:    "txt",
				Operator: "match",
			},
		}

		condition := &config.RuleCondition{
			Type:     "composite",
			Value:    conditions,
			Operator: "or",
		}

		engine := NewEngine()
		result := engine.matchComposite(file, condition)
		assert.True(t, result)
	})
}

// TestParseSizeString tests parsing of size strings
func TestParseSizeString(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
		wantErr  bool
	}{
		{"1024", 1024, false},
		{"1KB", 1024, false},
		{"1MB", 1024 * 1024, false},
		{"1GB", 1024 * 1024 * 1024, false},
		{"512KB", 512 * 1024, false},
		{"2.5MB", int64(2.5 * 1024 * 1024), false},
		{"invalid", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := parseSizeString(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

// TestNoMatchingRules tests behavior when no rules match
func TestNoMatchingRules(t *testing.T) {
	file := &analyzer.FileMetadata{
		Path:      "/test/image.jpg",
		Extension: "jpg",
	}

	rule := &config.Rule{
		Name:     "text-only",
		Priority: 10,
		Condition: &config.RuleCondition{
			Type:     "extension",
			Value:    "txt",
			Operator: "match",
		},
		Action: &config.RuleAction{
			Type:   "move",
			Target: "Documents",
		},
	}

	engine := NewEngine()
	engine.LoadRules([]*config.Rule{rule})

	matchedRules := engine.Match(file)
	assert.Equal(t, 0, len(matchedRules))

	actions := engine.Apply(file, matchedRules)
	assert.Nil(t, actions)
}

// TestEmptyRules tests behavior with no rules loaded
func TestEmptyRules(t *testing.T) {
	file := &analyzer.FileMetadata{
		Path: "/test/file.txt",
	}

	engine := NewEngine()
	engine.LoadRules([]*config.Rule{})

	matchedRules := engine.Match(file)
	assert.Equal(t, 0, len(matchedRules))

	actions := engine.Apply(file, matchedRules)
	assert.Nil(t, actions)
}
