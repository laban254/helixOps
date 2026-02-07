package output

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"helixops/internal/config"
	"helixops/internal/models"
)

// SlackSender sends alerts to Slack
type SlackSender struct {
	webhookURL string
	client     *http.Client
}

// NewSlackSender creates a new Slack sender
func NewSlackSender(webhookURL string) *SlackSender {
	return &SlackSender{
		webhookURL: webhookURL,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// SlackBlock represents a Slack message block
type SlackBlock struct {
	Type      string            `json:"type"`
	Text      *SlackText        `json:"text,omitempty"`
	Fields    []SlackField      `json:"fields,omitempty"`
	Accessory *SlackAccessory   `json:"accessory,omitempty"`
}

// SlackText represents text in Slack
type SlackText struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// SlackField represents a field in Slack
type SlackField struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// SlackAccessory represents an accessory element
type SlackAccessory struct {
	Type  string `json:"type"`
	Text  *SlackText `json:"text,omitempty"`
	URL   string `json:"url,omitempty"`
}

// SlackMessage represents a Slack message
type SlackMessage struct {
	Blocks []SlackBlock `json:"blocks"`
}

// SendAnalysis sends an analysis result to Slack
func (s *SlackSender) SendAnalysis(result *models.AnalysisResult) error {
	if s.webhookURL == "" {
		return fmt.Errorf("slack webhook URL not configured")
	}

	message := s.buildMessage(result)
	body, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, s.webhookURL, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("slack returned status: %d", resp.StatusCode)
	}

	return nil
}

// buildMessage creates a Slack message from analysis result
func (s *SlackSender) buildMessage(result *models.AnalysisResult) SlackMessage {
	emoji := "🔍"
	if result.Severity == "critical" {
		emoji = "🚨"
	} else if result.Severity == "warning" {
		emoji = "⚠️"
	}

	return SlackMessage{
		Blocks: []SlackBlock{
			{
				Type: "header",
				Text: &SlackText{
					Type: "plain_text",
					Text: fmt.Sprintf("%s Alert: %s on %s", emoji, result.AlertName, result.ServiceName),
				},
			},
			{
				Type: "section",
				Fields: []SlackField{
					{
						Type: "mrkdwn",
						Text: fmt.Sprintf("*Severity:*\n%s", result.Severity),
					},
					{
						Type: "mrkdwn",
						Text: fmt.Sprintf("*Confidence:*\n%s", result.Confidence),
					},
				},
			},
			{
				Type: "section",
				Text: &SlackText{
					Type: "mrkdwn",
					Text: fmt.Sprintf("*AI Analysis:*\n%s", result.RootCause),
				},
			},
			{
				Type: "section",
				Fields: []SlackField{
					{
						Type: "mrkdwn",
						Text: fmt.Sprintf("*Latency:*\n%.2fms (baseline: %.2fms)", result.Metrics.LatencyP99, result.Metrics.BaselineLatency),
					},
					{
						Type: "mrkdwn",
						Text: fmt.Sprintf("*Error Rate:*\n%.2f%% (baseline: %.2f%%)", result.Metrics.ErrorRate*100, result.Metrics.BaselineErrorRate*100),
					},
				},
			},
			{
				Type: "divider",
			},
			{
				Type: "context",
				Fields: []SlackField{
					{
						Type: "mrkdwn",
						Text: fmt.Sprintf("Analyzed at: %s | ID: %s", result.AnalyzedAt.Format(time.RFC3339), result.ID),
					},
				},
			},
		},
	}
}

// SlackConfig creates a Slack sender from config
func NewSlackSenderFromConfig(cfg config.SlackOutputConfig) *SlackSender {
	return NewSlackSender(cfg.WebhookURL)
}
