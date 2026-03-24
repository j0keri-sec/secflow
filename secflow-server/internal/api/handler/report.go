package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/secflow/server/internal/model"
	"github.com/secflow/server/internal/repository"
	"github.com/secflow/server/pkg/auth"
)

// ReportHandler handles report generation and management.
type ReportHandler struct {
	repo *repository.ReportRepository
}

// NewReportHandler creates a new ReportHandler.
func NewReportHandler(repo *repository.ReportRepository) *ReportHandler {
	return &ReportHandler{repo: repo}
}

// List godoc  GET /api/v1/reports
func (h *ReportHandler) List(c *gin.Context) {
	page, pageSize := paginate(c)
	items, total, err := h.repo.List(c.Request.Context(), bson.D{}, page, pageSize)
	if err != nil {
		log.Error().Err(err).Msg("list reports")
		Err(c, http.StatusInternalServerError, "database error")
		return
	}
	OK(c, PageResult(items, total, page, pageSize))
}

// CreateReportRequest is the request body for POST /api/v1/reports.
type CreateReportRequest struct {
	Title       string `json:"title"       binding:"required"`
	Description string `json:"description"`
	Type        string `json:"type"        binding:"required"` // daily|weekly|monthly|custom
	DateFrom    string `json:"date_from"`
	DateTo      string `json:"date_to"`
}

// Create godoc  POST /api/v1/reports
func (h *ReportHandler) Create(c *gin.Context) {
	var req CreateReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Err(c, http.StatusBadRequest, err.Error())
		return
	}

	claims := auth.ClaimsFromCtx(c)
	ownerID, _ := bson.ObjectIDFromHex(claims.UserID)

	period := req.DateFrom + " ~ " + req.DateTo
	if req.DateFrom == "" && req.DateTo == "" {
		period = time.Now().Format("2006-01-02")
	}

	report := &model.Report{
		Title:       req.Title,
		Description: req.Description,
		Status:      model.ReportPending,
		Period:      period,
		CreatedBy:   ownerID,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := h.repo.Create(c.Request.Context(), report); err != nil {
		log.Error().Err(err).Msg("create report")
		Err(c, http.StatusInternalServerError, "create failed")
		return
	}

	// In production, enqueue an async report generation job here.
	OK(c, report)
}

// Delete godoc  DELETE /api/v1/reports/:id
func (h *ReportHandler) Delete(c *gin.Context) {
	oid, err := bson.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		Err(c, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.repo.Delete(c.Request.Context(), oid); err != nil {
		Err(c, http.StatusInternalServerError, "delete failed")
		return
	}
	OK(c, nil)
}
