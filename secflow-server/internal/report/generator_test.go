package report

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGenerator_ExportFormats(t *testing.T) {
	// Test generator creation
	gen := &Generator{}
	_ = gen

	// Test constants
	assert.Equal(t, ReportType("weekly"), ReportTypeWeekly)
	assert.Equal(t, ReportType("monthly"), ReportTypeMonthly)
	assert.Equal(t, ReportType("daily"), ReportTypeDaily)

	assert.Equal(t, ExportFormat("md"), FormatMarkdown)
	assert.Equal(t, ExportFormat("html"), FormatHTML)
	assert.Equal(t, ExportFormat("pdf"), FormatPDF)
}

func TestReportData_Structure(t *testing.T) {
	data := &ReportData{
		Title:       "Test Report",
		ReportType:  ReportTypeWeekly,
		DateFrom:    time.Date(2024, 11, 11, 0, 0, 0, 0, time.UTC),
		DateTo:      time.Date(2024, 11, 15, 0, 0, 0, 0, time.UTC),
		GeneratedAt: time.Now(),
	}

	assert.Equal(t, "Test Report", data.Title)
	assert.Equal(t, ReportTypeWeekly, data.ReportType)
	assert.Equal(t, 2024, data.DateFrom.Year())
	assert.Equal(t, 11, int(data.DateFrom.Month()))
}

func TestMarkdownToHTML(t *testing.T) {
	gen := &Generator{}

	// Test basic Markdown to HTML conversion
	md := []byte("# Hello\n\nThis is **bold** and *italic*.\n\n- Item 1\n- Item 2\n\n| Col1 | Col2 |\n|------|------|\n| A    | B    |")
	
	html, err := gen.MarkdownToHTML(md)
	assert.NoError(t, err)
	assert.NotEmpty(t, html)

	// Check for HTML tags (goldmark adds id to headings)
	assert.Contains(t, string(html), "<h1")
	assert.Contains(t, string(html), "Hello</h1>")
	assert.Contains(t, string(html), "<strong>bold</strong>")
	assert.Contains(t, string(html), "<em>italic</em>")
	assert.Contains(t, string(html), "<ul>")
	assert.Contains(t, string(html), "<table>")
}

func TestGenerateHTML(t *testing.T) {
	gen := &Generator{}

	data := &ReportData{
		Title:       "周安全报告",
		ReportType:  ReportTypeWeekly,
		DateFrom:    time.Date(2024, 11, 11, 0, 0, 0, 0, time.UTC),
		DateTo:      time.Date(2024, 11, 15, 0, 0, 0, 0, time.UTC),
		GeneratedAt: time.Now(),
		Sources:     []string{"NVD"},
	}

	html, err := gen.GenerateHTML(data)
	assert.NoError(t, err)
	assert.NotEmpty(t, html)

	content := string(html)
	// Check HTML structure
	assert.True(t, strings.HasPrefix(content, "<!DOCTYPE html>"))
	assert.Contains(t, content, "<html lang=\"zh-CN\">")
	assert.Contains(t, content, "<title>周安全报告</title>")
	assert.Contains(t, content, "<meta charset=\"UTF-8\">")
	assert.Contains(t, content, "SecFlow 安全情报平台")
}

func TestExportToFile(t *testing.T) {
	gen := &Generator{}

	data := &ReportData{
		Title:       "周安全报告",
		ReportType:  ReportTypeWeekly,
		DateFrom:    time.Date(2024, 11, 11, 0, 0, 0, 0, time.UTC),
		DateTo:      time.Date(2024, 11, 15, 0, 0, 0, 0, time.UTC),
		GeneratedAt: time.Now(),
		Sources:     []string{"NVD"},
	}

	// Test Markdown export
	md, filename, err := gen.ExportToFile(data, FormatMarkdown)
	assert.NoError(t, err)
	assert.NotEmpty(t, md)
	assert.Equal(t, "secflow_weekly_20241111.md", filename)

	// Test HTML export
	html, filename, err := gen.ExportToFile(data, FormatHTML)
	assert.NoError(t, err)
	assert.NotEmpty(t, html)
	assert.Equal(t, "secflow_weekly_20241111.html", filename)
	assert.True(t, strings.HasPrefix(string(html), "<!DOCTYPE html>"))
}
