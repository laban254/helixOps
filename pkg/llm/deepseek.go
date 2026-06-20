package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"helixops/internal/config"
)

// DeepSeekProvider implements the Provider interface for interacting with the DeepSeek API.
// DeepSeek exposes an OpenAI-compatible Chat Completions endpoint, so it reuses the OpenAI
// request/response wire types defined in openai.go.
type DeepSeekProvider struct {
	client      *DeepSeekClient
	model       string
	temperature float64
	maxTokens   int
}

// DeepSeekClient handles low-level HTTP interactions with DeepSeek endpoints.
type DeepSeekClient struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

// NewDeepSeekProvider initializes the DeepSeek integration with the given authentication and model parameters.
func NewDeepSeekProvider(apiKey, model string, temperature float64, maxTokens int) (*DeepSeekProvider, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("DeepSeek API key is required")
	}
	if model == "" {
		model = "deepseek-chat"
	}

	return &DeepSeekProvider{
		client: &DeepSeekClient{
			apiKey:  apiKey,
			baseURL: "https://api.deepseek.com/v1",
			client: &http.Client{
				Timeout: 60 * time.Second,
			},
		},
		model:       model,
		temperature: temperature,
		maxTokens:   maxTokens,
	}, nil
}

// Analyze issues a prompt to the configured DeepSeek model and returns the generated diagnostic response.
func (p *DeepSeekProvider) Analyze(ctx context.Context, prompt string) (string, error) {
	req := OpenAIChatRequest{
		Model: p.model,
		Messages: []Message{
			{
				Role:    "system",
				Content: "You are an SRE assistant analyzing incidents. Follow the output format requested in the user's message exactly.",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Temperature: p.temperature,
		MaxTokens:   p.maxTokens,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.client.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.client.apiKey)

	resp, err := p.client.client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("DeepSeek API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var chatResp OpenAIChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	return chatResp.Choices[0].Message.Content, nil
}

// Name identifies this provider instance as "deepseek".
func (p *DeepSeekProvider) Name() string {
	return "deepseek"
}

// GetModel exposes the configured DeepSeek model string.
func (p *DeepSeekProvider) GetModel() string {
	return p.model
}

// Health verifies the API is reachable and the key is accepted by issuing a
// lightweight authenticated GET to the (OpenAI-compatible) models endpoint.
func (p *DeepSeekProvider) Health(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.client.baseURL+"/models", nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+p.client.apiKey)

	resp, err := p.client.client.Do(req)
	if err != nil {
		return fmt.Errorf("DeepSeek not available: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("DeepSeek returned status: %d", resp.StatusCode)
	}
	return nil
}

// NewDeepSeekProviderFromConfig constructs a DeepSeekProvider using a standard LLMConfig block.
func NewDeepSeekProviderFromConfig(cfg config.LLMConfig) (*DeepSeekProvider, error) {
	return NewDeepSeekProvider(cfg.APIKey, cfg.Model, cfg.Temperature, cfg.MaxTokens)
}
