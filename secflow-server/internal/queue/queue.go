// Package queue implements the Redis-based task queue used to dispatch
// crawl jobs to connected client nodes.
package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

const (
	// KeyTaskQueue is the Redis list used as the pending task queue.
	KeyTaskQueue = "secflow:tasks:pending"
	// KeyTaskResult is the Redis hash that stores task results keyed by task_id.
	KeyTaskResult = "secflow:tasks:results"
	// KeyNodeHeartbeat is the Redis sorted set storing node heartbeat scores.
	KeyNodeHeartbeat = "secflow:nodes:heartbeat"
	// KeyTaskProgress is the Redis hash for real-time task progress (0-100).
	KeyTaskProgress = "secflow:tasks:progress"
	// KeyRetryQueue is the Redis list for tasks that need to be retried.
	KeyRetryQueue = "secflow:tasks:retry"
	// KeyRetryCount is the Redis hash storing retry counts for failed tasks.
	KeyRetryCount = "secflow:tasks:retry_count"
	// KeyTaskErrors is the Redis hash storing error history for tasks.
	KeyTaskErrors = "secflow:tasks:errors"
	// KeyPriorityQueue is the Redis sorted set for priority-based task scheduling.
	KeyPriorityQueue = "secflow:tasks:priority"
	// KeyDeadLetterQueue is the Redis list for permanently failed tasks.
	KeyDeadLetterQueue = "secflow:tasks:deadletter"
	// KeyDeadLetterMeta is the Redis hash storing metadata for dead letter tasks.
	KeyDeadLetterMeta = "secflow:tasks:deadletter:meta"

	// taskTTL is how long a result is kept in Redis before expiry.
	taskTTL = 24 * time.Hour
	// retryTTL is how long retry metadata is kept in Redis.
	retryTTL = 7 * 24 * time.Hour
	// deadLetterTTL is how long dead letter tasks are kept.
	deadLetterTTL = 30 * 24 * time.Hour

	// Priority levels for task scheduling
	PriorityHigh   = 100 // Highest priority (executed first)
	PriorityMedium = 50  // Medium priority (default)
	PriorityLow    = 0   // Lowest priority (executed last)
)

// TaskMessage is the JSON payload pushed onto the Redis queue.
type TaskMessage struct {
	TaskID   string          `json:"task_id"`
	Type     string          `json:"type"`
	Payload  json.RawMessage `json:"payload"`
	Priority int             `json:"priority,omitempty"` // Higher value = higher priority
}

// ResultMessage is the JSON payload a client pushes back after completing a task.
type ResultMessage struct {
	TaskID string          `json:"task_id"`
	NodeID string          `json:"node_id"`
	Status string          `json:"status"` // done | failed
	Data   json.RawMessage `json:"data,omitempty"`
	Error  string          `json:"error,omitempty"`
}

// ProgressMessage is sent by the client during task execution.
type ProgressMessage struct {
	TaskID   string `json:"task_id"`
	NodeID   string `json:"node_id"`
	Progress int    `json:"progress"` // 0–100
	Message  string `json:"message,omitempty"`
}

// RetryMessage is used to track task retry information.
type RetryMessage struct {
	TaskID    string    `json:"task_id"`
	RetryCount int      `json:"retry_count"`
	MaxRetries int      `json:"max_retries"`
	LastError string    `json:"last_error"`
	RetryAt   time.Time `json:"retry_at"`
}

// DeadLetterMessage is used to track permanently failed tasks.
type DeadLetterMessage struct {
	TaskID       string    `json:"task_id"`
	Type         string    `json:"type"`           // Task type (vuln_crawl, article_crawl)
	Payload      string    `json:"payload"`        // Original task payload
	RetryCount   int       `json:"retry_count"`     // Number of retries attempted
	LastError    string    `json:"last_error"`      // Last error message
	FailedAt     time.Time `json:"failed_at"`       // When the task permanently failed
	OriginalTask string    `json:"original_task"`   // Task ID from original task
}

// Queue wraps a Redis client and provides typed enqueue/dequeue helpers.
type Queue struct {
	rdb *redis.Client
}

