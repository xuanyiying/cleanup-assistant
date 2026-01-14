package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/xuanyiying/cleanup-cli/internal/ai"
	"github.com/xuanyiying/cleanup-cli/internal/analyzer"
	"github.com/xuanyiying/cleanup-cli/internal/config"
)

// OllamaClient implements the Client interface
type OllamaClient struct {
	config     *config.OllamaConfig
	httpClient *http.Client
}

// NewClient creates a new Ollama client with the given configuration
func NewClient(cfg *config.OllamaConfig) *OllamaClient {
	if cfg == nil {
		cfg = &config.OllamaConfig{
			BaseURL: "http://localhost:11434",
			Model:   "llama3.2",
			Timeout: 30 * time.Second,
		}
	}

	if cfg.BaseURL == "" {
		cfg.BaseURL = "http://localhost:11434"
	}
	if cfg.Model == "" {
		cfg.Model = "llama3.2"
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}

	return &OllamaClient{
		config: cfg,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
	}
}

// CheckHealth verifies that Ollama service is running
func (c *OllamaClient) CheckHealth(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.config.BaseURL+"/api/tags", nil)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("ollama service unavailable at %s: %w", c.config.BaseURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ollama health check failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// generateRequest represents a request to the Ollama generate endpoint
type generateRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

// generateResponse represents a response from the Ollama generate endpoint
type generateResponse struct {
	Model              string `json:"model"`
	CreatedAt          string `json:"created_at"`
	Response           string `json:"response"`
	Done               bool   `json:"done"`
	Context            []int  `json:"context"`
	TotalDuration      int64  `json:"total_duration"`
	LoadDuration       int64  `json:"load_duration"`
	PromptEvalCount    int    `json:"prompt_eval_count"`
	PromptEvalDuration int64  `json:"prompt_eval_duration"`
	EvalCount          int    `json:"eval_count"`
	EvalDuration       int64  `json:"eval_duration"`
}

// Analyze sends a prompt to Ollama and returns the analysis result
func (c *OllamaClient) Analyze(ctx context.Context, prompt string, contextStr string) (*ai.AnalysisResult, error) {
	if prompt == "" {
		return nil, fmt.Errorf("prompt cannot be empty")
	}

	fullPrompt := prompt
	if contextStr != "" {
		fullPrompt = fmt.Sprintf("%s\n\nContext: %s", prompt, contextStr)
	}

	reqBody := generateRequest{
		Model:  c.config.Model,
		Prompt: fullPrompt,
		Stream: false,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.config.BaseURL+"/api/generate", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create analyze request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send analyze request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("analyze request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var genResp generateResponse
	if err := json.NewDecoder(resp.Body).Decode(&genResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &ai.AnalysisResult{
		Success: genResp.Done,
		Content: genResp.Response,
		Tokens:  genResp.EvalCount,
	}, nil
}

// SuggestName generates name suggestions for a file based on its metadata
func (c *OllamaClient) SuggestName(ctx context.Context, file *analyzer.FileMetadata) ([]string, error) {
	if file == nil {
		return nil, fmt.Errorf("file metadata cannot be nil")
	}

	prompt := ai.GenerateNameSuggestionPrompt(file)
	if prompt == "" {
		return nil, fmt.Errorf("could not generate prompt")
	}

	result, err := c.Analyze(ctx, prompt, "")
	if err != nil {
		return nil, err
	}

	if !result.Success {
		return nil, fmt.Errorf("analysis did not complete successfully")
	}

	return []string{ai.CleanSuggestedName(result.Content)}, nil
}

// SuggestCategory generates category suggestions for a file based on content analysis
func (c *OllamaClient) SuggestCategory(ctx context.Context, file *analyzer.FileMetadata) ([]string, error) {
	if file == nil {
		return nil, fmt.Errorf("file metadata cannot be nil")
	}

	prompt := ai.GenerateCategorySuggestionPrompt(file)
	if prompt == "" {
		return []string{}, nil
	}

	result, err := c.Analyze(ctx, prompt, "")
	if err != nil {
		return nil, err
	}

	if !result.Success {
		return nil, fmt.Errorf("analysis did not complete successfully")
	}

	return []string{ai.CleanSuggestedCategory(result.Content)}, nil
}
