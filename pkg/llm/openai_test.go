package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenAIProviderAnalyze(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		assert.Equal(t, "/v1/chat/completions", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Contains(t, r.Header.Get("Authorization"), "Bearer ")

		// Parse request body
		var req OpenAIChatRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)
		assert.Equal(t, "gpt-4o", req.Model)
		assert.Len(t, req.Messages, 2)

		// Return response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(OpenAIChatResponse{
			ID:      "test-id",
			Object:  "chat.completion",
			Created: time.Now().Unix(),
			Model:   "gpt-4o",
			Choices: []Choice{
				{
					Index: 0,
					Message: Message{
						Role:    "assistant",
						Content: "Test analysis response",
					},
					FinishReason: "stop",
				},
			},
		})
	}))
	defer server.Close()

	// Create provider
	provider, err := NewOpenAIProvider("test-api-key", "gpt-4o", 0.1, 1000)
	require.NoError(t, err)
	provider.client.baseURL = server.URL

	// Analyze
	result, err := provider.Analyze(context.Background(), "Test prompt")
	require.NoError(t, err)
	assert.Equal(t, "Test analysis response", result)
}

func TestOpenAIProviderAnalyzeError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": {"message": "Invalid API key"}}`))
	}))
	defer server.Close()

	provider, err := NewOpenAIProvider("invalid-key", "gpt-4o", 0.1, 1000)
	require.NoError(t, err)
	provider.client.baseURL = server.URL

	_, err = provider.Analyze(context.Background(), "Test prompt")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "API error")
}

func TestOpenAIProviderNoChoices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(OpenAIChatResponse{
			ID:      "test-id",
			Choices: []Choice{},
		})
	}))
	defer server.Close()

	provider, err := NewOpenAIProvider("test-key", "gpt-4o", 0.1, 1000)
	require.NoError(t, err)
	provider.client.baseURL = server.URL

	_, err = provider.Analyze(context.Background(), "Test prompt")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no choices")
}

func TestOpenAIProviderName(t *testing.T) {
	provider, err := NewOpenAIProvider("test-key", "gpt-4o", 0.1, 1000)
	require.NoError(t, err)
	assert.Equal(t, "openai", provider.Name())
	assert.Equal(t, "gpt-4o", provider.GetModel())
}

func TestNewOpenAIProviderMissingKey(t *testing.T) {
	_, err := NewOpenAIProvider("", "gpt-4o", 0.1, 1000)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "API key is required")
}
