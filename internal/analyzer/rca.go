// Package analyzer defines the core LLM-based root cause analysis component.
package analyzer

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"helixops/internal/clients/tempo"
	"helixops/internal/models"
	"helixops/pkg/llm"

	"github.com/google/uuid"
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
### ROLE
You are the Lead SRE Investigator for HelixOps. Your mission is to perform a high-fidelity Root Cause Analysis (RCA) based on provided Telemetry Context.

### OPERATIONAL CONSTRAINTS
1. EVIDENCE-ONLY: Never assume a cause. Every claim must be backed by a specific log entry, a metric spike, or a code diff provided in the context.
2. ADMIT IGNORANCE: If the provided data is insufficient to identify the root cause, state "INSUFFICIENT DATA" and list specifically what is missing.
3. NO HALLUCINATION: Do not invent service names, error codes, or timestamps. Use only what is in the prompt context.

### OUTPUT FORMAT (Markdown)
Your response must strictly follow this structure:

# Incident Analysis: [Brief Title]
**Confidence Score:** [0-100%%]
**Status:** [Confirmed / Probable / Inconclusive]

## 1. Executive Summary
[A 2-sentence summary of what happened and the immediate impact.]

## 2. Evidence Trail
- **Metric Spike:** [Describe metric change and timestamp]
- **Key Log Entry:** [Quote the specific log line]
- **Suspect Commit:** [Commit Hash/Author] - [Briefly explain the link]

## 3. Root Cause Analysis
[Detailed explanation of the failure chain.]

## 4. Recommended Action
- [Immediate Mitigation Step]
- [Long-term Prevention Step]

---
TELEMETRY CONTEXT:

ALERT:
- Service: %s
- Alert Name: %s
- Severity: %s
- Started: %s
- Summary: %s
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

	// Parse JSON response to extract structured data
	rootCause, confidence, nextSteps := parseLLMResponse(response)

	result := &models.AnalysisResult{
		ID:          uuid.New().String(),
		ServiceName: ctxData.ServiceName,
		AlertName:   ctxData.Alert.Name,
		Severity:    ctxData.Alert.Severity,
		Summary:     ctxData.Alert.Summary,
		RootCause:   rootCause,
		Metrics:     ctxData.Metrics,
		Commits:     ctxData.RecentCommits,
		Confidence:  confidence,
		NextSteps:   nextSteps,
		AnalyzedAt:  time.Now(),
	}

	return result, nil
}

// parseLLMResponse extracts structured data from the Markdown response
func parseLLMResponse(response string) (rootCause, confidence string, nextSteps []string) {
	confidence = "medium"

	// Extract Confidence Score
	confRe := regexp.MustCompile(`(?i)\*\*Confidence Score:\*\*\s*(.+)`)
	if match := confRe.FindStringSubmatch(response); len(match) > 1 {
		confidence = strings.TrimSpace(match[1])
	}

	// Extract Next Steps (Recommended Action)
	actionSplit := strings.Split(response, "## 4. Recommended Action")
	if len(actionSplit) > 1 {
		lines := strings.Split(actionSplit[1], "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ") {
				nextSteps = append(nextSteps, strings.TrimPrefix(strings.TrimPrefix(line, "- "), "* "))
			}
		}
	}

	// Set RootCause as the main body of analysis to be embedded into Slack/Markdown format
	if len(actionSplit) > 0 {
		rootCause = strings.TrimSpace(actionSplit[0])
	} else {
		rootCause = strings.TrimSpace(response)
	}

	return rootCause, confidence, nextSteps
}

// buildContextPrompt creates a detailed RCA prompt with metrics and commits
func (a *Analyzer) buildContextPrompt(ctx *models.AnalysisContext) string {
	return fmt.Sprintf(`
### ROLE
You are the Lead SRE Investigator for HelixOps. Your mission is to perform a high-fidelity Root Cause Analysis (RCA) based on provided Telemetry Context (Metrics, Logs, and Git Commits).

### OPERATIONAL CONSTRAINTS
1. EVIDENCE-ONLY: Never assume a cause. Every claim must be backed by a specific log entry, a metric spike, or a code diff provided in the context.
2. ADMIT IGNORANCE: If the provided data is insufficient to identify the root cause, state "INSUFFICIENT DATA" and list specifically what is missing.
3. NO HALLUCINATION: Do not invent service names, error codes, or timestamps. Use only what is in the prompt context.

### OUTPUT FORMAT (Markdown)
Your response must strictly follow this structure:

# Incident Analysis: [Brief Title]
**Confidence Score:** [0-100%%]
**Status:** [Confirmed / Probable / Inconclusive]

## 1. Executive Summary
[A 2-sentence summary of what happened and the immediate impact.]

## 2. Evidence Trail
- **Metric Spike:** [Describe metric change and timestamp]
- **Key Log Entry:** [Quote the specific log line]
- **Suspect Commit:** [Commit Hash/Author] - [Briefly explain the link]

## 3. Root Cause Analysis
[Detailed explanation of the failure chain.]

## 4. Recommended Action
- [Immediate Mitigation Step]
- [Long-term Prevention Step]

---
TELEMETRY CONTEXT:

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
