// Package llm defines the interfaces and factories for connecting to various Large Language Models.
package llm

import (
	"context"
	"fmt"

	"helixops/internal/config"
)

// Provider establishes the common contract for all supported LLM integrations.
type Provider interface {
	Analyze(ctx context.Context, prompt string) (string, error)
	Name() string
}

// ProviderType represents a supported backend LLM provider.
type ProviderType string

const (
	ProviderOpenAI    ProviderType = "openai"
	ProviderAnthropic ProviderType = "anthropic"
	ProviderOllama    ProviderType = "ollama"
)

// NewProvider evaluates the configuration to instantiate and route to the correct LLM backend implementation.
func NewProvider(cfg config.LLMConfig) (Provider, error) {
	providerType := ProviderType(cfg.ProviderType())

	switch providerType {
	case ProviderOpenAI:
		return NewOpenAIProvider(cfg.APIKey, cfg.Model, cfg.Temperature, cfg.MaxTokens)
	case ProviderAnthropic:
		return NewAnthropicProvider(cfg.APIKey, cfg.Model, cfg.Temperature, cfg.MaxTokens)
	case ProviderOllama:
		return NewOllamaProvider(cfg.OllamaURL, cfg.OllamaModel, cfg.Temperature)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", cfg.Provider)
	}
}
