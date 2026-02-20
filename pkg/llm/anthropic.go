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

// AnthropicProvider implements the Provider interface for interacting with the Anthropic Messages API.
type AnthropicProvider struct {
	client      *AnthropicClient
	model       string
	temperature float64
	maxTokens   int
}

// AnthropicClient handles low-level HTTP interactions with Anthropic endpoints.
type AnthropicClient struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

// AnthropicMessage defines a single conversational turn in the Anthropic prompt format.
type AnthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// AnthropicRequest models the payload for the Anthropic v1/messages endpoint.
type AnthropicRequest struct {
	Model       string          `json:"model"`
	Messages    []AnthropicMessage `json:"messages"`
	Temperature float64        `json:"temperature,omitempty"`
	MaxTokens   int            `json:"max_tokens,omitempty"`
}

// AnthropicResponse captures the results from the Anthropic v1/messages endpoint.
type AnthropicResponse struct {
	ID       string            `json:"id"`
	Type     string            `json:"type"`
	Role     string            `json:"role"`
	Content  []AnthropicContent `json:"content"`
	Model    string            `json:"model"`
	StopReason string         `json:"stop_reason"`
	Usage    AnthropicUsage    `json:"usage"`
}

// AnthropicContent encapsulates a single generated text or media block from Anthropic.
type AnthropicContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// AnthropicUsage tracks the token consumption for a given Anthropic API request.
type AnthropicUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// NewAnthropicProvider initializes the Anthropic integration with the given authentication and model parameters.
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

// Analyze issues a prompt to the configured Anthropic model and returns the generated diagnostic response.
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

// Name identifies this provider instance as "anthropic".
func (p *AnthropicProvider) Name() string {
	return "anthropic"
}

// GetModel exposes the configured Anthropic model string.
func (p *AnthropicProvider) GetModel() string {
	return p.model
}

// NewAnthropicProviderFromConfig constructs an AnthropicProvider using a standard LLMConfig block.
func NewAnthropicProviderFromConfig(cfg config.LLMConfig) (*AnthropicProvider, error) {
	return NewAnthropicProvider(cfg.APIKey, cfg.Model, cfg.Temperature, cfg.MaxTokens)
}
