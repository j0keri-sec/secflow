// Package scheduler implements the task scheduler that dispatches tasks
// from the Redis queue to connected client nodes.
package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/secflow/server/config"
	"github.com/secflow/server/internal/metrics"
	"github.com/secflow/server/internal/model"
	"github.com/secflow/server/internal/queue"
	"github.com/secflow/server/internal/repository"
	"github.com/secflow/server/internal/ws"
)

// Scheduler dispatches tasks from the queue to connected nodes.
type Scheduler struct {
	queue         *queue.Queue
	taskRepo      *repository.TaskRepo
	nodeRepo      *repository.NodeRepo
	hub           *ws.Hub
	stopCh        chan struct{}
	maxRetries    int           // Maximum retry attempts for failed tasks
	retryInterval time.Duration // Base retry interval
	batchSize     int           // Number of tasks to dispatch in one batch
	taskTimeout   time.Duration // Default task timeout
	timeoutCheck  time.Duration // Timeout check interval
}

// NewWithConfig creates a new task scheduler with configuration.
func NewWithConfig(q *queue.Queue, tr *repository.TaskRepo, nr *repository.NodeRepo, h *ws.Hub, cfg config.SchedulerConfig) *Scheduler {
	s := &Scheduler{
		queue:         q,
		taskRepo:      tr,
		nodeRepo:      nr,
		hub:           h,
		stopCh:        make(chan struct{}),
		maxRetries:    cfg.MaxRetries,
		retryInterval: cfg.RetryInterval,
		batchSize:     cfg.BatchSize,
		taskTimeout:   cfg.TaskTimeout,
		timeoutCheck:  cfg.TimeoutCheck,
	}
	
	// Ensure values are within valid ranges
	if s.maxRetries < 0 {
		s.maxRetries = 3
	}
	if s.batchSize < 1 {
		s.batchSize = 1
	}
	if s.batchSize > 100 {
		s.batchSize = 100
	}
	if s.taskTimeout == 0 {
		s.taskTimeout = 30 * time.Minute
	}
	if s.timeoutCheck == 0 {
		s.timeoutCheck = 1 * time.Minute
	}
	
	return s
}

// New creates a new task scheduler with default values.
func New(q *queue.Queue, tr *repository.TaskRepo, h *ws.Hub) *Scheduler {
	return &Scheduler{
		queue:         q,
		taskRepo:      tr,
		hub:           h,
		stopCh:        make(chan struct{}),
		maxRetries:    3,               // Default: retry 3 times
		retryInterval: 5 * time.Minute, // Default: retry after 5 minutes
		batchSize:     3,               // Default: dispatch 3 tasks at once
		taskTimeout:   30 * time.Minute, // Default: 30 minute timeout
		timeoutCheck:  1 * time.Minute, // Default: check every minute
	}
}

// SetMaxRetries sets the maximum number of retry attempts.
func (s *Scheduler) SetMaxRetries(maxRetries int) {
	s.maxRetries = maxRetries
}

// SetRetryInterval sets the base retry interval.
func (s *Scheduler) SetRetryInterval(interval time.Duration) {
	s.retryInterval = interval
}

// SetHub injects the WebSocket hub (resolves circular dependency).
func (s *Scheduler) SetHub(hub *ws.Hub) {
	s.hub = hub
}

// SetBatchSize sets the number of tasks to dispatch in one batch.
func (s *Scheduler) SetBatchSize(batchSize int) {
	if batchSize < 1 {
		batchSize = 1
	}
	if batchSize > 100 {
		batchSize = 100 // Cap at 100
	}
	s.batchSize = batchSize
}

// Start begins the scheduler loop.
func (s *Scheduler) Start(ctx context.Context) {
	log.Info().Msg("task scheduler started")
	go s.run(ctx)
}

// Stop stops the scheduler.
func (s *Scheduler) Stop() {
	close(s.stopCh)
}

func (s *Scheduler) run(ctx context.Context) {
	// Primary task queue ticker
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	
	// Retry queue ticker (check every 10 seconds)
	retryTicker := time.NewTicker(10 * time.Second)
	defer retryTicker.Stop()
	
	// Timeout checker ticker (check every 1 minute)
	timeoutTicker := time.NewTicker(1 * time.Minute)
	defer timeoutTicker.Stop()
	
	// Batch heartbeat ticker (every 30 seconds)
	heartbeatTicker := time.NewTicker(30 * time.Second)
	defer heartbeatTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.tryDispatch(ctx)
		case <-retryTicker.C:
			s.processRetryQueue(ctx)
		case <-timeoutTicker.C:
			s.checkTaskTimeouts(ctx)
		case <-heartbeatTicker.C:
			s.updateNodeHeartbeatsBatch(ctx)
		}
	}
}

