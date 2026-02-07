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

func TestAnthropicProviderAnalyze(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/messages", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Contains(t, r.Header.Get("x-api-key"), "test-key")
		assert.Contains(t, r.Header.Get("anthropic-version"), "2023-06-01")

		var req AnthropicRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)
		assert.Equal(t, "claude-3-5-sonnet", req.Model)
		assert.Len(t, req.Messages, 1)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(AnthropicResponse{
			ID:   "test-id",
			Type: "message",
			Role: "assistant",
			Content: []AnthropicContent{
				{
					Type: "text",
					Text: "Claude analysis response",
				},
			},
			Model:      "claude-3-5-sonnet",
			StopReason: "end_turn",
		})
	}))
	defer server.Close()

	provider, err := NewAnthropicProvider("test-api-key", "claude-3-5-sonnet", 0.1, 1000)
	require.NoError(t, err)
	provider.client.baseURL = server.URL

	result, err := provider.Analyze(context.Background(), "Test prompt")
	require.NoError(t, err)
	assert.Equal(t, "Claude analysis response", result)
}

func TestAnthropicProviderAnalyzeError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": {"type": "authentication_error", "message": "Invalid API key"}}`))
	}))
	defer server.Close()

	provider, err := NewAnthropicProvider("invalid-key", "claude-3-5-sonnet", 0.1, 1000)
	require.NoError(t, err)
	provider.client.baseURL = server.URL

	_, err = provider.Analyze(context.Background(), "Test prompt")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "API error")
}

func TestAnthropicProviderNoContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(AnthropicResponse{
			ID:      "test-id",
			Content: []AnthropicContent{},
		})
	}))
	defer server.Close()

	provider, err := NewAnthropicProvider("test-key", "claude-3-5-sonnet", 0.1, 1000)
	require.NoError(t, err)
	provider.client.baseURL = server.URL

	_, err = provider.Analyze(context.Background(), "Test prompt")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no content")
}

func TestAnthropicProviderName(t *testing.T) {
	provider, err := NewAnthropicProvider("test-key", "claude-3-5-sonnet", 0.1, 1000)
	require.NoError(t, err)
	assert.Equal(t, "anthropic", provider.Name())
	assert.Equal(t, "claude-3-5-sonnet", provider.GetModel())
}

func TestNewAnthropicProviderMissingKey(t *testing.T) {
	_, err := NewAnthropicProvider("", "claude-3-5-sonnet", 0.1, 1000)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "API key is required")
}
