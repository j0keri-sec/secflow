package pusher

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// WeChatWork implements WeChat Work robot push.
type WeChatWork struct {
	webhookURL string
	timeout    time.Duration
}

// NewWeChatWork creates a new WeChatWork pusher.
func NewWeChatWork(webhookURL, _ string) *WeChatWork {
	// WeChat Work doesn't use sign secret like DingTalk or Lark
	return &WeChatWork{
		webhookURL: webhookURL,
		timeout:    10 * time.Second,
	}
}

// WithTimeout sets the request timeout.
func (w *WeChatWork) WithTimeout(timeout time.Duration) *WeChatWork {
	w.timeout = timeout
	return w
}

// PushText sends a plain text message.
func (w *WeChatWork) PushText(ctx context.Context, text string) error {
	msg := weChatWorkMessage{
		MsgType: "text",
		Text: weChatWorkText{
			Content: text,
		},
	}
	return w.send(ctx, msg)
}

// PushMarkdown sends a markdown formatted message.
func (w *WeChatWork) PushMarkdown(ctx context.Context, title, content string) error {
	// WeChat Work supports markdown in text messages
	msg := weChatWorkMessage{
		MsgType: "markdown",
		Markdown: weChatWorkMarkdown{
			Content: fmt.Sprintf("**%s**\n\n%s", title, content),
		},
	}
	return w.send(ctx, msg)
}

// send sends a message to WeChat Work.
func (w *WeChatWork) send(ctx context.Context, msg interface{}) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal message: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, w.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, w.webhookURL, jsonBody(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: w.timeout}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	var result weChatWorkResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	if result.ErrCode != 0 {
		return fmt.Errorf("wechat work error %d: %s", result.ErrCode, result.ErrMsg)
	}

	return nil
}

// Message structures for WeChat Work
type weChatWorkText struct {
	Content             string   `json:"content"`
	MentionedList       []string `json:"mentioned_list,omitempty"`
	MentionedMobileList []string `json:"mentioned_mobile_list,omitempty"`
}

type weChatWorkMarkdown struct {
	Content string `json:"content"`
}

type weChatWorkMessage struct {
	MsgType  string              `json:"msgtype"`
	Text     weChatWorkText      `json:"text,omitempty"`
	Markdown weChatWorkMarkdown  `json:"markdown,omitempty"`
}

type weChatWorkResponse struct {
	ErrCode int    `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
}
