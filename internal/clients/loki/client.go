// Package loki provides a client to interface with Grafana Loki for log aggregation and LogQL querying.
package loki

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// Client handles authenticated LogQL queries against a specified Loki instance.
type Client struct {
	baseURL string
	client  *http.Client
	timeout time.Duration
}

// NewClient creates a new Loki client
func NewClient(baseURL string, timeout time.Duration) *Client {
	if baseURL == "" {
		baseURL = "http://localhost:3100"
	}
	return &Client{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: timeout,
		},
		timeout: timeout,
	}
}

// LogResponse represents Loki query response
type LogResponse struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Stream map[string]string `json:"stream"`
			Values [][]string        `json:"values"`
		} `json:"result"`
	} `json:"data"`
}

// Query executes a LogQL query and returns log entries
func (c *Client) Query(ctx context.Context, query string, start, end time.Time, limit int) ([]LogEntry, error) {
	params := url.Values{}
	params.Set("query", query)
	params.Set("start", start.Format(time.RFC3339Nano))
	params.Set("end", end.Format(time.RFC3339Nano))
	params.Set("limit", fmt.Sprintf("%d", limit))

	req, err := c.newRequest(ctx, http.MethodGet, "/loki/api/v1/query_range", params)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var result LogResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	entries := make([]LogEntry, 0)
	for _, res := range result.Data.Result {
		for _, value := range res.Values {
			if len(value) < 2 {
				continue
			}
			timestamp, err := time.Parse(time.RFC3339Nano, value[0])
			if err != nil {
				continue
			}
			entries = append(entries, LogEntry{
				Timestamp: timestamp,
				Message:   value[1],
				Service:   res.Stream["service"],
				Level:     res.Stream["level"],
			})
		}
	}

	return entries, nil
}

// QueryErrorLogs fetches error logs for a service
func (c *Client) QueryErrorLogs(ctx context.Context, serviceName string, start, end time.Time, limit int) ([]LogEntry, error) {
	query := fmt.Sprintf(`{service="%s"} |= "error"`, serviceName)
	return c.Query(ctx, query, start, end, limit)
}

// newRequest creates a new HTTP request
func (c *Client) newRequest(ctx context.Context, method, path string, params url.Values) (*http.Request, error) {
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}

	u.Path = path
	u.RawQuery = params.Encode()

	req, err := http.NewRequestWithContext(ctx, method, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	return req, nil
}
