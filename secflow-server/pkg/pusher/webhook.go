package pusher

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Webhook implements generic webhook push.
type Webhook struct {
	url       string
	method    string
	headers   map[string]string
	timeout   time.Duration
}

// NewWebhook creates a new Webhook pusher.
func NewWebhook(url, method string) *Webhook {
	if method == "" {
		method = http.MethodPost
	}

	return &Webhook{
		url:     url,
		method:  method,
		headers: make(map[string]string),
		timeout: 10 * time.Second,
	}
}

// WithTimeout sets the request timeout.
func (w *Webhook) WithTimeout(timeout time.Duration) *Webhook {
	w.timeout = timeout
	return w
}

// WithHeader adds a custom header.
func (w *Webhook) WithHeader(key, value string) *Webhook {
	w.headers[key] = value
	return w
}

// PushText sends a plain text message.
func (w *Webhook) PushText(ctx context.Context, text string) error {
	payload := map[string]interface{}{
		"text": text,
		"type": "text",
	}
	return w.send(ctx, payload)
}

// PushMarkdown sends a markdown formatted message.
func (w *Webhook) PushMarkdown(ctx context.Context, title, content string) error {
	payload := map[string]interface{}{
		"title":   title,
		"content": content,
		"type":    "markdown",
	}
	return w.send(ctx, payload)
}

// PushRaw sends a raw message with custom format.
func (w *Webhook) PushRaw(ctx context.Context, raw *RawMessage) error {
	return w.send(ctx, raw.Payload)
}

// send sends a message to the webhook.
func (w *Webhook) send(ctx context.Context, payload interface{}) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, w.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, w.method, w.url, jsonBody(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	// Set default headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "SecFlow-Pusher/1.0")

	// Set custom headers
	for key, value := range w.headers {
		req.Header.Set(key, value)
	}

	client := &http.Client{Timeout: w.timeout}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	// Check for success status codes
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	return nil
}
