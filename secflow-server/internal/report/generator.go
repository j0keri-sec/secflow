// Package report provides security report generation functionality.
package report

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/secflow/server/internal/repository"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

// ReportType represents the type of report to generate.
type ReportType string

const (
	ReportTypeWeekly  ReportType = "weekly"
	ReportTypeMonthly ReportType = "monthly"
	ReportTypeDaily  ReportType = "daily"
)

// ExportFormat represents the export file format.
type ExportFormat string

const (
	FormatMarkdown ExportFormat = "md"
	FormatHTML     ExportFormat = "html"
	FormatPDF     ExportFormat = "pdf"
)

// DataSource represents a data source for report generation.
type DataSource string

const (
	DataSourceNVD      DataSource = "nvd"
	DataSourceCNVD     DataSource = "cnvd"
	DataSourceCNNVD    DataSource = "cnnvd"
	DataSourceXianzhi  DataSource = "xianzhi"
	DataSourceQianxin  DataSource = "qianxin"
	DataSourceAnquanke DataSource = "anquanke"
)

// AIModelType represents the AI model to use for report summarization.
type AIModelType string

const (
	AIModelNone    AIModelType = ""
	AIModelGPT4    AIModelType = "gpt-4"
	AIModelClaude3 AIModelType = "claude-3"
	AIModelGemini  AIModelType = "gemini-pro"
)

// ReportConfig holds the configuration for report generation.
type ReportConfig struct {
	// Report metadata
	Title      string
	ReportType ReportType
	DateFrom   time.Time
	DateTo     time.Time
	
	// Data sources (which sources to include)
	Sources []DataSource
	
	// AI model (empty means no AI summarization)
	AIModel AIModelType
	
	// Export formats
	Formats []ExportFormat
	
	// User who created the report
	CreatedBy string
}

// ReportData holds all data needed to generate a report.
type ReportData struct {
	// Report metadata
	Title       string
	ReportType  ReportType
	DateFrom    time.Time
	DateTo      time.Time
	GeneratedAt time.Time
	
	// Config used
	Config *ReportConfig

	// Vulnerability statistics (per source)
	VulnStats map[DataSource]*repository.VulnStats

	// Top vulnerabilities (per source)
	TopVulns map[DataSource][]repository.VulnItem

	// Combined top vulnerabilities
	AllTopVulns []repository.VulnItem

	// Events/News (from articles)
	Events []repository.EventItem

	// AI summarization (if enabled)
	AISummary *AISummary

	// Sources included
	Sources []string

	// Data sources (as strings for display)
	DataSources []string
}

// AISummary holds AI-generated summary content.
type AISummary struct {
	Model      string
	Summary    string   // 本周概况摘要
	Highlights []string // 重点关注
	Advice     string   // 安全建议
	Trends     string   // 趋势分析
}

// AI summarizer interface
type AISummarizer interface {
	IsEnabled() bool
	GetModel() AIModelType
	Summarize(ctx context.Context, data *ReportData) (*AISummary, error)
}

// Generator generates security reports.
type Generator struct {
	vulnRepo    *repository.VulnRepo
	articleRepo *repository.ArticleRepository
	aiSummarizer AISummarizer
}

// NewGenerator creates a new report generator.
func NewGenerator(vulnRepo *repository.VulnRepo, articleRepo *repository.ArticleRepository) *Generator {
	return &Generator{
		vulnRepo:    vulnRepo,
		articleRepo: articleRepo,
		aiSummarizer: nil, // AI disabled by default
	}
}

// NewGeneratorWithAI creates a new report generator with AI enabled.
func NewGeneratorWithAI(vulnRepo *repository.VulnRepo, articleRepo *repository.ArticleRepository, apiKey string, groupID string, model AIModelType) *Generator {
	var summarizer AISummarizer
	if apiKey != "" && groupID != "" {
		summarizer = NewMinimaxService(apiKey, groupID, model)
	}
	return &Generator{
		vulnRepo:    vulnRepo,
		articleRepo: articleRepo,
		aiSummarizer: summarizer,
	}
}