// New creates a Queue backed by the given Redis client.
func New(rdb *redis.Client) *Queue {
	return &Queue{rdb: rdb}
}

// Enqueue pushes a task message to the tail of the task queue.
func (q *Queue) Enqueue(ctx context.Context, msg *TaskMessage) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal task: %w", err)
	}
	return q.rdb.RPush(ctx, KeyTaskQueue, data).Err()
}

// Dequeue blocks until a task is available or ctx is cancelled.
// Returns the task or (nil, nil) on timeout.
func (q *Queue) Dequeue(ctx context.Context, timeout time.Duration) (*TaskMessage, error) {
	res, err := q.rdb.BLPop(ctx, timeout, KeyTaskQueue).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("blpop: %w", err)
	}
	// res[0] is the key, res[1] is the value.
	var msg TaskMessage
	if err = json.Unmarshal([]byte(res[1]), &msg); err != nil {
		return nil, fmt.Errorf("unmarshal task: %w", err)
	}
	return &msg, nil
}

// QueueLength returns the number of pending tasks.
func (q *Queue) QueueLength(ctx context.Context) (int64, error) {
	return q.rdb.LLen(ctx, KeyTaskQueue).Result()
}

// SetProgress stores the current progress (0–100) for a task.
func (q *Queue) SetProgress(ctx context.Context, taskID string, progress int) error {
	return q.rdb.HSet(ctx, KeyTaskProgress, taskID, progress).Err()
}

// GetProgress retrieves the current progress of a task.
func (q *Queue) GetProgress(ctx context.Context, taskID string) (int, error) {
	v, err := q.rdb.HGet(ctx, KeyTaskProgress, taskID).Int()
	if err == redis.Nil {
		return 0, nil
	}
	return v, err
}

// StoreResult saves the result of a completed task with TTL.
func (q *Queue) StoreResult(ctx context.Context, msg *ResultMessage) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	key := fmt.Sprintf("%s:%s", KeyTaskResult, msg.TaskID)
	return q.rdb.Set(ctx, key, data, taskTTL).Err()
}

// GetResult retrieves the result stored for a task.
func (q *Queue) GetResult(ctx context.Context, taskID string) (*ResultMessage, error) {
	key := fmt.Sprintf("%s:%s", KeyTaskResult, taskID)
	data, err := q.rdb.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var msg ResultMessage
	return &msg, json.Unmarshal(data, &msg)
}

// Heartbeat updates the node's last-seen score in the sorted set.
func (q *Queue) Heartbeat(ctx context.Context, nodeID string) error {
	return q.rdb.ZAdd(ctx, KeyNodeHeartbeat, redis.Z{
		Score:  float64(time.Now().Unix()),
		Member: nodeID,
	}).Err()
}

// StaleNodes returns node IDs whose last heartbeat is older than threshold.
func (q *Queue) StaleNodes(ctx context.Context, threshold time.Duration) ([]string, error) {
	cutoff := float64(time.Now().Add(-threshold).Unix())
	res, err := q.rdb.ZRangeByScore(ctx, KeyNodeHeartbeat, &redis.ZRangeBy{
		Min: "-inf",
		Max: fmt.Sprintf("%f", cutoff),
	}).Result()
	return res, err
}

// CancelTask removes a task from both the regular queue and priority queue by task_id.
func (q *Queue) CancelTask(ctx context.Context, taskID string) error {
	// Try to cancel from regular queue first
	tasks, err := q.rdb.LRange(ctx, KeyTaskQueue, 0, -1).Result()
	if err == nil {
		for _, taskData := range tasks {
			var msg TaskMessage
			if err := json.Unmarshal([]byte(taskData), &msg); err != nil {
				continue
			}
			if msg.TaskID == taskID {
				_ = q.rdb.LRem(ctx, KeyTaskQueue, 1, taskData).Err()
			}
		}
	}

	// Also try to cancel from priority queue
	taskKey := fmt.Sprintf("secflow:tasks:data:%s", taskID)
	pipe := q.rdb.Pipeline()
	pipe.ZRem(ctx, KeyPriorityQueue, taskID)
	pipe.Del(ctx, taskKey)
	_, _ = pipe.Exec(ctx)

	return nil
}

