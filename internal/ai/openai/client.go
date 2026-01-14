package openai

import (
	"context"
	"fmt"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/xuanyiying/cleanup-cli/internal/ai"
	"github.com/xuanyiying/cleanup-cli/internal/analyzer"
	"github.com/xuanyiying/cleanup-cli/internal/config"
)

type OpenAIClient struct {
	client *openai.Client
	config *config.OpenAIConfig
}

func NewClient(cfg *config.OpenAIConfig) *OpenAIClient {
	opts := []option.RequestOption{
		option.WithAPIKey(cfg.APIKey),
	}

	if cfg.BaseURL != "" {
		opts = append(opts, option.WithBaseURL(cfg.BaseURL))
	}

	client := openai.NewClient(opts...)
	return &OpenAIClient{
		client: &client,
		config: cfg,
	}
}

func (c *OpenAIClient) CheckHealth(ctx context.Context) error {
	// A simple way to check health is to do a cheap completion
	_, err := c.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage("ping"),
		},
		Model:     openai.ChatModel(c.config.Model),
		MaxTokens: openai.Int(1),
	})

	if err != nil {
		return fmt.Errorf("openai health check failed: %w", err)
	}
	return nil
}

func (c *OpenAIClient) Analyze(ctx context.Context, prompt string, contextContent string) (*ai.AnalysisResult, error) {
	fullPrompt := prompt
	if contextContent != "" {
		fullPrompt = fmt.Sprintf("%s\n\nContext:\n%s", prompt, contextContent)
	}

	resp, err := c.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(fullPrompt),
		},
		Model: openai.ChatModel(c.config.Model),
	})

	if err != nil {
		return nil, fmt.Errorf("openai analysis failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return &ai.AnalysisResult{Success: false}, nil
	}

	return &ai.AnalysisResult{
		Success: true,
		Content: resp.Choices[0].Message.Content,
		Tokens:  int(resp.Usage.TotalTokens),
	}, nil
}

func (c *OpenAIClient) SuggestName(ctx context.Context, file *analyzer.FileMetadata) ([]string, error) {
	prompt := ai.GenerateNameSuggestionPrompt(file)
	if prompt == "" {
		return nil, fmt.Errorf("could not generate prompt")
	}

	result, err := c.Analyze(ctx, prompt, "")
	if err != nil {
		return nil, err
	}

	if !result.Success {
		return nil, fmt.Errorf("analysis failed")
	}

	return []string{ai.CleanSuggestedName(result.Content)}, nil
}

func (c *OpenAIClient) SuggestCategory(ctx context.Context, file *analyzer.FileMetadata) ([]string, error) {
	prompt := ai.GenerateCategorySuggestionPrompt(file)
	if prompt == "" {
		return []string{}, nil
	}

	result, err := c.Analyze(ctx, prompt, "")
	if err != nil {
		return nil, err
	}

	if !result.Success {
		return nil, fmt.Errorf("analysis failed")
	}

	return []string{ai.CleanSuggestedCategory(result.Content)}, nil
}
