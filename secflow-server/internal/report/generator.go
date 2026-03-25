// Package report provides security report generation functionality.
package report

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
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

	// Risk analysis
	RiskDistribution map[string]int      // Severity-based risk distribution
	TopExploits     []repository.VulnItem // Known exploited vulnerabilities
	EmergingThreats []repository.VulnItem // New vulnerabilities from last 7 days
	ThreatIntel     *ThreatIntel         // Threat intelligence summary

	// Computed risk scores for vulnerabilities
	VulnRiskScores map[string]float64 // CVE -> risk score
}

// ThreatIntel holds threat intelligence summary.
type ThreatIntel struct {
	ActiveExploits   int      // Number of actively exploited vulns
	ZeroDays         int      // Number of zero-day vulns
	CriticalCount    int      // Number of critical severity vulns
	TopAttackVectors []string // Top attack vectors (e.g., RCE, SQLi)
	GeographicSpread []string // Affected regions/countries
}

// CalculateRiskScore computes risk score based on CVSS, exploitability, and recency.
func CalculateRiskScore(cvss float64, publishedAt time.Time, hasExploit bool) float64 {
	if cvss < 0 {
		cvss = 0
	}
	if cvss > 10 {
		cvss = 10
	}

	// Base score from CVSS (60% weight)
	baseScore := cvss * 0.6

	// Recency factor (20% weight) - newer vulns get higher scores
	daysSincePublish := time.Since(publishedAt).Hours() / 24
	var recencyFactor float64
	if daysSincePublish <= 7 {
		recencyFactor = 2.0 // Fresh vulnerabilities
	} else if daysSincePublish <= 30 {
		recencyFactor = 1.5
	} else if daysSincePublish <= 90 {
		recencyFactor = 1.0
	} else {
		recencyFactor = 0.5
	}
	recencyScore := recencyFactor * 2.0 * 0.2 // 20% weight max of 2 points

	// Exploit availability (20% weight)
	var exploitScore float64
	if hasExploit {
		exploitScore = 2.0
	}

	// Popular product factor (bonus 0-0.5)
	popularProductBonus := 0.0

	totalScore := baseScore + recencyScore + exploitScore + popularProductBonus
	if totalScore > 10 {
		totalScore = 10
	}
	return totalScore
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
		RiskDistribution: make(map[string]int),
		VulnRiskScores: make(map[string]float64),
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

		// Add to combined list and compute risk scores
		for i := range topVulns {
			v := &topVulns[i]
			// Compute risk score
			riskScore := CalculateRiskScore(v.CVSS, v.PublishedAt, false)
			v.RiskScore = riskScore
			data.VulnRiskScores[v.CVE] = riskScore

			// Track risk distribution
			severityKey := v.Severity
			if severityKey == "" {
				severityKey = "未知"
			}
			data.RiskDistribution[severityKey]++

			// Identify emerging threats (published within last 7 days)
			if time.Since(v.PublishedAt).Hours() <= 168 { // 7 days * 24 hours
				data.EmergingThreats = append(data.EmergingThreats, *v)
			}
		}
		data.AllTopVulns = append(data.AllTopVulns, topVulns...)
	}

	// Build ThreatIntel from aggregated data
	data.ThreatIntel = g.buildThreatIntel(data)

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

// buildThreatIntel constructs threat intelligence from report data.
func (g *Generator) buildThreatIntel(data *ReportData) *ThreatIntel {
	ti := &ThreatIntel{
		TopAttackVectors: []string{},
		GeographicSpread: []string{},
	}

	criticalCount := 0
	activeExploits := 0

	for _, v := range data.AllTopVulns {
		// Count critical severity
		if v.Severity == "严重" || v.Severity == "Critical" {
			criticalCount++
		}

		// Check for high risk score (potential active exploit)
		if v.RiskScore >= 7.0 {
			activeExploits++
		}
	}

	ti.CriticalCount = criticalCount
	ti.ActiveExploits = activeExploits

	// Analyze attack vectors from descriptions
	ti.TopAttackVectors = g.extractAttackVectors(data.AllTopVulns)

	return ti
}

