package report

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewMinimaxService(t *testing.T) {
	// Test with no API key
	svc := NewMinimaxService("", "", AIModelNone)
	assert.False(t, svc.IsEnabled())

	// Test with API key but no groupID
	svc = NewMinimaxService("test-key", "", AIModelNone)
	assert.False(t, svc.IsEnabled())

	// Test with all parameters
	svc = NewMinimaxService("test-key", "group-123", AIModelGPT4)
	assert.True(t, svc.IsEnabled())
	assert.Equal(t, AIModelGPT4, svc.GetModel())
}

func TestMinimaxService_GetModelName(t *testing.T) {
	svc := &MinimaxService{model: AIModelGPT4}
	assert.Equal(t, MinimaxModelAbab6_5g, svc.getModelName())

	svc.model = AIModelGemini
	assert.Equal(t, MinimaxModelAbab6_5s, svc.getModelName())

	svc.model = AIModelClaude3
	assert.Equal(t, MinimaxModelAbab6_5g, svc.getModelName())
}

func TestMinimaxService_ParseSummaryResponse(t *testing.T) {
	svc := &MinimaxService{model: AIModelGPT4}

	response := `【概况】
本周安全态势整体平稳，共收录漏洞36个。

【重点】
1. Microsoft .NET 存在远程代码执行漏洞
2. Adobe Commerce 存在跨站脚本漏洞
3. Google Chrome 存在未明漏洞

【建议】
建议及时更新受影响产品的安全补丁

【趋势】
本周漏洞数量较上周持平，高危漏洞占比略有上升`

	summary := svc.parseSummaryResponse(response)

	assert.Equal(t, "gpt-4", summary.Model)
	assert.Contains(t, summary.Summary, "安全态势")
	assert.Len(t, summary.Highlights, 3)
	assert.Contains(t, summary.Trends, "持平")
}