// updateNodeHeartbeatsBatch updates heartbeats for all connected nodes using Pipeline.
func (s *Scheduler) updateNodeHeartbeatsBatch(ctx context.Context) {
	// Get all connected nodes
	nodes := s.hub.ConnectedNodes()
	if len(nodes) == 0 {
		return
	}
	
	// Use Pipeline to batch update heartbeats
	updated, err := s.queue.HeartbeatBatch(ctx, nodes)
	if err != nil {
		log.Error().Err(err).Msg("scheduler: failed to batch update heartbeats")
		return
	}
	
	log.Debug().Int("node_count", len(nodes)).Int("updated", updated).Msg("scheduler: batch heartbeat update completed")
}

func (s *Scheduler) tryDispatch(ctx context.Context) {
	log.Info().Msg("scheduler: checking for tasks to dispatch")
	
	// Get connected nodes
	nodes := s.hub.ConnectedNodes()
	log.Info().Int("node_count", len(nodes)).Msg("scheduler: connected nodes")
	if len(nodes) == 0 {
		log.Info().Msg("scheduler: no connected nodes, skipping dispatch")
		return
	}

	// Get the best node for task dispatching
	// Use intelligent load balancing if node repo is available
	var nodeID string
	if s.nodeRepo != nil {
		// Use intelligent load balancing with performance metrics
		nodeID = s.hub.GetBestNodeIntelligent(func(nodeID string) (model.NodeTaskStats, error) {
			node, err := s.nodeRepo.GetByNodeID(ctx, nodeID)
			if err != nil {
				return model.NodeTaskStats{}, err
			}
			return node.TaskStats, nil
		})
	} else {
		// Fallback to simple task count based balancing
		nodeID = s.hub.GetBestNode()
	}
	
	log.Debug().Str("selected_node_id", nodeID).Msg("scheduler: selected best node")
	if nodeID == "" {
		log.Warn().Msg("scheduler: no available nodes")
		metrics.SchedulerDispatchCount.WithLabelValues("no_nodes").Inc()
		return
	}

	// First, try to dequeue batch from priority queue
	log.Info().Int("batch_size", s.batchSize).Msg("scheduler: attempting to dequeue batch from priority queue")
	priorityTasks, err := s.queue.DequeueBatchPriority(ctx, s.batchSize)
	if err != nil {
		log.Error().Err(err).Msg("scheduler: dequeue batch from priority queue failed")
		return
	}
	
	var tasks []queue.TaskMessage
	// If no priority tasks, try regular queue
	if len(priorityTasks) == 0 {
		log.Info().Msg("scheduler: no priority tasks, checking regular queue")
		regularTasks, err := s.queue.DequeueBatch(ctx, s.batchSize)
		if err != nil {
			log.Error().Err(err).Msg("scheduler: dequeue batch failed")
			return
		}
		tasks = regularTasks
	} else {
		tasks = priorityTasks
	}
	
	if len(tasks) == 0 {
		log.Info().Msg("scheduler: no tasks available in queue")
		return // No tasks available
	}
	
	log.Info().
		Int("task_count", len(tasks)).
		Str("node_id", nodeID).
		Msg("scheduler: dequeued tasks")

	// Update task status to dispatched for all tasks
	var taskMsgs []*ws.Message
	for _, msg := range tasks {
		log.Debug().Str("task_id", msg.TaskID).Msg("scheduler: updating task status to dispatched")
		if err := s.taskRepo.UpdateStatus(ctx, msg.TaskID, model.TaskDispatched, ""); err != nil {
			log.Error().Err(err).Str("task_id", msg.TaskID).Msg("scheduler: failed to update task status")
			continue
		}
		
		// Create task payload
		taskPayload := TaskPayload{
			TaskID:  msg.TaskID,
			Type:    msg.Type,
			Payload: msg.Payload,
		}
		payloadBytes, _ := json.Marshal(taskPayload)
		
		// Create task message
		taskMsg := &ws.Message{
			Type:    ws.MsgTypeTask,
			Version: ws.ProtocolVersion,
			Payload: payloadBytes,
		}
		taskMsgs = append(taskMsgs, taskMsg)
	}
	
	if len(taskMsgs) == 0 {
		log.Warn().Msg("scheduler: no valid tasks to dispatch")
		return
	}

	// Increment task count for the selected node (by number of tasks)
	for i := 0; i < len(taskMsgs); i++ {
		s.hub.IncTaskCount(nodeID)
	}
	log.Debug().Str("node_id", nodeID).Int("task_count", len(taskMsgs)).Msg("scheduler: incremented node task count")

	log.Debug().Str("node_id", nodeID).Int("task_count", len(taskMsgs)).Msg("scheduler: sending tasks to node")
	if !s.hub.SendBatch(nodeID, taskMsgs) {
		log.Error().Str("node_id", nodeID).Int("task_count", len(taskMsgs)).Msg("scheduler: failed to send tasks to node")
		// Record metrics
		metrics.SchedulerDispatchCount.WithLabelValues("failed").Inc()

		// Decrement task counts since send failed
		for i := 0; i < len(taskMsgs); i++ {
			s.hub.DecTaskCount(nodeID)
		}

		// Re-queue the failed tasks
		for _, msg := range tasks {
			if err := s.queue.Enqueue(ctx, &msg); err != nil {
				log.Error().Err(err).Str("task_id", msg.TaskID).Msg("scheduler: failed to re-queue task")
			}
		}
		return
	}
	log.Debug().Str("node_id", nodeID).Int("task_count", len(taskMsgs)).Msg("scheduler: successfully sent tasks to node")
	
	// Record metrics
	metrics.SchedulerDispatchCount.WithLabelValues("success").Inc()
	metrics.SchedulerBatchSize.Observe(float64(len(tasks)))

	// Update task status to running for all tasks
	for _, msg := range tasks {
		log.Debug().Str("task_id", msg.TaskID).Str("node_id", nodeID).Msg("scheduler: updating task status to running")
		if err := s.taskRepo.UpdateStatus(ctx, msg.TaskID, model.TaskRunning, nodeID); err != nil {
			log.Error().Err(err).Str("task_id", msg.TaskID).Msg("scheduler: failed to update task status to running")
		}
		// Record task start time for timeout tracking
		if err := s.taskRepo.UpdateTaskStartedAt(ctx, msg.TaskID); err != nil {
			log.Warn().Err(err).Str("task_id", msg.TaskID).Msg("scheduler: failed to update task started_at")
		}
	}

	log.Info().
		Int("task_count", len(tasks)).
		Str("node_id", nodeID).
		Msg("tasks dispatched")
}

