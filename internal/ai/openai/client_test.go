package openai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xuanyiying/cleanup-cli/internal/analyzer"
	"github.com/xuanyiying/cleanup-cli/internal/config"
)

// Mock response structures for OpenAI API
type chatCompletionResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []choice `json:"choices"`
	Usage   usage    `json:"usage"`
}

type choice struct {
	Index        int     `json:"index"`
	Message      message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

func TestOpenAIClient_SuggestName(t *testing.T) {
	// Setup mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/chat/completions", r.URL.Path)
		assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))

		resp := chatCompletionResponse{
			ID:      "chatcmpl-123",
			Object:  "chat.completion",
			Created: time.Now().Unix(),
			Model:   "gpt-4",
			Choices: []choice{
				{
					Index: 0,
					Message: message{
						Role:    "assistant",
						Content: "suggested-filename",
					},
					FinishReason: "stop",
				},
			},
			Usage: usage{
				PromptTokens:     10,
				CompletionTokens: 5,
				TotalTokens:      15,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Setup client
	cfg := &config.OpenAIConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Model:   "gpt-4",
		Timeout: 10 * time.Second,
	}
	client := NewClient(cfg)

	// Test SuggestName
	file := &analyzer.FileMetadata{
		Name:           "test.txt",
		MimeType:       "text/plain",
		ContentPreview: "This is a test file content.",
	}

	names, err := client.SuggestName(context.Background(), file)
	require.NoError(t, err)
	assert.Len(t, names, 1)
	assert.Equal(t, "suggested-filename", names[0])
}

func TestOpenAIClient_SuggestCategory(t *testing.T) {
	// Setup mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/chat/completions", r.URL.Path)

		resp := chatCompletionResponse{
			ID:      "chatcmpl-123",
			Object:  "chat.completion",
			Created: time.Now().Unix(),
			Model:   "gpt-4",
			Choices: []choice{
				{
					Index: 0,
					Message: message{
						Role:    "assistant",
						Content: "report",
					},
					FinishReason: "stop",
				},
			},
			Usage: usage{
				PromptTokens:     10,
				CompletionTokens: 5,
				TotalTokens:      15,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Setup client
	cfg := &config.OpenAIConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Model:   "gpt-4",
		Timeout: 10 * time.Second,
	}
	client := NewClient(cfg)

	// Test SuggestCategory
	file := &analyzer.FileMetadata{
		Name:           "test.txt",
		MimeType:       "text/plain",
		ContentPreview: "This is a test file content.",
	}

	categories, err := client.SuggestCategory(context.Background(), file)
	require.NoError(t, err)
	assert.Len(t, categories, 1)
	assert.Equal(t, "report", categories[0]) // Expecting lowercase from CleanSuggestedCategory
}

func TestOpenAIClient_CheckHealth(t *testing.T) {
	// Setup mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/chat/completions", r.URL.Path)

		resp := chatCompletionResponse{
			ID:      "chatcmpl-123",
			Object:  "chat.completion",
			Created: time.Now().Unix(),
			Model:   "gpt-4",
			Choices: []choice{
				{
					Index: 0,
					Message: message{
						Role:    "assistant",
						Content: "pong",
					},
					FinishReason: "stop",
				},
			},
			Usage: usage{
				PromptTokens:     5,
				CompletionTokens: 1,
				TotalTokens:      6,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Setup client
	cfg := &config.OpenAIConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Model:   "gpt-4",
		Timeout: 10 * time.Second,
	}
	client := NewClient(cfg)

	// Test CheckHealth
	err := client.CheckHealth(context.Background())
	require.NoError(t, err)
}

func TestOpenAIClient_CheckHealth_Error(t *testing.T) {
	// Setup mock server that returns 500
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	// Setup client
	cfg := &config.OpenAIConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Model:   "gpt-4",
		Timeout: 10 * time.Second,
	}
	client := NewClient(cfg)

	// Test CheckHealth
	err := client.CheckHealth(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "openai health check failed")
}
