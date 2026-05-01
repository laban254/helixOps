// Package orchestrator coordinates data collection from PromQL, Loki, GitHub, and Tempo to build diagnostic context.
package orchestrator

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"helixops/internal/clients/github"
	"helixops/internal/clients/loki"
	"helixops/internal/clients/prometheus"
	"helixops/internal/clients/tempo"
	"helixops/internal/config"
	"helixops/internal/models"
)

// Orchestrator coordinates asynchronous data collection from multiple external APIs to build a unified incident context.
type Orchestrator struct {
	promClient   *prometheus.Client
	githubClient *github.Client
	lokiClient   *loki.Client
	tempoClient  *tempo.Client
	cfg          *config.Config
}

// New initializes a new Orchestrator instance with the necessary infrastructure clients.
func New(prom *prometheus.Client, gh *github.Client, loki *loki.Client, tempoClient *tempo.Client, cfg *config.Config) *Orchestrator {
	return &Orchestrator{
		promClient:   prom,
		githubClient: gh,
		lokiClient:   loki,
		tempoClient:  tempoClient,
		cfg:          cfg,
	}
}

// PrepareContext gathers metrics, traces, and commits concurrently for a given service within an incident time window.
func (o *Orchestrator) PrepareContext(ctx context.Context, serviceName string, alertTime time.Time) (*models.AnalysisContext, error) {
	slog.Info("prepare_context.start", "service", serviceName)

	// Calculate time windows
	metricsWindow := o.cfg.Analysis.GetMetricsWindowDuration()
	commitsLookback := o.cfg.Analysis.GetCommitsLookbackDuration()
	logsLookback := o.cfg.Analysis.GetLogsLookbackDuration()

	metricsStart := alertTime.Add(-metricsWindow)
	metricsEnd := alertTime

	commitsSince := alertTime.Add(-commitsLookback)
	logsStart := alertTime.Add(-logsLookback)

	// Fetch data concurrently, tagging each result with its source so we can
	// capture non-fatal errors per-source and continue processing.
	type result struct {
		source  string
		metrics models.MetricsSummary
		commits []models.CommitInfo
		traces  tempo.TraceContext
		logs    []models.LogEntry
		err     error
	}

	resultCh := make(chan result, 4)

	go func() {
		metrics, err := o.fetchMetrics(ctx, serviceName, metricsStart, metricsEnd)
		resultCh <- result{source: "prometheus", metrics: metrics, err: err}
	}()

	go func() {
		commits, err := o.fetchCommits(ctx, serviceName, commitsSince)
		resultCh <- result{source: "github", commits: commits, err: err}
	}()

	go func() {
		traces, err := o.fetchTraces(ctx, serviceName, metricsStart, metricsEnd)
		resultCh <- result{source: "tempo", traces: traces, err: err}
	}()

	go func() {
		logs, err := o.fetchLogs(ctx, serviceName, logsStart, metricsEnd)
		resultCh <- result{source: "loki", logs: logs, err: err}
	}()

	ctxResult := &models.AnalysisContext{
		ServiceName: serviceName,
		TimeWindow: models.TimeWindow{
			Start:    metricsStart,
			End:      metricsEnd,
			Duration: metricsWindow.String(),
		},
		Errors: make(map[string]string),
	}

	for i := 0; i < 4; i++ {
		r := <-resultCh
		if r.err != nil {
			// Record non-fatal error by source so analyzer + operators can see gaps
			ctxResult.Errors[r.source] = r.err.Error()
			slog.Warn("fetch_error", "source", r.source, "error", r.err)
		}
		if len(r.commits) > 0 && len(ctxResult.RecentCommits) == 0 {
			ctxResult.RecentCommits = r.commits
		}
		if (r.metrics.LatencyP99 > 0 || r.metrics.ErrorRate > 0) && (ctxResult.Metrics == models.MetricsSummary{}) {
			ctxResult.Metrics = r.metrics
		}
		if r.traces.TraceCount > 0 && ctxResult.Traces.TraceCount == 0 {
			ctxResult.Traces = r.traces
		}
		if len(r.logs) > 0 && len(ctxResult.ErrorLogs) == 0 {
			ctxResult.ErrorLogs = r.logs
		}
	}

	slog.Info("prepare_context.done", "service", serviceName, "errors", ctxResult.Errors)
	return ctxResult, nil
}

