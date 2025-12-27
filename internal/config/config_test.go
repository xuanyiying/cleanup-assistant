package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"pgregory.net/rapid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Property 8: Rule Configuration Round-Trip
// For any valid rule configuration, saving to file and loading back SHALL produce an equivalent rule set.
// Validates: Requirements 6.1
func TestConfigurationRoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random configuration
		config := generateRandomConfig(t)

		// Create temporary directory for config file
		tmpDir := os.TempDir()
		configPath := filepath.Join(tmpDir, "cleanup_test.yaml")
		defer os.Remove(configPath)

		// Save configuration
		manager := NewManager(configPath)
		err := manager.Save(config)
		require.NoError(t, err)

		// Load configuration back
		loadedManager := NewManager(configPath)
		loadedConfig, err := loadedManager.Load()
		require.NoError(t, err)

		// Verify round-trip equivalence
		assert.Equal(t, config.Ollama.BaseURL, loadedConfig.Ollama.BaseURL)
		assert.Equal(t, config.Ollama.Model, loadedConfig.Ollama.Model)
		assert.Equal(t, config.Ollama.Timeout, loadedConfig.Ollama.Timeout)

		assert.Equal(t, config.DefaultStrategy.UseAI, loadedConfig.DefaultStrategy.UseAI)
		assert.Equal(t, config.DefaultStrategy.CreateFolders, loadedConfig.DefaultStrategy.CreateFolders)
		assert.Equal(t, config.DefaultStrategy.ConflictStrategy, loadedConfig.DefaultStrategy.ConflictStrategy)

		assert.Equal(t, config.TransactionLogPath, loadedConfig.TransactionLogPath)
		assert.Equal(t, config.TrashPath, loadedConfig.TrashPath)

		assert.Equal(t, len(config.Rules), len(loadedConfig.Rules))
		for i, rule := range config.Rules {
			assert.Equal(t, rule.Name, loadedConfig.Rules[i].Name)
			assert.Equal(t, rule.Priority, loadedConfig.Rules[i].Priority)
			assert.Equal(t, rule.Condition.Type, loadedConfig.Rules[i].Condition.Type)
			assert.Equal(t, rule.Condition.Operator, loadedConfig.Rules[i].Condition.Operator)
			assert.Equal(t, rule.Action.Type, loadedConfig.Rules[i].Action.Type)
			assert.Equal(t, rule.Action.Target, loadedConfig.Rules[i].Action.Target)
			assert.Equal(t, rule.Action.Template, loadedConfig.Rules[i].Action.Template)
		}
	})
}

// Test that default configuration is applied when no config file exists
func TestDefaultConfiguration(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "nonexistent.yaml")

	manager := NewManager(configPath)
	config, err := manager.Load()

	require.NoError(t, err)
	assert.Equal(t, "http://localhost:11434", config.Ollama.BaseURL)
	assert.Equal(t, "llama3.2", config.Ollama.Model)
	assert.Equal(t, 30*time.Second, config.Ollama.Timeout)
	assert.True(t, config.DefaultStrategy.UseAI)
	assert.True(t, config.DefaultStrategy.CreateFolders)
	assert.Equal(t, "suffix", config.DefaultStrategy.ConflictStrategy)
}

// Test that configuration can be saved and loaded multiple times
func TestConfigurationPersistence(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "cleanup.yaml")

	// Create initial configuration
	config := &CleanupConfig{
		Ollama: OllamaConfig{
			BaseURL: "http://localhost:11434",
			Model:   "llama3.2",
			Timeout: 30 * time.Second,
		},
		DefaultStrategy: &OrganizeStrategy{
			UseAI:            true,
			CreateFolders:    true,
			ConflictStrategy: "suffix",
		},
		TransactionLogPath: "/tmp/transactions.json",
		TrashPath:          "/tmp/trash",
		Rules: []*Rule{
			{
				Name:     "test-rule",
				Priority: 10,
				Condition: &RuleCondition{
					Type:     "extension",
					Value:    "jpg,png",
					Operator: "match",
				},
				Action: &RuleAction{
					Type:     "move",
					Target:   "Pictures",
					Template: "",
				},
			},
		},
	}

	// Save configuration
	manager := NewManager(configPath)
	err := manager.Save(config)
	require.NoError(t, err)

	// Verify file was created
	_, err = os.Stat(configPath)
	require.NoError(t, err)

	// Load and verify
	loadedConfig, err := manager.Load()
	require.NoError(t, err)
	assert.Equal(t, 1, len(loadedConfig.Rules))
	assert.Equal(t, "test-rule", loadedConfig.Rules[0].Name)
}

// Helper function to generate random configuration
func generateRandomConfig(t *rapid.T) *CleanupConfig {
	baseURL := rapid.StringMatching(`http://localhost:\d{4,5}`).Draw(t, "baseURL")
	model := rapid.StringMatching(`[a-z0-9\-\.]+`).Draw(t, "model")
	timeout := time.Duration(rapid.IntRange(5, 120).Draw(t, "timeout")) * time.Second

	useAI := rapid.Bool().Draw(t, "useAI")
	createFolders := rapid.Bool().Draw(t, "createFolders")
	conflictStrategy := rapid.SampledFrom([]string{"skip", "suffix", "overwrite", "prompt"}).Draw(t, "conflictStrategy")

	transactionLogPath := rapid.StringMatching(`/tmp/[a-z0-9_]+\.json`).Draw(t, "transactionLogPath")
	trashPath := rapid.StringMatching(`/tmp/[a-z0-9_]+`).Draw(t, "trashPath")

	// Generate random rules
	numRules := rapid.IntRange(0, 3).Draw(t, "numRules")
	rules := make([]*Rule, numRules)
	for i := 0; i < numRules; i++ {
		ruleName := rapid.StringMatching(`[a-z0-9\-]+`).Draw(t, "ruleName")
		priority := rapid.IntRange(1, 100).Draw(t, "priority")

		rules[i] = &Rule{
			Name:     ruleName,
			Priority: priority,
			Condition: &RuleCondition{
				Type:     "extension",
				Value:    "jpg,png,gif",
				Operator: "match",
			},
			Action: &RuleAction{
				Type:     "move",
				Target:   "Pictures",
				Template: "",
			},
		}
	}

	return &CleanupConfig{
		Ollama: OllamaConfig{
			BaseURL: baseURL,
			Model:   model,
			Timeout: timeout,
		},
		DefaultStrategy: &OrganizeStrategy{
			UseAI:            useAI,
			CreateFolders:    createFolders,
			ConflictStrategy: conflictStrategy,
		},
		TransactionLogPath: transactionLogPath,
		TrashPath:          trashPath,
		Rules:              rules,
	}
}