// GenerateReport generates a report based on the configuration.
func (g *Generator) GenerateReport(ctx context.Context, config *ReportConfig) (*ReportData, error) {
	data := &ReportData{
		Title:       config.Title,
		ReportType:  config.ReportType,
		DateFrom:    config.DateFrom,
		DateTo:      config.DateTo,
		GeneratedAt: time.Now(),
		Config:      config,
		VulnStats:   make(map[DataSource]*repository.VulnStats),
		TopVulns:   make(map[DataSource][]repository.VulnItem),
		DataSources: g.formatSources(config.Sources),
	}

	// If no sources specified, use all
	sources := config.Sources
	if len(sources) == 0 {
		sources = []DataSource{DataSourceNVD, DataSourceCNVD}
	}

	// Fetch data from each source
	for _, source := range sources {
		stats, err := g.vulnRepo.GetStatsByDateRange(ctx, config.DateFrom, config.DateTo, string(source))
		if err != nil {
			continue // Skip on error, non-critical
		}
		data.VulnStats[source] = stats

		topVulns, err := g.vulnRepo.GetTopVulns(ctx, config.DateFrom, config.DateTo, string(source), 10)
		if err != nil {
			continue
		}
		data.TopVulns[source] = topVulns
		
		// Add to combined list
		data.AllTopVulns = append(data.AllTopVulns, topVulns...)
	}

	// Fetch security events (from articles)
	events, err := g.articleRepo.GetSecurityEvents(ctx, config.DateFrom, config.DateTo, 5)
	if err != nil {
		data.Events = []repository.EventItem{}
	} else {
		data.Events = events
	}

	// Set data sources as strings
	data.Sources = []string{"NVD (nvd.nist.gov)", "CNVD (cnvd.org.cn)", "CNNVD (cnnvd.org.cn)"}

	// AI summarization if configured
	if config.AIModel != AIModelNone && g.aiSummarizer != nil && g.aiSummarizer.IsEnabled() {
		aiSummary, err := g.aiSummarizer.Summarize(ctx, data)
		if err != nil {
			// Log error but don't fail the report
			data.AISummary = &AISummary{
				Model:   string(config.AIModel),
				Summary: "[AI 摘要生成失败，请稍后重试]",
			}
		} else {
			data.AISummary = aiSummary
		}
	}

	return data, nil
}

// formatSources converts DataSource slice to display strings.
func (g *Generator) formatSources(sources []DataSource) []string {
	result := make([]string, 0, len(sources))
	for _, s := range sources {
		switch s {
		case DataSourceNVD:
			result = append(result, "NVD")
		case DataSourceCNVD:
			result = append(result, "CNVD")
		case DataSourceCNNVD:
			result = append(result, "CNNVD")
		case DataSourceXianzhi:
			result = append(result, "先知社区")
		case DataSourceQianxin:
			result = append(result, "奇安信")
		case DataSourceAnquanke:
			result = append(result, "安全客")
		}
	}
	return result
}

