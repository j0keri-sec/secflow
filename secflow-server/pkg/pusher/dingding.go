package pusher

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// DingDing implements DingTalk robot push.
type DingDing struct {
	accessToken string
	signSecret  string
	timeout     time.Duration
}

// NewDingDing creates a new DingDing pusher.
func NewDingDing(accessToken, signSecret string) *DingDing {
	return &DingDing{
		accessToken: accessToken,
		signSecret:  signSecret,
		timeout:     10 * time.Second,
	}
}

// WithTimeout sets the request timeout.
func (d *DingDing) WithTimeout(timeout time.Duration) *DingDing {
	d.timeout = timeout
	return d
}

// PushText sends a plain text message.
func (d *DingDing) PushText(ctx context.Context, text string) error {
	msg := dingTalkTextMessage{
		MsgType: "text",
		Text: dingTalkText{
			Content: text,
		},
	}
	return d.send(ctx, msg)
}

// PushMarkdown sends a markdown formatted message.
func (d *DingDing) PushMarkdown(ctx context.Context, title, content string) error {
	// Special handling for empty lines in DingTalk markdown
	content = prepareDingTalkMarkdown(content)

	msg := dingTalkMarkdownMessage{
		MsgType: "markdown",
		Markdown: dingTalkMarkdown{
			Title: title,
			Text:  content,
		},
	}
	return d.send(ctx, msg)
}

// send sends a message to DingTalk.
func (d *DingDing) send(ctx context.Context, msg interface{}) error {
	webhook := "https://oapi.dingtalk.com/robot/send"
	
	// Generate signature if secret is provided
	if d.signSecret != "" {
		timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)
		signature := d.generateSign(timestamp, d.signSecret)
		webhook = fmt.Sprintf("%s?access_token=%s&timestamp=%s&sign=%s", 
			webhook, d.accessToken, timestamp, signature)
	} else {
		webhook = fmt.Sprintf("%s?access_token=%s", webhook, d.accessToken)
	}

	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal message: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, d.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhook, jsonBody(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: d.timeout}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	var result dingTalkResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	if result.ErrCode != 0 {
		return fmt.Errorf("dingtalk error %d: %s", result.ErrCode, result.ErrMsg)
	}

	return nil
}

// generateSign generates the signature for DingTalk webhook.
func (d *DingDing) generateSign(timestamp, secret string) string {
	stringToSign := fmt.Sprintf("%s\n%s", timestamp, secret)
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(stringToSign))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

// Message structures for DingTalk
type dingTalkText struct {
	Content string `json:"content"`
}

type dingTalkTextMessage struct {
	MsgType string        `json:"msgtype"`
	Text    dingTalkText `json:"text"`
}

type dingTalkMarkdown struct {
	Title string `json:"title"`
	Text  string `json:"text"`
}

type dingTalkMarkdownMessage struct {
	MsgType  string           `json:"msgtype"`
	Markdown dingTalkMarkdown `json:"markdown"`
}

type dingTalkResponse struct {
	ErrCode int    `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
}

// prepareDingTalkMarkdown prepares markdown content for DingTalk.
// DingTalk requires special handling for empty lines.
func prepareDingTalkMarkdown(content string) string {
	// Add spaces for empty lines to ensure proper rendering
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if strings.TrimSpace(line) == "" {
			lines[i] = "&nbsp;"
		}
	}
	return strings.Join(lines, "\n")
}