// processRetryQueue checks for tasks that are ready to be retried and moves them to the main queue.
func (s *Scheduler) processRetryQueue(ctx context.Context) {
	for {
		// Try to get a ready retry task
		msg, err := s.queue.DequeueRetry(ctx)
		if err != nil {
			log.Error().Err(err).Msg("scheduler: failed to dequeue retry task")
			return
		}
		if msg == nil {
			// No retry tasks ready
			return
		}
		
		// Move back to main queue
		log.Info().Str("task_id", msg.TaskID).Msg("scheduler: moving retry task back to main queue")
		if err := s.queue.Enqueue(ctx, msg); err != nil {
			log.Error().Err(err).Str("task_id", msg.TaskID).Msg("scheduler: failed to enqueue retry task")
			continue
		}
		
		// Update task status in MongoDB
		if err := s.taskRepo.UpdateStatus(ctx, msg.TaskID, model.TaskPending, ""); err != nil {
			log.Error().Err(err).Str("task_id", msg.TaskID).Msg("scheduler: failed to update task status for retry")
		}
	}
}

// HandleTaskFailure processes a failed task and decides whether to retry or mark as failed.
func (s *Scheduler) HandleTaskFailure(ctx context.Context, taskID string, nodeID string, errorMsg string) error {
	// Get current task
	task, err := s.taskRepo.GetByTaskID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}
	
	if task == nil {
		return fmt.Errorf("task not found: %s", taskID)
	}
	
	// Decrement node task count
	s.hub.DecTaskCount(nodeID)
	
	// Check if we've exceeded max retries
	retryCount := task.RetryCount
	maxRetries := task.MaxRetries
	if maxRetries <= 0 {
		maxRetries = s.maxRetries // Use scheduler default
	}
	
	if retryCount >= maxRetries {
		// Max retries exceeded, mark as failed
		log.Info().
			Str("task_id", taskID).
			Int("retry_count", retryCount).
			Int("max_retries", maxRetries).
			Msg("task failed after max retries, moving to dead letter queue")

		// Update task status to failed
		err := s.taskRepo.UpdateStatus(ctx, taskID, model.TaskFailed, "")
		if err != nil {
			return fmt.Errorf("failed to update task status: %w", err)
		}

		// Add to dead letter queue for later analysis/retry
		retryMsg := &queue.TaskMessage{
			TaskID:  taskID,
			Type:    string(task.Type),
			Payload: task.Payload,
		}
		if err := s.queue.EnqueueDeadLetter(ctx, retryMsg, retryCount, errorMsg); err != nil {
			log.Error().Err(err).Str("task_id", taskID).Msg("failed to add to dead letter queue")
		}

		// Record error
		s.queue.RecordError(ctx, taskID, errorMsg)

		return nil
	}
	
	// Schedule retry
	retryCount++
	log.Info().
		Str("task_id", taskID).
		Int("retry_count", retryCount).
		Int("max_retries", maxRetries).
		Msg("scheduling task retry")
	
	// Update task retry metadata
	task.RetryCount = retryCount
	task.LastRetryAt = time.Now()
	task.Error = errorMsg
	
	// Add to retry queue with exponential backoff
	retryMsg := &queue.TaskMessage{
		TaskID:  taskID,
		Type:    string(task.Type),
		Payload: task.Payload,
	}
	
	if err := s.queue.EnqueueRetry(ctx, retryMsg, retryCount, maxRetries, errorMsg); err != nil {
		return fmt.Errorf("failed to enqueue retry: %w", err)
	}
	
	// Update task in MongoDB
	if err := s.taskRepo.UpdateRetryMetadata(ctx, taskID, retryCount, maxRetries, errorMsg); err != nil {
		log.Warn().Err(err).Str("task_id", taskID).Msg("failed to update retry metadata")
	}
	
	return nil
}

