// Package pusher provides notification services for SecFlow.
// It supports multiple notification channels like DingTalk, Lark, WeChat Work, Slack, Telegram, etc.
package pusher

import (
	"context"
	"time"

	"github.com/secflow/server/internal/model"
)

// TextPusher is the interface for pushing text and markdown messages.
type TextPusher interface {
	// PushText sends a plain text message.
	PushText(ctx context.Context, text string) error
	// PushMarkdown sends a markdown formatted message.
	PushMarkdown(ctx context.Context, title, content string) error
}

// RawPusher is the interface for pushing raw messages.
type RawPusher interface {
	// PushRaw sends a raw message with custom format.
	PushRaw(ctx context.Context, raw *RawMessage) error
}

// VulnPusher is the interface for pushing vulnerability notifications.
type VulnPusher interface {
	// PushVuln sends a vulnerability notification.
	PushVuln(ctx context.Context, vuln *model.VulnRecord) error
}

// ArticlePusher is the interface for pushing article notifications.
type ArticlePusher interface {
	// PushArticle sends an article notification.
	PushArticle(ctx context.Context, article *model.Article) error
}

// RawMessage represents a raw message with custom format.
type RawMessage struct {
	Type    string
	Payload interface{}
}

// PusherFactory creates pushers from configuration.
type PusherFactory struct {
	timeout time.Duration
}

// NewPusherFactory creates a new PusherFactory.
func NewPusherFactory() *PusherFactory {
	return &PusherFactory{
		timeout: 10 * time.Second, // Default timeout for HTTP requests
	}
}

// WithTimeout sets the HTTP timeout for pushers.
func (f *PusherFactory) WithTimeout(timeout time.Duration) *PusherFactory {
	f.timeout = timeout
	return f
}

// CreateFromChannel creates a pusher from a PushChannel configuration.
func (f *PusherFactory) CreateFromChannel(channel *model.PushChannel) (TextPusher, error) {
	switch channel.Type {
	case "dingding":
		return NewDingDing(channel.Config["access_token"], channel.Config["sign_secret"]), nil
	case "lark":
		return NewLark(channel.Config["access_token"], channel.Config["sign_secret"]), nil
	case "wechat_work":
		return NewWeChatWork(channel.Config["access_token"], channel.Config["sign_secret"]), nil
	case "slack":
		return NewSlack(channel.Config["webhook_url"]), nil
	case "telegram":
		return NewTelegram(channel.Config["bot_token"], channel.Config["chat_id"]), nil
	case "webhook":
		return NewWebhook(channel.Config["url"], channel.Config["method"]), nil
	default:
		return nil, ErrUnsupportedChannel
	}
}
