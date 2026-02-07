package orchestrator

import (
	"context"
	"log"
	"time"

	"helixops/internal/clients/github"
	"helixops/internal/clients/loki"
	"helixops/internal/clients/prometheus"
	"helixops/internal/config"
	"helixops/internal/models"
)

// Orchestrator coordinates data collection from multiple sources
type Orchestrator struct {
	promClient   *prometheus.Client
	githubClient *github.Client
	lokiClient   *loki.Client
	cfg          *config.Config
}

// New creates a new orchestrator
func New(prom *prometheus.Client, gh *github.Client, loki *loki.Client, cfg *config.Config) *Orchestrator {
	return &Orchestrator{
		promClient:   prom,
		githubClient: gh,
		lokiClient:   loki,
		cfg:          cfg,
	}
}

// PrepareContext gathers all context data for RCA
func (o *Orchestrator) PrepareContext(ctx context.Context, serviceName string, alertTime time.Time) (*models.AnalysisContext, error) {
	log.Printf("Preparing context for service: %s", serviceName)

	// Calculate time windows
	metricsWindow := o.cfg.Analysis.GetMetricsWindowDuration()
	commitsLookback := o.cfg.Analysis.GetCommitsLookbackDuration()

	metricsStart := alertTime.Add(-metricsWindow)
	metricsEnd := alertTime

	commitsSince := alertTime.Add(-commitsLookback)

	// Fetch data concurrently
	type result struct {
		metrics models.MetricsSummary
		commits []models.CommitInfo
		err     error
	}

	resultCh := make(chan result, 2)

	go func() {
		metrics, err := o.fetchMetrics(ctx, serviceName, metricsStart, metricsEnd)
		resultCh <- result{metrics: metrics, err: err}
	}()

	go func() {
		commits, err := o.fetchCommits(ctx, serviceName, commitsSince)
		resultCh <- result{commits: commits, err: err}
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

	for i := 0; i < 2; i++ {
		r := <-resultCh
		if r.err != nil {
			log.Printf("Error fetching data: %v", r.err)
			aggregatedErr = r.err
		}
		if len(r.commits) > 0 {
			ctxResult.RecentCommits = r.commits
		}
		if r.metrics.LatencyP99 > 0 || r.metrics.ErrorRate > 0 {
			ctxResult.Metrics = r.metrics
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
	// TODO: Map service name to GitHub repo (from DB or config)
	repo := o.cfg.GitHub.APIURL
	if repo == "" {
		repo = serviceName // Fallback
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

// parseTime parses a time string
func parseTime(s string) time.Time {
	t, _ := time.Parse(time.RFC3339, s)
	return t
}
