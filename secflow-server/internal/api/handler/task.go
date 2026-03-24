package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/secflow/server/internal/metrics"
	"github.com/secflow/server/internal/model"
	"github.com/secflow/server/internal/queue"
	"github.com/secflow/server/internal/repository"
)

// TaskHandler manages task dispatch and status tracking.
type TaskHandler struct {
	TaskRepo  *repository.TaskRepo
	Queue     *queue.Queue
	scheduler interface {
		StopTask(ctx context.Context, taskID string) error
	}
}

// SchedulerStopper is the interface for schedulers that can stop tasks.
type SchedulerStopper interface {
	StopTask(ctx context.Context, taskID string) error
}

func NewTaskHandler(tr *repository.TaskRepo, q *queue.Queue) *TaskHandler {
	return &TaskHandler{TaskRepo: tr, Queue: q}
}

// NewTaskHandlerWithScheduler creates a TaskHandler with scheduler for stop functionality.
func NewTaskHandlerWithScheduler(tr *repository.TaskRepo, q *queue.Queue, scheduler SchedulerStopper) *TaskHandler {
	return &TaskHandler{TaskRepo: tr, Queue: q, scheduler: scheduler}
}

// SetScheduler sets the scheduler for stop functionality.
func (h *TaskHandler) SetScheduler(scheduler SchedulerStopper) {
	h.scheduler = scheduler
}

// CreateVulnCrawlRequest is the body for creating a vuln crawl task.
type CreateVulnCrawlRequest struct {
	Sources      []string `json:"sources"        binding:"required"`
	PageLimit    int      `json:"page_limit"`
	EnableGithub bool     `json:"enable_github"`
	Proxy        string   `json:"proxy"`
	Priority     int      `json:"priority"` // 0-100, higher = more important
}

// CreateVulnCrawl creates a new vuln crawl task and enqueues it.
//
// POST /api/v1/tasks/vuln-crawl
func (h *TaskHandler) CreateVulnCrawl(c *gin.Context) {
	var req CreateVulnCrawlRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	if req.PageLimit == 0 {
		req.PageLimit = 1
	}

	payload := model.VulnCrawlPayload{
		Sources:      req.Sources,
		PageLimit:    req.PageLimit,
		EnableGithub: req.EnableGithub,
		Proxy:        req.Proxy,
	}
	rawPayload, _ := json.Marshal(payload)

	taskID := uuid.New().String()
	
	// Set default priority if not specified
	priority := req.Priority
	if priority <= 0 {
		priority = queue.PriorityMedium // Default to medium priority
	}
	if priority > queue.PriorityHigh {
		priority = queue.PriorityHigh // Cap at maximum priority
	}
	
	t := &model.Task{
		TaskID:         taskID,
		Type:           model.TaskTypeVulnCrawl,
		Status:         model.TaskPending,
		Payload:        rawPayload,
		Priority:       priority,
		TimeoutSeconds: 1800, // 30 minutes timeout
		MaxRetries:     3,
	}
	if err := h.TaskRepo.Create(c, t); err != nil {
		fail(c, http.StatusInternalServerError, "failed to create task: "+err.Error())
		return
	}

	// Record metrics
	priorityLabel := "low"
	if priority >= queue.PriorityHigh {
		priorityLabel = "high"
	} else if priority >= queue.PriorityMedium {
		priorityLabel = "medium"
	}
	metrics.TaskCreatedCount.WithLabelValues("vuln_crawl", priorityLabel).Inc()

	// Enqueue to Redis with priority.
	msg := &queue.TaskMessage{
		TaskID:   taskID,
		Type:     string(model.TaskTypeVulnCrawl),
		Payload:  rawPayload,
		Priority: priority,
	}
	
	// Use priority queue if priority is set, otherwise use regular queue
	var enqueueErr error
	if priority > 0 {
		enqueueErr = h.Queue.EnqueueWithPriority(c, msg, priority)
	} else {
		enqueueErr = h.Queue.Enqueue(c, msg)
	}
	
	if enqueueErr != nil {
		fail(c, http.StatusInternalServerError, "failed to enqueue task: "+enqueueErr.Error())
		return
	}

	ok(c, gin.H{"task_id": taskID, "priority": priority})
}

// CreateArticleCrawlRequest is the body for creating an article crawl task.
type CreateArticleCrawlRequest struct {
	Sources []string `json:"sources" binding:"required"`
	Limit   int      `json:"limit"`
}

