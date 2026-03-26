package handler

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/secflow/server/internal/model"
	"github.com/secflow/server/internal/report"
	"github.com/secflow/server/internal/repository"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// ReportHandler handles report-related API endpoints.
type ReportHandler struct {
	gen        *report.Generator
	vulnRepo   *repository.VulnRepo
	reportRepo  *repository.ReportRepository
}

// NewReportHandler creates a new report handler.
func NewReportHandler(gen *report.Generator, vulnRepo *repository.VulnRepo, reportRepo *repository.ReportRepository) *ReportHandler {
	return &ReportHandler{
		gen:       gen,
		vulnRepo:  vulnRepo,
		reportRepo: reportRepo,
	}
}

// List handles GET /api/v1/reports
func (h *ReportHandler) List(c *gin.Context) {
	page := 1
	pageSize := 20
	if p := c.Query("page"); p != "" {
		fmt.Sscanf(p, "%d", &page)
	}
	if ps := c.Query("page_size"); ps != "" {
		fmt.Sscanf(ps, "%d", &pageSize)
	}

	reports, total, err := h.reportRepo.List(c.Request.Context(), bson.D{}, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"reports":   reports,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// Create handles POST /api/v1/reports
func (h *ReportHandler) Create(c *gin.Context) {
	var req struct {
		Title    string   `json:"title" binding:"required"`
		Period   string   `json:"period"`
		Content  string   `json:"content"`
		FilePath string   `json:"file_path"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	rep := &model.Report{
		Title:    req.Title,
		Period:   req.Period,
		Content:  req.Content,
		FilePath: req.FilePath,
		Status:   model.ReportDone,
	}

	if err := h.reportRepo.Create(c.Request.Context(), rep); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, rep)
}

// Delete handles DELETE /api/v1/reports/:id
func (h *ReportHandler) Delete(c *gin.Context) {
	idStr := c.Param("id")
	id, err := bson.ObjectIDFromHex(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := h.reportRepo.Delete(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

// GenerateReportRequest represents the request to generate a report.
type GenerateReportRequest struct {
	Title     string   `json:"title" binding:"required"`
	Type      string   `json:"type" binding:"required,oneof=weekly monthly daily"`
	Sources   []string `json:"sources"` // e.g., ["nvd", "cnvd", "xianzhi"]
	DateFrom  string   `json:"date_from"` // YYYY-MM-DD
	DateTo    string   `json:"date_to"`   // YYYY-MM-DD
	AIModel   string   `json:"ai_model"`  // gpt-4, claude-3, gemini-pro, or empty
	Formats   []string `json:"formats"`   // ["markdown", "html"]
}

// ExportReport handles GET /api/v1/reports/export
func (h *ReportHandler) ExportReport(c *gin.Context) {
	var req GenerateReportRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Parse dates
	dateTo := time.Now()
	dateFrom := dateTo.AddDate(0, 0, -7)

	if req.DateFrom != "" {
		parsed, err := time.Parse("2006-01-02", req.DateFrom)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid date_from format, use YYYY-MM-DD"})
			return
		}
		dateFrom = parsed
	}

	if req.DateTo != "" {
		parsed, err := time.Parse("2006-01-02", req.DateTo)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid date_to format, use YYYY-MM-DD"})
			return
		}
		dateTo = parsed
	}

	// Convert sources
	var sources []report.DataSource
	for _, s := range req.Sources {
		sources = append(sources, report.DataSource(s))
	}

	// Convert type
	var reportType report.ReportType
	switch req.Type {
	case "weekly":
		reportType = report.ReportTypeWeekly
	case "monthly":
		reportType = report.ReportTypeMonthly
	case "daily":
		reportType = report.ReportTypeDaily
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid report type"})
		return
	}

	// Build config
	config := &report.ReportConfig{
		Title:      req.Title,
		ReportType: reportType,
		DateFrom:   dateFrom,
		DateTo:     dateTo,
		Sources:    sources,
		AIModel:   report.AIModelType(req.AIModel),
	}

	// Generate report data
	data, err := h.gen.GenerateReport(c.Request.Context(), config)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate report"})
		return
	}

	// Determine format
	formatStr := c.DefaultQuery("format", "html")
	format := report.ExportFormat(formatStr)
	if format != report.FormatMarkdown && format != report.FormatHTML {
		format = report.FormatHTML
	}

	// Export
	content, filename, err := h.gen.ExportToFile(data, format)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to export report"})
		return
	}

	// Set content type
	var contentType string
	switch format {
	case report.FormatMarkdown:
		contentType = "text/markdown; charset=utf-8"
	case report.FormatHTML:
		contentType = "text/html; charset=utf-8"
	default:
		contentType = "application/octet-stream"
	}

	c.Header("Content-Type", contentType)
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Data(http.StatusOK, contentType, content)
}

// PreviewReport handles GET /api/v1/reports/preview
func (h *ReportHandler) PreviewReport(c *gin.Context) {
	reqType := c.DefaultQuery("type", "weekly")
	dateFromStr := c.Query("date_from")
	dateToStr := c.Query("date_to")
	sourcesStr := c.QueryArray("sources")

	// Parse dates
	dateTo := time.Now()
	dateFrom := dateTo.AddDate(0, 0, -7)

	if dateFromStr != "" {
		parsed, err := time.Parse("2006-01-02", dateFromStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid date_from format"})
			return
		}
		dateFrom = parsed
	}

	if dateToStr != "" {
		parsed, err := time.Parse("2006-01-02", dateToStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid date_to format"})
			return
		}
		dateTo = parsed
	}

	// Convert type
	var reportType report.ReportType
	switch reqType {
	case "weekly":
		reportType = report.ReportTypeWeekly
	case "monthly":
		reportType = report.ReportTypeMonthly
	case "daily":
		reportType = report.ReportTypeDaily
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid report type"})
		return
	}

	// Convert sources
	var sources []report.DataSource
	for _, s := range sourcesStr {
		sources = append(sources, report.DataSource(s))
	}

	// Build config
	config := &report.ReportConfig{
		Title:      "安全报告预览",
		ReportType: reportType,
		DateFrom:   dateFrom,
		DateTo:     dateTo,
		Sources:    sources,
	}

	// Generate report
	data, err := h.gen.GenerateReport(c.Request.Context(), config)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate report"})
		return
	}

	content, _, err := h.gen.ExportToFile(data, report.FormatHTML)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to export report"})
		return
	}

	c.Data(http.StatusOK, "text/html; charset=utf-8", content)
}

// GetReportStats handles GET /api/v1/reports/stats
func (h *ReportHandler) GetReportStats(c *gin.Context) {
	dateFromStr := c.Query("date_from")
	dateToStr := c.Query("date_to")
	source := c.Query("source") // optional filter

	dateTo := time.Now()
	dateFrom := dateTo.AddDate(0, 0, -7)

	if dateFromStr != "" {
		parsed, err := time.Parse("2006-01-02", dateFromStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid date_from format"})
			return
		}
		dateFrom = parsed
	}

	if dateToStr != "" {
		parsed, err := time.Parse("2006-01-02", dateToStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid date_to format"})
			return
		}
		dateTo = parsed
	}

	stats, err := h.vulnRepo.GetStatsByDateRange(c.Request.Context(), dateFrom, dateTo, source)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get report statistics"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"date_from": dateFrom.Format("2006-01-02"),
		"date_to":   dateTo.Format("2006-01-02"),
		"source":    source,
		"stats":     stats,
	})
}

// GetDataSources handles GET /api/v1/reports/datasources
func (h *ReportHandler) GetDataSources(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"sources": []gin.H{
			{"id": "nvd", "name": "NVD", "description": "美国国家漏洞数据库"},
			{"id": "cnvd", "name": "CNVD", "description": "中国国家漏洞库"},
			{"id": "cnnvd", "name": "CNNVD", "description": "中国国家信息安全漏洞库"},
			{"id": "xianzhi", "name": "先知社区", "description": "阿里云先知社区"},
			{"id": "qianxin", "name": "奇安信", "description": "奇安信威胁情报"},
			{"id": "anquanke", "name": "安全客", "description": "安全客资讯"},
		},
	})
}

// GetAIModels handles GET /api/v1/reports/aimodels
func (h *ReportHandler) GetAIModels(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"models": []gin.H{
			{"id": "", "name": "不使用 AI", "description": "纯数据报告"},
			{"id": "gpt-4", "name": "GPT-4", "description": "OpenAI 最强模型"},
			{"id": "claude-3", "name": "Claude 3", "description": "Anthropic 大模型"},
			{"id": "gemini-pro", "name": "Gemini Pro", "description": "Google AI 模型"},
		},
	})
}
