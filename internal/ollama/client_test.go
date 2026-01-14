package ollama

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

func TestNewClient(t *testing.T) {
	tests := []struct {
		name     string
		config   *config.OllamaConfig
		expected *config.OllamaConfig
	}{
		{
			name:   "nil config uses defaults",
			config: nil,
			expected: &config.OllamaConfig{
				BaseURL: "http://localhost:11434",
				Model:   "llama3.2",
				Timeout: 30 * time.Second,
			},
		},
		{
			name: "partial config fills in defaults",
			config: &config.OllamaConfig{
				BaseURL: "http://custom:11434",
			},
			expected: &config.OllamaConfig{
				BaseURL: "http://custom:11434",
				Model:   "llama3.2",
				Timeout: 30 * time.Second,
			},
		},
		{
			name: "full config is preserved",
			config: &config.OllamaConfig{
				BaseURL: "http://custom:11434",
				Model:   "custom-model",
				Timeout: 60 * time.Second,
			},
			expected: &config.OllamaConfig{
				BaseURL: "http://custom:11434",
				Model:   "custom-model",
				Timeout: 60 * time.Second,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient(tt.config)
			assert.NotNil(t, client)
			assert.Equal(t, tt.expected.BaseURL, client.config.BaseURL)
			assert.Equal(t, tt.expected.Model, client.config.Model)
			assert.Equal(t, tt.expected.Timeout, client.config.Timeout)
		})
	}
}

func TestCheckHealth(t *testing.T) {
	tests := []struct {
		name         string
		statusCode   int
		responseBody string
		expectError  bool
		errorMsg     string
	}{
		{
			name:         "health check succeeds",
			statusCode:   http.StatusOK,
			responseBody: `{"models":[]}`,
			expectError:  false,
		},
		{
			name:         "service unavailable",
			statusCode:   http.StatusServiceUnavailable,
			responseBody: `{"error":"service unavailable"}`,
			expectError:  true,
			errorMsg:     "health check failed with status 503",
		},
		{
			name:         "not found",
			statusCode:   http.StatusNotFound,
			responseBody: `{"error":"not found"}`,
			expectError:  true,
			errorMsg:     "health check failed with status 404",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/api/tags", r.URL.Path)
				assert.Equal(t, http.MethodGet, r.Method)
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			client := NewClient(&config.OllamaConfig{
				BaseURL: server.URL,
				Model:   "test-model",
				Timeout: 5 * time.Second,
			})

			err := client.CheckHealth(context.Background())
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCheckHealthConnectionError(t *testing.T) {
	client := NewClient(&config.OllamaConfig{
		BaseURL: "http://invalid-host-that-does-not-exist:11434",
		Model:   "test-model",
		Timeout: 1 * time.Second,
	})

	err := client.CheckHealth(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ollama service unavailable")
}

func TestAnalyze(t *testing.T) {
	tests := []struct {
		name         string
		prompt       string
		context      string
		statusCode   int
		responseBody generateResponse
		expectError  bool
		errorMsg     string
	}{
		{
			name:       "successful analysis",
			prompt:     "What is this file?",
			context:    "file.txt",
			statusCode: http.StatusOK,
			responseBody: generateResponse{
				Model:     "llama3.2",
				Response:  "This is a text file",
				Done:      true,
				EvalCount: 10,
			},
			expectError: false,
		},
		{
			name:       "analysis with empty context",
			prompt:     "Analyze this",
			context:    "",
			statusCode: http.StatusOK,
			responseBody: generateResponse{
				Model:     "llama3.2",
				Response:  "Analysis result",
				Done:      true,
				EvalCount: 5,
			},
			expectError: false,
		},
		{
			name:        "empty prompt error",
			prompt:      "",
			context:     "",
			expectError: true,
			errorMsg:    "prompt cannot be empty",
		},
		{
			name:         "server error",
			prompt:       "What is this?",
			context:      "",
			statusCode:   http.StatusInternalServerError,
			responseBody: generateResponse{},
			expectError:  true,
			errorMsg:     "analyze request failed with status 500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/api/generate", r.URL.Path)
				assert.Equal(t, http.MethodPost, r.Method)

				var req generateRequest
				err := json.NewDecoder(r.Body).Decode(&req)
				require.NoError(t, err)
				assert.Equal(t, "test-model", req.Model)
				assert.False(t, req.Stream)

				w.WriteHeader(tt.statusCode)
				json.NewEncoder(w).Encode(tt.responseBody)
			}))
			defer server.Close()

			client := NewClient(&config.OllamaConfig{
				BaseURL: server.URL,
				Model:   "test-model",
				Timeout: 5 * time.Second,
			})

			result, err := client.Analyze(context.Background(), tt.prompt, tt.context)
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.responseBody.Response, result.Content)
				assert.Equal(t, tt.responseBody.Done, result.Success)
				assert.Equal(t, tt.responseBody.EvalCount, result.Tokens)
			}
		})
	}
}

