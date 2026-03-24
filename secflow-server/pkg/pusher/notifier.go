package pusher

import (
	"context"
	"fmt"

	"github.com/secflow/server/internal/model"
	"github.com/secflow/server/internal/repository"
)

// Notifier provides backward-compatible notification interface.
type Notifier struct {
	service      *Service
	channelRepo  *repository.PushChannelRepository
}

// NewNotifier creates a new notifier.
func NewNotifier(service *Service, channelRepo *repository.PushChannelRepository) *Notifier {
	return &Notifier{
		service:     service,
		channelRepo: channelRepo,
	}
}

// SendVuln sends vulnerability notification to enabled channels.
func (n *Notifier) SendVuln(ctx context.Context, vuln *model.VulnRecord) error {
	// Get all enabled channels
	channels, err := n.channelRepo.List(ctx)
	if err != nil {
		return fmt.Errorf("list channels: %w", err)
	}

	var enabledChannels []*model.PushChannel
	for _, ch := range channels {
		if ch.Enabled {
			enabledChannels = append(enabledChannels, ch)
		}
	}

	if len(enabledChannels) == 0 {
		return nil // No enabled channels
	}

	return n.service.PushVulnToChannels(ctx, vuln, enabledChannels)
}

// SendArticle sends article notification to enabled channels.
func (n *Notifier) SendArticle(ctx context.Context, article *model.Article) error {
	// Get all enabled channels
	channels, err := n.channelRepo.List(ctx)
	if err != nil {
		return fmt.Errorf("list channels: %w", err)
	}

	var enabledChannels []*model.PushChannel
	for _, ch := range channels {
		if ch.Enabled {
			enabledChannels = append(enabledChannels, ch)
		}
	}

	if len(enabledChannels) == 0 {
		return nil // No enabled channels
	}

	return n.service.PushArticleToChannels(ctx, article, enabledChannels)
}

// SendAlert sends alert notification to enabled channels.
func (n *Notifier) SendAlert(ctx context.Context, title, content string) error {
	// Get all enabled channels
	channels, err := n.channelRepo.List(ctx)
	if err != nil {
		return fmt.Errorf("list channels: %w", err)
	}

	var enabledChannels []*model.PushChannel
	for _, ch := range channels {
		if ch.Enabled {
			enabledChannels = append(enabledChannels, ch)
		}
	}

	if len(enabledChannels) == 0 {
		return nil // No enabled channels
	}

	return n.service.PushAlertToChannels(ctx, title, content, enabledChannels)
}

// SendTest sends a test message to all enabled channels.
func (n *Notifier) SendTest(ctx context.Context) error {
	channels, err := n.channelRepo.List(ctx)
	if err != nil {
		return fmt.Errorf("list channels: %w", err)
	}

	return n.service.PushTestMessage(ctx, channels)
}
