package pusher

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Telegram implements Telegram bot push.
type Telegram struct {
	botToken string
	chatID   string
	timeout  time.Duration
}

// NewTelegram creates a new Telegram pusher.
func NewTelegram(botToken, chatID string) *Telegram {
	return &Telegram{
		botToken: botToken,
		chatID:   chatID,
		timeout:  10 * time.Second,
	}
}

// WithTimeout sets the request timeout.
func (t *Telegram) WithTimeout(timeout time.Duration) *Telegram {
	t.timeout = timeout
	return t
}

// PushText sends a plain text message.
func (t *Telegram) PushText(ctx context.Context, text string) error {
	params := url.Values{}
	params.Add("chat_id", t.chatID)
	params.Add("text", text)
	params.Add("parse_mode", "HTML")

	return t.send(ctx, "sendMessage", params)
}

// PushMarkdown sends a markdown formatted message.
func (t *Telegram) PushMarkdown(ctx context.Context, title, content string) error {
	// Telegram supports HTML or MarkdownV2
	// Use HTML for better compatibility
	text := fmt.Sprintf("<b>%s</b>\n\n%s", escapeHTML(title), escapeHTML(content))

	params := url.Values{}
	params.Add("chat_id", t.chatID)
	params.Add("text", text)
	params.Add("parse_mode", "HTML")

	return t.send(ctx, "sendMessage", params)
}

// send sends a message to Telegram.
func (t *Telegram) send(ctx context.Context, method string, params url.Values) error {
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/%s", t.botToken, method)

	ctx, cancel := context.WithTimeout(ctx, t.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, strings.NewReader(params.Encode()))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: t.timeout}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	var result telegramResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	if !result.OK {
		return fmt.Errorf("telegram error: %s", result.Description)
	}

	return nil
}

// escapeHTML escapes HTML special characters.
func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

// Message structures for Telegram
type telegramResponse struct {
	OK          bool   `json:"ok"`
	Description string `json:"description,omitempty"`
}
