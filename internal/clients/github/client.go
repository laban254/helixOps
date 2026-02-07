package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// Client wraps GitHub API calls
type Client struct {
	baseURL string
	token   string
	client  *http.Client
}

// NewClient creates a new GitHub client
func NewClient(baseURL, token string) *Client {
	if baseURL == "" {
		baseURL = "https://api.github.com"
	}
	return &Client{
		baseURL: baseURL,
		token:   token,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Commit represents a GitHub commit
type Commit struct {
	SHA      string       `json:"sha"`
	Message  string       `json:"message"`
	Author   CommitAuthor `json:"author"`
	URL      string       `json:"html_url"`
	Committer CommitAuthor `json:"committer"`
}

// CommitAuthor represents commit author info
type CommitAuthor struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Date  string `json:"date"`
}

// ListCommitsResponse represents the GitHub commits API response
type ListCommitsResponse struct {
	SHA       string `json:"sha"`
	Commit    struct {
		Message string `json:"message"`
		Author  struct {
			Name  string `json:"name"`
			Email string `json:"email"`
			Date  string `json:"date"`
		} `json:"author"`
	} `json:"commit"`
	HTMLURL string `json:"html_url"`
}

// FetchCommits fetches recent commits for a repository
func (c *Client) FetchCommits(ctx context.Context, owner, repo string, since time.Time) ([]Commit, error) {
	path := fmt.Sprintf("/repos/%s/%s/commits", owner, repo)
	
	params := url.Values{}
	params.Set("since", since.Format(time.RFC3339))
	params.Set("per_page", "10")

	req, err := c.newRequest(ctx, http.MethodGet, path, params, nil)
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

	var commits []ListCommitsResponse
	if err := json.NewDecoder(resp.Body).Decode(&commits); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	result := make([]Commit, len(commits))
	for i, cmt := range commits {
		result[i] = Commit{
			SHA:      cmt.SHA,
			Message:  cmt.Commit.Message,
			Author: CommitAuthor{
				Name:  cmt.Commit.Author.Name,
				Email: cmt.Commit.Author.Email,
				Date:  cmt.Commit.Author.Date,
			},
			Committer: CommitAuthor{
				Name:  cmt.Commit.Committer.Name,
				Email: cmt.Commit.Committer.Email,
				Date:  cmt.Commit.Committer.Date,
			},
			URL: cmt.HTMLURL,
		}
	}

	return result, nil
}

// newRequest creates a new HTTP request with auth headers
func (c *Client) newRequest(ctx context.Context, method, path string, params url.Values, body interface{}) (*http.Request, error) {
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

	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	return req, nil
}

// FetchCommitsByRepo fetches commits using repo name format (owner/repo)
func (c *Client) FetchCommitsByRepo(ctx context.Context, repo string, since time.Time) ([]Commit, error) {
	parts := splitRepo(repo)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid repo format: %s (expected owner/repo)", repo)
	}
	return c.FetchCommits(ctx, parts[0], parts[1], since)
}

// splitRepo splits "owner/repo" into [owner, repo]
func splitRepo(repo string) []string {
	for i := 0; i < len(repo); i++ {
		if repo[i] == '/' {
			return []string{repo[:i], repo[i+1:]}
		}
	}
	return []string{repo, ""}
}
