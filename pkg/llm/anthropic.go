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

// AnthropicProvider implements Provider for Anthropic
type AnthropicProvider struct {
	client      *AnthropicClient
	model       string
	temperature float64
	maxTokens   int
}

// AnthropicClient is a simple Anthropic API client
type AnthropicClient struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

// AnthropicMessage represents an Anthropic message
type AnthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// AnthropicRequest represents the Anthropic API request
type AnthropicRequest struct {
	Model       string          `json:"model"`
	Messages    []AnthropicMessage `json:"messages"`
	Temperature float64        `json:"temperature,omitempty"`
	MaxTokens   int            `json:"max_tokens,omitempty"`
}

// AnthropicResponse represents the Anthropic API response
type AnthropicResponse struct {
	ID       string            `json:"id"`
	Type     string            `json:"type"`
	Role     string            `json:"role"`
	Content  []AnthropicContent `json:"content"`
	Model    string            `json:"model"`
	StopReason string         `json:"stop_reason"`
	Usage    AnthropicUsage    `json:"usage"`
}

// AnthropicContent represents content in the response
type AnthropicContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// AnthropicUsage represents token usage
type AnthropicUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// NewAnthropicProvider creates a new Anthropic provider
func NewAnthropicProvider(apiKey, model string, temperature float64, maxTokens int) (*AnthropicProvider, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("Anthropic API key is required")
	}
	if model == "" {
		model = "claude-3-5-sonnet-20241022"
	}

	return &AnthropicProvider{
		client: &AnthropicClient{
			apiKey:  apiKey,
			baseURL: "https://api.anthropic.com/v1",
			client: &http.Client{
				Timeout: 60 * time.Second,
			},
		},
		model:       model,
		temperature: temperature,
		maxTokens:   maxTokens,
	}, nil
}

// Analyze sends a prompt to Anthropic and returns the response
func (p *AnthropicProvider) Analyze(ctx context.Context, prompt string) (string, error) {
	req := AnthropicRequest{
		Model: p.model,
		Messages: []AnthropicMessage{
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

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.client.baseURL+"/messages", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.client.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := p.client.client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Anthropic API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var anthropicResp AnthropicResponse
	if err := json.NewDecoder(resp.Body).Decode(&anthropicResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(anthropicResp.Content) == 0 {
		return "", fmt.Errorf("no content in response")
	}

	return anthropicResp.Content[0].Text, nil
}

// Name returns the provider name
func (p *AnthropicProvider) Name() string {
	return "anthropic"
}

// GetModel returns the model name
func (p *AnthropicProvider) GetModel() string {
	return p.model
}

// AnthropicConfig creates an Anthropic provider from config
func NewAnthropicProviderFromConfig(cfg config.LLMConfig) (*AnthropicProvider, error) {
	return NewAnthropicProvider(cfg.APIKey, cfg.Model, cfg.Temperature, cfg.MaxTokens)
}
