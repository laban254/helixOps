// Package analyzer defines the core LLM-based root cause analysis component.
package analyzer

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"helixops/internal/models"
	"helixops/internal/clients/tempo"
	"helixops/pkg/llm"
)

// Analyzer utilizes an underlying LLM provider to perform Root Cause Analysis on incident data.
type Analyzer struct {
	provider llm.Provider
}

// New initializes a new Analyzer with the given LLM provider.
func New(provider llm.Provider) *Analyzer {
	return &Analyzer{
		provider: provider,
	}
}

// Analyze performs a rapid RCA on a firing alert without full diagnostic context.
func (a *Analyzer) Analyze(ctx context.Context, alert models.AlertItem) (*models.AnalysisResult, error) {
	// Build prompt
	prompt := a.buildPrompt(alert)

	// Call LLM
	response, err := a.provider.Analyze(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("LLM analysis failed: %w", err)
	}

	// Parse response
	result := &models.AnalysisResult{
		ID:          uuid.New().String(),
		ServiceName: alert.GetLabel("service_name"),
		AlertName:   alert.Labels["alertname"],
		Severity:    alert.Labels["severity"],
		Summary:     alert.GetAnnotation("summary"),
		RootCause:   response,
		Confidence:  "medium",
		AnalyzedAt:  time.Now(),
	}

	return result, nil
}

// buildPrompt creates the RCA prompt for the LLM
func (a *Analyzer) buildPrompt(alert models.AlertItem) string {
	return fmt.Sprintf(`
You are an SRE analyzing an incident. Given the following alert data, identify the most likely root cause.

ALERT:
- Service: %s
- Alert Name: %s
- Severity: %s
- Started: %s
- Summary: %s

Based on this data, provide:
1. Most likely root cause (2-3 sentences)
2. Confidence level (high/medium/low)
3. Suggested next steps (3 bullet points)

Respond in JSON format:
{
  "root_cause": "...",
  "confidence": "...",
  "next_steps": ["...", "...", "..."]
}
`,
		alert.GetLabel("service_name"),
		alert.Labels["alertname"],
		alert.Labels["severity"],
		alert.StartsAt.Format(time.RFC3339),
		alert.GetAnnotation("summary"),
	)
}

// AnalyzeWithContext performs a comprehensive RCA utilizing metrics, distributed traces, logs, and recent code commits.
func (a *Analyzer) AnalyzeWithContext(ctx context.Context, ctxData *models.AnalysisContext) (*models.AnalysisResult, error) {
	prompt := a.buildContextPrompt(ctxData)

	response, err := a.provider.Analyze(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("LLM analysis failed: %w", err)
	}

	result := &models.AnalysisResult{
		ID:          uuid.New().String(),
		ServiceName: ctxData.ServiceName,
		AlertName:   ctxData.Alert.Name,
		Severity:    ctxData.Alert.Severity,
		Summary:     ctxData.Alert.Summary,
		RootCause:   response,
		Metrics:     ctxData.Metrics,
		Commits:     ctxData.RecentCommits,
		Confidence:  "medium",
		AnalyzedAt:  time.Now(),
	}

	return result, nil
}

// buildContextPrompt creates a detailed RCA prompt with metrics and commits
func (a *Analyzer) buildContextPrompt(ctx *models.AnalysisContext) string {
	return fmt.Sprintf(`
You are an SRE analyzing an incident. Given the following data, identify the most likely root cause.

ALERT:
- Service: %s
- Alert Name: %s
- Severity: %s
- Started: %s
- Summary: %s

METRICS:
- Latency P99: %.2fms
- Error Rate: %.2f%%
- Requests/sec: %.2f

BASELINE:
- Latency: %.2fms
- Error Rate: %.2f%%

DISTRIBUTED TRACES:
- P99 Latency: %.2fms
- Slow Spans (>500ms): %d
- Error Spans: %d

%s

RECENT COMMITS (%d commits):
%s

Based on this data, provide:
1. Most likely root cause (2-3 sentences)
2. Confidence level (high/medium/low)
3. Suggested next steps (3 bullet points)

Respond in JSON format:
{
  "root_cause": "...",
  "confidence": "...",
  "next_steps": ["...", "...", "..."]
}
`,
		ctx.ServiceName,
		ctx.Alert.Name,
		ctx.Alert.Severity,
		ctx.Alert.StartedAt.Format(time.RFC3339),
		ctx.Alert.Summary,
		ctx.Metrics.LatencyP99,
		ctx.Metrics.ErrorRate*100,
		ctx.Metrics.RPS,
		ctx.Metrics.BaselineLatency,
		ctx.Metrics.BaselineErrorRate*100,
		ctx.Traces.P99Latency,
		len(ctx.Traces.SlowSpans),
		len(ctx.Traces.ErrorSpans),
		formatSpans(ctx.Traces.SlowSpans),
		len(ctx.RecentCommits),
		formatCommits(ctx.RecentCommits),
	)
}

// formatCommits formats commits for the prompt
func formatCommits(commits []models.CommitInfo) string {
	if len(commits) == 0 {
		return "No recent commits found."
	}

	result := ""
	for i, c := range commits {
		if i >= 10 {
			break
		}
		result += fmt.Sprintf("- %s: %s (by %s)\n", c.SHA[:7], truncate(c.Message, 50), c.Author)
	}
	return result
}

// truncate truncates a string
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// formatSpans formats spans for the prompt
func formatSpans(spans []tempo.Span) string {
	if len(spans) == 0 {
		return ""
	}

	result := ""
	for i, s := range spans {
		if i >= 10 { // limit to top 10 spans
			break
		}
		result += fmt.Sprintf("- Service: %s\n  Operation: %s\n  Duration: %dms\n  Status: %s\n", s.ServiceName, s.OperationName, s.DurationMs, s.Status)
	}
	return result
}
