// Package notify provides notification services for SecFlow.
package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/secflow/server/internal/model"
)

// Notifier defines the interface for sending notifications.
type Notifier interface {
	SendVuln(ctx context.Context, vuln *model.VulnRecord) error
	SendArticle(ctx context.Context, article *model.Article) error
	SendAlert(ctx context.Context, title, content string) error
}

// Config holds configuration for notification channels.
type Config struct {
	DingTalkWebhook string
	FeiShuWebhook   string
	Enabled         bool
}

// MultiNotifier sends notifications to multiple channels.
type MultiNotifier struct {
	config Config
	client *http.Client
}

// New creates a new MultiNotifier.
func New(config Config) *MultiNotifier {
	return &MultiNotifier{
		config: config,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// SendVuln sends a vulnerability notification.
func (n *MultiNotifier) SendVuln(ctx context.Context, vuln *model.VulnRecord) error {
	if !n.config.Enabled {
		return nil
	}

	var errs []error

	if n.config.DingTalkWebhook != "" {
		if err := n.sendDingTalkVuln(ctx, vuln); err != nil {
			errs = append(errs, fmt.Errorf("dingtalk: %w", err))
		}
	}

	if n.config.FeiShuWebhook != "" {
		if err := n.sendFeiShuVuln(ctx, vuln); err != nil {
			errs = append(errs, fmt.Errorf("feishu: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("notification errors: %v", errs)
	}
	return nil
}

// SendArticle sends an article notification.
func (n *MultiNotifier) SendArticle(ctx context.Context, article *model.Article) error {
	if !n.config.Enabled {
		return nil
	}

	var errs []error

	if n.config.DingTalkWebhook != "" {
		if err := n.sendDingTalkArticle(ctx, article); err != nil {
			errs = append(errs, fmt.Errorf("dingtalk: %w", err))
		}
	}

	if n.config.FeiShuWebhook != "" {
		if err := n.sendFeiShuArticle(ctx, article); err != nil {
			errs = append(errs, fmt.Errorf("feishu: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("notification errors: %v", errs)
	}
	return nil
}

// SendAlert sends a generic alert notification.
func (n *MultiNotifier) SendAlert(ctx context.Context, title, content string) error {
	if !n.config.Enabled {
		return nil
	}

	var errs []error

	if n.config.DingTalkWebhook != "" {
		if err := n.sendDingTalkAlert(ctx, title, content); err != nil {
			errs = append(errs, fmt.Errorf("dingtalk: %w", err))
		}
	}

	if n.config.FeiShuWebhook != "" {
		if err := n.sendFeiShuAlert(ctx, title, content); err != nil {
			errs = append(errs, fmt.Errorf("feishu: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("notification errors: %v", errs)
	}
	return nil
}

// DingTalk message structures
type dingTalkMarkdown struct {
	Title string `json:"title"`
	Text  string `json:"text"`
}

type dingTalkMessage struct {
	MsgType  string           `json:"msgtype"`
	Markdown dingTalkMarkdown `json:"markdown"`
}

func (n *MultiNotifier) sendDingTalkVuln(ctx context.Context, vuln *model.VulnRecord) error {
	severityEmoji := map[model.SeverityLevel]string{
		model.SeverityCritical: "🔴",
		model.SeverityHigh:     "🟠",
		model.SeverityMedium:   "🟡",
		model.SeverityLow:      "🟢",
	}

	emoji := severityEmoji[vuln.Severity]
	if emoji == "" {
		emoji = "⚪"
	}

	text := fmt.Sprintf("## %s %s\n\n"+
		"**CVE:** %s\n\n"+
		"**Severity:** %s\n\n"+
		"**Source:** %s\n\n"+
		"**Disclosure:** %s\n\n"+
		"**Description:**\n%s\n\n"+
		"[View Details](%s)",
		emoji, vuln.Title,
		vuln.CVE,
		vuln.Severity,
		vuln.Source,
		vuln.Disclosure,
		vuln.Description,
		vuln.URL,
	)

	msg := dingTalkMessage{
		MsgType: "markdown",
		Markdown: dingTalkMarkdown{
			Title: fmt.Sprintf("%s %s", emoji, vuln.Title),
			Text:  text,
		},
	}

	return n.sendDingTalk(ctx, msg)
}

func (n *MultiNotifier) sendDingTalkArticle(ctx context.Context, article *model.Article) error {
	text := fmt.Sprintf("## 📰 %s\n\n"+
		"**Source:** %s\n\n"+
		"**Published:** %s\n\n"+
		"**Summary:**\n%s\n\n"+
		"[Read More](%s)",
		article.Title,
		article.Source,
		article.PublishedAt.Format("2006-01-02"),
		article.Summary,
		article.URL,
	)

	msg := dingTalkMessage{
		MsgType: "markdown",
		Markdown: dingTalkMarkdown{
			Title: article.Title,
			Text:  text,
		},
	}

	return n.sendDingTalk(ctx, msg)
}

func (n *MultiNotifier) sendDingTalkAlert(ctx context.Context, title, content string) error {
	text := fmt.Sprintf("## 🚨 %s\n\n%s", title, content)

	msg := dingTalkMessage{
		MsgType: "markdown",
		Markdown: dingTalkMarkdown{
			Title: title,
			Text:  text,
		},
	}

	return n.sendDingTalk(ctx, msg)
}

func (n *MultiNotifier) sendDingTalk(ctx context.Context, msg dingTalkMessage) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshaling message: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, n.config.DingTalkWebhook, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := n.client.Do(req)
	if err != nil {
		return fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

// FeiShu message structures
type feiShuContent struct {
	Text string `json:"text"`
}

type feiShuMessage struct {
	MsgType string        `json:"msg_type"`
	Content feiShuContent `json:"content"`
}

func (n *MultiNotifier) sendFeiShuVuln(ctx context.Context, vuln *model.VulnRecord) error {
	severityEmoji := map[model.SeverityLevel]string{
		model.SeverityCritical: "🔴",
		model.SeverityHigh:     "🟠",
		model.SeverityMedium:   "🟡",
		model.SeverityLow:      "🟢",
	}

	emoji := severityEmoji[vuln.Severity]
	if emoji == "" {
		emoji = "⚪"
	}

	text := fmt.Sprintf("%s **%s**\n\n"+
		"CVE: %s\n"+
		"Severity: %s\n"+
		"Source: %s\n"+
		"Disclosure: %s\n\n"+
		"%s\n\n"+
		"Details: %s",
		emoji, vuln.Title,
		vuln.CVE,
		vuln.Severity,
		vuln.Source,
		vuln.Disclosure,
		vuln.Description,
		vuln.URL,
	)

	msg := feiShuMessage{
		MsgType: "text",
		Content: feiShuContent{Text: text},
	}

	return n.sendFeiShu(ctx, msg)
}

func (n *MultiNotifier) sendFeiShuArticle(ctx context.Context, article *model.Article) error {
	text := fmt.Sprintf("📰 **%s**\n\n"+
		"Source: %s\n"+
		"Published: %s\n\n"+
		"%s\n\n"+
		"Read more: %s",
		article.Title,
		article.Source,
		article.PublishedAt.Format("2006-01-02"),
		article.Summary,
		article.URL,
	)

	msg := feiShuMessage{
		MsgType: "text",
		Content: feiShuContent{Text: text},
	}

	return n.sendFeiShu(ctx, msg)
}

func (n *MultiNotifier) sendFeiShuAlert(ctx context.Context, title, content string) error {
	text := fmt.Sprintf("🚨 **%s**\n\n%s", title, content)

	msg := feiShuMessage{
		MsgType: "text",
		Content: feiShuContent{Text: text},
	}

	return n.sendFeiShu(ctx, msg)
}

func (n *MultiNotifier) sendFeiShu(ctx context.Context, msg feiShuMessage) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshaling message: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, n.config.FeiShuWebhook, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := n.client.Do(req)
	if err != nil {
		return fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}
