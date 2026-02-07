package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config holds all configuration for the agent
type Config struct {
	App        AppConfig        `mapstructure:"app"`
	Prometheus PrometheusConfig `mapstructure:"prometheus"`
	Loki       LokiConfig       `mapstructure:"loki"`
	GitHub     GitHubConfig     `mapstructure:"github"`
	LLM        LLMConfig        `mapstructure:"llm"`
	Output     OutputConfig     `mapstructure:"output"`
	Analysis   AnalysisConfig   `mapstructure:"analysis"`
}

type AppConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	LogLevel string `mapstructure:"log_level"`
}

type PrometheusConfig struct {
	URL     string `mapstructure:"url"`
	Timeout string `mapstructure:"timeout"`
}

type LokiConfig struct {
	URL     string `mapstructure:"url"`
	Timeout string `mapstructure:"timeout"`
}

type GitHubConfig struct {
	APIURL   string `mapstructure:"api_url"`
	TokenEnv string `mapstructure:"token_env"`
	Token    string `mapstructure:"-"`
}

type LLMConfig struct {
	Provider    string `mapstructure:"provider"`
	Model       string `mapstructure:"model"`
	Temperature float64 `mapstructure:"temperature"`
	MaxTokens   int    `mapstructure:"max_tokens"`
	OllamaURL   string `mapstructure:"ollama_url"`
	OllamaModel string `mapstructure:"ollama_model"`
	APIKey      string `mapstructure:"-"`
}

type OutputConfig struct {
	Slack    SlackOutputConfig    `mapstructure:"slack"`
	Discord  DiscordOutputConfig  `mapstructure:"discord"`
	Markdown MarkdownOutputConfig `mapstructure:"markdown"`
}

type SlackOutputConfig struct {
	WebhookURLEnv string `mapstructure:"webhook_url_env"`
	WebhookURL   string `mapstructure:"-"`
	Enabled      bool   `mapstructure:"enabled"`
}

type DiscordOutputConfig struct {
	WebhookURLEnv string `mapstructure:"webhook_url_env"`
	WebhookURL   string `mapstructure:"-"`
	Enabled      bool   `mapstructure:"enabled"`
}

type MarkdownOutputConfig struct {
	OutputDir string `mapstructure:"output_dir"`
	Enabled   bool   `mapstructure:"enabled"`
}

type AnalysisConfig struct {
	MetricsWindow   string `mapstructure:"metrics_window"`
	CommitsLookback string `mapstructure:"commits_lookback"`
	LogsLookback    string `mapstructure:"logs_lookback"`
}

// GetTimeoutDuration returns the timeout as a time.Duration
func (c *PrometheusConfig) GetTimeoutDuration() time.Duration {
	d, _ := time.ParseDuration(c.Timeout)
	if d == 0 {
		return 30 * time.Second
	}
	return d
}

func (c *LokiConfig) GetTimeoutDuration() time.Duration {
	d, _ := time.ParseDuration(c.Timeout)
	if d == 0 {
		return 30 * time.Second
	}
	return d
}

// GetCommitsLookbackDuration returns the commits lookback as a time.Duration
func (c *AnalysisConfig) GetCommitsLookbackDuration() time.Duration {
	d, _ := time.ParseDuration(c.CommitsLookback)
	if d == 0 {
		return 24 * time.Hour
	}
	return d
}

func (c *AnalysisConfig) GetLogsLookbackDuration() time.Duration {
	d, _ := time.ParseDuration(c.LogsLookback)
	if d == 0 {
		return 1 * time.Hour
	}
	return d
}

func (c *AnalysisConfig) GetMetricsWindowDuration() time.Duration {
	d, _ := time.ParseDuration(c.MetricsWindow)
	if d == 0 {
		return 15 * time.Minute
	}
	return d
}

// Load loads configuration from config.yaml or environment variables
func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")
	viper.AddConfigPath("/etc/helixops")

	// Allow environment variables to override config
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Set defaults
	viper.SetDefault("app.host", "0.0.0.0")
	viper.SetDefault("app.port", 8080)
	viper.SetDefault("app.log_level", "info")
	viper.SetDefault("prometheus.timeout", "30s")
	viper.SetDefault("loki.timeout", "30s")
	viper.SetDefault("llm.provider", "openai")
	viper.SetDefault("llm.model", "gpt-4o")
	viper.SetDefault("llm.temperature", 0.1)
	viper.SetDefault("llm.max_tokens", 1000)
	viper.SetDefault("analysis.metrics_window", "15m")
	viper.SetDefault("analysis.commits_lookback", "24h")
	viper.SetDefault("analysis.logs_lookback", "1h")

	// Read config file
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Get API keys from environment if token_env is set
	if cfg.GitHub.TokenEnv != "" {
		cfg.GitHub.Token = os.Getenv(cfg.GitHub.TokenEnv)
	}

	if cfg.LLM.Provider != "ollama" {
		apiKeyEnv := "OPENAI_API_KEY"
		if cfg.LLM.Provider == "anthropic" {
			apiKeyEnv = "ANTHROPIC_API_KEY"
		}
		cfg.LLM.APIKey = os.Getenv(apiKeyEnv)
	}

	if cfg.Output.Slack.WebhookURLEnv != "" {
		cfg.Output.Slack.WebhookURL = os.Getenv(cfg.Output.Slack.WebhookURLEnv)
	}

	if cfg.Output.Discord.WebhookURLEnv != "" {
		cfg.Output.Discord.WebhookURL = os.Getenv(cfg.Output.Discord.WebhookURLEnv)
	}

	return &cfg, nil
}

// ProviderType returns the LLM provider type
func (c *LLMConfig) ProviderType() string {
	return strings.ToLower(c.Provider)
}
