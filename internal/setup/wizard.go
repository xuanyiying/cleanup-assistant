package setup

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
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
	
	// Base URL with validation
	for {
		cfg.BaseURL = prompt(reader, "Ollama Base URL", cfg.BaseURL)
		if validateURL(cfg.BaseURL) {
			// Test connection
			fmt.Print("Testing connection... ")
			if testOllamaConnection(cfg.BaseURL) {
				fmt.Println("✓ Connected successfully")
				break
			} else {
				fmt.Println("✗ Connection failed")
				retry := prompt(reader, "Retry? (y/n)", "y")
				if strings.ToLower(retry) != "y" {
					break
				}
			}
		} else {
			fmt.Println("Invalid URL format. Please enter a valid URL (e.g., http://localhost:11434)")
		}
	}

	cfg.Model = prompt(reader, "Model Name", cfg.Model)
	cfg.ModelPath = prompt(reader, "Model Storage Path (optional)", cfg.ModelPath)

	// Interactive params with validation
	for {
		tempStr := prompt(reader, "Temperature (0.0-1.0, optional)", "")
		if tempStr == "" {
			break
		}
		if val, err := strconv.ParseFloat(tempStr, 64); err == nil && val >= 0 && val <= 1 {
			if cfg.Params == nil {
				cfg.Params = make(map[string]interface{})
			}
			cfg.Params["temperature"] = val
			break
		} else {
			fmt.Println("Invalid temperature. Must be between 0.0 and 1.0")
		}
	}

	// Timeout with validation
	for {
		timeoutStr := prompt(reader, "Timeout (seconds)", fmt.Sprintf("%.0f", cfg.Timeout.Seconds()))
		if val, err := strconv.Atoi(timeoutStr); err == nil && val > 0 && val <= 300 {
			cfg.Timeout = time.Duration(val) * time.Second
			break
		} else {
			fmt.Println("Invalid timeout. Must be between 1 and 300 seconds")
		}
	}
}

func configureOpenAI(reader *bufio.Reader, cfg *config.OpenAIConfig) {
	fmt.Println("\n--- OpenAI Configuration ---")
	
	// Base URL with validation
	for {
		cfg.BaseURL = prompt(reader, "API Base URL", cfg.BaseURL)
		if validateURL(cfg.BaseURL) {
			break
		} else {
			fmt.Println("Invalid URL format. Please enter a valid URL")
		}
	}

	// API Key with validation
	for {
		cfg.APIKey = prompt(reader, "API Key", "") // No default for security
		if cfg.APIKey != "" && len(cfg.APIKey) > 10 {
			break
		} else {
			fmt.Println("API Key is required and must be at least 10 characters")
		}
	}

	cfg.Model = prompt(reader, "Model Name", cfg.Model)

	// Timeout with validation
	for {
		timeoutStr := prompt(reader, "Timeout (seconds)", fmt.Sprintf("%.0f", cfg.Timeout.Seconds()))
		if val, err := strconv.Atoi(timeoutStr); err == nil && val > 0 && val <= 300 {
			cfg.Timeout = time.Duration(val) * time.Second
			break
		} else {
			fmt.Println("Invalid timeout. Must be between 1 and 300 seconds")
		}
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

// validateURL validates a URL format
func validateURL(urlStr string) bool {
	if urlStr == "" {
		return false
	}
	return strings.HasPrefix(urlStr, "http://") || strings.HasPrefix(urlStr, "https://")
}

// testOllamaConnection tests connection to Ollama server
func testOllamaConnection(baseURL string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", baseURL+"/api/tags", nil)
	if err != nil {
		return false
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

// ValidateConfig validates a configuration
func ValidateConfig(cfg *config.CleanupConfig) []string {
	var errors []string

	// Validate AI provider
	if cfg.AI.Provider != "ollama" && cfg.AI.Provider != "openai" {
		errors = append(errors, "AI provider must be 'ollama' or 'openai'")
	}

	// Validate Ollama config
	if cfg.AI.Provider == "ollama" {
		if cfg.Ollama.BaseURL == "" {
			errors = append(errors, "Ollama base URL is required")
		}
		if cfg.Ollama.Model == "" {
			errors = append(errors, "Ollama model is required")
		}
		if cfg.Ollama.Timeout < time.Second {
			errors = append(errors, "Ollama timeout must be at least 1 second")
		}
	}

	// Validate OpenAI config
	if cfg.AI.Provider == "openai" {
		if cfg.AI.OpenAI.BaseURL == "" {
			errors = append(errors, "OpenAI base URL is required")
		}
		if cfg.AI.OpenAI.APIKey == "" {
			errors = append(errors, "OpenAI API key is required")
		}
		if cfg.AI.OpenAI.Model == "" {
			errors = append(errors, "OpenAI model is required")
		}
		if cfg.AI.OpenAI.Timeout < time.Second {
			errors = append(errors, "OpenAI timeout must be at least 1 second")
		}
	}

	return errors
}
