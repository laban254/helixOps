// Package postmortem orchestrates the AI-driven generation of incident resolution reports.
package postmortem

import (
	"context"
	"fmt"
	"time"
	"github.com/google/uuid"

	"helixops/internal/models"
	"helixops/pkg/llm"
	"helixops/internal/remediation"
)

// Postmortem encapsulates the timeline, context, and actionable takeaways of a resolved incident.
type Postmortem struct {
	ID                 string
	IncidentName       string
	Date               time.Time
	Duration           time.Duration
	RootCause          string
	Impact             string
	DetectionMethod    string
	ActionItems        []string
	RemediationRules   []remediation.Suggestion
	Markdown           string
}

// Generator orchestrates the compilation of metrics, traces, and LLM summaries into a coherent postmortem.
type Generator struct {
	provider llm.Provider
	rules    *remediation.Engine
}

// NewGenerator initializes a Generator with the necessary LLM provider and rule engine dependencies.
func NewGenerator(provider llm.Provider, rules *remediation.Engine) *Generator {
	return &Generator{
		provider: provider,
		rules:    rules,
	}
}

// Generate executes the postmortem creation workflow, invoking the LLM and rule engine concurrently.
func (g *Generator) Generate(ctx context.Context, ac *models.AnalysisContext) (*Postmortem, error) {
	// 1. Get LLM Postmortem Summary
	prompt := g.buildPrompt(ac)
	llmResponse, err := g.provider.Analyze(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("postmortem generation failed: %w", err)
	}

	// 2. Fetch Rule-Based Remediations
	ruleSuggestions := g.rules.GetSuggestions(ac.Alert)

	pm := &Postmortem{
		ID:               uuid.New().String(),
		IncidentName:     fmt.Sprintf("Incident: %s on %s", ac.Alert.Name, ac.ServiceName),
		Date:             time.Now(),
		Duration:         time.Since(ac.Alert.StartedAt),
		RemediationRules: ruleSuggestions,
		// LLM Response acts as the bulk markdown body for now, which we merge below
	}

	// 3. Assemble Markdown
	pm.Markdown = g.assembleMarkdown(pm, llmResponse)

	return pm, nil
}

func (g *Generator) buildPrompt(ctx *models.AnalysisContext) string {
	return fmt.Sprintf(`
You are an expert SRE writing a formal incident postmortem.
An alert that was previously firing has now RESOLVED.

INCIDENT DETAILS:
- Service: %s
- Alert: %s
- Started: %s
- Resolved: %s
- Total Duration: %s

Please write a structured postmortem with the following sections in Markdown:
## 1. Summary
## 2. Impact
## 3. Root Cause Analysis
## 4. Resolution and Recovery
## 5. What went well & What went wrong
## 6. Action Items (LLM Suggested)

Use this alert context to inform your writeup:
- Alert Summary: %s
- Commits found during window: %d
`, 
		ctx.ServiceName, 
		ctx.Alert.Name, 
		ctx.Alert.StartedAt.Format(time.RFC3339),
		time.Now().Format(time.RFC3339),
		time.Since(ctx.Alert.StartedAt).String(),
		ctx.Alert.Summary,
		len(ctx.RecentCommits),
	)
}

func (g *Generator) assembleMarkdown(pm *Postmortem, llmBody string) string {
	md := fmt.Sprintf("# %s\n", pm.IncidentName)
	md += fmt.Sprintf("**Date:** %s\n", pm.Date.Format("2006-01-02 15:04:05"))
	md += fmt.Sprintf("**Duration:** %s\n\n", pm.Duration.String())
	
	md += llmBody + "\n\n"

	md += "## Automated Rule-Based Suggestions\n"
	if len(pm.RemediationRules) == 0 {
		md += "No automated rules matched this incident type.\n"
	} else {
		for _, rule := range pm.RemediationRules {
			md += fmt.Sprintf("### %s\n", rule.Title)
			md += fmt.Sprintf("%s\n\n", rule.Description)
			md += fmt.Sprintf("```bash\n%s\n```\n\n", rule.Action)
		}
	}

	return md
}