// CreateArticleCrawl creates a new article crawl task and enqueues it.
//
// POST /api/v1/tasks/article-crawl
func (h *TaskHandler) CreateArticleCrawl(c *gin.Context) {
	var req CreateArticleCrawlRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	if req.Limit == 0 {
		req.Limit = 10
	}

	payload := map[string]interface{}{
		"type":    "article_crawl",
		"sources": req.Sources,
		"limit":   req.Limit,
	}
	rawPayload, _ := json.Marshal(payload)

	taskID := uuid.New().String()
	t := &model.Task{
		TaskID:         taskID,
		Type:           model.TaskTypeArticleCrawl,
		Status:         model.TaskPending,
		Payload:        rawPayload,
		TimeoutSeconds: 1800, // 30 minutes timeout
		MaxRetries:     3,
	}
	if err := h.TaskRepo.Create(c, t); err != nil {
		fail(c, http.StatusInternalServerError, "failed to create task")
		return
	}

	// Enqueue to Redis.
	msg := &queue.TaskMessage{
		TaskID:  taskID,
		Type:    string(model.TaskTypeArticleCrawl),
		Payload: rawPayload,
	}
	if err := h.Queue.Enqueue(c, msg); err != nil {
		fail(c, http.StatusInternalServerError, "failed to enqueue task")
		return
	}

	ok(c, gin.H{"task_id": taskID})
}

// List returns paginated tasks with optional status filter.
//
// GET /api/v1/tasks
func (h *TaskHandler) List(c *gin.Context) {
	page, pageSize := pageParams(c)
	status := model.TaskStatus(c.Query("status"))
	items, total, err := h.TaskRepo.List(c, status, page, pageSize)
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	okPage(c, total, page, pageSize, items)
}

// Get returns a single task by task_id.
//
// GET /api/v1/tasks/:id
func (h *TaskHandler) Get(c *gin.Context) {
	t, err := h.TaskRepo.GetByTaskID(c, c.Param("id"))
	if err != nil {
		fail(c, http.StatusNotFound, "task not found")
		return
	}
	// Enrich with real-time progress from Redis.
	progress, _ := h.Queue.GetProgress(c, t.TaskID)
	ok(c, gin.H{"task": t, "progress": progress})
}

// Delete removes a task by task_id and cancels it from queue if pending.
//
// DELETE /api/v1/tasks/:id
func (h *TaskHandler) Delete(c *gin.Context) {
	taskID := c.Param("id")

	// Get task first to check status
	t, err := h.TaskRepo.GetByTaskID(c, taskID)
	if err != nil {
		fail(c, http.StatusNotFound, "task not found")
		return
	}

	// For running tasks, notify client via WebSocket before deletion
	if t.Status == model.TaskRunning {
		if h.scheduler != nil {
			if err := h.scheduler.StopTask(c.Request.Context(), taskID); err != nil {
				log.Warn().Str("task_id", taskID).Err(err).Msg("task handler: failed to stop task via scheduler")
			}
		}
	}

	// Cancel task from queue if it's pending or dispatched
	if t.Status == model.TaskPending || t.Status == model.TaskDispatched {
		if err := h.Queue.CancelTask(c, taskID); err != nil {
			log.Warn().Str("task_id", taskID).Err(err).Msg("failed to cancel task in queue")
		}
	}

	// Delete from database
	if err := h.TaskRepo.DeleteByTaskID(c, taskID); err != nil {
		fail(c, http.StatusInternalServerError, "failed to delete task: "+err.Error())
		return
	}

	ok(c, nil)
}

// Stop cancels a running or pending task.
//
// POST /api/v1/tasks/:id/stop
func (h *TaskHandler) Stop(c *gin.Context) {
	taskID := c.Param("id")

	t, err := h.TaskRepo.GetByTaskID(c, taskID)
	if err != nil {
		fail(c, http.StatusNotFound, "task not found")
		return
	}

	// Only allow stopping pending, dispatched, or running tasks
	if t.Status != model.TaskPending && t.Status != model.TaskDispatched && t.Status != model.TaskRunning {
		fail(c, http.StatusBadRequest, "cannot stop task with status: "+string(t.Status))
		return
	}

	// Use scheduler to stop task if available (handles queue cancel + WebSocket notify)
	if h.scheduler != nil {
		if err := h.scheduler.StopTask(c.Request.Context(), taskID); err != nil {
			log.Warn().Str("task_id", taskID).Err(err).Msg("task handler: scheduler stop failed")
		}
	} else {
		// Fallback: cancel from queue directly
		if err := h.Queue.CancelTask(c, taskID); err != nil {
			log.Warn().Str("task_id", taskID).Err(err).Msg("failed to cancel task in queue")
		}
	}

	// Update status to failed with cancellation message
	if err := h.TaskRepo.UpdateStatus(c, taskID, model.TaskFailed, ""); err != nil {
		fail(c, http.StatusInternalServerError, "failed to stop task: "+err.Error())
		return
	}

	ok(c, gin.H{"task_id": taskID, "status": "failed"})
}