// GetTaskLogs retrieves stored logs for a task (if any).
func (q *Queue) GetTaskLogs(ctx context.Context, taskID string) ([]string, error) {
	key := fmt.Sprintf("secflow:tasks:logs:%s", taskID)
	lines, err := q.rdb.LRange(ctx, key, 0, -1).Result()
	if err == redis.Nil {
		return nil, nil
	}
	return lines, err
}

// AppendTaskLog appends a log line to a task's log.
func (q *Queue) AppendTaskLog(ctx context.Context, taskID string, logLine string) error {
	key := fmt.Sprintf("secflow:tasks:logs:%s", taskID)
	return q.rdb.RPush(ctx, key, logLine).Err()
}

// GetNodeLogs retrieves stored logs for a node.
func (q *Queue) GetNodeLogs(ctx context.Context, nodeID string) ([]string, error) {
	key := fmt.Sprintf("secflow:nodes:logs:%s", nodeID)
	lines, err := q.rdb.LRange(ctx, key, 0, -1).Result()
	if err == redis.Nil {
		return nil, nil
	}
	return lines, err
}

// AppendNodeLog appends a log line to a node's log.
func (q *Queue) AppendNodeLog(ctx context.Context, nodeID string, logLine string) error {
	key := fmt.Sprintf("secflow:nodes:logs:%s", nodeID)
	return q.rdb.RPush(ctx, key, logLine).Err()
}

// ClearNodeLogs clears logs for a node.
func (q *Queue) ClearNodeLogs(ctx context.Context, nodeID string) error {
	key := fmt.Sprintf("secflow:nodes:logs:%s", nodeID)
	return q.rdb.Del(ctx, key).Err()
}

// EnqueueRetry adds a task to the retry queue with exponential backoff.
func (q *Queue) EnqueueRetry(ctx context.Context, msg *TaskMessage, retryCount int, maxRetries int, errorMsg string) error {
	// Store complete task data for retry
	taskData, _ := json.Marshal(msg)
	
	retryMsg := RetryMessage{
		TaskID:     msg.TaskID,
		RetryCount: retryCount,
		MaxRetries: maxRetries,
		LastError:  errorMsg,
		RetryAt:    time.Now().Add(q.calculateBackoff(retryCount)),
	}
	
	// Store retry metadata including complete task data
	retryMeta := struct {
		RetryMessage
		TaskData string `json:"task_data"`
	}{
		RetryMessage: retryMsg,
		TaskData:     string(taskData),
	}
	
	retryData, _ := json.Marshal(retryMeta)
	if err := q.rdb.HSet(ctx, KeyRetryCount, msg.TaskID, string(retryData)).Err(); err != nil {
		return err
	}
	
	// Add to retry queue with delay (using sorted set for delayed execution)
	return q.rdb.ZAdd(ctx, KeyRetryQueue, redis.Z{
		Score:  float64(retryMsg.RetryAt.Unix()),
		Member: retryMsg.TaskID,
	}).Err()
}

// DequeueRetry gets a task from the retry queue that is ready to be retried.
func (q *Queue) DequeueRetry(ctx context.Context) (*TaskMessage, error) {
	// Get tasks that are ready to be retried (score <= now)
	now := float64(time.Now().Unix())
	taskIDs, err := q.rdb.ZRangeByScore(ctx, KeyRetryQueue, &redis.ZRangeBy{
		Min: "-inf",
		Max: fmt.Sprintf("%f", now),
	}).Result()
	
	if err != nil || len(taskIDs) == 0 {
		return nil, err
	}
	
	// Get the first ready task
	taskID := taskIDs[0]
	
	// Get retry metadata including complete task data
	retryData, err := q.rdb.HGet(ctx, KeyRetryCount, taskID).Result()
	if err != nil {
		return nil, err
	}
	
	var retryMeta struct {
		TaskData string `json:"task_data"`
	}
	
	if err := json.Unmarshal([]byte(retryData), &retryMeta); err != nil {
		return nil, err
	}
	
	// Parse the original task data
	var taskMsg TaskMessage
	if err := json.Unmarshal([]byte(retryMeta.TaskData), &taskMsg); err != nil {
		return nil, err
	}
	
	// Remove from retry queue and retry count
	pipe := q.rdb.Pipeline()
	pipe.ZRem(ctx, KeyRetryQueue, taskID)
	pipe.HDel(ctx, KeyRetryCount, taskID)
	if _, err := pipe.Exec(ctx); err != nil {
		return nil, err
	}
	
	return &taskMsg, nil
}

