package tempo

import "time"

// Trace represents a complete distributed trace containing multiple spans.
type Trace struct {
	TraceID string `json:"traceID"`
	Spans   []Span `json:"spans"`
}

// Span represents a single timed operation within a larger trace.
type Span struct {
	SpanID        string    `json:"spanID"`
	TraceID       string    `json:"traceID"`
	ServiceName   string    `json:"serviceName"`
	OperationName string    `json:"operationName"`
	StartTime     time.Time `json:"startTime"`
	DurationMs    int64     `json:"durationMs"`
	Status        string    `json:"status"` // e.g., "ok", "error"
}

// TraceContext aggregates related traces and spans for use in RCA prompts.
type TraceContext struct {
	SlowSpans  []Span  `json:"slowSpans"`
	ErrorSpans []Span  `json:"errorSpans"`
	TraceCount int     `json:"traceCount"`
	P99Latency float64 `json:"p99Latency"`
}