// ============================================
// Dead Letter Queue API Endpoints
// ============================================

// ListDeadLetters returns paginated dead letter tasks.
//
// GET /api/v1/dead-letters
func (h *TaskHandler) ListDeadLetters(c *gin.Context) {
	page, pageSize := pageParams(c)
	offset := int((page - 1) * pageSize)
	limit := int(pageSize)

	items, total, err := h.Queue.ListDeadLetters(c, offset, limit)
	if err != nil {
		fail(c, http.StatusInternalServerError, "failed to list dead letters: "+err.Error())
		return
	}

	okPage(c, total, page, pageSize, items)
}

// GetDeadLetter returns a single dead letter task by task_id.
//
// GET /api/v1/dead-letters/:id
func (h *TaskHandler) GetDeadLetter(c *gin.Context) {
	taskID := c.Param("id")

	dl, err := h.Queue.GetDeadLetter(c, taskID)
	if err != nil {
		fail(c, http.StatusInternalServerError, "failed to get dead letter: "+err.Error())
		return
	}

	if dl == nil {
		fail(c, http.StatusNotFound, "dead letter not found")
		return
	}

	ok(c, gin.H{"dead_letter": dl})
}

// RetryDeadLetter moves a dead letter task back to the main queue for retry.
//
// POST /api/v1/dead-letters/:id/retry
func (h *TaskHandler) RetryDeadLetter(c *gin.Context) {
	taskID := c.Param("id")

	// Get dead letter info first
	dl, err := h.Queue.GetDeadLetter(c, taskID)
	if err != nil {
		fail(c, http.StatusInternalServerError, "failed to get dead letter: "+err.Error())
		return
	}

	if dl == nil {
		fail(c, http.StatusNotFound, "dead letter not found")
		return
	}

	// Requeue to main queue
	if err := h.Queue.RequeueDeadLetter(c, taskID); err != nil {
		fail(c, http.StatusInternalServerError, "failed to requeue dead letter: "+err.Error())
		return
	}

	// Create new task in database
	t := &model.Task{
		TaskID:         fmt.Sprintf("%s-retry-%d", taskID, time.Now().Unix()),
		Type:           model.TaskType(dl.Type),
		Status:         model.TaskPending,
		Payload:        json.RawMessage(dl.Payload),
		TimeoutSeconds: 1800,
		MaxRetries:     3,
	}
	if err := h.TaskRepo.Create(c, t); err != nil {
		log.Warn().Err(err).Str("task_id", t.TaskID).Msg("failed to create retry task in database")
	}

	ok(c, gin.H{
		"message":     "dead letter requeued for retry",
		"new_task_id": t.TaskID,
	})
}

// DeleteDeadLetter removes a dead letter task permanently.
//
// DELETE /api/v1/dead-letters/:id
func (h *TaskHandler) DeleteDeadLetter(c *gin.Context) {
	taskID := c.Param("id")

	if err := h.Queue.RemoveDeadLetter(c, taskID); err != nil {
		fail(c, http.StatusInternalServerError, "failed to delete dead letter: "+err.Error())
		return
	}

	ok(c, gin.H{"message": "dead letter deleted"})
}

// GetDeadLetterStats returns statistics about the dead letter queue.
//
// GET /api/v1/dead-letters/stats
func (h *TaskHandler) GetDeadLetterStats(c *gin.Context) {
	count, err := h.Queue.DeadLetterQueueLength(c)
	if err != nil {
		fail(c, http.StatusInternalServerError, "failed to get dead letter stats: "+err.Error())
		return
	}

	// Get counts by type
	items, _, err := h.Queue.ListDeadLetters(c, 0, 100)
	if err != nil {
		fail(c, http.StatusInternalServerError, "failed to get dead letter details: "+err.Error())
		return
	}

	typeCounts := make(map[string]int)
	for _, item := range items {
		typeCounts[item.Type]++
	}

	ok(c, gin.H{
		"total_count": count,
		"by_type":     typeCounts,
	})
}
