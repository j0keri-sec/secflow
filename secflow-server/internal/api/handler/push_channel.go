package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/secflow/server/internal/model"
	"github.com/secflow/server/internal/repository"
)

// PushChannelHandler manages notification channel CRUD.
type PushChannelHandler struct {
	repo *repository.PushChannelRepository
}

// NewPushChannelHandler creates a new PushChannelHandler.
func NewPushChannelHandler(repo *repository.PushChannelRepository) *PushChannelHandler {
	return &PushChannelHandler{repo: repo}
}

// List godoc  GET /api/v1/push-channels
func (h *PushChannelHandler) List(c *gin.Context) {
	items, err := h.repo.List(c.Request.Context())
	if err != nil {
		Err(c, http.StatusInternalServerError, "database error")
		return
	}
	OK(c, items)
}

// Create godoc  POST /api/v1/push-channels
func (h *PushChannelHandler) Create(c *gin.Context) {
	var ch model.PushChannel
	if err := c.ShouldBindJSON(&ch); err != nil {
		Err(c, http.StatusBadRequest, err.Error())
		return
	}
	ch.CreatedAt = time.Now()
	ch.UpdatedAt = time.Now()
	if err := h.repo.Create(c.Request.Context(), &ch); err != nil {
		log.Error().Err(err).Msg("create push channel")
		Err(c, http.StatusInternalServerError, "create failed")
		return
	}
	OK(c, ch)
}

// Update godoc  PATCH /api/v1/push-channels/:id
func (h *PushChannelHandler) Update(c *gin.Context) {
	oid, err := bson.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		Err(c, http.StatusBadRequest, "invalid id")
		return
	}
	var patch map[string]any
	if err := c.ShouldBindJSON(&patch); err != nil {
		Err(c, http.StatusBadRequest, err.Error())
		return
	}
	patch["updated_at"] = time.Now()
	if err := h.repo.Update(c.Request.Context(), oid, patch); err != nil {
		Err(c, http.StatusInternalServerError, "update failed")
		return
	}
	OK(c, nil)
}

// Delete godoc  DELETE /api/v1/push-channels/:id
func (h *PushChannelHandler) Delete(c *gin.Context) {
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
