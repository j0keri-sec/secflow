package pusher

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Common errors
var (
	// ErrUnsupportedChannel is returned when trying to create an unsupported channel type.
	ErrUnsupportedChannel = errors.New("unsupported channel type")
	
	// ErrInvalidConfig is returned when the channel configuration is invalid.
	ErrInvalidConfig = errors.New("invalid channel configuration")
	
	// ErrPushFailed is returned when a push operation fails.
	ErrPushFailed = errors.New("push failed")
)

// PushError represents a push error with channel context.
type PushError struct {
	Channel string
	Err     error
}

// Error implements the error interface.
func (e *PushError) Error() string {
	return fmt.Sprintf("%s: %v", e.Channel, e.Err)
}

// Unwrap returns the underlying error.
func (e *PushError) Unwrap() error {
	return e.Err
}

// jsonBody creates a JSON request body.
func jsonBody(data []byte) *bytes.Reader {
	return bytes.NewReader(data)
}

// escapeMarkdown escapes special markdown characters.
func escapeMarkdown(s string) string {
	// Characters that need escaping in markdown: \ ` * _ { } [ ] ( ) # + - . !
	replacer := strings.NewReplacer(
		"\\", "\\\\",
		"`", "\\`",
		"*", "\\*",
		"_", "\\_",
		"{", "\\{",
		"}", "\\}",
		"[", "\\[",
		"]", "\\]",
		"(", "\\(",
		")", "\\)",
		"#", "\\#",
		"+", "\\+",
		"-", "\\-",
		".", "\\.",
		"!", "\\!",
	)
	return replacer.Replace(s)
}

// truncate truncates a string to the specified length.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// truncateWords truncates a string to the specified number of words.
func truncateWords(s string, maxWords int) string {
	words := strings.Fields(s)
	if len(words) <= maxWords {
		return s
	}
	return strings.Join(words[:maxWords], " ") + "..."
}

// formatSeverity formats severity level with emoji.
func formatSeverity(severity string) string {
	emojiMap := map[string]string{
		"严重": "🔴",
		"高危": "🟠",
		"中危": "🟡",
		"低危": "🟢",
		"critical": "🔴",
		"high":     "🟠",
		"medium":   "🟡",
		"low":      "🟢",
	}
	
	if emoji, ok := emojiMap[strings.ToLower(severity)]; ok {
		return emoji
	}
	return "⚪"
}

// formatTime formats time in a human-readable way.
func formatTime(t time.Time) string {
	return t.Format("2006-01-02 15:04:05")
}

// joinWithLimit joins strings with a separator, limiting the total length.
func joinWithLimit(strs []string, sep string, maxLen int) string {
	if len(strs) == 0 {
		return ""
	}
	
	var result strings.Builder
	currentLen := 0
	
	for i, s := range strs {
		if i > 0 {
			if currentLen+len(sep) > maxLen {
				break
			}
			result.WriteString(sep)
			currentLen += len(sep)
		}
		
		if currentLen+len(s) > maxLen {
			remaining := maxLen - currentLen - 3 // 3 for "..."
			if remaining > 0 {
				result.WriteString(s[:remaining])
				result.WriteString("...")
			}
			break
		}
		
		result.WriteString(s)
		currentLen += len(s)
	}
	
	return result.String()
}
