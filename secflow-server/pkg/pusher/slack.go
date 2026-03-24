package pusher

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Slack implements Slack webhook push.
type Slack struct {
	webhookURL string
	timeout    time.Duration
}

// NewSlack creates a new Slack pusher.
func NewSlack(webhookURL string) *Slack {
	return &Slack{
		webhookURL: webhookURL,
		timeout:    10 * time.Second,
	}
}

// WithTimeout sets the request timeout.
func (s *Slack) WithTimeout(timeout time.Duration) *Slack {
	s.timeout = timeout
	return s
}

// PushText sends a plain text message.
func (s *Slack) PushText(ctx context.Context, text string) error {
	msg := slackMessage{
		Text: text,
	}
	return s.send(ctx, msg)
}

// PushMarkdown sends a markdown formatted message.
func (s *Slack) PushMarkdown(ctx context.Context, title, content string) error {
	msg := slackMessage{
		Text:     title,
		Markdown: true,
		Attachments: []slackAttachment{
			{
				Color: "#36a64f",
				Title: title,
				Text:  content,
			},
		},
	}
	return s.send(ctx, msg)
}

// send sends a message to Slack.
func (s *Slack) send(ctx context.Context, msg interface{}) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal message: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.webhookURL, jsonBody(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: s.timeout}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	return nil
}

// Message structures for Slack
type slackAttachment struct {
	Color string `json:"color"`
	Title string `json:"title"`
	Text  string `json:"text"`
}

type slackMessage struct {
	Text        string            `json:"text"`
	Markdown    bool              `json:"mrkdwn,omitempty"`
	Attachments []slackAttachment `json:"attachments,omitempty"`
}
