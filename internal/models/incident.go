package models

import (
	"time"

	"helixops/internal/clients/tempo"
)

// AnalysisResult represents the result of RCA analysis
type AnalysisResult struct {
	ID          string    `json:"id"`
	ServiceName string    `json:"service_name"`
	AlertName   string    `json:"alert_name"`
	Severity    string    `json:"severity"`
	Summary     string    `json:"summary"`
	RootCause   string    `json:"root_cause"`
	Confidence  string    `json:"confidence"`
	NextSteps   []string  `json:"next_steps"`
	Metrics     MetricsSummary `json:"metrics"`
	Commits     []CommitInfo    `json:"commits"`
	AnalyzedAt  time.Time `json:"analyzed_at"`
}

// MetricsSummary represents golden signals metrics
type MetricsSummary struct {
	LatencyP99   float64 `json:"latency_p99"`
	LatencyAvg   float64 `json:"latency_avg"`
	ErrorRate    float64 `json:"error_rate"`
	RPS          float64 `json:"requests_per_second"`
	MemoryUsage  float64 `json:"memory_usage"`
	
	// Baseline values for comparison
	BaselineLatency   float64 `json:"baseline_latency"`
	BaselineErrorRate float64 `json:"baseline_error_rate"`
	BaselineRPS       float64 `json:"baseline_rps"`
}

// CommitInfo represents a GitHub commit
type CommitInfo struct {
	SHA       string    `json:"sha"`
	Message   string    `json:"message"`
	Author    string    `json:"author"`
	Email     string    `json:"email"`
	URL       string    `json:"url"`
	Timestamp time.Time `json:"timestamp"`
	PRNumber  int       `json:"pr_number,omitempty"`
}

// AnalysisContext holds all data needed for RCA
type AnalysisContext struct {
	ServiceName   string                 `json:"service_name"`
	Alert         AlertInfo              `json:"alert"`
	Metrics       MetricsSummary         `json:"metrics"`
	RecentCommits []CommitInfo           `json:"recent_commits"`
	ErrorLogs     []LogEntry             `json:"error_logs,omitempty"`
	Traces        tempo.TraceContext     `json:"traces,omitempty"`
	TimeWindow    TimeWindow             `json:"time_window"`
}

// AlertInfo represents simplified alert data for analysis
type AlertInfo struct {
	Name      string            `json:"name"`
	Severity  string            `json:"severity"`
	Summary   string            `json:"summary"`
	Labels    map[string]string `json:"labels"`
	StartedAt time.Time         `json:"started_at"`
}

// TimeWindow represents the time range for queries
type TimeWindow struct {
	Start    time.Time `json:"start"`
	End      time.Time `json:"end"`
	Duration string    `json:"duration"`
}

// LogEntry represents a log entry from Loki
type LogEntry struct {
	Timestamp   time.Time `json:"timestamp"`
	Level       string    `json:"level"`
	Message     string    `json:"message"`
	Service     string    `json:"service"`
	Error       string    `json:"error,omitempty"`
	StackTrace  string    `json:"stack_trace,omitempty"`
}
