package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"helixops/internal/config"
)

// OllamaProvider implements the Provider interface for interacting with localized Ollama instances.
type OllamaProvider struct {
	url        string
	model      string
	temperature float64
	client     *http.Client
}

// OllamaRequest models the payload for the Ollama /api/generate endpoint.
type OllamaRequest struct {
	Model       string  `json:"model"`
	Prompt      string  `json:"prompt"`
	Temperature float64 `json:"temperature,omitempty"`
	Stream      bool    `json:"stream,omitempty"`
}

// OllamaResponse captures the results from the Ollama /api/generate endpoint.
type OllamaResponse struct {
	Response   string `json:"response"`
	Done       bool   `json:"done"`
	TotalDuration int64 `json:"total_duration,omitempty"`
	LoadDuration int64  `json:"load_duration,omitempty"`
	SampleCount int64   `json:"sample_count,omitempty"`
	SampleDuration int64 `json:"sample_duration,omitempty"`
	PromptEvalCount int64 `json:"prompt_eval_count,omitempty"`
	EvalCount  int64   `json:"eval_count,omitempty"`
}

// NewOllamaProvider initializes the Ollama integration with the given host URL and model parameters.
func NewOllamaProvider(url, model string, temperature float64) (*OllamaProvider, error) {
	if url == "" {
		url = "http://localhost:11434"
	}
	if model == "" {
		model = "llama3"
	}

	url = strings.TrimSuffix(url, "/")

	return &OllamaProvider{
		url:        url,
		model:      model,
		temperature: temperature,
		client: &http.Client{
			Timeout: 300 * time.Second,
		},
	}, nil
}

// Analyze issues a prompt to the configured local Ollama instance and returns the generated diagnostic response.
func (p *OllamaProvider) Analyze(ctx context.Context, prompt string) (string, error) {
	req := OllamaRequest{
		Model:       p.model,
		Prompt:      prompt,
		Temperature: p.temperature,
		Stream:      false,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.url+"/api/generate", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Ollama API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var ollamaResp OllamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return ollamaResp.Response, nil
}

// Name identifies this provider instance as "ollama".
func (p *OllamaProvider) Name() string {
	return "ollama"
}

// GetModel exposes the configured Ollama model string.
func (p *OllamaProvider) GetModel() string {
	return p.model
}

// Health pings the Ollama daemon to ensure it is responsive.
func (p *OllamaProvider) Health(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.url+"/api/tags", nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("Ollama not available: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Ollama returned status: %d", resp.StatusCode)
	}

	return nil
}

// ListModels queries the local Ollama daemon for all currently pulled models.
func (p *OllamaProvider) ListModels(ctx context.Context) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.url+"/api/tags", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to list models: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	models := make([]string, len(result.Models))
	for i, m := range result.Models {
		models[i] = m.Name
	}

	return models, nil
}

// NewOllamaProviderFromConfig constructs an OllamaProvider using a standard LLMConfig block.
func NewOllamaProviderFromConfig(cfg config.LLMConfig) (*OllamaProvider, error) {
	return NewOllamaProvider(cfg.OllamaURL, cfg.OllamaModel, cfg.Temperature)
}
