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
	// Include a short DATA GAPS section so the LLM knows which sources failed
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

// Matchers for parsing the LLM's Markdown response. They are intentionally lenient:
// small/local models (e.g. Ollama qwen2.5) rarely reproduce the requested headers
// verbatim, so we tolerate variations in heading level, numbering, bold markers,
// and synonyms ("Recommended Action", "Next Steps", "Action Items", "Remediation").
var (
	confidenceRe = regexp.MustCompile(`(?i)\*{0,2}\s*Confidence(?:\s+Score)?\s*\*{0,2}\s*[:\-]\s*\*{0,2}\s*([0-9]{1,3}\s*%?|high|medium|low|confirmed|probable|inconclusive)`)
	actionHeadRe = regexp.MustCompile(`(?im)^\s*#{0,6}\s*\*{0,2}\s*(?:\d+[.)]\s*)?(?:Recommended\s+Actions?|Next\s+Steps?|Action\s+Items?|Remediations?)\b.*$`)
	headingRe    = regexp.MustCompile(`(?m)^\s*#{1,6}\s`)
	bulletRe     = regexp.MustCompile(`^\s*(?:[-*+]|\d+[.)])\s+(.*)$`)
)

// parseLLMResponse extracts structured data from the Markdown response. It degrades
// gracefully: if the model omits a confidence score or an actions section, it falls
// back to sensible defaults and treats the whole response as the root-cause body.
func parseLLMResponse(response string) (rootCause, confidence string, nextSteps []string) {
	confidence = "medium"

	if match := confidenceRe.FindStringSubmatch(response); len(match) > 1 {
		confidence = strings.TrimSpace(match[1])
	}

	// Locate the "recommended actions" section regardless of exact heading text.
	if loc := actionHeadRe.FindStringIndex(response); loc != nil {
		rootCause = strings.TrimSpace(response[:loc[0]])
		nextSteps = parseBullets(response[loc[1]:])
	} else {
		rootCause = strings.TrimSpace(response)
	}

	return rootCause, confidence, nextSteps
}

// parseBullets collects bullet/numbered list items from the start of section until
// the next Markdown heading (which marks a different section).
func parseBullets(section string) []string {
	var steps []string
	for _, line := range strings.Split(section, "\n") {
		if headingRe.MatchString(line) {
			break
		}
		if m := bulletRe.FindStringSubmatch(strings.TrimSpace(line)); len(m) > 1 {
			if step := strings.TrimSpace(m[1]); step != "" {
				steps = append(steps, step)
			}
		}
	}
	return steps
}

// buildContextPrompt creates a detailed RCA prompt with metrics and commits
func (a *Analyzer) buildContextPrompt(ctx *models.AnalysisContext) string {
	// Build the main prompt, then append a DATA GAPS section if any sources failed.
	main := fmt.Sprintf(`
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

	// Append data gaps if present
	if ctx.Errors != nil && len(ctx.Errors) > 0 {
		gaps := "\nDATA GAPS:\n"
		for k, v := range ctx.Errors {
			gaps += fmt.Sprintf("- %s: %s\n", k, v)
		}
		main += gaps
	}

	return main
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
