// Package notify provides notification services for SecFlow.
package notify

import (
	"context"
	"crypto/tls"
	"fmt"
	"html"
	"net"
	"net/smtp"
	"strings"
	"time"
)

// EmailConfig holds SMTP configuration for sending emails.
type EmailConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
	FromName string
	UseTLS   bool
}

// EmailSender handles sending emails via SMTP.
type EmailSender struct {
	config EmailConfig
}

// NewEmailSender creates a new email sender with the given configuration.
func NewEmailSender(config EmailConfig) *EmailSender {
	return &EmailSender{config: config}
}

// Send sends an email to the specified recipient.
func (e *EmailSender) Send(ctx context.Context, to, subject, body string) error {
	if e.config.Host == "" {
		return fmt.Errorf("email sender not configured: SMTP host is empty")
	}

	// Build email message
	from := e.config.From
	if e.config.FromName != "" {
		from = fmt.Sprintf("%s <%s>", e.config.FromName, e.config.From)
	}

	msg := buildEmail(from, to, subject, body)

	// SMTP server address
	addr := fmt.Sprintf("%s:%d", e.config.Host, e.config.Port)

	// Authentication
	auth := smtp.PlainAuth("", e.config.Username, e.config.Password, e.config.Host)

	// Choose sending method based on TLS setting
	if e.config.UseTLS {
		// Implicit TLS (typically port 465)
		return e.sendWithTLS(addr, auth, from, to, msg)
	}
	// Use STARTTLS (typically port 587) for opportunistic TLS
	return e.sendWithSTARTTLS(addr, auth, from, to, msg)
}

// sendWithTLS sends email using implicit TLS (port 465).
func (e *EmailSender) sendWithTLS(addr string, auth smtp.Auth, from, to string, msg []byte) error {
	// Connect to SMTP server with TLS
	tlsConfig := &tls.Config{
		ServerName: e.config.Host,
	}

	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("TLS connection failed: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, e.config.Host)
	if err != nil {
		return fmt.Errorf("create SMTP client: %w", err)
	}
	defer client.Close()

	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("SMTP auth: %w", err)
	}

	if err := client.Mail(from); err != nil {
		return fmt.Errorf("SMTP MAIL FROM: %w", err)
	}

	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("SMTP RCPT TO: %w", err)
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("SMTP DATA: %w", err)
	}

	if _, err := w.Write(msg); err != nil {
		return fmt.Errorf("write email body: %w", err)
	}

	if err := w.Close(); err != nil {
		return fmt.Errorf("close email writer: %w", err)
	}

	return client.Quit()
}

// sendWithSTARTTLS sends email with explicit STARTTLS upgrade (port 587).
func (e *EmailSender) sendWithSTARTTLS(addr string, auth smtp.Auth, from, to string, msg []byte) error {
	dialer := &net.Dialer{Timeout: 30 * time.Second}
	conn, err := dialer.Dial("tcp", addr)
	if err != nil {
		return fmt.Errorf("dial SMTP server: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, e.config.Host)
	if err != nil {
		return fmt.Errorf("create SMTP client: %w", err)
	}
	defer client.Close()

	// Send EHLO
	if err := client.Hello("secflow"); err != nil {
		return fmt.Errorf("SMTP EHLO: %w", err)
	}

	// Upgrade to TLS
	tlsConfig := &tls.Config{
		ServerName: e.config.Host,
	}
	if err := client.StartTLS(tlsConfig); err != nil {
		return fmt.Errorf("STARTTLS upgrade: %w", err)
	}

	// Re-authenticate after TLS
	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("SMTP auth after STARTTLS: %w", err)
	}

	if err := client.Mail(from); err != nil {
		return fmt.Errorf("SMTP MAIL FROM: %w", err)
	}

	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("SMTP RCPT TO: %w", err)
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("SMTP DATA: %w", err)
	}

	if _, err := w.Write(msg); err != nil {
		return fmt.Errorf("write email body: %w", err)
	}

	if err := w.Close(); err != nil {
		return fmt.Errorf("close email writer: %w", err)
	}

	return client.Quit()
}

// buildEmail constructs a RFC 822 email message.
func buildEmail(from, to, subject, body string) []byte {
	var sb strings.Builder
	// Escape CRLF in headers to prevent header injection
	subject = strings.ReplaceAll(subject, "\r", "")
	subject = strings.ReplaceAll(subject, "\n", "")
	sb.WriteString(fmt.Sprintf("From: %s\r\n", from))
	sb.WriteString(fmt.Sprintf("To: %s\r\n", to))
	sb.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	sb.WriteString("MIME-Version: 1.0\r\n")
	sb.WriteString("Content-Type: text/html; charset=\"UTF-8\"\r\n")
	sb.WriteString("Content-Transfer-Encoding: 8bit\r\n")
	sb.WriteString("\r\n")
	sb.WriteString(body)
	sb.WriteString("\r\n")
	return []byte(sb.String())
}

// SendPasswordReset sends a password reset email to the user.
func (e *EmailSender) SendPasswordReset(ctx context.Context, toEmail, username, resetToken, resetURL string) error {
	subject := "重置您的 SecFlow 密码 / Reset Your SecFlow Password"

	// Escape HTML entities to prevent XSS in email
	safeUsername := html.EscapeString(username)

	// HTML body with professional styling
	body := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Password Reset</title>
</head>
<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; max-width: 600px; margin: 0 auto; padding: 20px; color: #333;">
    <div style="background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%); padding: 30px; border-radius: 10px 10px 0 0;">
        <h1 style="color: white; margin: 0; font-size: 24px;">SecFlow</h1>
    </div>
    <div style="background: #f9f9f9; padding: 30px; border-radius: 0 0 10px 10px; border: 1px solid #e0e0e0;">
        <p style="font-size: 16px;">Hi <strong>%s</strong>,</p>
        <p style="font-size: 14px; line-height: 1.6;">We received a request to reset your SecFlow password. If you didn't make this request, you can safely ignore this email.</p>
        <p style="font-size: 14px; line-height: 1.6;">Your password reset token:</p>
        <div style="background: #fff; border: 2px dashed #667eea; padding: 15px; border-radius: 5px; margin: 20px 0; text-align: center;">
            <code style="font-size: 18px; font-family: 'Courier New', monospace; color: #667eea; word-break: break-all;">%s</code>
        </div>
        <p style="font-size: 12px; color: #666;">This token will expire in 15 minutes.</p>
        <p style="font-size: 14px; line-height: 1.6;">Or click the link below to reset your password:</p>
        <div style="text-align: center; margin: 25px 0;">
            <a href="%s" style="display: inline-block; background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%); color: white; padding: 12px 30px; border-radius: 5px; text-decoration: none; font-weight: bold;">Reset Password</a>
        </div>
        <hr style="border: none; border-top: 1px solid #e0e0e0; margin: 25px 0;">
        <p style="font-size: 12px; color: #999;">This email was sent by SecFlow Security Platform.<br>Please do not reply to this email.</p>
    </div>
</body>
</html>
`, safeUsername, resetToken, resetURL)

	return e.Send(ctx, toEmail, subject, body)
}
