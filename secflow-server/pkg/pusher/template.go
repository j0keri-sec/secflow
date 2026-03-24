package pusher

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"
	"time"

	"github.com/secflow/server/internal/model"
)

const (
	// Default templates for vulnerability notifications
	vulnTemplate = `# {{ .SeverityEmoji }} {{ .Title }}

- **CVE编号**: {{ if .CVE }}**{{ .CVE }}**{{ else }}暂无{{ end }}
- **危害定级**: **{{ .Severity }}**
- **漏洞标签**: {{ range $i, $tag := .Tags }}{{ if $i }}, {{ end }}**{{ $tag }}**{{ end }}
- **披露日期**: **{{ .Disclosure }}**
- **信息来源**: [{{ .Source }}]({{ .From }})
{{ if .ReportedBy }}- **报告节点**: {{ .ReportedBy }}{{ end }}

{{ if .Description }}### **漏洞描述**
{{ .Description }}
{{ end }}

{{ if .Solutions }}### **修复方案**
{{ .Solutions }}

{{ end -}}
{{ if .References }}### **参考链接**
{{ range $i, $ref := .References }}{{ inc $i }}. [{{ $ref }}]({{ $ref }})
{{ end }}
{{ end -}}
{{ if .CVE }}### **开源检索**
{{ if .GithubSearch }}{{ range $i, $ref := .GithubSearch }}{{ inc $i }}. [{{ $ref }}]({{ $ref }})
{{ end }}
{{ else }}暂未找到
{{ end -}}{{ end -}}`

	articleTemplate = `# 📰 {{ .Title }}

- **作者**: {{ if .Author }}{{ .Author }}{{ else }}未知{{ end }}
- **来源**: **{{ .Source }}**
- **发布日期**: **{{ .PublishedAt }}**
- **阅读链接**: [{{ .URL }}]({{ .URL }})
{{ if .ReportedBy }}- **报告节点**: {{ .ReportedBy }}{{ end }}

{{ if .Summary }}### **文章摘要**
{{ .Summary }}

{{ end -}}
{{ if .Tags }}### **标签分类**
{{ range $i, $tag := .Tags }}{{ if $i }}, {{ end }}**{{ $tag }}**{{ end }}
{{ end -}}`

	alertTemplate = `# 🚨 {{ .Title }}

{{ .Content }}

---
时间: {{ .Time }}
来源: SecFlow Alert System`

	initialTemplate = `# SecFlow 初始化完成

**版本**: {{ .Version }}
**本地漏洞数量**: {{ .VulnCount }}
**检查周期**: {{ .Interval }}

### **成功的数据源**
{{ range .Provider }}- {{ .Name }} ({{ .Count }} 条)
{{ end }}

### **失败的数据源**
{{ if .FailedProvider }}{{ range .FailedProvider }}- {{ .Name }}: {{ .Error }}
{{ end }}{{ else }}无{{ end }}`
)

var (
	funcMap = template.FuncMap{
		"inc": func(i int) int { return i + 1 },
	}

	vulnTpl     = template.Must(template.New("vuln").Funcs(funcMap).Parse(vulnTemplate))
	articleTpl  = template.Must(template.New("article").Funcs(funcMap).Parse(articleTemplate))
	alertTpl    = template.Must(template.New("alert").Parse(alertTemplate))
	initialTpl  = template.Must(template.New("initial").Parse(initialTemplate))
)

// RenderVuln renders a vulnerability notification.
func RenderVuln(vuln *model.VulnRecord) string {
	data := struct {
		*model.VulnRecord
		SeverityEmoji string
	}{
		VulnRecord:    vuln,
		SeverityEmoji: formatSeverity(string(vuln.Severity)),
	}

	var buf bytes.Buffer
	if err := vulnTpl.Execute(&buf, data); err != nil {
		return fmt.Sprintf("Failed to render vulnerability: %v", err)
	}
	return buf.String()
}

// RenderArticle renders an article notification.
func RenderArticle(article *model.Article) string {
	data := struct {
		*model.Article
		PublishedAt string
	}{
		Article:     article,
		PublishedAt: formatTime(article.PublishedAt),
	}

	var buf bytes.Buffer
	if err := articleTpl.Execute(&buf, data); err != nil {
		return fmt.Sprintf("Failed to render article: %v", err)
	}
	return buf.String()
}

// RenderAlert renders an alert notification.
func RenderAlert(title, content string) string {
	data := struct {
		Title   string
		Content string
		Time    string
	}{
		Title:   title,
		Content: content,
		Time:    formatTime(time.Now()),
	}

	var buf bytes.Buffer
	if err := alertTpl.Execute(&buf, data); err != nil {
		return fmt.Sprintf("Failed to render alert: %v", err)
	}
	return buf.String()
}

// RenderInitial renders an initialization message.
func RenderInitial(version string, vulnCount int, interval string, providers []ProviderInfo, failed []FailedProvider) string {
	data := struct {
		Version        string
		VulnCount      int
		Interval       string
		Provider       []ProviderInfo
		FailedProvider []FailedProvider
	}{
		Version:        version,
		VulnCount:      vulnCount,
		Interval:       interval,
		Provider:       providers,
		FailedProvider: failed,
	}

	var buf bytes.Buffer
	if err := initialTpl.Execute(&buf, data); err != nil {
		return fmt.Sprintf("Failed to render initial message: %v", err)
	}
	return buf.String()
}

// ProviderInfo represents a provider's information.
type ProviderInfo struct {
	Name  string
	Count int
}

// FailedProvider represents a failed provider.
type FailedProvider struct {
	Name  string
	Error string
}

// ShortVulnInfo creates a short summary of a vulnerability.
func ShortVulnInfo(vuln *model.VulnRecord) string {
	severityEmoji := formatSeverity(string(vuln.Severity))
	tags := joinWithLimit(vuln.Tags, ", ", 50)
	
	return fmt.Sprintf("%s %s [%s] - %s", 
		severityEmoji,
		truncate(vuln.Title, 100),
		tags,
		vuln.CVE,
	)
}

// ShortArticleInfo creates a short summary of an article.
func ShortArticleInfo(article *model.Article) string {
	return fmt.Sprintf("📰 %s - %s (%s)",
		truncate(article.Title, 100),
		article.Source,
		article.PublishedAt.Format("2006-01-02"),
	)
}

// TruncateMarkdown truncates markdown content while preserving structure.
func TruncateMarkdown(content string, maxLines int) string {
	lines := strings.Split(content, "\n")
	if len(lines) <= maxLines {
		return content
	}

	truncated := lines[:maxLines]
	truncated = append(truncated, "\n*... (truncated)*")
	return strings.Join(truncated, "\n")
}
