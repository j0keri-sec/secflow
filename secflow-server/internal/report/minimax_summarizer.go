package report

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// MinimaxService provides AI-powered report summarization using Minimax API.
type MinimaxService struct {
	apiKey  string
	groupID string
	model   AIModelType
	enabled bool
	client  *http.Client
}

// Minimax API endpoint
const minimaxAPIURL = "https://api.minimax.chat/v1/text/chatcompletion_pro"

const (
	// Minimax models
	MinimaxModelAbab6_5s = "abab6.5s"
	MinimaxModelAbab6_5g = "abab6.5g"
	MinimaxModelAbab5_5  = "abab5.5"
)

// NewMinimaxService creates a new Minimax AI service.
func NewMinimaxService(apiKey string, groupID string, model AIModelType) *MinimaxService {
	if apiKey == "" || groupID == "" || model == AIModelNone {
		return &MinimaxService{enabled: false}
	}

	return &MinimaxService{
		apiKey:  apiKey,
		groupID: groupID,
		model:   model,
		enabled: true,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// IsEnabled returns whether AI summarization is available.
func (s *MinimaxService) IsEnabled() bool {
	return s.enabled
}

// GetModel returns the current AI model.
func (s *MinimaxService) GetModel() AIModelType {
	return s.model
}

// MinimaxRequest represents the API request body.
type MinimaxRequest struct {
	Model       string       `json:"model"`
	Messages    []MinimaxMsg `json:"messages"`
	Temperature float64      `json:"temperature,omitempty"`
	MaxTokens  int          `json:"max_tokens,omitempty"`
}

// MinimaxMsg represents a chat message.
type MinimaxMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// MinimaxResponse represents the API response.
type MinimaxResponse struct {
	ID      string   `json:"id"`
	Choices []Choice `json:"choices"`
	Usage   Usage   `json:"usage"`
}

// Choice represents a response choice.
type Choice struct {
	FinishReason string     `json:"finish_reason"`
	Message     MinimaxMsg `json:"message"`
}

// Usage represents token usage.
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens     int `json:"total_tokens"`
}

// Summarize generates an AI summary for the report data.
func (s *MinimaxService) Summarize(ctx context.Context, data *ReportData) (*AISummary, error) {
	if !s.enabled {
		return nil, fmt.Errorf("Minimax service is not enabled")
	}

	// Build the prompt
	prompt := s.buildSummarizePrompt(data)

	// Prepare messages
	messages := []MinimaxMsg{
		{
			Role:    "system",
			Content: "你是一个专业的网络安全分析师，负责总结安全周报的关键信息。",
		},
		{
			Role:    "user",
			Content: prompt,
		},
	}

	// Get model name
	modelName := s.getModelName()

	// Create request
	reqBody := MinimaxRequest{
		Model:       modelName,
		Messages:    messages,
		Temperature: 0.7,
		MaxTokens:   2000,
	}

	reqJSON, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", minimaxAPIURL, bytes.NewBuffer(reqJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.apiKey))

	// Send request
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Minimax API call failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Minimax API error: status=%d, body=%s", resp.StatusCode, string(body))
	}

	// Parse response
	var minimaxResp MinimaxResponse
	if err := json.Unmarshal(body, &minimaxResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(minimaxResp.Choices) == 0 {
		return nil, fmt.Errorf("no response from Minimax")
	}

	content := minimaxResp.Choices[0].Message.Content

	// Parse the response
	return s.parseSummaryResponse(content), nil
}

// buildSummarizePrompt builds a prompt for the AI to summarize the report.
func (s *MinimaxService) buildSummarizePrompt(data *ReportData) string {
	var buf bytes.Buffer

	buf.WriteString("请分析以下安全周报数据，生成简洁的摘要：\n\n")

	// Add stats
	buf.WriteString("【漏洞统计】\n")
	buf.WriteString(fmt.Sprintf("报告周期：%s 至 %s\n",
		data.DateFrom.Format("2006/01/02"), data.DateTo.Format("2006/01/02")))

	totalVulns := 0
	for _, stats := range data.VulnStats {
		if stats != nil {
			totalVulns += stats.Total
		}
	}
	buf.WriteString(fmt.Sprintf("本周共收录漏洞：%d 个\n\n", totalVulns))

	// Add top vulnerabilities
	if len(data.AllTopVulns) > 0 {
		buf.WriteString("【重点漏洞 TOP 5】\n")
		for i, v := range data.AllTopVulns {
			if i >= 5 {
				break
			}
			buf.WriteString(fmt.Sprintf("%d. %s (%s)\n", i+1, v.Name, v.CVE))
		}
		buf.WriteString("\n")
	}

	// Add events
	if len(data.Events) > 0 {
		buf.WriteString("【安全事件】\n")
		for i, e := range data.Events {
			if i >= 3 {
				break
			}
			buf.WriteString(fmt.Sprintf("- %s\n", e.Title))
		}
		buf.WriteString("\n")
	}

	buf.WriteString("请生成以下格式的摘要：\n")
	buf.WriteString("【概况】一句话总结本周安全态势\n")
	buf.WriteString("【重点】列出2-3个最值得关注的问题\n")
	buf.WriteString("【建议】给出简要的安全改进建议\n")
	buf.WriteString("【趋势】分析漏洞趋势（上升/下降/持平）\n")

	return buf.String()
}

// parseSummaryResponse parses the AI response into structured summary.
func (s *MinimaxService) parseSummaryResponse(content string) *AISummary {
	summary := &AISummary{
		Model:      string(s.model),
		Highlights: []string{},
	}

	lines := strings.Split(content, "\n")
	currentSection := ""

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Detect section headers
		lower := strings.ToLower(line)
		if strings.Contains(lower, "概况") || strings.Contains(lower, "总结") || strings.Contains(lower, "摘要") {
			currentSection = "summary"
			continue
		} else if strings.Contains(lower, "重点") || strings.Contains(lower, "关注") || strings.Contains(lower, "关键") {
			currentSection = "highlights"
			continue
		} else if strings.Contains(lower, "建议") || strings.Contains(lower, "修复") {
			currentSection = "advice"
			continue
		} else if strings.Contains(lower, "趋势") || strings.Contains(lower, "分析") {
			currentSection = "trends"
			continue
		}

		// Remove markdown formatting
		line = strings.TrimPrefix(line, "**")
		line = strings.TrimSuffix(line, "**")
		line = strings.TrimPrefix(line, "- ")
		line = strings.TrimPrefix(line, "* ")
		line = strings.Trim(line, "-* ")

		switch currentSection {
		case "summary":
			if summary.Summary == "" {
				summary.Summary = line
			}
		case "highlights":
			if line != "" && !strings.HasPrefix(line, "【") {
				summary.Highlights = append(summary.Highlights, line)
			}
		case "advice":
			if summary.Advice == "" {
				summary.Advice = line
			} else {
				summary.Advice += " " + line
			}
		case "trends":
			if summary.Trends == "" {
				summary.Trends = line
			}
		}
	}

	return summary
}

// getModelName returns the Minimax model name.
func (s *MinimaxService) getModelName() string {
	switch s.model {
	case AIModelClaude3:
		return MinimaxModelAbab6_5g
	case AIModelGemini:
		return MinimaxModelAbab6_5s
	default:
		return MinimaxModelAbab6_5g
	}
}


