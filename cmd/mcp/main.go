// Package main provides the entry point for the HelixOps MCP (Model Context Protocol) server.
package main

import (
	"log/slog"
	"os"

	"helixops/internal/analyzer"
	"helixops/internal/clients/github"
	"helixops/internal/clients/loki"
	"helixops/internal/clients/prometheus"
	"helixops/internal/config"
	"helixops/internal/logging"
	mcpsrv "helixops/internal/mcp"
	"helixops/internal/orchestrator"
	"helixops/pkg/llm"

	"github.com/mark3labs/mcp-go/server"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("config.load_failed", "error", err)
		os.Exit(1)
	}

	// Initialize the minimal set of clients required to run the MCP tools.
	promClient := prometheus.NewClient(cfg.Prometheus.URL, cfg.Prometheus.GetTimeoutDuration())
	githubClient := github.NewClient(cfg.GitHub.APIURL, cfg.GitHub.Token)
	lokiClient := loki.NewClient(cfg.Loki.URL, cfg.Loki.GetTimeoutDuration())

	llmProvider, err := llm.NewProvider(cfg.LLM)
	// Initialize structured logging
	logging.Init("helixops-mcp", cfg.App.LogLevel)
	slog.Info("starting", "service", "helixops-mcp")
	if err != nil {
		slog.Error("llm.provider_failed", "error", err)
		os.Exit(1)
	}

	orch := orchestrator.New(promClient, githubClient, lokiClient, nil, cfg)
	anlz := analyzer.New(llmProvider)

	// Initialize the core MCP server instance.
	s := server.NewMCPServer(
		"helixops-mcp",
		"1.0.0",
	)

	// Bind HelixOps specific tools (Metrics, RCA, Logs, Commits) to the MCP server.
	helixServerWrapper := mcpsrv.New(cfg, orch, anlz)
	helixServerWrapper.RegisterTools(s)

	slog.Info("HelixOps MCP Server listening on stdio...")
	// Start serving the MCP protocol over standard input/output streams.
	if err := server.ServeStdio(s); err != nil {
		slog.Error("server.error", "error", err)
		os.Exit(1)
	}
}