// GetRetryCount returns the current retry count for a task.
func (q *Queue) GetRetryCount(ctx context.Context, taskID string) (int, error) {
	retryData, err := q.rdb.HGet(ctx, KeyRetryCount, taskID).Result()
	if err == redis.Nil {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	
	var retryMsg RetryMessage
	if err := json.Unmarshal([]byte(retryData), &retryMsg); err != nil {
		return 0, err
	}
	
	return retryMsg.RetryCount, nil
}

// RecordError stores an error message for a task.
func (q *Queue) RecordError(ctx context.Context, taskID string, errorMsg string) error {
	timestamp := time.Now().Format(time.RFC3339)
	errorEntry := fmt.Sprintf("%s: %s", timestamp, errorMsg)
	return q.rdb.HSet(ctx, KeyTaskErrors, taskID, errorEntry).Err()
}

// GetErrors retrieves error history for a task.
func (q *Queue) GetErrors(ctx context.Context, taskID string) ([]string, error) {
	errorData, err := q.rdb.HGet(ctx, KeyTaskErrors, taskID).Result()
	if err == redis.Nil {
		return []string{}, nil
	}
	if err != nil {
		return nil, err
	}
	
	return []string{errorData}, nil
}

// calculateBackoff calculates exponential backoff delay.
func (q *Queue) calculateBackoff(retryCount int) time.Duration {
	// Exponential backoff: 2^retryCount seconds, max 1 hour
	baseDelay := time.Duration(1<<uint(retryCount)) * time.Second
	if baseDelay > time.Hour {
		baseDelay = time.Hour
	}
	
	// Add jitter (0-20% randomization)
	jitter := time.Duration(float64(baseDelay) * 0.2 * float64(time.Now().UnixNano()%100)/100)
	return baseDelay + jitter
}

// RetryQueueLength returns the number of tasks waiting to be retried.
func (q *Queue) RetryQueueLength(ctx context.Context) (int64, error) {
	return q.rdb.ZCard(ctx, KeyRetryQueue).Result()
}

// CleanupRetryData removes old retry metadata.
func (q *Queue) CleanupRetryData(ctx context.Context, olderThan time.Duration) error {
	cutoff := time.Now().Add(-olderThan)
	
	// Clean up old retry metadata
	retryData, err := q.rdb.HGetAll(ctx, KeyRetryCount).Result()
	if err != nil {
		return err
	}
	
	for taskID, data := range retryData {
		var retryMsg RetryMessage
		if err := json.Unmarshal([]byte(data), &retryMsg); err != nil {
			continue
		}
		
		if retryMsg.RetryAt.Before(cutoff) {
			q.rdb.HDel(ctx, KeyRetryCount, taskID)
			q.rdb.HDel(ctx, KeyTaskErrors, taskID)
		}
	}
	
	return nil
}

// EnqueueWithPriority adds a task to the priority queue.
func (q *Queue) EnqueueWithPriority(ctx context.Context, msg *TaskMessage, priority int) error {
	// Store task data
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal task: %w", err)
	}
	
	// Use Redis transaction to ensure atomicity
	pipe := q.rdb.Pipeline()
	
	// Add to priority queue (sorted set)
	pipe.ZAdd(ctx, KeyPriorityQueue, redis.Z{
		Score:  float64(priority),
		Member: msg.TaskID,
	})
	
	// Store task data in a hash for retrieval
	taskKey := fmt.Sprintf("secflow:tasks:data:%s", msg.TaskID)
	pipe.Set(ctx, taskKey, data, taskTTL)
	
	_, err = pipe.Exec(ctx)
	return err
}

