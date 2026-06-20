// Package config provides configuration structures and loading logic for HelixOps.
package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config represents the root configuration structure for the HelixOps agent.
type Config struct {
	App        AppConfig        `mapstructure:"app"`
	Prometheus PrometheusConfig `mapstructure:"prometheus"`
	Loki       LokiConfig       `mapstructure:"loki"`
	Tempo      TempoConfig      `mapstructure:"tempo"`
	GitHub     GitHubConfig     `mapstructure:"github"`
	LLM        LLMConfig        `mapstructure:"llm"`
	Output     OutputConfig     `mapstructure:"output"`
	Analysis   AnalysisConfig   `mapstructure:"analysis"`
	Database   DatabaseConfig   `mapstructure:"database"`
}

// AppConfig defines application-level settings such as host and port.
type AppConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	LogLevel string `mapstructure:"log_level"`
}

// PrometheusConfig defines connection and timeout settings for the Prometheus TSDB.
type PrometheusConfig struct {
	URL     string `mapstructure:"url"`
	Timeout string `mapstructure:"timeout"`
}

// LokiConfig defines connection and timeout settings for the Grafana Loki log aggregation system.
type LokiConfig struct {
	URL     string `mapstructure:"url"`
	Timeout string `mapstructure:"timeout"`
}

// TempoConfig defines connection settings for the Grafana Tempo distributed tracing backend.
type TempoConfig struct {
	URL                 string `mapstructure:"url"`
	Timeout             string `mapstructure:"timeout"`
	Enabled             bool   `mapstructure:"enabled"`
	SlowSpanThresholdMs int    `mapstructure:"slow_span_threshold_ms"`
	SearchLimit         int    `mapstructure:"search_limit"`
}

// GitHubConfig defines settings for interacting with the GitHub REST API.
type GitHubConfig struct {
	APIURL         string            `mapstructure:"api_url"`
	TokenEnv       string            `mapstructure:"token_env"`
	Token          string            `mapstructure:"-"`
	DefaultOrg     string            `mapstructure:"default_org"`
	ServiceMapping map[string]string `mapstructure:"service_mapping"` // service_name -> owner/repo
}

// LLMConfig defines the selected Language Model provider and its operational parameters.
type LLMConfig struct {
	Provider    string  `mapstructure:"provider"`
	Model       string  `mapstructure:"model"`
	Temperature float64 `mapstructure:"temperature"`
	MaxTokens   int     `mapstructure:"max_tokens"`
	OllamaURL   string  `mapstructure:"ollama_url"`
	OllamaModel string  `mapstructure:"ollama_model"`
	APIKey      string  `mapstructure:"-"`
}

// OutputConfig defines the notification channels and serialization targets for RCA reports.
type OutputConfig struct {
	Slack    SlackOutputConfig    `mapstructure:"slack"`
	Markdown MarkdownOutputConfig `mapstructure:"markdown"`
	// Future: Discord, Teams, PagerDuty, Webhooks
}

// SlackOutputConfig defines settings for the Slack incoming webhook integration.
type SlackOutputConfig struct {
	WebhookURLEnv string `mapstructure:"webhook_url_env"`
	WebhookURL    string `mapstructure:"-"`
	Enabled       bool   `mapstructure:"enabled"`
}

// MarkdownOutputConfig defines settings for locally generating Markdown incident reports.
type MarkdownOutputConfig struct {
	OutputDir string `mapstructure:"output_dir"`
	Enabled   bool   `mapstructure:"enabled"`
}

// AnalysisConfig defines the time boundaries and lookback windows for fetching RCA data.
type AnalysisConfig struct {
	MetricsWindow   string `mapstructure:"metrics_window"`
	CommitsLookback string `mapstructure:"commits_lookback"`
	LogsLookback    string `mapstructure:"logs_lookback"`
}

// DatabaseConfig defines database settings (supports postgres and sqlite).
type DatabaseConfig struct {
	Type     string `mapstructure:"type"`     // "postgres" or "sqlite"
	Path     string `mapstructure:"path"`     // SQLite file path (used when type=sqlite)
	Host     string `mapstructure:"host"`     // PostgreSQL host
	Port     int    `mapstructure:"port"`     // PostgreSQL port
	User     string `mapstructure:"user"`     // PostgreSQL user
	Password string `mapstructure:"-"`        // from env var
	DBName   string `mapstructure:"dbname"`   // PostgreSQL database name
	SSLMode  string `mapstructure:"sslmode"`  // PostgreSQL SSL mode
	Enabled  bool   `mapstructure:"enabled"`
}

