package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/secflow/server/internal/repository"
)

// AuditLogHandler handles audit log queries.
type AuditLogHandler struct {
	repo *repository.AuditLogRepository
}

// NewAuditLogHandler creates a new AuditLogHandler.
func NewAuditLogHandler(repo *repository.AuditLogRepository) *AuditLogHandler {
	return &AuditLogHandler{repo: repo}
}

// List godoc  GET /api/v1/audit-logs
//
//	Query: page, page_size, keyword (username/ip/resource), action
func (h *AuditLogHandler) List(c *gin.Context) {
	page, pageSize := paginate(c)

	filter := bson.D{}
	if kw := c.Query("keyword"); kw != "" {
		filter = append(filter, bson.E{Key: "$or", Value: bson.A{
			bson.D{{Key: "username", Value: bson.D{{Key: "$regex", Value: kw}, {Key: "$options", Value: "i"}}}},
			bson.D{{Key: "ip", Value: bson.D{{Key: "$regex", Value: kw}}}},
			bson.D{{Key: "resource", Value: bson.D{{Key: "$regex", Value: kw}, {Key: "$options", Value: "i"}}}},
		}})
	}
	if action := c.Query("action"); action != "" {
		filter = append(filter, bson.E{Key: "action", Value: action})
	}

	items, total, err := h.repo.List(c.Request.Context(), filter, page, pageSize)
	if err != nil {
		log.Error().Err(err).Msg("list audit logs")
		Err(c, http.StatusInternalServerError, "database error")
		return
	}
	OK(c, PageResult(items, total, page, pageSize))
}