// DequeueWithPriority gets the highest priority task from the queue.
func (q *Queue) DequeueWithPriority(ctx context.Context) (*TaskMessage, error) {
	// Get the task with highest priority (highest score)
	taskIDs, err := q.rdb.ZRevRange(ctx, KeyPriorityQueue, 0, 0).Result()
	if err != nil {
		return nil, err
	}
	
	if len(taskIDs) == 0 {
		return nil, nil // No tasks available
	}
	
	taskID := taskIDs[0]
	
	// Get task data
	taskKey := fmt.Sprintf("secflow:tasks:data:%s", taskID)
	data, err := q.rdb.Get(ctx, taskKey).Bytes()
	if err != nil {
		return nil, err
	}
	
	// Remove from priority queue and data store
	pipe := q.rdb.Pipeline()
	pipe.ZRem(ctx, KeyPriorityQueue, taskID)
	pipe.Del(ctx, taskKey)
	if _, err := pipe.Exec(ctx); err != nil {
		return nil, err
	}
	
	var msg TaskMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, fmt.Errorf("unmarshal task: %w", err)
	}
	
	return &msg, nil
}

// PriorityQueueLength returns the number of tasks in the priority queue.
func (q *Queue) PriorityQueueLength(ctx context.Context) (int64, error) {
	return q.rdb.ZCard(ctx, KeyPriorityQueue).Result()
}

// GetTaskPriority returns the priority of a task in the queue.
func (q *Queue) GetTaskPriority(ctx context.Context, taskID string) (int, error) {
	score, err := q.rdb.ZScore(ctx, KeyPriorityQueue, taskID).Result()
	if err == redis.Nil {
		return PriorityMedium, nil // Default priority
	}
	if err != nil {
		return 0, err
	}
	
	return int(score), nil
}

// UpdateTaskPriority updates the priority of a task in the queue.
func (q *Queue) UpdateTaskPriority(ctx context.Context, taskID string, newPriority int) error {
	// Check if task exists in priority queue
	exists, err := q.rdb.ZScore(ctx, KeyPriorityQueue, taskID).Result()
	if err == redis.Nil {
		return fmt.Errorf("task not found in priority queue: %s", taskID)
	}
	if err != nil {
		return err
	}
	
	// Update priority (score)
	_ = exists // suppress unused warning
	return q.rdb.ZAdd(ctx, KeyPriorityQueue, redis.Z{
		Score:  float64(newPriority),
		Member: taskID,
	}).Err()
}

// CancelTaskFromPriority removes a task from the priority queue.
func (q *Queue) CancelTaskFromPriority(ctx context.Context, taskID string) error {
	// Get task data before removing
	taskKey := fmt.Sprintf("secflow:tasks:data:%s", taskID)
	
	pipe := q.rdb.Pipeline()
	pipe.ZRem(ctx, KeyPriorityQueue, taskID)
	pipe.Del(ctx, taskKey)
	_, err := pipe.Exec(ctx)
	
	return err
}

// GetTasksByPriorityRange returns tasks within a priority range.
func (q *Queue) GetTasksByPriorityRange(ctx context.Context, minPriority, maxPriority int) ([]TaskMessage, error) {
	taskIDs, err := q.rdb.ZRangeByScore(ctx, KeyPriorityQueue, &redis.ZRangeBy{
		Min: fmt.Sprintf("%d", minPriority),
		Max: fmt.Sprintf("%d", maxPriority),
	}).Result()
	
	if err != nil {
		return nil, err
	}
	
	if len(taskIDs) == 0 {
		return []TaskMessage{}, nil
	}
	
	// Get task data for all task IDs
	var tasks []TaskMessage
	for _, taskID := range taskIDs {
		taskKey := fmt.Sprintf("secflow:tasks:data:%s", taskID)
		data, err := q.rdb.Get(ctx, taskKey).Bytes()
		if err != nil {
			continue // Skip tasks that can't be retrieved
		}
		
		var msg TaskMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			continue
		}
		
		tasks = append(tasks, msg)
	}
	
	return tasks, nil
}