// GetTimeoutDuration returns the timeout as a time.Duration
func (c *PrometheusConfig) GetTimeoutDuration() time.Duration {
	d, _ := time.ParseDuration(c.Timeout)
	if d == 0 {
		return 30 * time.Second
	}
	return d
}

// GetTimeoutDuration parses the configured string timeout into a time.Duration.
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

// GetLogsLookbackDuration parses the configured log lookback window into a time.Duration.
func (c *AnalysisConfig) GetLogsLookbackDuration() time.Duration {
	d, _ := time.ParseDuration(c.LogsLookback)
	if d == 0 {
		return 1 * time.Hour
	}
	return d
}

// GetMetricsWindowDuration parses the configured metrics gathering window into a time.Duration.
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
	viper.SetDefault("tempo.timeout", "30s")
	viper.SetDefault("tempo.enabled", true)
	viper.SetDefault("tempo.slow_span_threshold_ms", 500)
	viper.SetDefault("tempo.search_limit", 20)
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

	if cfg.LLM.ProviderType() != "ollama" {
		apiKeyEnv := "OPENAI_API_KEY"
		switch cfg.LLM.ProviderType() {
		case "anthropic":
			apiKeyEnv = "ANTHROPIC_API_KEY"
		case "deepseek":
			apiKeyEnv = "DEEPSEEK_API_KEY"
		}
		cfg.LLM.APIKey = os.Getenv(apiKeyEnv)
	}

	if cfg.Output.Slack.WebhookURLEnv != "" {
		cfg.Output.Slack.WebhookURL = os.Getenv(cfg.Output.Slack.WebhookURLEnv)
	}

	return &cfg, nil
}

// Validate checks required configuration fields and returns an error describing any missing fields.
func (c *Config) Validate() error {
	var errs []string

	if strings.TrimSpace(c.Prometheus.URL) == "" {
		errs = append(errs, "prometheus.url is required")
	}
	if strings.TrimSpace(c.Loki.URL) == "" {
		errs = append(errs, "loki.url is required")
	}
	if c.Tempo.Enabled && strings.TrimSpace(c.Tempo.URL) == "" {
		errs = append(errs, "tempo.url is required when tempo.enabled is true")
	}
	if strings.TrimSpace(c.App.LogLevel) == "" {
		errs = append(errs, "app.log_level is required")
	}
	if c.LLM.ProviderType() != "ollama" && strings.TrimSpace(c.LLM.APIKey) == "" {
		errs = append(errs, "LLM API key is missing for provider: "+c.LLM.Provider)
	}
	if c.Output.Markdown.Enabled && strings.TrimSpace(c.Output.Markdown.OutputDir) == "" {
		errs = append(errs, "output.markdown.output_dir is required when markdown output is enabled")
	}
	if c.Database.Enabled {
		switch strings.ToLower(c.Database.Type) {
		case "sqlite":
			if strings.TrimSpace(c.Database.Path) == "" {
				errs = append(errs, "database.path is required when using sqlite")
			}
		case "postgres", "":
			if strings.TrimSpace(c.Database.Host) == "" {
				errs = append(errs, "database.host is required when database.enabled is true")
			}
			if c.Database.Port == 0 {
				errs = append(errs, "database.port is required when database.enabled is true")
			}
			if strings.TrimSpace(c.Database.User) == "" {
				errs = append(errs, "database.user is required when database.enabled is true")
			}
			if strings.TrimSpace(c.Database.DBName) == "" {
				errs = append(errs, "database.dbname is required when database.enabled is true")
			}
		default:
			errs = append(errs, "database.type must be 'postgres' or 'sqlite'")
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("configuration validation failed: %s", strings.Join(errs, "; "))
	}
	return nil
}

// ProviderType returns the LLM provider type
func (c *LLMConfig) ProviderType() string {
	return strings.ToLower(c.Provider)
}
