// Package pusher provides notification services for SecFlow.
// It supports multiple notification channels like DingTalk, Lark, WeChat Work, Slack, Telegram, Discord, etc.
package pusher

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Discord implements TextPusher for Discord webhooks.
type Discord struct {
	webhookURL string
	client     *http.Client
}

// NewDiscord creates a new Discord pusher with the given webhook URL.
// The webhook URL should be in the format: https://discord.com/api/webhooks/{webhook_id}/{webhook_token}
func NewDiscord(webhookURL string) *Discord {
	return &Discord{
		webhookURL: webhookURL,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// DiscordMessage represents a Discord webhook message payload.
type DiscordMessage struct {
	Content   string            `json:"content,omitempty"`
	Username  string            `json:"username,omitempty"`
	AvatarURL string            `json:"avatar_url,omitempty"`
	Embeds    []DiscordEmbed    `json:"embeds,omitempty"`
}

// DiscordEmbed represents an embedded message in Discord.
type DiscordEmbed struct {
	Title       string          `json:"title,omitempty"`
	Description string          `json:"description,omitempty"`
	Color       int             `json:"color,omitempty"`
	URL         string          `json:"url,omitempty"`
	Fields      []DiscordField  `json:"fields,omitempty"`
	Footer      *DiscordFooter  `json:"footer,omitempty"`
	Timestamp   string          `json:"timestamp,omitempty"`
}

// DiscordField represents a field in a Discord embed.
type DiscordField struct {
	Name   string `json:"name,omitempty"`
	Value  string `json:"value,omitempty"`
	Inline bool   `json:"inline,omitempty"`
}

// DiscordFooter represents a footer in a Discord embed.
type DiscordFooter struct {
	Text    string `json:"text,omitempty"`
	IconURL string `json:"icon_url,omitempty"`
}

// PushText sends a plain text message to Discord.
func (d *Discord) PushText(ctx context.Context, text string) error {
	msg := DiscordMessage{
		Content: text,
	}
	return d.send(ctx, msg)
}

// PushMarkdown sends a markdown formatted message to Discord as an embed.
func (d *Discord) PushMarkdown(ctx context.Context, title, content string) error {
	msg := DiscordMessage{
		Username: "SecFlow",
		Embeds: []DiscordEmbed{
			{
				Title:       title,
				Description: content,
				Color:       0x3498db, // Blue color
				Timestamp:   time.Now().Format(time.RFC3339),
			},
		},
	}
	return d.send(ctx, msg)
}

// PushVulnAsEmbed sends a vulnerability notification as a rich Discord embed.
func (d *Discord) PushVulnAsEmbed(ctx context.Context, vuln *VulnInfo) error {
	// Determine color based on severity
	color := 0x3498db // Default blue
	switch vuln.Severity {
	case "CRITICAL":
		color = 0xe74c3c // Red
	case "HIGH":
		color = 0xe67e22 // Orange
	case "MEDIUM":
		color = 0xf1c40f // Yellow
	case "LOW":
		color = 0x2ecc71 // Green
	}

	// Build fields
	fields := []DiscordField{
		{Name: "Severity", Value: vuln.Severity, Inline: true},
		{Name: "Source", Value: vuln.Source, Inline: true},
	}
	if vuln.CVE != "" {
		fields = append(fields, DiscordField{Name: "CVE", Value: vuln.CVE, Inline: true})
	}

	msg := DiscordMessage{
		Username: "SecFlow Vuln Alert",
		Embeds: []DiscordEmbed{
			{
				Title:       vuln.Title,
				Description: truncateString(vuln.Description, 500),
				Color:       color,
				URL:         vuln.URL,
				Fields:      fields,
				Timestamp:   time.Now().Format(time.RFC3339),
				Footer: &DiscordFooter{
					Text: "SecFlow Security Intelligence",
				},
			},
		},
	}
	return d.send(ctx, msg)
}

// send sends a message to Discord.
func (d *Discord) send(ctx context.Context, msg DiscordMessage) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal discord message: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, d.webhookURL, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("send message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("discord returned status %d", resp.StatusCode)
	}

	return nil
}

// truncateString truncates a string to a maximum length.
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// VulnInfo contains vulnerability information for notifications.
type VulnInfo struct {
	Title       string
	Description string
	Severity    string
	CVE         string
	Source      string
	URL         string
}