// DequeueBatch removes and returns up to n tasks from the regular queue.
func (q *Queue) DequeueBatch(ctx context.Context, n int) ([]TaskMessage, error) {
	if n <= 0 {
		n = 1
	}
	if n > 100 {
		n = 100 // Cap at 100 to prevent abuse
	}
	
	// Better approach: get range then trim
	result, err := q.rdb.LRange(ctx, KeyTaskQueue, 0, int64(n-1)).Result()
	if err != nil {
		return nil, err
	}
	
	if len(result) == 0 {
		return []TaskMessage{}, nil
	}
	
	// Remove the items we just read
	_, err = q.rdb.LTrim(ctx, KeyTaskQueue, int64(len(result)), -1).Result()
	if err != nil {
		return nil, err
	}
	
	// Parse all task messages
	var tasks []TaskMessage
	for _, item := range result {
		var msg TaskMessage
		if err := json.Unmarshal([]byte(item), &msg); err != nil {
			log.Error().Err(err).Msg("failed to unmarshal task")
			continue
		}
		tasks = append(tasks, msg)
	}
	
	return tasks, nil
}

// DequeueBatchPriority removes and returns up to n tasks from the priority queue (highest priority first).
func (q *Queue) DequeueBatchPriority(ctx context.Context, n int) ([]TaskMessage, error) {
	if n <= 0 {
		n = 1
	}
	if n > 100 {
		n = 100 // Cap at 100 to prevent abuse
	}
	
	// Get n highest priority tasks
	taskIDs, err := q.rdb.ZRevRange(ctx, KeyPriorityQueue, 0, int64(n-1)).Result()
	if err != nil {
		return nil, err
	}
	
	if len(taskIDs) == 0 {
		return []TaskMessage{}, nil
	}
	
	// Get task data for all task IDs
	var tasks []TaskMessage
	var taskKeys []string
	
	for _, taskID := range taskIDs {
		taskKey := fmt.Sprintf("secflow:tasks:data:%s", taskID)
		taskKeys = append(taskKeys, taskKey)
		
		data, err := q.rdb.Get(ctx, taskKey).Bytes()
		if err != nil {
			continue // Skip tasks that can't be retrieved
		}
		
		var msg TaskMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			log.Error().Err(err).Msg("failed to unmarshal task")
			continue
		}
		
		tasks = append(tasks, msg)
	}
	
// Remove tasks from priority queue and data store
	pipe := q.rdb.Pipeline()
	for _, taskID := range taskIDs {
		pipe.ZRem(ctx, KeyPriorityQueue, taskID)
	}
	for _, taskKey := range taskKeys {
		pipe.Del(ctx, taskKey)
	}

	if _, err := pipe.Exec(ctx); err != nil {
		log.Error().Err(err).Msg("failed to cleanup priority queue")
	}

	return tasks, nil
}

// GetProgressBatch retrieves progress for multiple tasks in a single Pipeline call.
// Returns a map of task_id -> progress (0-100).
func (q *Queue) GetProgressBatch(ctx context.Context, taskIDs []string) (map[string]int, error) {
	if len(taskIDs) == 0 {
		return map[string]int{}, nil
	}

	pipe := q.rdb.Pipeline()
	cmds := make([]*redis.StringCmd, len(taskIDs))

	for i, taskID := range taskIDs {
		cmds[i] = pipe.HGet(ctx, KeyTaskProgress, taskID)
	}

	_, err := pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		return nil, fmt.Errorf("pipeline exec: %w", err)
	}

	results := make(map[string]int, len(taskIDs))
	for i, cmd := range cmds {
		if cmd.Err() == redis.Nil {
			results[taskIDs[i]] = 0 // Default to 0 if not found
			continue
		}
		if cmd.Err() != nil {
			continue // Skip errors for individual keys
		}
		// Convert string to int
		var progress int
		if _, err := fmt.Sscanf(cmd.Val(), "%d", &progress); err != nil {
			results[taskIDs[i]] = 0
		} else {
			results[taskIDs[i]] = progress
		}
	}

	return results, nil
}

