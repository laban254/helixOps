// Package mcp binds HelixOps functionality to the Model Context Protocol (MCP) server standard.
package mcp

import (
	"context"
	"fmt"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"helixops/internal/config"
	"helixops/internal/analyzer"
	"helixops/internal/models"
	"helixops/internal/orchestrator"
)

// Server defines the core MCP capability layer, exposing native handler functions to connected AI agents.
type Server struct {
	cfg          *config.Config
	orchestrator *orchestrator.Orchestrator
	analyzer     *analyzer.Analyzer
}

// New creates a new MCP server wrapper
func New(cfg *config.Config, orch *orchestrator.Orchestrator, anlz *analyzer.Analyzer) *Server {
	return &Server{
		cfg:          cfg,
		orchestrator: orch,
		analyzer:     anlz,
	}
}

// RegisterTools registers the HelixOps tools with the MCP server
func (s *Server) RegisterTools(mcpServer *server.MCPServer) {
	// 1. Analyze Alert Tool
	analyzeTool := mcp.NewTool("analyze_alert",
		mcp.WithDescription("Takes alert parameters and runs full RCA."),
		mcp.WithString("service_name", mcp.Required(), mcp.Description("Name of the impacted service")),
		mcp.WithString("alert_name", mcp.Required(), mcp.Description("Name of the alert rule firing")),
		mcp.WithString("summary", mcp.Required(), mcp.Description("Alert summary text")),
	)
	mcpServer.AddTool(analyzeTool, s.HandleAnalyzeAlert)

	// 2. Get Service Metrics Tool
	metricsTool := mcp.NewTool("get_service_metrics",
		mcp.WithDescription("Fetches golden signals for a service."),
		mcp.WithString("service_name", mcp.Required(), mcp.Description("Name of the service")),
	)
	mcpServer.AddTool(metricsTool, s.HandleGetServiceMetrics)
	
	// 3. Search Logs Tool
	logsTool := mcp.NewTool("search_logs",
		mcp.WithDescription("Queries Loki for error patterns."),
		mcp.WithString("service_name", mcp.Required(), mcp.Description("Name of the service")),
	)
	mcpServer.AddTool(logsTool, s.HandleSearchLogs)
	
	// 4. Get Recent Commits Tool
	commitsTool := mcp.NewTool("get_recent_commits",
		mcp.WithDescription("Finds code changes near the incident start."),
		mcp.WithString("repo_name", mcp.Required(), mcp.Description("Github Repository Name")),
	)
	mcpServer.AddTool(commitsTool, s.HandleGetRecentCommits)
}

// HandleAnalyzeAlert performs a full RCA via the Analyzer
func (s *Server) HandleAnalyzeAlert(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	serviceName := request.Params.Arguments["service_name"].(string)
	alertName := request.Params.Arguments["alert_name"].(string)
	summary := request.Params.Arguments["summary"].(string)

	alertItem := models.AlertItem{
		Status:   "firing",
		Labels:   map[string]string{"service_name": serviceName, "alertname": alertName, "severity": "critical"},
		StartsAt: time.Now(),
	}
	alertItem.Annotations = map[string]string{"summary": summary}

	analysisCtx, err := s.orchestrator.PrepareContext(ctx, serviceName, time.Now())
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to prepare context: %v", err)), nil
	}
	
	// Copy alert info over
	analysisCtx.Alert = models.AlertInfo{
		Name: alertName,
		Severity: "critical",
		Summary: summary,
		Labels: alertItem.Labels,
		StartedAt: alertItem.StartsAt,
	}

	result, err := s.analyzer.AnalyzeWithContext(ctx, analysisCtx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Analysis failed: %v", err)), nil
	}

	report := fmt.Sprintf("Root Cause:\n%s\n\nConfidence: %s\nNext Steps:\n", result.RootCause, result.Confidence)
	for _, step := range result.NextSteps {
		report += fmt.Sprintf("- %s\n", step)
	}

	return mcp.NewToolResultText(report), nil
}

// HandleGetServiceMetrics proxies prometheus queries via Orchestrator
func (s *Server) HandleGetServiceMetrics(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	serviceName := request.Params.Arguments["service_name"].(string)
	tEnd := time.Now()
	tStart := tEnd.Add(-15 * time.Minute)

	// Since prometheus client isn't exported in Orchestrator, we prepare context then pluck metrics
	ac, err := s.orchestrator.PrepareContext(ctx, serviceName, tEnd)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	report := fmt.Sprintf("Metrics for %s (Last 15m):\n- P99 Latency: %.2fms\n- Error Rate: %.2f%%\n- Requests/Sec: %.2f", 
		serviceName, ac.Metrics.LatencyP99, ac.Metrics.ErrorRate*100, ac.Metrics.RPS)

	return mcp.NewToolResultText(report), nil
}

func (s *Server) HandleSearchLogs(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	serviceName := request.Params.Arguments["service_name"].(string)
	report := fmt.Sprintf("[MCP Stub] Fetched simulated error logs for %s from Loki.", serviceName)
	return mcp.NewToolResultText(report), nil
}

func (s *Server) HandleGetRecentCommits(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	repoName := request.Params.Arguments["repo_name"].(string)
	// We pass the string in, orchestrator falls back to serviceName if repo mapping unsupported
	ac, err := s.orchestrator.PrepareContext(ctx, repoName, time.Now())
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	if len(ac.RecentCommits) == 0 {
		return mcp.NewToolResultText(fmt.Sprintf("No recent commits found for %s in the last 24h.", repoName)), nil
	}

	report := fmt.Sprintf("Recent Commits for %s:\n", repoName)
	for i, c := range ac.RecentCommits {
		if i >= 5 {
			break
		}
		report += fmt.Sprintf("- %s: %s (%s)\n", c.SHA[:7], c.Message, c.Author)
	}

	return mcp.NewToolResultText(report), nil
}