// GenerateMarkdown generates a Markdown report.
func (g *Generator) GenerateMarkdown(data *ReportData) ([]byte, error) {
	var buf bytes.Buffer

	// Title
	buf.WriteString(fmt.Sprintf("# %s\n\n", data.Title))
	buf.WriteString(fmt.Sprintf("**报告周期**: %s - %s  \n", 
		data.DateFrom.Format("2006/01/02"), data.DateTo.Format("2006/01/02")))
	buf.WriteString(fmt.Sprintf("**数据来源**: %s  \n", strings.Join(data.DataSources, ", ")))
	if data.AISummary != nil {
		buf.WriteString(fmt.Sprintf("**AI 模型**: %s  \n", data.AISummary.Model))
	}
	buf.WriteString(fmt.Sprintf("**生成时间**: %s\n\n", data.GeneratedAt.Format("2006-01-02 15:04:05")))
	buf.WriteString("---\n\n")

	// AI Summary (if available)
	if data.AISummary != nil {
		buf.WriteString("## 🤖 AI 智能摘要\n\n")
		if data.AISummary.Summary != "" {
			buf.WriteString(fmt.Sprintf("%s\n\n", data.AISummary.Summary))
		}
		if len(data.AISummary.Highlights) > 0 {
			buf.WriteString("**重点关注**:\n")
			for _, h := range data.AISummary.Highlights {
				buf.WriteString(fmt.Sprintf("- %s\n", h))
			}
			buf.WriteString("\n")
		}
		if data.AISummary.Advice != "" {
			buf.WriteString(fmt.Sprintf("**安全建议**: %s\n\n", data.AISummary.Advice))
		}
		buf.WriteString("---\n\n")
	}

	// 0x00 Summary
	buf.WriteString("## 0x00 本周安全概况\n\n")
	
	// Aggregate stats
	totalVulns := 0
	for _, stats := range data.VulnStats {
		if stats != nil {
			totalVulns += stats.Total
		}
	}
	buf.WriteString(fmt.Sprintf("本周共收录安全漏洞 **%d** 个。\n\n", totalVulns))
	
	// Source breakdown
	if len(data.VulnStats) > 0 {
		buf.WriteString("**漏洞来源统计**:\n\n")
		for source, stats := range data.VulnStats {
			if stats != nil {
				sourceName := g.getSourceName(source)
				buf.WriteString(fmt.Sprintf("- %s: %d 个\n", sourceName, stats.Total))
				
				// Category breakdown
				if len(stats.ByCategory) > 0 {
					for category, count := range stats.ByCategory {
						buf.WriteString(fmt.Sprintf("  - %s: %d\n", category, count))
					}
				}
			}
		}
		buf.WriteString("\n")
	}

	// Top vulns table
	if len(data.AllTopVulns) > 0 {
		buf.WriteString("**本周关注漏洞 (TOP 10)**:\n\n")
		buf.WriteString("| 序号 | 漏洞名称 | CVE编号 | 厂商 | 来源 |\n")
		buf.WriteString("|------|----------|---------|------|------|\n")
		
		for i, v := range data.AllTopVulns {
			if i >= 10 {
				break
			}
			cve := v.CVE
			if cve == "" {
				cve = "-"
			}
			vendor := v.Vendor
			if vendor == "" {
				vendor = "-"
			}
			source := v.Source
			if source == "" {
				source = "-"
			}
			buf.WriteString(fmt.Sprintf("| %d | %s | %s | %s | %s |\n", i+1, v.Name, cve, vendor, source))
		}
		buf.WriteString("\n")
	}

	// 0x01 Detailed vulns
	if len(data.AllTopVulns) > 0 {
		buf.WriteString("## 0x01 重要漏洞列表\n\n")
		
		count := 0
		for _, v := range data.AllTopVulns {
			if count >= 10 {
				break
			}
			count++
			
			buf.WriteString(fmt.Sprintf("### %d. %s\n\n", count, v.Name))
			if v.CVE != "" {
				buf.WriteString(fmt.Sprintf("**CVE编号**: %s  \n", v.CVE))
			}
			if v.Severity != "" {
				buf.WriteString(fmt.Sprintf("**严重性**: %s  \n", v.Severity))
			}
			if v.Vendor != "" {
				buf.WriteString(fmt.Sprintf("**厂商**: %s  \n", v.Vendor))
			}
			if v.Source != "" {
				buf.WriteString(fmt.Sprintf("**来源**: %s  \n", v.Source))
			}
			if v.Description != "" {
				buf.WriteString(fmt.Sprintf("**描述**: %s  \n", v.Description))
			}
			if v.Solutions != "" {
				buf.WriteString(fmt.Sprintf("**修复建议**: %s  \n", v.Solutions))
			}
			buf.WriteString("\n---\n\n")
		}
	}

	// Events
	if len(data.Events) > 0 {
		buf.WriteString("## 本周安全事件\n\n")
		for i, e := range data.Events {
			buf.WriteString(fmt.Sprintf("%d. %s  \n", i+1, e.Title))
		}
		buf.WriteString("\n")
	}

	// Sources
	buf.WriteString("## 数据来源\n\n")
	for _, s := range data.Sources {
		buf.WriteString(fmt.Sprintf("- %s\n", s))
	}

	return buf.Bytes(), nil
}

func (g *Generator) getSourceName(source DataSource) string {
	switch source {
	case DataSourceNVD:
		return "NVD"
	case DataSourceCNVD:
		return "CNVD"
	case DataSourceCNNVD:
		return "CNNVD"
	case DataSourceXianzhi:
		return "先知社区"
	case DataSourceQianxin:
		return "奇安信"
	case DataSourceAnquanke:
		return "安全客"
	default:
		return string(source)
	}
}

// MarkdownToHTML converts Markdown content to HTML using goldmark.
func (g *Generator) MarkdownToHTML(mdContent []byte) ([]byte, error) {
	md := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			extension.Table,
			extension.Strikethrough,
			extension.Linkify,
			extension.Typographer,
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithRendererOptions(
			html.WithHardWraps(),
			html.WithXHTML(),
			html.WithUnsafe(),
		),
	)

	var buf bytes.Buffer
	if err := md.Convert(mdContent, &buf); err != nil {
		return nil, fmt.Errorf("failed to convert markdown to html: %w", err)
	}

	return buf.Bytes(), nil
}

// GenerateHTML generates an HTML report from Markdown using goldmark.
func (g *Generator) GenerateHTML(data *ReportData) ([]byte, error) {
	mdContent, err := g.GenerateMarkdown(data)
	if err != nil {
		return nil, err
	}

	htmlContent, err := g.MarkdownToHTML(mdContent)
	if err != nil {
		return nil, err
	}

	return g.wrapHTMLTemplate(data, htmlContent), nil
}