// checkTaskTimeouts checks for running tasks that have exceeded their timeout and cancels them.
func (s *Scheduler) checkTaskTimeouts(ctx context.Context) {
	log.Debug().Msg("scheduler: checking for task timeouts")
	
	// Get timed out tasks
	timedOutTasks, err := s.taskRepo.GetTimedOutTasks(ctx)
	if err != nil {
		log.Error().Err(err).Msg("scheduler: failed to get timed out tasks")
		return
	}
	
	if len(timedOutTasks) == 0 {
		log.Debug().Msg("scheduler: no timed out tasks found")
		return
	}
	
	log.Info().Int("count", len(timedOutTasks)).Msg("scheduler: found timed out tasks, cancelling")
	
	// Cancel each timed out task
	for _, task := range timedOutTasks {
		log.Info().
			Str("task_id", task.TaskID).
			Str("node_id", task.AssignedTo).
			Int("timeout_seconds", task.TimeoutSeconds).
			Msg("scheduler: cancelling timed out task")
		
		// Send cancel message to node if connected
		if task.AssignedTo != "" {
			cancelMsg := &ws.Message{
				Type:    ws.MsgTypeTaskCancel,
				Version: ws.ProtocolVersion,
				Payload: []byte(fmt.Sprintf(`{"task_id":"%s"}`, task.TaskID)),
			}
			if !s.hub.Send(task.AssignedTo, cancelMsg) {
				log.Warn().Str("task_id", task.TaskID).Str("node_id", task.AssignedTo).Msg("scheduler: failed to send cancel message to node")
			}
		}
		
		// Handle task failure with timeout error
		timeoutErr := fmt.Sprintf("task timed out after %d seconds", task.TimeoutSeconds)
		if err := s.HandleTaskFailure(ctx, task.TaskID, task.AssignedTo, timeoutErr); err != nil {
			log.Error().Err(err).Str("task_id", task.TaskID).Msg("scheduler: failed to handle task timeout")
		}
	}
}

// TaskPayload represents the task data sent to clients.
type TaskPayload struct {
	TaskID  string          `json:"task_id"`
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// BatchTaskPayload represents multiple tasks sent to clients in one message.
type BatchTaskPayload struct {
	Tasks []TaskPayload `json:"tasks"` // Array of tasks
}

// StopTask stops a task by cancelling it from the queue and notifying the client.
func (s *Scheduler) StopTask(ctx context.Context, taskID string) error {
	// Cancel from queue (both regular and priority)
	if err := s.queue.CancelTask(ctx, taskID); err != nil {
		log.Warn().Str("task_id", taskID).Err(err).Msg("scheduler: failed to cancel task from queue")
	}

	// Get task to find assigned node
	task, err := s.taskRepo.GetByTaskID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("get task: %w", err)
	}

	// Notify client via WebSocket if task was assigned
	if task != nil && task.AssignedTo != "" {
		cancelMsg := &ws.Message{
			Type:    ws.MsgTypeTaskCancel,
			Version: ws.ProtocolVersion,
			Payload: []byte(fmt.Sprintf(`{"task_id":"%s"}`, taskID)),
		}
		if !s.hub.Send(task.AssignedTo, cancelMsg) {
			log.Warn().Str("task_id", taskID).Str("node_id", task.AssignedTo).Msg("scheduler: failed to send cancel message to node")
		}
		// Decrement task count since we're cancelling
		s.hub.DecTaskCount(task.AssignedTo)
	}

	return nil
}
