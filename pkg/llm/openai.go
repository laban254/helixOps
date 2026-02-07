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

// OpenAIProvider implements Provider for OpenAI
type OpenAIProvider struct {
	client      *OpenAIClient
	model       string
	temperature float64
	maxTokens   int
}

// OpenAIClient is a simple OpenAI API client
type OpenAIClient struct {
	apiKey string
	baseURL string
	client  *http.Client
}

// OpenAIChatRequest represents the OpenAI chat completion request
type OpenAIChatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float64   `json:"temperature,omitempty"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
}

// Message represents a chat message
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// OpenAIChatResponse represents the OpenAI chat completion response
type OpenAIChatResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

// Choice represents a chat completion choice
type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

// Usage represents token usage
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// NewOpenAIProvider creates a new OpenAI provider
func NewOpenAIProvider(apiKey, model string, temperature float64, maxTokens int) (*OpenAIProvider, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("OpenAI API key is required")
	}
	if model == "" {
		model = "gpt-4o"
	}

	return &OpenAIProvider{
		client: &OpenAIClient{
			apiKey:  apiKey,
			baseURL: "https://api.openai.com/v1",
			client: &http.Client{
				Timeout: 60 * time.Second,
			},
		},
		model:       model,
		temperature: temperature,
		maxTokens:   maxTokens,
	}, nil
}

// Analyze sends a prompt to OpenAI and returns the response
func (p *OpenAIProvider) Analyze(ctx context.Context, prompt string) (string, error) {
	req := OpenAIChatRequest{
		Model: p.model,
		Messages: []Message{
			{
				Role:    "system",
				Content: "You are an SRE assistant analyzing incidents. Respond with JSON only.",
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
		return "", fmt.Errorf("OpenAI API error (status %d): %s", resp.StatusCode, string(respBody))
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

// Name returns the provider name
func (p *OpenAIProvider) Name() string {
	return "openai"
}

// GetModel returns the model name
func (p *OpenAIProvider) GetModel() string {
	return p.model
}

// OpenAIConfig creates an OpenAI provider from config
func NewOpenAIProviderFromConfig(cfg config.LLMConfig) (*OpenAIProvider, error) {
	return NewOpenAIProvider(cfg.APIKey, cfg.Model, cfg.Temperature, cfg.MaxTokens)
}
