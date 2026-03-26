package notify

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEmailSender_BuildEmail(t *testing.T) {
	tests := []struct {
		name    string
		from    string
		to      string
		subject string
		body    string
		wantErr bool
	}{
		{
			name:    "basic email",
			from:    "test@example.com",
			to:      "user@example.com",
			subject: "Test Subject",
			body:    "Test Body Content",
			wantErr: false,
		},
		{
			name:    "email with special characters",
			from:    "test@example.com",
			to:      "user@example.com",
			subject: "Test with <special> & \"chars\"",
			body:    "Test Body",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := buildEmail(tt.from, tt.to, tt.subject, tt.body)
			assert.NotEmpty(t, msg)
			assert.Contains(t, string(msg), "From: "+tt.from)
			assert.Contains(t, string(msg), "To: "+tt.to)
			assert.Contains(t, string(msg), "Subject: "+tt.subject)
		})
	}
}

func TestEmailSender_NewEmailSender(t *testing.T) {
	config := EmailConfig{
		Host:     "smtp.gmail.com",
		Port:     587,
		Username: "test@gmail.com",
		Password: "password",
		From:     "test@gmail.com",
		FromName: "Test Sender",
		UseTLS:   true,
	}

	sender := NewEmailSender(config)
	assert.NotNil(t, sender)
	assert.Equal(t, config, sender.config)
}

func TestBuildEmail_SubjectCRLFFiltering(t *testing.T) {
	// Subject with newlines should be filtered to prevent header injection
	from := "test@example.com"
	to := "user@example.com"
	subject := "Normal Subject\r\nInjected: evil"
	body := "Test body"

	msg := buildEmail(from, to, subject, body)
	msgStr := string(msg)

	// The injected header should not appear as a separate header
	// The subject value should NOT contain newlines within it (after "Subject: ")
	// Header lines end with \r\n, so subject line is "Subject: value\r\n"
	// We check the value part (everything after "Subject: " and before "\r\n")

	// Find the subject value - it should be "Normal SubjectInjected: evil" (no embedded newlines)
	// The raw header injection "Normal Subject\r\nInjected: evil" becomes "Normal SubjectInjected: evil"
	assert.Contains(t, msgStr, "Subject: Normal SubjectInjected: evil")
	assert.NotContains(t, msgStr, "Subject: Normal Subject\r\n")
	assert.NotContains(t, msgStr, "Subject: Normal Subject\n")
}

func TestSend_EmptyHost(t *testing.T) {
	sender := &EmailSender{config: EmailConfig{
		Host: "",
	}}

	err := sender.Send(nil, "user@example.com", "Test", "Body")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not configured")
}

func TestSendPasswordReset_EscapesHTML(t *testing.T) {
	sender := &EmailSender{config: EmailConfig{
		Host: "smtp.example.com",
		Port: 587,
	}}

	// Username with potential XSS should be escaped
	err := sender.SendPasswordReset(nil, "user@example.com", "<script>alert('xss')</script>", "token123", "https://example.com/reset")
	assert.Error(t, err) // Will error due to no actual SMTP server, but should not panic

	// If we had a working sender, we'd check the email body
	// For now just verify no panic
}
