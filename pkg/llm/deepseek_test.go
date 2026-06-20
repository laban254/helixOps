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

func TestDeepSeekProviderAnalyze(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/chat/completions", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Contains(t, r.Header.Get("Authorization"), "Bearer ")

		var req OpenAIChatRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)
		assert.Equal(t, "deepseek-chat", req.Model)
		assert.Len(t, req.Messages, 2)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(OpenAIChatResponse{
			ID:      "test-id",
			Object:  "chat.completion",
			Created: time.Now().Unix(),
			Model:   "deepseek-chat",
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

	provider, err := NewDeepSeekProvider("test-api-key", "deepseek-chat", 0.1, 1000)
	require.NoError(t, err)
	provider.client.baseURL = server.URL

	result, err := provider.Analyze(context.Background(), "Test prompt")
	require.NoError(t, err)
	assert.Equal(t, "Test analysis response", result)
}

func TestDeepSeekProviderAnalyzeError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": {"message": "Invalid API key"}}`))
	}))
	defer server.Close()

	provider, err := NewDeepSeekProvider("invalid-key", "deepseek-chat", 0.1, 1000)
	require.NoError(t, err)
	provider.client.baseURL = server.URL

	_, err = provider.Analyze(context.Background(), "Test prompt")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "API error")
}

func TestDeepSeekProviderName(t *testing.T) {
	provider, err := NewDeepSeekProvider("test-key", "deepseek-chat", 0.1, 1000)
	require.NoError(t, err)
	assert.Equal(t, "deepseek", provider.Name())
	assert.Equal(t, "deepseek-chat", provider.GetModel())
}

func TestDeepSeekProviderDefaultModel(t *testing.T) {
	provider, err := NewDeepSeekProvider("test-key", "", 0.1, 1000)
	require.NoError(t, err)
	assert.Equal(t, "deepseek-chat", provider.GetModel())
}

func TestNewDeepSeekProviderMissingKey(t *testing.T) {
	_, err := NewDeepSeekProvider("", "deepseek-chat", 0.1, 1000)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "API key is required")
}

func TestDeepSeekProviderHealth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/models", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Contains(t, r.Header.Get("Authorization"), "Bearer ")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	provider, err := NewDeepSeekProvider("test-key", "deepseek-chat", 0.1, 1000)
	require.NoError(t, err)
	provider.client.baseURL = server.URL

	require.NoError(t, provider.Health(context.Background()))
}

func TestDeepSeekProviderHealthUnauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	provider, err := NewDeepSeekProvider("bad-key", "deepseek-chat", 0.1, 1000)
	require.NoError(t, err)
	provider.client.baseURL = server.URL

	assert.Error(t, provider.Health(context.Background()))
}
