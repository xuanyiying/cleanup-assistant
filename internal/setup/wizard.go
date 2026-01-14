package setup

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/xuanyiying/cleanup-cli/internal/config"
)

// RunSetup runs the interactive setup wizard
func RunSetup(mgr *config.Manager) error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("===========================================")
	fmt.Println("   Welcome to Cleanup Assistant Setup")
	fmt.Println("===========================================")
	fmt.Println("It looks like this is your first run.")
	fmt.Println("Let's configure the AI model settings.")
	fmt.Println()

	// Load default config
	cfg, _ := mgr.Load() // Loads defaults since file doesn't exist

	// 1. Select Provider
	provider := prompt(reader, "Select AI Provider (ollama/openai)", cfg.AI.Provider)
	for provider != "ollama" && provider != "openai" {
		fmt.Println("Invalid provider. Please choose 'ollama' or 'openai'.")
		provider = prompt(reader, "Select AI Provider (ollama/openai)", cfg.AI.Provider)
	}
	cfg.AI.Provider = provider

	if provider == "ollama" {
		configureOllama(reader, &cfg.Ollama)
	} else {
		configureOpenAI(reader, &cfg.AI.OpenAI)
	}

	// Confirm settings
	fmt.Println("\nConfiguration Summary:")
	fmt.Println("----------------------")
	fmt.Printf("Provider: %s\n", cfg.AI.Provider)
	if cfg.AI.Provider == "ollama" {
		fmt.Printf("Base URL: %s\n", cfg.Ollama.BaseURL)
		fmt.Printf("Model: %s\n", cfg.Ollama.Model)
		if cfg.Ollama.ModelPath != "" {
			fmt.Printf("Model Path: %s\n", cfg.Ollama.ModelPath)
		}
		if len(cfg.Ollama.Params) > 0 {
			fmt.Printf("Params: %v\n", cfg.Ollama.Params)
		}
	} else {
		fmt.Printf("Base URL: %s\n", cfg.AI.OpenAI.BaseURL)
		fmt.Printf("Model: %s\n", cfg.AI.OpenAI.Model)
		fmt.Printf("API Key: %s\n", maskString(cfg.AI.OpenAI.APIKey))
	}
	fmt.Println("----------------------")

	confirm := prompt(reader, "Save configuration? (y/n)", "y")
	if strings.ToLower(confirm) != "y" {
		return fmt.Errorf("setup cancelled by user")
	}

	// Save config
	if err := mgr.Save(cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Println("\nConfiguration saved successfully!")
	fmt.Println("You can change these settings later in ~/.cleanuprc.yaml")
	fmt.Println()
	return nil
}

func configureOllama(reader *bufio.Reader, cfg *config.OllamaConfig) {
	fmt.Println("\n--- Ollama Configuration ---")
	cfg.BaseURL = prompt(reader, "Ollama Base URL", cfg.BaseURL)
	cfg.Model = prompt(reader, "Model Name", cfg.Model)
	cfg.ModelPath = prompt(reader, "Model Storage Path (optional)", cfg.ModelPath)

	// Interactive params
	tempStr := prompt(reader, "Temperature (0.0-1.0, optional)", "")
	if tempStr != "" {
		if val, err := strconv.ParseFloat(tempStr, 64); err == nil {
			if cfg.Params == nil {
				cfg.Params = make(map[string]interface{})
			}
			cfg.Params["temperature"] = val
		} else {
			fmt.Println("Invalid temperature, ignoring.")
		}
	}

	// Timeout
	timeoutStr := prompt(reader, "Timeout (seconds)", fmt.Sprintf("%.0f", cfg.Timeout.Seconds()))
	if val, err := strconv.Atoi(timeoutStr); err == nil {
		cfg.Timeout = time.Duration(val) * time.Second
	}
}

func configureOpenAI(reader *bufio.Reader, cfg *config.OpenAIConfig) {
	fmt.Println("\n--- OpenAI Configuration ---")
	cfg.BaseURL = prompt(reader, "API Base URL", cfg.BaseURL)
	cfg.APIKey = prompt(reader, "API Key", "") // No default for security
	cfg.Model = prompt(reader, "Model Name", cfg.Model)

	// Timeout
	timeoutStr := prompt(reader, "Timeout (seconds)", fmt.Sprintf("%.0f", cfg.Timeout.Seconds()))
	if val, err := strconv.Atoi(timeoutStr); err == nil {
		cfg.Timeout = time.Duration(val) * time.Second
	}
}

func prompt(reader *bufio.Reader, label string, defaultValue string) string {
	if defaultValue != "" {
		fmt.Printf("%s [%s]: ", label, defaultValue)
	} else {
		fmt.Printf("%s: ", label)
	}

	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if input == "" {
		return defaultValue
	}
	return input
}

func maskString(s string) string {
	if len(s) <= 4 {
		return "****"
	}
	return s[:2] + "****" + s[len(s)-2:]
}