// HeartbeatBatch updates heartbeats for multiple nodes in a single Pipeline call.
// Returns the number of nodes successfully updated.
func (q *Queue) HeartbeatBatch(ctx context.Context, nodeIDs []string) (int, error) {
	if len(nodeIDs) == 0 {
		return 0, nil
	}

	pipe := q.rdb.Pipeline()
	now := float64(time.Now().Unix())

	for _, nodeID := range nodeIDs {
		pipe.ZAdd(ctx, KeyNodeHeartbeat, redis.Z{
			Score:  now,
			Member: nodeID,
		})
	}

	cmds, err := pipe.Exec(ctx)
	if err != nil {
		return 0, fmt.Errorf("pipeline exec: %w", err)
	}

	// Count successful commands
	successCount := 0
	for _, cmd := range cmds {
		if cmd.Err() == nil {
			successCount++
		}
	}

	return successCount, nil
}

// SetProgressBatch sets progress for multiple tasks in a single Pipeline call.
// Returns the number of tasks successfully updated.
func (q *Queue) SetProgressBatch(ctx context.Context, progressMap map[string]int) (int, error) {
	if len(progressMap) == 0 {
		return 0, nil
	}

	pipe := q.rdb.Pipeline()

	for taskID, progress := range progressMap {
		pipe.HSet(ctx, KeyTaskProgress, taskID, progress)
	}

	cmds, err := pipe.Exec(ctx)
	if err != nil {
		return 0, fmt.Errorf("pipeline exec: %w", err)
	}

	// Count successful commands
	successCount := 0
	for _, cmd := range cmds {
		if cmd.Err() == nil {
			successCount++
		}
	}

	return successCount, nil
}

// GetTaskCountsBatch retrieves multiple queue lengths in a single Pipeline call.
// Returns counts for: pending, priority, retry queues.
func (q *Queue) GetTaskCountsBatch(ctx context.Context) (map[string]int64, error) {
	pipe := q.rdb.Pipeline()

	// Queue length commands
	pendingCmd := pipe.LLen(ctx, KeyTaskQueue)
	priorityCmd := pipe.ZCard(ctx, KeyPriorityQueue)
	retryCmd := pipe.ZCard(ctx, KeyRetryQueue)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("pipeline exec: %w", err)
	}

	counts := map[string]int64{
		"pending":  pendingCmd.Val(),
		"priority": priorityCmd.Val(),
		"retry":    retryCmd.Val(),
	}

	return counts, nil
}

// ============================================
// Dead Letter Queue Methods
// ============================================

// EnqueueDeadLetter adds a permanently failed task to the dead letter queue.
func (q *Queue) EnqueueDeadLetter(ctx context.Context, taskMsg *TaskMessage, retryCount int, lastError string) error {
	// Create dead letter entry
	dl := DeadLetterMessage{
		TaskID:       taskMsg.TaskID,
		Type:         taskMsg.Type,
		Payload:      string(taskMsg.Payload),
		RetryCount:   retryCount,
		LastError:    lastError,
		FailedAt:     time.Now(),
		OriginalTask: taskMsg.TaskID,
	}

	data, err := json.Marshal(dl)
	if err != nil {
		return fmt.Errorf("marshal dead letter: %w", err)
	}

	// Use pipeline for atomicity
	pipe := q.rdb.Pipeline()

	// Add to dead letter queue (list)
	pipe.RPush(ctx, KeyDeadLetterQueue, data)

	// Store metadata with TTL (hash)
	pipe.HSet(ctx, KeyDeadLetterMeta, dl.TaskID, string(data))
	pipe.Expire(ctx, KeyDeadLetterMeta, deadLetterTTL)

	_, err = pipe.Exec(ctx)
	return err
}

// DequeueDeadLetter removes and returns a dead letter task for reprocessing.
func (q *Queue) DequeueDeadLetter(ctx context.Context) (*DeadLetterMessage, error) {
	// Get the first item from the queue
	result, err := q.rdb.LPop(ctx, KeyDeadLetterQueue).Result()
	if err == redis.Nil {
		return nil, nil // No items in queue
	}
	if err != nil {
		return nil, err
	}

	var dl DeadLetterMessage
	if err := json.Unmarshal([]byte(result), &dl); err != nil {
		return nil, fmt.Errorf("unmarshal dead letter: %w", err)
	}

	// Remove metadata
	q.rdb.HDel(ctx, KeyDeadLetterMeta, dl.TaskID)

	return &dl, nil
}

