// Package remediation provides a fast, rule-based engine for suggesting incident fixes.
package remediation

import (
	"strings"
	"helixops/internal/models"
)

// Suggestion defines an actionable, context-aware remediation step for an alert.
type Suggestion struct {
	Title       string
	Description string
	Action      string // E.g., a CLI command, link, or Terraform snippet
}

// Engine evaluates incoming alerts against a set of predefined heuristic rules.
type Engine struct{}

// NewEngine initializes a generic heuristic remediation engine.
func NewEngine() *Engine {
	return &Engine{}
}

// GetSuggestions parses the alert's labels and triggers any matching heuristic rules for immediate action.
func (e *Engine) GetSuggestions(alert models.AlertInfo) []Suggestion {
	var suggestions []Suggestion
	alertName := strings.ToLower(alert.Name)

	if strings.Contains(alertName, "highlatency") || strings.Contains(alertName, "latency") {
		suggestions = append(suggestions, Suggestion{
			Title:       "Check Database Query Performance",
			Description: "High latency is often caused by unoptimized queries or missing indexes.",
			Action:      "Review slow query logs in your database provider or check APM traces for bottleneck spans.",
		})
		suggestions = append(suggestions, Suggestion{
			Title:       "Scale Up Service Replicas",
			Description: "If CPU/Memory is also high, the service might be underprovisioned for current traffic.",
			Action:      "kubectl scale deployment " + alert.Labels["service_name"] + " --replicas=3",
		})
	}

	if strings.Contains(alertName, "errorrate") || strings.Contains(alertName, "high_error_rate") {
		suggestions = append(suggestions, Suggestion{
			Title:       "Investigate Recent Deployments",
			Description: "Spikes in error rates strongly correlate with recent code deployments.",
			Action:      "Check GitHub Actions or ArgoCD for recent rollouts to this service.",
		})
		suggestions = append(suggestions, Suggestion{
			Title:       "Check Downstream Dependencies",
			Description: "Ensure that upstream endpoints or databases are not rejecting connections or timing out.",
			Action:      "Review error logs in Loki for 'connection refused' or 'timeout' errors.",
		})
	}

	if strings.Contains(alertName, "cpu") || strings.Contains(alertName, "throttling") {
		suggestions = append(suggestions, Suggestion{
			Title:       "Review CPU Limits",
			Description: "The container might be getting heavily throttled by Kubernetes CPU limits.",
			Action:      "Consider increasing the CPU limit in the pod's resources configuration.",
		})
	}

	if strings.Contains(alertName, "memory") || strings.Contains(alertName, "oom") {
		suggestions = append(suggestions, Suggestion{
			Title:       "Investigate Memory Leaks",
			Description: "If memory climbs steadily until OOMKilled, there may be a memory leak.",
			Action:      "Capture a heap profile (pprof) and analyze memory allocations.",
		})
	}

	return suggestions
}