// wrapHTMLTemplate wraps content in a complete HTML page.
func (g *Generator) wrapHTMLTemplate(data *ReportData, content []byte) []byte {
	var buf bytes.Buffer

	buf.WriteString(`<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <meta name="description" content="`)
	buf.WriteString(fmt.Sprintf("SecFlow %s - %s 至 %s",
		g.getReportTypeName(data.ReportType),
		data.DateFrom.Format("2006/01/02"),
		data.DateTo.Format("2006/01/02")))
	buf.WriteString(`">
    <title>`)
	buf.WriteString(data.Title)
	buf.WriteString(`</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
            font-size: 15px;
            line-height: 1.8;
            color: #24292e;
            max-width: 900px;
            margin: 0 auto;
            padding: 40px 20px;
            background: #fff;
        }
        h1 {
            font-size: 28px;
            color: #1a1a2e;
            margin-bottom: 10px;
            padding-bottom: 15px;
            border-bottom: 3px solid #0066cc;
        }
        h2 {
            font-size: 20px;
            color: #1a1a2e;
            margin: 35px 0 15px;
            padding-left: 12px;
            border-left: 5px solid #0066cc;
        }
        h3 {
            font-size: 17px;
            color: #333;
            margin: 25px 0 10px;
        }
        .meta {
            color: #666;
            font-size: 14px;
            margin-bottom: 25px;
            padding: 15px;
            background: #f6f8fa;
            border-radius: 6px;
        }
        .meta span { margin-right: 25px; }
        .ai-summary {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            padding: 20px;
            border-radius: 8px;
            margin: 20px 0;
        }
        .ai-summary h2 { color: white; border-left-color: white; }
        .ai-summary .highlights {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
            gap: 10px;
            margin: 15px 0;
        }
        .ai-summary .highlight {
            background: rgba(255,255,255,0.2);
            padding: 10px 15px;
            border-radius: 6px;
        }
        hr {
            border: none;
            border-top: 1px solid #e1e4e8;
            margin: 30px 0;
        }
        table {
            width: 100%;
            border-collapse: collapse;
            margin: 20px 0;
            font-size: 14px;
        }
        th, td {
            border: 1px solid #dfe2e5;
            padding: 10px 14px;
            text-align: left;
        }
        th { background: #f6f8fa; font-weight: 600; }
        tr:nth-child(even) { background: #fafbfc; }
        code {
            background: #f1f1f1;
            padding: 2px 6px;
            border-radius: 3px;
            font-size: 0.9em;
        }
        pre {
            background: #f6f8fa;
            padding: 15px;
            border-radius: 6px;
            overflow-x: auto;
        }
        pre code { background: none; padding: 0; }
        blockquote {
            border-left: 4px solid #0066cc;
            padding-left: 15px;
            color: #666;
            margin: 15px 0;
        }
        ul, ol { padding-left: 25px; }
        li { margin: 5px 0; }
        .footer {
            text-align: center;
            margin-top: 50px;
            padding-top: 25px;
            border-top: 1px solid #e1e4e8;
            color: #999;
            font-size: 13px;
        }
        a { color: #0066cc; text-decoration: none; }
        a:hover { text-decoration: underline; }
    </style>
</head>
<body>
    <h1>`)
	buf.WriteString(data.Title)
	buf.WriteString(`</h1>
    <div class="meta">
        <span><strong>报告周期</strong>: `)
	buf.WriteString(fmt.Sprintf("%s - %s",
		data.DateFrom.Format("2006/01/02"),
		data.DateTo.Format("2006/01/02")))
	buf.WriteString(`</span>
        <span><strong>数据来源</strong>: `)
	buf.WriteString(strings.Join(data.DataSources, ", "))
	buf.WriteString(`</span>
        <span><strong>生成时间</strong>: `)
	buf.WriteString(data.GeneratedAt.Format("2006-01-02 15:04:05"))
	buf.WriteString(`</span>
    </div>
    <div class="content">
`)
	buf.Write(content)
	buf.WriteString(`
    </div>
    <div class="footer">
        <p>本报告由 <strong>SecFlow 安全情报平台</strong> 自动生成</p>
    </div>
</body>
</html>`)

	return buf.Bytes()
}

// ExportToFile exports the report to a file with the given format.
func (g *Generator) ExportToFile(data *ReportData, format ExportFormat) ([]byte, string, error) {
	var content []byte
	var ext string

	switch format {
	case FormatMarkdown:
		content, _ = g.GenerateMarkdown(data)
		ext = "md"
	case FormatHTML:
		content, _ = g.GenerateHTML(data)
		ext = "html"
	case FormatPDF:
		content, _ = g.GenerateHTML(data)
		ext = "html"
	default:
		return nil, "", fmt.Errorf("unsupported format: %s", format)
	}

	filename := fmt.Sprintf("secflow_%s_%s.%s",
		data.ReportType,
		data.DateFrom.Format("20060102"),
		ext)

	return content, filename, nil
}

func (g *Generator) getReportTypeName(t ReportType) string {
	switch t {
	case ReportTypeWeekly:
		return "周"
	case ReportTypeMonthly:
		return "月"
	case ReportTypeDaily:
		return "日"
	default:
		return "安全"
	}
}

// sanitizeString removes potentially problematic characters.
func sanitizeString(s string) string {
	return strings.TrimSpace(s)
}
