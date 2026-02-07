package prometheus

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// Client wraps Prometheus HTTP API calls
type Client struct {
	baseURL string
	client  *http.Client
	timeout time.Duration
}

// NewClient creates a new Prometheus client
func NewClient(baseURL string, timeout time.Duration) *Client {
	return &Client{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: timeout,
		},
		timeout: timeout,
	}
}

// QueryResult represents a Prometheus query result
type QueryResult struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Metric map[string]string `json:"metric"`
			Value  []interface{}      `json:"value"`
		} `json:"result"`
	} `json:"data"`
}

// Query executes an instant query and returns the first value
func (c *Client) Query(ctx context.Context, query string) (float64, error) {
	params := url.Values{
		"query": []string{query},
	}

	resp, err := c.doRequest(ctx, "/api/v1/query", params)
	if err != nil {
		return 0, err
	}

	var result QueryResult
	if err := json.Unmarshal(resp, &result); err != nil {
		return 0, fmt.Errorf("failed to parse response: %w", err)
	}

	if result.Status != "success" {
		return 0, fmt.Errorf("query failed: %s", result.Status)
	}

	if len(result.Data.Result) == 0 {
		return 0, nil
	}

	// Get the first value
	if len(result.Data.Result[0].Value) < 2 {
		return 0, nil
	}

	value, ok := result.Data.Result[0].Value[1].(string)
	if !ok {
		return 0, fmt.Errorf("invalid value type")
	}

	var f float64
	_, err = fmt.Sscanf(value, "%f", &f)
	if err != nil {
		return 0, fmt.Errorf("failed to parse value: %w", err)
	}

	return f, nil
}

// QueryRange executes a range query
func (c *Client) QueryRange(ctx context.Context, query string, start, end time.Time, step string) (*QueryResult, error) {
	params := url.Values{
		"query": []string{query},
		"start": []string{start.Format(time.RFC3339)},
		"end":   []string{end.Format(time.RFC3339)},
		"step":  []string{step},
	}

	resp, err := c.doRequest(ctx, "/api/v1/query_range", params)
	if err != nil {
		return nil, err
	}

	var result QueryResult
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if result.Status != "success" {
		return nil, fmt.Errorf("query failed: %s", result.Status)
	}

	return &result, nil
}

// doRequest makes an HTTP request to Prometheus
func (c *Client) doRequest(ctx context.Context, path string, params url.Values) ([]byte, error) {
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}

	u.Path = path
	u.RawQuery = params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
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

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return body, nil
}

// QueryLatencyP99 returns the p99 latency for a service
func (c *Client) QueryLatencyP99(ctx context.Context, serviceName string, start, end time.Time) (float64, error) {
	query := fmt.Sprintf(
		"histogram_quantile(0.99, sum(rate(http_request_duration_seconds_bucket{service='%s'}[5m])) by (le))",
		serviceName,
	)
	return c.Query(ctx, query)
}

// QueryErrorRate returns the error rate for a service
func (c *Client) QueryErrorRate(ctx context.Context, serviceName string, start, end time.Time) (float64, error) {
	query := fmt.Sprintf(
		"sum(rate(http_requests_total{service='%s',status=~'5..'}[5m])) / sum(rate(http_requests_total{service='%s'}[5m]))",
		serviceName, serviceName,
	)
	return c.Query(ctx, query)
}

// QueryRPS returns requests per second for a service
func (c *Client) QueryRPS(ctx context.Context, serviceName string, start, end time.Time) (float64, error) {
	query := fmt.Sprintf(
		"sum(rate(http_requests_total{service='%s'}[5m]))",
		serviceName,
	)
	return c.Query(ctx, query)
}
