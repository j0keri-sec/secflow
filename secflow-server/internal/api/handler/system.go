package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/secflow/server/internal/model"
	"github.com/secflow/server/internal/repository"
)

// SystemHandler manages system settings including task schedules.
type SystemHandler struct {
	scheduleRepo *repository.TaskScheduleRepo
}

// NewSystemHandler creates a new SystemHandler.
func NewSystemHandler(scheduleRepo *repository.TaskScheduleRepo) *SystemHandler {
	return &SystemHandler{scheduleRepo: scheduleRepo}
}

// TaskScheduleResponse is the API response for task schedule settings.
type TaskScheduleResponse struct {
	Type     string   `json:"type"`
	Enabled  bool     `json:"enabled"`
	Interval int      `json:"interval"` // in minutes
	Sources  []string `json:"sources"`
}

// GetTaskSchedules returns all task schedule configurations.
//
// GET /api/v1/task-schedules
func (h *SystemHandler) GetTaskSchedules(c *gin.Context) {
	schedules, err := h.scheduleRepo.List(c)
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	// Build response with defaults for types not yet configured
	response := make([]TaskScheduleResponse, 0, 2)
	typeConfigs := map[string]bool{}
	for _, s := range schedules {
		typeConfigs[string(s.Type)] = true
		response = append(response, TaskScheduleResponse{
			Type:     string(s.Type),
			Enabled:  s.Enabled,
			Interval: s.Interval,
			Sources:  s.Sources,
		})
	}

	// Add default configs for types not yet in database
	if !typeConfigs[string(model.TaskTypeVulnCrawl)] {
		response = append(response, TaskScheduleResponse{
			Type:     string(model.TaskTypeVulnCrawl),
			Enabled:  false,
			Interval: 30,
			Sources:  []string{},
		})
	}
	if !typeConfigs[string(model.TaskTypeArticleCrawl)] {
		response = append(response, TaskScheduleResponse{
			Type:     string(model.TaskTypeArticleCrawl),
			Enabled:  false,
			Interval: 60,
			Sources:  []string{},
		})
	}

	ok(c, response)
}

// UpdateTaskScheduleRequest is the request body for updating a task schedule.
type UpdateTaskScheduleRequest struct {
	Type     string   `json:"type" binding:"required"`
	Enabled  bool     `json:"enabled"`
	Interval int      `json:"interval" binding:"min=1,max=1440"`
	Sources  []string `json:"sources"`
}

// UpdateTaskSchedule updates a task schedule configuration.
//
// PUT /api/v1/task-schedules
func (h *SystemHandler) UpdateTaskSchedule(c *gin.Context) {
	var req UpdateTaskScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}

	if req.Type != string(model.TaskTypeVulnCrawl) && req.Type != string(model.TaskTypeArticleCrawl) {
		fail(c, http.StatusBadRequest, "invalid task type: "+req.Type)
		return
	}

	schedule := &model.TaskSchedule{
		Type:     model.TaskType(req.Type),
		Enabled:  req.Enabled,
		Interval: req.Interval,
		Sources:  req.Sources,
	}

	if err := h.scheduleRepo.Upsert(c, schedule); err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	ok(c, TaskScheduleResponse{
		Type:     string(schedule.Type),
		Enabled:  schedule.Enabled,
		Interval: schedule.Interval,
		Sources:  schedule.Sources,
	})
}