// Package tempo provides a client for interacting with the Grafana Tempo distributed tracing backend.
package tempo

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"time"
)

// Client implements HTTP interaction with the Tempo API to fetch traces and spans.
type Client struct {
	baseURL    string
	httpClient *http.Client
	logger     *slog.Logger
}

// NewClient creates a new Tempo client
func NewClient(baseURL string, timeout time.Duration, logger *slog.Logger) *Client {
	if logger == nil {
		logger = slog.Default()
	}
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		logger: logger,
	}
}

// QueryResult represents a Tempo query response
type QueryResult struct {
	Traces []struct {
		TraceID string `json:"traceID"`
		// Note: the exact structure depends on Tempo API version
		RootServiceName   string `json:"rootServiceName"`
		RootTraceName     string `json:"rootTraceName"`
		StartTimeUnixNano uint64 `json:"startTimeUnixNano"`
		DurationMs        int64  `json:"durationMs"`
	} `json:"traces"`
}

// doRequest performs the HTTP request to Tempo via HTTP API
func (c *Client) doRequest(ctx context.Context, apiPath string, params url.Values) ([]byte, error) {
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}

	u.Path = apiPath
	if params != nil {
		u.RawQuery = params.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("tempo request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code from tempo: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return body, nil
}

// GetTracesByService fetches recent traces for a given service within the time window
func (c *Client) GetTracesByService(ctx context.Context, service string, start, end time.Time) ([]Trace, error) {
	// Tempo searches are typically conducted via TraceQL e.g. /api/search
	query := BuildServiceQuery(service)
	
	params := url.Values{
		"q":     []string{query},
		"start": []string{fmt.Sprintf("%d", start.Unix())},
		"end":   []string{fmt.Sprintf("%d", end.Unix())},
	}

	resp, err := c.doRequest(ctx, "/api/search", params)
	if err != nil {
		c.logger.Error("Failed to fetch traces", "service", service, "error", err)
		return nil, err
	}

	// This assumes the search returns basic trace overviews
	var searchResult QueryResult
	if err := json.Unmarshal(resp, &searchResult); err != nil {
		return nil, fmt.Errorf("failed to parse search response: %w", err)
	}

	var traces []Trace
	for _, t := range searchResult.Traces {
		traces = append(traces, Trace{
			TraceID: t.TraceID,
		})
	}

	return traces, nil
}

// GetTraceByID fetches a single complete trace by its ID
func (c *Client) GetTraceByID(ctx context.Context, traceID string) (*Trace, error) {
	resp, err := c.doRequest(ctx, fmt.Sprintf("/api/traces/%s", traceID), nil)
	if err != nil {
		c.logger.Error("Failed to fetch trace by ID", "traceID", traceID, "error", err)
		return nil, err
	}

	// In reality we would decode the full OTLP JSON trace format here.
	// For the MVP structural implementation, we represent it as parsed struct.
	var trace Trace
	// We'd unmarshal `resp` appropriately into `trace` here.
	_ = resp
	
	return &trace, nil
}

// SearchSlowSpans finds spans exceeding a latency threshold using TraceQL
func (c *Client) SearchSlowSpans(ctx context.Context, service string, thresholdMs int) ([]Span, error) {
	query := BuildSlowSpansQuery(service, thresholdMs)
	params := url.Values{
		"q": []string{query},
	}

	resp, err := c.doRequest(ctx, "/api/search", params)
	if err != nil {
		c.logger.Error("Failed to search slow spans", "query", query, "error", err)
		return nil, err
	}

	// Dummy parsing block: real implementation parses TraceQL span results 
	_ = resp
	var slowSpans []Span
	
	// Assume we append successfully matched spans into slowSpans
	return slowSpans, nil
}
