package ai

import (
	"context"

	"github.com/xuanyiying/cleanup-cli/internal/analyzer"
)

// AnalysisResult represents the result of an analysis request
type AnalysisResult struct {
	Success bool
	Content string
	Tokens  int
}

// Client defines the interface for AI operations
type Client interface {
	CheckHealth(ctx context.Context) error
	Analyze(ctx context.Context, prompt string, context string) (*AnalysisResult, error)
	SuggestName(ctx context.Context, file *analyzer.FileMetadata) ([]string, error)
	SuggestCategory(ctx context.Context, file *analyzer.FileMetadata) ([]string, error)
}
