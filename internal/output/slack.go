package output

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"helixops/internal/config"
	"helixops/internal/models"
	"helixops/internal/postmortem"
)

// SlackSender handles the dispatch of rich-text incident notifications to a Slack webhook.
type SlackSender struct {
	webhookURL string
	client     *http.Client
}

// NewSlackSender initializes a SlackSender with a configured webhook URL and HTTP client.
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

// SendPostmortem sends a generated postmortem to Slack
func (s *SlackSender) SendPostmortem(pm *postmortem.Postmortem) error {
	if s.webhookURL == "" {
		return fmt.Errorf("slack webhook URL not configured")
	}

	message := s.buildPostmortemMessage(pm)
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

// buildMessage constructs a visually formatted Slack block kit payload from an analysis result.
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

// NewSlackSenderFromConfig constructs a SlackSender using the provided configuration block.
func NewSlackSenderFromConfig(cfg config.SlackOutputConfig) *SlackSender {
	return NewSlackSender(cfg.WebhookURL)
}

// buildPostmortemMessage creates a Slack message from a postmortem
func (s *SlackSender) buildPostmortemMessage(pm *postmortem.Postmortem) SlackMessage {
	blocks := []SlackBlock{
		{
			Type: "header",
			Text: &SlackText{
				Type: "plain_text",
				Text: fmt.Sprintf("✅ Resolved: %s", pm.IncidentName),
			},
		},
		{
			Type: "section",
			Fields: []SlackField{
				{
					Type: "mrkdwn",
					Text: fmt.Sprintf("*Duration:*\n%s", pm.Duration.String()),
				},
				{
					Type: "mrkdwn",
					Text: fmt.Sprintf("*Date:*\n%s", pm.Date.Format(time.RFC822)),
				},
			},
		},
		{
			Type: "section",
			Text: &SlackText{
				Type: "mrkdwn",
				Text: "*HelixOps Automated Postmortem Generated*\nThe incident has been resolved and a detailed postmortem is available focusing on Timeline, Root Cause, and Impact.",
			},
		},
	}

	if len(pm.RemediationRules) > 0 {
		blocks = append(blocks, SlackBlock{Type: "divider"})
		blocks = append(blocks, SlackBlock{
			Type: "section",
			Text: &SlackText{
				Type: "mrkdwn",
				Text: "*Suggested Fixes (Rule Engine)*",
			},
		})

		for i, rule := range pm.RemediationRules {
			if i >= 3 { // Limit to top 3 rules
				break
			}
			blocks = append(blocks, SlackBlock{
				Type: "section",
				Text: &SlackText{
					Type: "mrkdwn",
					Text: fmt.Sprintf(">*%s*\n>%s\n>`%s`", rule.Title, rule.Description, rule.Action),
				},
			})
		}
	}

	return SlackMessage{Blocks: blocks}
}
