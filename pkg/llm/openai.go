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

// OpenAIProvider implements the Provider interface for interacting with the OpenAI API.
type OpenAIProvider struct {
	client      *OpenAIClient
	model       string
	temperature float64
	maxTokens   int
}

// OpenAIClient handles low-level HTTP interactions with OpenAI endpoints.
type OpenAIClient struct {
	apiKey string
	baseURL string
	client  *http.Client
}

// OpenAIChatRequest models the payload for the OpenAI v1/chat/completions endpoint.
type OpenAIChatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float64   `json:"temperature,omitempty"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
}

// Message defines a single conversational turn in the prompt.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// OpenAIChatResponse captures the results from the OpenAI v1/chat/completions endpoint.
type OpenAIChatResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

// Choice encapsulates a single generated text candidate from OpenAI.
type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

// Usage tracks the token consumption for a given API request.
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// NewOpenAIProvider initializes the OpenAI integration with the given authentication and model parameters.
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

// Analyze issues a prompt to the configured OpenAI model and returns the generated diagnostic response.
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

// Name identifies this provider instance as "openai".
func (p *OpenAIProvider) Name() string {
	return "openai"
}

// GetModel exposes the configured OpenAI model string.
func (p *OpenAIProvider) GetModel() string {
	return p.model
}

// NewOpenAIProviderFromConfig constructs an OpenAIProvider using a standard LLMConfig block.
func NewOpenAIProviderFromConfig(cfg config.LLMConfig) (*OpenAIProvider, error) {
	return NewOpenAIProvider(cfg.APIKey, cfg.Model, cfg.Temperature, cfg.MaxTokens)
}
