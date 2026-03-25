package pusher

import (
	"context"
	"fmt"

	"github.com/secflow/server/internal/model"
)

// Service provides high-level push notification services.
type Service struct {
	factory *PusherFactory
}

// NewService creates a new push service.
func NewService() *Service {
	return &Service{
		factory: NewPusherFactory(),
	}
}

// collectPushers creates TextPusher instances from enabled channels.
func (s *Service) collectPushers(channels []*model.PushChannel) ([]TextPusher, error) {
	if len(channels) == 0 {
		return nil, nil
	}

	var pushers []TextPusher
	for _, channel := range channels {
		if !channel.Enabled {
			continue
		}

		pusher, err := s.factory.CreateFromChannel(channel)
		if err != nil {
			return nil, fmt.Errorf("create pusher for channel %s: %w", channel.Name, err)
		}
		pushers = append(pushers, pusher)
	}

	return pushers, nil
}

// PushVulnToChannels pushes a vulnerability notification to multiple channels.
func (s *Service) PushVulnToChannels(ctx context.Context, vuln *model.VulnRecord, channels []*model.PushChannel) error {
	pushers, err := s.collectPushers(channels)
	if err != nil {
		return err
	}
	if len(pushers) == 0 {
		return nil
	}

	// Render the vulnerability message
	markdown := RenderVuln(vuln)
	title := fmt.Sprintf("%s %s", formatSeverity(string(vuln.Severity)), vuln.Title)

	// Send to all channels
	multi := NewMulti(pushers...)
	return multi.PushMarkdown(ctx, title, markdown)
}

// PushArticleToChannels pushes an article notification to multiple channels.
func (s *Service) PushArticleToChannels(ctx context.Context, article *model.Article, channels []*model.PushChannel) error {
	pushers, err := s.collectPushers(channels)
	if err != nil {
		return err
	}
	if len(pushers) == 0 {
		return nil
	}

	// Render the article message
	markdown := RenderArticle(article)

	// Send to all channels
	multi := NewMulti(pushers...)
	return multi.PushMarkdown(ctx, article.Title, markdown)
}

// PushAlertToChannels pushes an alert notification to multiple channels.
func (s *Service) PushAlertToChannels(ctx context.Context, title, content string, channels []*model.PushChannel) error {
	pushers, err := s.collectPushers(channels)
	if err != nil {
		return err
	}
	if len(pushers) == 0 {
		return nil
	}

	// Render the alert message
	markdown := RenderAlert(title, content)

	// Send to all channels
	multi := NewMulti(pushers...)
	return multi.PushMarkdown(ctx, title, markdown)
}

// PushTestMessage sends a test message to all channels.
func (s *Service) PushTestMessage(ctx context.Context, channels []*model.PushChannel) error {
	pushers, err := s.collectPushers(channels)
	if err != nil {
		return err
	}
	if len(pushers) == 0 {
		return nil
	}

	text := "🔔 SecFlow 推送测试\n\n这是一条测试消息，用于验证推送通道配置是否正确。"

	// Send to all channels
	multi := NewMulti(pushers...)
	return multi.PushText(ctx, text)
}

// ValidateChannel validates a push channel configuration.
func (s *Service) ValidateChannel(ctx context.Context, channel *model.PushChannel) error {
	if channel.Type == "" {
		return fmt.Errorf("channel type is required")
	}

	switch channel.Type {
	case "dingding":
		if channel.Config["access_token"] == "" {
			return fmt.Errorf("access_token is required for dingding")
		}
	case "lark":
		if channel.Config["access_token"] == "" {
			return fmt.Errorf("access_token is required for lark")
		}
	case "wechat_work":
		if channel.Config["webhook_url"] == "" {
			return fmt.Errorf("webhook_url is required for wechat_work")
		}
	case "slack":
		if channel.Config["webhook_url"] == "" {
			return fmt.Errorf("webhook_url is required for slack")
		}
	case "telegram":
		if channel.Config["bot_token"] == "" {
			return fmt.Errorf("bot_token is required for telegram")
		}
		if channel.Config["chat_id"] == "" {
			return fmt.Errorf("chat_id is required for telegram")
		}
	case "webhook":
		if channel.Config["url"] == "" {
			return fmt.Errorf("url is required for webhook")
		}
	default:
		return fmt.Errorf("unsupported channel type: %s", channel.Type)
	}

	return nil
}

// TestChannel sends a test message to a specific channel.
func (s *Service) TestChannel(ctx context.Context, channel *model.PushChannel) error {
	if err := s.ValidateChannel(ctx, channel); err != nil {
		return err
	}

	pusher, err := s.factory.CreateFromChannel(channel)
	if err != nil {
		return fmt.Errorf("create pusher: %w", err)
	}

	text := fmt.Sprintf("🔔 SecFlow 推送测试 - %s\n\n通道类型: %s\n配置验证: ✓", 
		channel.Name, channel.Type)

	return pusher.PushText(ctx, text)
}