// fetchMetrics retrieves golden signals metrics from Prometheus
func (o *Orchestrator) fetchMetrics(ctx context.Context, serviceName string, start, end time.Time) (models.MetricsSummary, error) {
	metrics := models.MetricsSummary{}

	var errs []string

	latency, err := o.promClient.QueryLatencyP99(ctx, serviceName, start, end)
	if err != nil {
		errs = append(errs, fmt.Sprintf("latency: %v", err))
	} else {
		metrics.LatencyP99 = latency
	}

	errorRate, err := o.promClient.QueryErrorRate(ctx, serviceName, start, end)
	if err != nil {
		errs = append(errs, fmt.Sprintf("error_rate: %v", err))
	} else {
		metrics.ErrorRate = errorRate
	}

	rps, err := o.promClient.QueryRPS(ctx, serviceName, start, end)
	if err != nil {
		errs = append(errs, fmt.Sprintf("rps: %v", err))
	} else {
		metrics.RPS = rps
	}

	if len(errs) > 0 {
		return metrics, fmt.Errorf(strings.Join(errs, "; "))
	}

	return metrics, nil
}

// fetchCommits retrieves recent commits from GitHub
func (o *Orchestrator) fetchCommits(ctx context.Context, serviceName string, since time.Time) ([]models.CommitInfo, error) {
	// Map service name to GitHub repo using config mapping
	repo := ""
	if o.cfg.GitHub.ServiceMapping != nil {
		if mapped, ok := o.cfg.GitHub.ServiceMapping[serviceName]; ok {
			repo = mapped
		}
	}

	// Fallback: use default org + service name as repo
	if repo == "" {
		if o.cfg.GitHub.DefaultOrg != "" {
			repo = o.cfg.GitHub.DefaultOrg + "/" + serviceName
		} else {
			repo = serviceName // Last resort fallback
		}
	}

	commits, err := o.githubClient.FetchCommitsByRepo(ctx, repo, since)
	if err != nil {
		slog.Warn("fetch_error", "source", "github", "error", err)
		return nil, err
	}

	result := make([]models.CommitInfo, len(commits))
	for i, c := range commits {
		result[i] = models.CommitInfo{
			SHA:       c.SHA,
			Message:   c.Message,
			Author:    c.Author.Name,
			Email:     c.Author.Email,
			URL:       c.URL,
			Timestamp: parseTime(c.Author.Date),
		}
	}

	return result, nil
}

// HealthCheck verifies that orchestrator is properly initialized
func (o *Orchestrator) HealthCheck(ctx context.Context) bool {
	// Basic check: orchestrator is initialized with clients
	return o.promClient != nil || o.githubClient != nil || o.lokiClient != nil
}

// parseTime parses a time string
func parseTime(s string) time.Time {
	t, _ := time.Parse(time.RFC3339, s)
	return t
}

// fetchTraces retrieves trace context from Tempo
func (o *Orchestrator) fetchTraces(ctx context.Context, serviceName string, start, end time.Time) (tempo.TraceContext, error) {
	var traceCtx tempo.TraceContext

	if o.tempoClient == nil {
		return traceCtx, nil
	}

	traces, err := o.tempoClient.GetTracesByService(ctx, serviceName, start, end)
	if err != nil {
		slog.Warn("fetch_error", "source", "tempo", "error", err)
		return traceCtx, err
	}
	traceCtx.TraceCount = len(traces)

	slowSpans, err := o.tempoClient.SearchSlowSpans(ctx, serviceName, 500)
	if err == nil {
		traceCtx.SlowSpans = slowSpans
	}

	return traceCtx, nil
}

// fetchLogs retrieves error logs from Loki
func (o *Orchestrator) fetchLogs(ctx context.Context, serviceName string, start, end time.Time) ([]models.LogEntry, error) {
	if o.lokiClient == nil {
		return nil, nil
	}

	// Fetch error logs for the service
	logs, err := o.lokiClient.QueryErrorLogs(ctx, serviceName, start, end, 50)
	if err != nil {
		slog.Warn("fetch_error", "source", "loki", "error", err)
		return nil, err
	}

	// Convert Loki LogEntry to models.LogEntry
	result := make([]models.LogEntry, len(logs))
	for i, log := range logs {
		result[i] = models.LogEntry{
			Timestamp: log.Timestamp,
			Message:   log.Message,
			Service:   log.Service,
			Level:     log.Level,
		}
	}

	slog.Info("logs.fetched", "count", len(result), "service", serviceName)
	return result, nil
}
