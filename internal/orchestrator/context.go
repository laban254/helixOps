// Package orchestrator coordinates data collection from PromQL, Loki, GitHub, and Tempo to build diagnostic context.
package orchestrator

import (
	"context"
	"log"
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
	log.Printf("Preparing context for service: %s", serviceName)

	// Calculate time windows
	metricsWindow := o.cfg.Analysis.GetMetricsWindowDuration()
	commitsLookback := o.cfg.Analysis.GetCommitsLookbackDuration()
	logsLookback := o.cfg.Analysis.GetLogsLookbackDuration()

	metricsStart := alertTime.Add(-metricsWindow)
	metricsEnd := alertTime

	commitsSince := alertTime.Add(-commitsLookback)
	logsStart := alertTime.Add(-logsLookback)

	// Fetch data concurrently
	type result struct {
		metrics models.MetricsSummary
		commits []models.CommitInfo
		traces  tempo.TraceContext
		logs    []models.LogEntry
		err     error
	}

	resultCh := make(chan result, 4)

	go func() {
		metrics, err := o.fetchMetrics(ctx, serviceName, metricsStart, metricsEnd)
		resultCh <- result{metrics: metrics, err: err}
	}()

	go func() {
		commits, err := o.fetchCommits(ctx, serviceName, commitsSince)
		resultCh <- result{commits: commits, err: err}
	}()

	go func() {
		traces, err := o.fetchTraces(ctx, serviceName, metricsStart, metricsEnd)
		resultCh <- result{traces: traces, err: err}
	}()

	go func() {
		logs, err := o.fetchLogs(ctx, serviceName, logsStart, metricsEnd)
		resultCh <- result{logs: logs, err: err}
	}()

	// Collect results
	var aggregatedErr error
	ctxResult := &models.AnalysisContext{
		ServiceName: serviceName,
		TimeWindow: models.TimeWindow{
			Start:    metricsStart,
			End:      metricsEnd,
			Duration: metricsWindow.String(),
		},
	}

	for i := 0; i < 4; i++ {
		r := <-resultCh
		if r.err != nil {
			log.Printf("Error fetching data: %v", r.err)
		}
		if len(r.commits) > 0 {
			ctxResult.RecentCommits = r.commits
		}
		if r.metrics.LatencyP99 > 0 || r.metrics.ErrorRate > 0 {
			ctxResult.Metrics = r.metrics
		}
		if r.traces.TraceCount > 0 {
			ctxResult.Traces = r.traces
		}
		if len(r.logs) > 0 {
			ctxResult.ErrorLogs = r.logs
		}
	}

	return ctxResult, aggregatedErr
}

// fetchMetrics retrieves golden signals metrics from Prometheus
func (o *Orchestrator) fetchMetrics(ctx context.Context, serviceName string, start, end time.Time) (models.MetricsSummary, error) {
	metrics := models.MetricsSummary{}

	latency, err := o.promClient.QueryLatencyP99(ctx, serviceName, start, end)
	if err != nil {
		log.Printf("Failed to query latency: %v", err)
	} else {
		metrics.LatencyP99 = latency
	}

	errorRate, err := o.promClient.QueryErrorRate(ctx, serviceName, start, end)
	if err != nil {
		log.Printf("Failed to query error rate: %v", err)
	} else {
		metrics.ErrorRate = errorRate
	}

	rps, err := o.promClient.QueryRPS(ctx, serviceName, start, end)
	if err != nil {
		log.Printf("Failed to query RPS: %v", err)
	} else {
		metrics.RPS = rps
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
		log.Printf("Failed to fetch commits: %v", err)
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
		log.Printf("Failed to fetch traces: %v", err)
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
		log.Printf("Failed to fetch error logs: %v", err)
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

	log.Printf("Fetched %d error logs for service %s", len(result), serviceName)
	return result, nil
}