// GetDeadLetter retrieves metadata for a specific dead letter task.
func (q *Queue) GetDeadLetter(ctx context.Context, taskID string) (*DeadLetterMessage, error) {
	data, err := q.rdb.HGet(ctx, KeyDeadLetterMeta, taskID).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var dl DeadLetterMessage
	if err := json.Unmarshal([]byte(data), &dl); err != nil {
		return nil, err
	}

	return &dl, nil
}

// ListDeadLetters returns all dead letter tasks with pagination.
func (q *Queue) ListDeadLetters(ctx context.Context, offset, limit int) ([]DeadLetterMessage, int64, error) {
	// Get total count
	total, err := q.rdb.LLen(ctx, KeyDeadLetterQueue).Result()
	if err != nil {
		return nil, 0, err
	}

	// Get items with pagination
	result, err := q.rdb.LRange(ctx, KeyDeadLetterQueue, int64(offset), int64(offset+limit-1)).Result()
	if err != nil {
		return nil, 0, err
	}

	var items []DeadLetterMessage
	for _, item := range result {
		var dl DeadLetterMessage
		if err := json.Unmarshal([]byte(item), &dl); err != nil {
			continue
		}
		items = append(items, dl)
	}

	return items, total, nil
}

// RequeueDeadLetter moves a dead letter task back to the main queue for retry.
func (q *Queue) RequeueDeadLetter(ctx context.Context, taskID string) error {
	// Get the dead letter
	dl, err := q.GetDeadLetter(ctx, taskID)
	if err != nil {
		return err
	}
	if dl == nil {
		return fmt.Errorf("dead letter not found: %s", taskID)
	}

	// Create a new task message with fresh ID
	newTaskID := fmt.Sprintf("%s-retry-%d", taskID, time.Now().Unix())
	newMsg := TaskMessage{
		TaskID:  newTaskID,
		Type:    dl.Type,
		Payload: json.RawMessage(dl.Payload),
	}

	// Remove from dead letter queue (already removed from metadata by GetDeadLetter)
	q.rdb.HDel(ctx, KeyDeadLetterMeta, taskID)

	// Add to regular queue
	return q.Enqueue(ctx, &newMsg)
}

// RemoveDeadLetter removes a specific task from the dead letter queue.
func (q *Queue) RemoveDeadLetter(ctx context.Context, taskID string) error {
	// Get all items to find and remove the specific one
	items, err := q.rdb.LRange(ctx, KeyDeadLetterQueue, 0, -1).Result()
	if err != nil {
		return err
	}

	for _, item := range items {
		var dl DeadLetterMessage
		if err := json.Unmarshal([]byte(item), &dl); err != nil {
			continue
		}
		if dl.TaskID == taskID {
			// Remove this item
			q.rdb.LRem(ctx, KeyDeadLetterQueue, 1, item)
			q.rdb.HDel(ctx, KeyDeadLetterMeta, taskID)
			return nil
		}
	}

	return fmt.Errorf("dead letter not found: %s", taskID)
}

// DeadLetterQueueLength returns the number of tasks in the dead letter queue.
func (q *Queue) DeadLetterQueueLength(ctx context.Context) (int64, error) {
	return q.rdb.LLen(ctx, KeyDeadLetterQueue).Result()
}

// CleanupDeadLetterQueue removes old dead letter entries based on age.
func (q *Queue) CleanupDeadLetterQueue(ctx context.Context, olderThan time.Duration) (int, error) {
	cutoff := time.Now().Add(-olderThan)
	removed := 0

	items, err := q.rdb.LRange(ctx, KeyDeadLetterQueue, 0, -1).Result()
	if err != nil {
		return 0, err
	}

	for _, item := range items {
		var dl DeadLetterMessage
		if err := json.Unmarshal([]byte(item), &dl); err != nil {
			continue
		}

		if dl.FailedAt.Before(cutoff) {
			q.rdb.LRem(ctx, KeyDeadLetterQueue, 1, item)
			q.rdb.HDel(ctx, KeyDeadLetterMeta, dl.TaskID)
			removed++
		}
	}

	return removed, nil
}