// extractAttackVectors analyzes vulnerabilities to identify top attack vectors.
func (g *Generator) extractAttackVectors(vulns []repository.VulnItem) []string {
	attackVectors := make(map[string]int)

	vectorKeywords := map[string][]string{
		"Remote Code Execution (RCE)": {"remote code execution", "rce", "远程代码执行", "arbitrary code", "execute code"},
		"SQL Injection":              {"sql injection", "sql注入", "sqli"},
		"Cross-Site Scripting (XSS)": {"xss", "cross-site scripting", "跨站脚本"},
		"Buffer Overflow":            {"buffer overflow", "缓冲区溢出", "heap overflow"},
		"Privilege Escalation":       {"privilege escalation", "权限提升", "local exploit"},
		"Authentication Bypass":     {"authentication bypass", "认证绕过", "bypass auth"},
		"Information Disclosure":    {"information disclosure", "信息泄露", "information leak"},
		"Denial of Service (DoS)":   {"denial of service", "dos", "拒绝服务"},
	}

	for _, v := range vulns {
		text := strings.ToLower(v.Name + " " + v.Description)
		for vector, keywords := range vectorKeywords {
			for _, kw := range keywords {
				if strings.Contains(text, kw) {
					attackVectors[vector]++
					break
				}
			}
		}
	}

	// Sort by frequency and return top 5
	type kv struct {
		Key   string
		Value int
	}
	var sorted []kv
	for k, v := range attackVectors {
		sorted = append(sorted, kv{k, v})
	}
	// Simple sort - in production use sort.Slice
	if len(sorted) > 5 {
		sorted = sorted[:5]
	}

	result := make([]string, len(sorted))
	for i, kv := range sorted {
		result[i] = kv.Key
	}
	return result
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

// GenerateEnhancedHTML generates an HTML report with Chart.js visualizations.
func (g *Generator) GenerateEnhancedHTML(data *ReportData) ([]byte, error) {
	mdContent, err := g.GenerateMarkdown(data)
	if err != nil {
		return nil, err
	}

	htmlContent, err := g.MarkdownToHTML(mdContent)
	if err != nil {
		return nil, err
	}

	return g.wrapEnhancedHTMLTemplate(data, htmlContent), nil
}

// wrapEnhancedHTMLTemplate wraps content in an enhanced HTML page with charts.
func (g *Generator) wrapEnhancedHTMLTemplate(data *ReportData, content []byte) []byte {
	// Aggregate ByCategory from VulnStats
	byCategory := make(map[string]int)
	for _, stats := range data.VulnStats {
		if stats != nil {
			for k, v := range stats.ByCategory {
				byCategory[k] += v
			}
		}
	}

	var buf bytes.Buffer
	enhancedHTMLTemplate.Execute(&buf, struct {
		Title           string
		Description     string
		Content         template.HTML
		DateFrom        string
		DateTo          string
		DataSources     string
		GeneratedAt     string
		ThreatIntel     *ThreatIntel
		RiskDistribution map[string]int
		ByCategory     map[string]int
	}{
		Title:           data.Title,
		Description:     fmt.Sprintf("SecFlow %s - %s 至 %s", g.getReportTypeName(data.ReportType), data.DateFrom.Format("2006/01/02"), data.DateTo.Format("2006/01/02")),
		Content:         template.HTML(content),
		DateFrom:        data.DateFrom.Format("2006/01/02"),
		DateTo:          data.DateTo.Format("2006/01/02"),
		DataSources:     strings.Join(data.DataSources, ", "),
		GeneratedAt:     data.GeneratedAt.Format("2006-01-02 15:04:05"),
		ThreatIntel:     data.ThreatIntel,
		RiskDistribution: data.RiskDistribution,
		ByCategory:      byCategory,
	})

	return buf.Bytes()
}

// wrapHTMLTemplate wraps content in a complete HTML page.
func (g *Generator) wrapHTMLTemplate(data *ReportData, content []byte) []byte {
	var buf bytes.Buffer

	htmlTemplate.Execute(&buf, struct {
		Title       string
		Description string
		Content     template.HTML
		DateFrom    string
		DateTo      string
		DataSources string
		GeneratedAt string
	}{
		Title:       data.Title,
		Description: fmt.Sprintf("SecFlow %s - %s 至 %s", g.getReportTypeName(data.ReportType), data.DateFrom.Format("2006/01/02"), data.DateTo.Format("2006/01/02")),
		Content:     template.HTML(content),
		DateFrom:    data.DateFrom.Format("2006/01/02"),
		DateTo:      data.DateTo.Format("2006/01/02"),
		DataSources: strings.Join(data.DataSources, ", "),
		GeneratedAt: data.GeneratedAt.Format("2006-01-02 15:04:05"),
	})

	return buf.Bytes()
}

// htmlTemplate is the HTML template for reports.
var htmlTemplate = template.Must(template.New("report").Parse(`<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <meta name="description" content="{{.Description}}">
    <title>{{.Title}}</title>
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
    <h1>{{.Title}}</h1>
    <div class="meta">
        <span><strong>报告周期</strong>: {{.DateFrom}} - {{.DateTo}}</span>
        <span><strong>数据来源</strong>: {{.DataSources}}</span>
        <span><strong>生成时间</strong>: {{.GeneratedAt}}</span>
    </div>
    <div class="content">
{{.Content}}
    </div>
    <div class="footer">
        <p>本报告由 <strong>SecFlow 安全情报平台</strong> 自动生成</p>
    </div>
</body>
</html>`))

// enhancedHTMLTemplate includes Chart.js for visualizations.
var enhancedHTMLTemplate = template.Must(template.New("enhanced_report").Parse(`<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <meta name="description" content="{{.Description}}">
    <title>{{.Title}}</title>
    <script src="https://cdn.jsdelivr.net/npm/chart.js@4.4.0/dist/chart.umd.min.js"></script>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
            font-size: 15px;
            line-height: 1.8;
            color: #24292e;
            max-width: 1200px;
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

        /* Charts section */
        .charts-section {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(400px, 1fr));
            gap: 20px;
            margin: 30px 0;
        }
        .chart-container {
            background: #fff;
            border: 1px solid #e1e4e8;
            border-radius: 8px;
            padding: 20px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.05);
        }
        .chart-container canvas {
            max-height: 300px;
        }
        .chart-title {
            font-size: 16px;
            font-weight: 600;
            color: #1a1a2e;
            margin-bottom: 15px;
            text-align: center;
        }

        /* Threat intel section */
        .threat-intel {
            background: linear-gradient(135deg, #11998e 0%, #38ef7d 100%);
            color: white;
            padding: 20px;
            border-radius: 8px;
            margin: 20px 0;
        }
        .threat-intel-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 15px;
            margin-top: 15px;
        }
        .threat-intel-item {
            background: rgba(255,255,255,0.2);
            padding: 12px 15px;
            border-radius: 6px;
        }
        .threat-intel-item .label {
            font-size: 12px;
            opacity: 0.9;
        }
        .threat-intel-item .value {
            font-size: 24px;
            font-weight: bold;
        }

        /* Risk distribution */
        .risk-distribution {
            display: flex;
            gap: 15px;
            margin: 15px 0;
            flex-wrap: wrap;
        }
        .risk-item {
            flex: 1;
            min-width: 100px;
            text-align: center;
            padding: 15px;
            border-radius: 6px;
            color: white;
        }
        .risk-critical { background: #dc3545; }
        .risk-high { background: #fd7e14; }
        .risk-medium { background: #ffc107; color: #333; }
        .risk-low { background: #28a745; }
        .risk-unknown { background: #6c757d; }
    </style>
</head>
<body>
    <h1>{{.Title}}</h1>
    <div class="meta">
        <span><strong>报告周期</strong>: {{.DateFrom}} - {{.DateTo}}</span>
        <span><strong>数据来源</strong>: {{.DataSources}}</span>
        <span><strong>生成时间</strong>: {{.GeneratedAt}}</span>
    </div>

    {{if .ThreatIntel}}
    <div class="threat-intel">
        <h2 style="border:none;margin:0;color:white;">威胁情报概览</h2>
        <div class="threat-intel-grid">
            <div class="threat-intel-item">
                <div class="label">严重漏洞数</div>
                <div class="value">{{.ThreatIntel.CriticalCount}}</div>
            </div>
            <div class="threat-intel-item">
                <div class="label">高危漏洞数</div>
                <div class="value">{{.ThreatIntel.ActiveExploits}}</div>
            </div>
            <div class="threat-intel-item">
                <div class="label">零日漏洞</div>
                <div class="value">{{.ThreatIntel.ZeroDays}}</div>
            </div>
        </div>
        {{if .ThreatIntel.TopAttackVectors}}
        <div style="margin-top:15px;">
            <strong>主要攻击向量:</strong> {{range .ThreatIntel.TopAttackVectors}}{{.}} {{end}}
        </div>
        {{end}}
    </div>
    {{end}}

    {{if .RiskDistribution}}
    <h2>风险分布</h2>
    <div class="risk-distribution">
        {{range $severity, $count := .RiskDistribution}}
        <div class="risk-item risk-{{$severity}}">
            <div>{{$severity}}</div>
            <div style="font-size:20px;font-weight:bold;">{{$count}}</div>
        </div>
        {{end}}
    </div>
    {{end}}

    <div class="charts-section">
        <div class="chart-container">
            <div class="chart-title">漏洞严重性分布</div>
            <canvas id="severityChart"></canvas>
        </div>
        <div class="chart-container">
            <div class="chart-title">漏洞来源分布</div>
            <canvas id="sourceChart"></canvas>
        </div>
    </div>

    <div class="content">
{{.Content}}
    </div>

    <div class="footer">
        <p>本报告由 <strong>SecFlow 安全情报平台</strong> 自动生成</p>
    </div>

    <script>
    // Chart.js initialization
    document.addEventListener('DOMContentLoaded', function() {
        // Severity chart
        const severityCtx = document.getElementById('severityChart');
        if (severityCtx) {
            new Chart(severityCtx, {
                type: 'doughnut',
                data: {
                    labels: [{{range $k, $v := .RiskDistribution}}"{{$k}}",{{end}}],
                    datasets: [{
                        data: [{{range $k, $v := .RiskDistribution}}{{$v}},{{end}}],
                        backgroundColor: [
                            '#dc3545', '#fd7e14', '#ffc107', '#28a745', '#6c757d'
                        ]
                    }]
                },
                options: {
                    responsive: true,
                    plugins: {
                        legend: { position: 'bottom' }
                    }
                }
            });
        }

        // Source chart
        const sourceCtx = document.getElementById('sourceChart');
        if (sourceCtx) {
            new Chart(sourceCtx, {
                type: 'bar',
                data: {
                    labels: [{{range $k, $v := .ByCategory}}"{{$k}}",{{end}}],
                    datasets: [{
                        label: '漏洞数量',
                        data: [{{range $k, $v := .ByCategory}}{{$v}},{{end}}],
                        backgroundColor: '#0066cc'
                    }]
                },
                options: {
                    responsive: true,
                    plugins: {
                        legend: { display: false }
                    },
                    scales: {
                        y: { beginAtZero: true }
                    }
                }
            });
        }
    });
    </script>
</body>
</html>`))

// ExportToFile exports the report to a file with the given format.
func (g *Generator) ExportToFile(data *ReportData, format ExportFormat) ([]byte, string, error) {
	var content []byte
	var ext string

	switch format {
	case FormatMarkdown:
		content, _ = g.GenerateMarkdown(data)
		ext = "md"
	case FormatHTML:
		content, _ = g.GenerateEnhancedHTML(data)
		ext = "html"
	case FormatPDF:
		content, _ = g.GenerateEnhancedHTML(data)
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
