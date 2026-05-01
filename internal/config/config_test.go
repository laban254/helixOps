package config_test

import (
	"strings"
	"testing"

	"helixops/internal/config"
)

func TestValidate_MissingFields(t *testing.T) {
	cfg := &config.Config{}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error for empty config, got nil")
	}
	if !strings.Contains(err.Error(), "prometheus.url") {
		t.Fatalf("expected missing prometheus.url error, got: %v", err)
	}
}

func TestValidate_TempoEnabledMissingURL(t *testing.T) {
	cfg := &config.Config{
		App:        config.AppConfig{LogLevel: "info"},
		Prometheus: config.PrometheusConfig{URL: "http://prom:9090"},
		Loki:       config.LokiConfig{URL: "http://loki:3100"},
		Tempo:      config.TempoConfig{Enabled: true, URL: ""},
		LLM:        config.LLMConfig{Provider: "openai", APIKey: "key"},
	}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error for tempo.enabled without URL, got nil")
	}
	if !strings.Contains(err.Error(), "tempo.url") {
		t.Fatalf("expected tempo.url error, got: %v", err)
	}
}

func TestValidate_DatabaseRequiredFields(t *testing.T) {
	cfg := &config.Config{
		App:        config.AppConfig{LogLevel: "info"},
		Prometheus: config.PrometheusConfig{URL: "http://prom:9090"},
		Loki:       config.LokiConfig{URL: "http://loki:3100"},
		LLM:        config.LLMConfig{Provider: "openai", APIKey: "key"},
		Database:   config.DatabaseConfig{Enabled: true},
	}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error for missing DB fields, got nil")
	}
	if !strings.Contains(err.Error(), "database.host") {
		t.Fatalf("expected database.host error, got: %v", err)
	}
}

func TestValidate_Success(t *testing.T) {
	cfg := &config.Config{
		App:        config.AppConfig{LogLevel: "info"},
		Prometheus: config.PrometheusConfig{URL: "http://prom:9090"},
		Loki:       config.LokiConfig{URL: "http://loki:3100"},
		LLM:        config.LLMConfig{Provider: "openai", APIKey: "key"},
		Output:     config.OutputConfig{Markdown: config.MarkdownOutputConfig{Enabled: false}},
	}
	err := cfg.Validate()
	if err != nil {
		t.Fatalf("expected no validation error, got: %v", err)
	}
}