func TestAnalyzeContextCombination(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req generateRequest
		json.NewDecoder(r.Body).Decode(&req)

		// Verify that prompt and context are combined
		assert.Contains(t, req.Prompt, "Main prompt")
		assert.Contains(t, req.Prompt, "Context: Additional context")

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(generateResponse{
			Response: "Result",
			Done:     true,
		})
	}))
	defer server.Close()

	client := NewClient(&config.OllamaConfig{
		BaseURL: server.URL,
		Model:   "test-model",
		Timeout: 5 * time.Second,
	})

	result, err := client.Analyze(context.Background(), "Main prompt", "Additional context")
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSuggestName(t *testing.T) {
	tests := []struct {
		name        string
		file        *analyzer.FileMetadata
		statusCode  int
		expectError bool
		errorMsg    string
	}{
		{
			name: "successful name suggestion",
			file: &analyzer.FileMetadata{
				Name:           "document.pdf",
				Extension:      "pdf",
				Size:           1024,
				MimeType:       "application/pdf",
				ContentPreview: "Sample content",
			},
			statusCode:  http.StatusOK,
			expectError: false,
		},
		{
			name:        "nil file error",
			file:        nil,
			expectError: true,
			errorMsg:    "file metadata cannot be nil",
		},
		{
			name: "analysis failure",
			file: &analyzer.FileMetadata{
				Name:      "test.txt",
				Extension: "txt",
			},
			statusCode:  http.StatusOK,
			expectError: true,
			errorMsg:    "analysis did not complete successfully",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				json.NewEncoder(w).Encode(generateResponse{
					Response: "Suggested names",
					Done:     !tt.expectError || tt.file == nil,
				})
			}))
			defer server.Close()

			client := NewClient(&config.OllamaConfig{
				BaseURL: server.URL,
				Model:   "test-model",
				Timeout: 5 * time.Second,
			})

			result, err := client.SuggestName(context.Background(), tt.file)
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
			}
		})
	}
}

func TestSuggestCategory(t *testing.T) {
	tests := []struct {
		name        string
		file        *analyzer.FileMetadata
		statusCode  int
		expectError bool
		errorMsg    string
	}{
		{
			name: "successful category suggestion",
			file: &analyzer.FileMetadata{
				Name:           "resume.txt",
				Extension:      "txt",
				MimeType:       "text/plain",
				ContentPreview: "JOHN DOE\nEmail: john@example.com\nExperience: Senior Software Engineer",
			},
			statusCode:  http.StatusOK,
			expectError: false,
		},
		{
			name:        "nil file error",
			file:        nil,
			expectError: true,
			errorMsg:    "file metadata cannot be nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				json.NewEncoder(w).Encode(generateResponse{
					Response: "resume",
					Done:     true,
				})
			}))
			defer server.Close()

			client := NewClient(&config.OllamaConfig{
				BaseURL: server.URL,
				Model:   "test-model",
				Timeout: 5 * time.Second,
			})

			result, err := client.SuggestCategory(context.Background(), tt.file)
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
			}
		})
	}
}

func TestAnalyzeTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(generateResponse{
			Response: "Result",
			Done:     true,
		})
	}))
	defer server.Close()

	client := NewClient(&config.OllamaConfig{
		BaseURL: server.URL,
		Model:   "test-model",
		Timeout: 500 * time.Millisecond,
	})

	_, err := client.Analyze(context.Background(), "test prompt", "")
	assert.Error(t, err)
}

func TestCheckHealthTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(&config.OllamaConfig{
		BaseURL: server.URL,
		Model:   "test-model",
		Timeout: 500 * time.Millisecond,
	})

	err := client.CheckHealth(context.Background())
	assert.Error(t, err)
}

func TestAnalyzeWithContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req generateRequest
		json.NewDecoder(r.Body).Decode(&req)

		// Verify context is included in prompt
		assert.Contains(t, req.Prompt, "Context:")

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(generateResponse{
			Response:  "Result with context",
			Done:      true,
			EvalCount: 15,
		})
	}))
	defer server.Close()

	client := NewClient(&config.OllamaConfig{
		BaseURL: server.URL,
		Model:   "test-model",
		Timeout: 5 * time.Second,
	})

	result, err := client.Analyze(context.Background(), "prompt", "context info")
	assert.NoError(t, err)
	assert.Equal(t, "Result with context", result.Content)
	assert.Equal(t, 15, result.Tokens)
}
