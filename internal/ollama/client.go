package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/user/cleanup-cli/internal/analyzer"
)

// Config holds Ollama client configuration
type Config struct {
	BaseURL string        // default: http://localhost:11434
	Model   string        // default: llama3.2
	Timeout time.Duration // default: 30s
}

// AnalysisResult represents the result of an analysis request
type AnalysisResult struct {
	Success bool
	Content string
	Tokens  int
}

// Client defines the interface for Ollama operations
type Client interface {
	CheckHealth(ctx context.Context) error
	Analyze(ctx context.Context, prompt string, context string) (*AnalysisResult, error)
	SuggestName(ctx context.Context, file *analyzer.FileMetadata) ([]string, error)
	SuggestCategory(ctx context.Context, file *analyzer.FileMetadata) ([]string, error)
}

// OllamaClient implements the Client interface
type OllamaClient struct {
	config     *Config
	httpClient *http.Client
}

// NewClient creates a new Ollama client with the given configuration
func NewClient(config *Config) *OllamaClient {
	if config == nil {
		config = &Config{
			BaseURL: "http://localhost:11434",
			Model:   "llama3.2",
			Timeout: 30 * time.Second,
		}
	}

	if config.BaseURL == "" {
		config.BaseURL = "http://localhost:11434"
	}
	if config.Model == "" {
		config.Model = "llama3.2"
	}
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	return &OllamaClient{
		config: config,
		httpClient: &http.Client{
			Timeout: config.Timeout,
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
func (c *OllamaClient) Analyze(ctx context.Context, prompt string, contextStr string) (*AnalysisResult, error) {
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

	return &AnalysisResult{
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

	// Build a detailed prompt based on file type and content
	var prompt string
	
	// Check if it's a document type
	isDocument := strings.HasPrefix(file.MimeType, "text/") || 
		strings.Contains(file.MimeType, "pdf") ||
		strings.Contains(file.MimeType, "document") ||
		strings.Contains(file.MimeType, "word") ||
		strings.Contains(file.MimeType, "excel") ||
		strings.Contains(file.MimeType, "powerpoint")
	
	if file.ContentPreview != "" && len(file.ContentPreview) > 20 {
		if isDocument {
			// For documents, use optimized prompt
			prompt = fmt.Sprintf(
				"You are a file naming expert. Analyze the document content and create a descriptive filename.\n\n"+
					"Document content:\n%s\n\n"+
					"Requirements:\n"+
					"1. Extract the main topic or purpose from the content\n"+
					"2. Create a clear, specific filename (e.g., 'quarterly-sales-report-2024', 'team-meeting-notes-jan')\n"+
					"3. Use only lowercase letters, numbers, and hyphens\n"+
					"4. Keep it between 15-50 characters\n"+
					"5. Do NOT include file extension\n"+
					"6. Do NOT use quotes or special characters\n\n"+
					"Output ONLY the filename:",
				file.ContentPreview,
			)
		} else {
			// For other files with content
			prompt = fmt.Sprintf(
				"Create a descriptive filename based on this content:\n\n%s\n\n"+
					"Rules:\n"+
					"- Use lowercase with hyphens\n"+
					"- Be specific and concise\n"+
					"- Maximum 40 characters\n"+
					"- No file extension\n\n"+
					"Filename:",
				file.ContentPreview,
			)
		}
	} else {
		// For files without content preview
		prompt = fmt.Sprintf(
			"Suggest a better filename for: %s (type: %s)\n\n"+
				"Output a concise, descriptive name using lowercase and hyphens. "+
				"No extension. Maximum 30 characters.\n\n"+
				"Filename:",
			file.Name, file.MimeType,
		)
	}

	result, err := c.Analyze(ctx, prompt, "")
	if err != nil {
		return nil, err
	}

	if !result.Success {
		return nil, fmt.Errorf("analysis did not complete successfully")
	}

	// Parse the response and clean it up
	suggestedName := strings.TrimSpace(result.Content)
	
	// Take first line only
	lines := strings.Split(suggestedName, "\n")
	suggestedName = strings.TrimSpace(lines[0])
	
	// Remove common prefixes that AI might add
	suggestedName = strings.TrimPrefix(suggestedName, "Filename:")
	suggestedName = strings.TrimPrefix(suggestedName, "filename:")
	suggestedName = strings.TrimPrefix(suggestedName, "Suggested:")
	suggestedName = strings.TrimPrefix(suggestedName, "suggested:")
	suggestedName = strings.TrimSpace(suggestedName)
	
	// Remove quotes and backticks
	suggestedName = strings.Trim(suggestedName, "\"'`")
	
	// Remove any file extension if AI added one
	if ext := filepath.Ext(suggestedName); ext != "" {
		suggestedName = strings.TrimSuffix(suggestedName, ext)
	}
	
	// Clean up: replace spaces with hyphens, convert to lowercase
	suggestedName = strings.ToLower(suggestedName)
	suggestedName = strings.ReplaceAll(suggestedName, " ", "-")
	suggestedName = strings.ReplaceAll(suggestedName, "_", "-")
	
	// Remove multiple consecutive hyphens
	for strings.Contains(suggestedName, "--") {
		suggestedName = strings.ReplaceAll(suggestedName, "--", "-")
	}
	
	// Trim hyphens from start and end
	suggestedName = strings.Trim(suggestedName, "-")

	// Validate the suggested name
	if suggestedName == "" || len(suggestedName) > 100 || len(suggestedName) < 3 {
		return []string{}, nil
	}

	return []string{suggestedName}, nil
}

// SuggestCategory generates category suggestions for a file based on content analysis
func (c *OllamaClient) SuggestCategory(ctx context.Context, file *analyzer.FileMetadata) ([]string, error) {
	if file == nil {
		return nil, fmt.Errorf("file metadata cannot be nil")
	}

	// If no content preview, return empty
	if file.ContentPreview == "" || len(file.ContentPreview) < 20 {
		return []string{}, nil
	}

	prompt := fmt.Sprintf(
		"Analyze the document content and categorize it by its purpose/scenario.\n\n"+
			"Document content:\n%s\n\n"+
			"Based on the content, identify the document's primary purpose/scenario.\n"+
			"Return ONE category from these options:\n"+
			"- resume (简历、CV、个人简历)\n"+
			"- interview (面试题、面试准备、面试笔记)\n"+
			"- meeting (会议记录、会议纪要、讨论记录)\n"+
			"- report (报告、分析报告、工作报告)\n"+
			"- proposal (提案、建议书、项目提案)\n"+
			"- contract (合同、协议、条款)\n"+
			"- invoice (发票、账单、收据)\n"+
			"- guide (指南、教程、说明书)\n"+
			"- notes (笔记、备忘录、草稿)\n"+
			"- other (其他)\n\n"+
			"Output ONLY the category name (lowercase):",
		file.ContentPreview,
	)

	result, err := c.Analyze(ctx, prompt, "")
	if err != nil {
		return nil, err
	}

	if !result.Success {
		return nil, fmt.Errorf("analysis did not complete successfully")
	}

	// Parse the response
	category := strings.TrimSpace(result.Content)
	category = strings.ToLower(category)
	
	// Take first line only
	lines := strings.Split(category, "\n")
	category = strings.TrimSpace(lines[0])
	
	// Remove common prefixes
	category = strings.TrimPrefix(category, "category:")
	category = strings.TrimPrefix(category, "Category:")
	category = strings.TrimSpace(category)
	
	// Remove quotes
	category = strings.Trim(category, "\"'`")
	
	// Validate category
	validCategories := map[string]bool{
		"resume":    true,
		"interview": true,
		"meeting":   true,
		"report":    true,
		"proposal":  true,
		"contract":  true,
		"invoice":   true,
		"guide":     true,
		"notes":     true,
		"other":     true,
	}
	
	if !validCategories[category] {
		return []string{"other"}, nil
	}

	return []string{category}, nil
}
