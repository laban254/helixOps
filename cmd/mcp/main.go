// Package main provides the entry point for the HelixOps MCP (Model Context Protocol) server.
package main

import (
	"log"
	"log/slog"
	
	"github.com/mark3labs/mcp-go/server"
	"helixops/internal/config"
	mcpsrv "helixops/internal/mcp"
	"helixops/internal/orchestrator"
	"helixops/internal/analyzer"
	"helixops/pkg/llm"
	"helixops/internal/clients/prometheus"
	"helixops/internal/clients/github"
	"helixops/internal/clients/loki"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize the minimal set of clients required to run the MCP tools.
	promClient := prometheus.NewClient(cfg.Prometheus.URL, cfg.Prometheus.GetTimeoutDuration())
	githubClient := github.NewClient(cfg.GitHub.APIURL, cfg.GitHub.Token)
	lokiClient := loki.NewClient(cfg.Loki.URL, cfg.Loki.GetTimeoutDuration())

	llmProvider, err := llm.NewProvider(cfg.LLM)
	if err != nil {
		log.Fatalf("Failed to create LLM provider: %v", err)
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
		log.Fatalf("Server error: %v", err)
	}
}
