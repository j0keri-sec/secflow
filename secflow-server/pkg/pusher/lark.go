package pusher

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Lark implements Lark/Feishu robot push.
type Lark struct {
	webhookURL string
	signSecret string
	timeout    time.Duration
}

// NewLark creates a new Lark pusher.
func NewLark(webhookURL, signSecret string) *Lark {
	// If webhookURL is just an access token, construct full URL
	if !strings.HasPrefix(webhookURL, "http") {
		webhookURL = "https://open.feishu.cn/open-apis/bot/v2/hook/" + webhookURL
	}

	return &Lark{
		webhookURL: webhookURL,
		signSecret: signSecret,
		timeout:    10 * time.Second,
	}
}

// WithTimeout sets the request timeout.
func (l *Lark) WithTimeout(timeout time.Duration) *Lark {
	l.timeout = timeout
	return l
}

// PushText sends a plain text message.
func (l *Lark) PushText(ctx context.Context, text string) error {
	msg := larkMessage{
		MsgType: "text",
		Content: larkTextContent{Text: text},
	}
	return l.send(ctx, msg)
}

// PushMarkdown sends a markdown formatted message.
func (l *Lark) PushMarkdown(ctx context.Context, title, content string) error {
	// Lark doesn't support markdown in the same way, use rich text card
	msg := larkMessage{
		MsgType: "interactive",
		Card: larkCard{
			Config: larkCardConfig{
				WideScreenMode: true,
			},
			Header: larkCardHeader{
				Title: larkCardTitle{
					Tag:     "plain_text",
					Content: title,
				},
			},
			Elements: []larkCardElement{
				larkCardDiv{
					Tag: "div",
					Text: larkCardText{
						Tag:     "lark_md",
						Content: prepareLarkMarkdown(content),
					},
				},
			},
		},
	}
	return l.send(ctx, msg)
}

// send sends a message to Lark.
func (l *Lark) send(ctx context.Context, msg interface{}) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal message: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, l.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, l.webhookURL, jsonBody(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: l.timeout}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	var result larkResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	if result.Code != 0 {
		return fmt.Errorf("lark error %d: %s", result.Code, result.Msg)
	}

	return nil
}

// Message structures for Lark
type larkTextContent struct {
	Text string `json:"text"`
}

type larkCardTitle struct {
	Tag     string `json:"tag"`
	Content string `json:"content"`
}

type larkCardHeader struct {
	Title larkCardTitle `json:"title"`
}

type larkCardConfig struct {
	WideScreenMode bool `json:"wide_screen_mode"`
}

type larkCardText struct {
	Tag     string `json:"tag"`
	Content string `json:"content"`
}

type larkCardDiv struct {
	Tag  string       `json:"tag"`
	Text larkCardText `json:"text"`
}

type larkCardElement interface{}

type larkCard struct {
	Config   larkCardConfig    `json:"config"`
	Header   larkCardHeader    `json:"header"`
	Elements []larkCardElement `json:"elements"`
}

type larkMessage struct {
	MsgType string          `json:"msg_type"`
	Content larkTextContent `json:"content,omitempty"`
	Card    larkCard        `json:"card,omitempty"`
}

type larkResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

// prepareLarkMarkdown prepares markdown content for Lark.
// Lark uses a subset of markdown and requires some adjustments.
func prepareLarkMarkdown(content string) string {
	// Remove special characters that Lark doesn't support well
	content = strings.ReplaceAll(content, "&nbsp;", "")
	
	// Ensure proper line breaks
	content = strings.ReplaceAll(content, "\n\n", "\n\n")
	
	return content
}
