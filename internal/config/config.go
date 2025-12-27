package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/viper"
)

// CleanupConfig represents the complete configuration for the cleanup CLI
type CleanupConfig struct {
	Ollama             OllamaConfig      `yaml:"ollama" mapstructure:"ollama"`
	Rules              []*Rule           `yaml:"rules" mapstructure:"rules"`
	DefaultStrategy    *OrganizeStrategy `yaml:"defaultStrategy" mapstructure:"defaultStrategy"`
	TransactionLogPath string            `yaml:"transactionLogPath" mapstructure:"transactionLogPath"`
	TrashPath          string            `yaml:"trashPath" mapstructure:"trashPath"`
	Exclude            *ExcludeConfig    `yaml:"exclude" mapstructure:"exclude"`
}

// ExcludeConfig represents files and directories to exclude from scanning
type ExcludeConfig struct {
	Extensions []string `yaml:"extensions" mapstructure:"extensions"` // 要排除的文件扩展名
	Patterns   []string `yaml:"patterns" mapstructure:"patterns"`     // 要排除的文件名模式
	Dirs       []string `yaml:"dirs" mapstructure:"dirs"`             // 要排除的目录名
}

// OllamaConfig represents Ollama service configuration
type OllamaConfig struct {
	BaseURL string        `yaml:"baseUrl" mapstructure:"baseUrl"`
	Model   string        `yaml:"model" mapstructure:"model"`
	Timeout time.Duration `yaml:"timeout" mapstructure:"timeout"`
}

// Rule represents a file organization rule
type Rule struct {
	Name      string         `yaml:"name" mapstructure:"name"`
	Priority  int            `yaml:"priority" mapstructure:"priority"`
	Condition *RuleCondition `yaml:"condition" mapstructure:"condition"`
	Action    *RuleAction    `yaml:"action" mapstructure:"action"`
}

// RuleCondition represents a condition for rule matching
type RuleCondition struct {
	Type     string      `yaml:"type" mapstructure:"type"`
	Value    interface{} `yaml:"value" mapstructure:"value"`
	Operator string      `yaml:"operator" mapstructure:"operator"`
}

// RuleAction represents an action to take when a rule matches
type RuleAction struct {
	Type     string `yaml:"type" mapstructure:"type"`
	Target   string `yaml:"target" mapstructure:"target"`
	Template string `yaml:"template" mapstructure:"template"`
}

// OrganizeStrategy represents the strategy for organizing files
type OrganizeStrategy struct {
	UseAI            bool   `yaml:"useAI" mapstructure:"useAI"`
	CreateFolders    bool   `yaml:"createFolders" mapstructure:"createFolders"`
	ConflictStrategy string `yaml:"conflictStrategy" mapstructure:"conflictStrategy"`
}

// Manager handles configuration loading and saving
type Manager struct {
	v    *viper.Viper
	path string
}

// NewManager creates a new configuration manager
func NewManager(configPath string) *Manager {
	return &Manager{
		v:    viper.New(),
		path: configPath,
	}
}

// Load loads configuration from file or returns defaults
func (m *Manager) Load() (*CleanupConfig, error) {
	// Set default values
	m.setDefaults()

	// If config file exists, load it
	if _, err := os.Stat(m.path); err == nil {
		m.v.SetConfigFile(m.path)
		m.v.SetConfigType("yaml")

		if err := m.v.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	// Unmarshal into struct
	var config CleanupConfig
	if err := m.v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &config, nil
}

// Save saves configuration to file
func (m *Manager) Save(config *CleanupConfig) error {
	// Ensure directory exists
	dir := filepath.Dir(m.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Marshal config to viper
	m.v.Set("ollama", config.Ollama)
	m.v.Set("rules", config.Rules)
	m.v.Set("defaultStrategy", config.DefaultStrategy)
	m.v.Set("transactionLogPath", config.TransactionLogPath)
	m.v.Set("trashPath", config.TrashPath)

	// Write to file
	if err := m.v.WriteConfigAs(m.path); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// Get retrieves a configuration value by key
func (m *Manager) Get(key string) interface{} {
	return m.v.Get(key)
}

// Set sets a configuration value by key
func (m *Manager) Set(key string, value interface{}) {
	m.v.Set(key, value)
}

// setDefaults sets default configuration values
func (m *Manager) setDefaults() {
	homeDir, _ := os.UserHomeDir()

	m.v.SetDefault("ollama.baseUrl", "http://localhost:11434")
	m.v.SetDefault("ollama.model", "llama3.2")
	m.v.SetDefault("ollama.timeout", 30*time.Second)

	m.v.SetDefault("defaultStrategy.useAI", true)
	m.v.SetDefault("defaultStrategy.createFolders", true)
	m.v.SetDefault("defaultStrategy.conflictStrategy", "suffix")

	m.v.SetDefault("transactionLogPath", filepath.Join(homeDir, ".cleanup", "transactions.json"))
	m.v.SetDefault("trashPath", filepath.Join(homeDir, ".cleanup", "trash"))

	m.v.SetDefault("rules", []*Rule{})
	
	// 默认排除常见的系统和版本控制文件
	m.v.SetDefault("exclude.extensions", []string{})
	m.v.SetDefault("exclude.patterns", []string{".DS_Store", "Thumbs.db", "desktop.ini"})
	m.v.SetDefault("exclude.dirs", []string{".git", ".svn", "node_modules", "__pycache__"})
}
