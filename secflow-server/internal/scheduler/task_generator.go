// Package scheduler implements automatic task generation for periodic crawling.
package scheduler

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/secflow/server/internal/model"
	"github.com/secflow/server/internal/queue"
	"github.com/secflow/server/internal/repository"
)

// TaskGenerator automatically creates crawl tasks on a schedule.
type TaskGenerator struct {
	taskRepo    *repository.TaskRepo
	queue       *queue.Queue
	nodeRepo    *repository.NodeRepo
	vulnSources []string
	articleSources []string
	stopCh      chan struct{}
}

// NewTaskGenerator creates a new automatic task generator.
func NewTaskGenerator(
	tr *repository.TaskRepo,
	q *queue.Queue,
	nr *repository.NodeRepo,
	vulnSources []string,
	articleSources []string,
) *TaskGenerator {
	return &TaskGenerator{
		taskRepo:       tr,
		queue:          q,
		nodeRepo:       nr,
		vulnSources:    vulnSources,
		articleSources: articleSources,
		stopCh:         make(chan struct{}),
	}
}

// Start begins the task generator loop.
func (tg *TaskGenerator) Start(ctx context.Context) {
	go tg.run(ctx)
}

// Stop stops the task generator.
func (tg *TaskGenerator) Stop() {
	close(tg.stopCh)
}

func (tg *TaskGenerator) run(ctx context.Context) {
	// Task generation is now controlled via API.
	// The generator waits for explicit trigger or stop signal.
	<-ctx.Done()
}

// generateVulnCrawlTask creates a new vulnerability crawl task.
func (tg *TaskGenerator) generateVulnCrawlTask(ctx context.Context) {
	sources := tg.vulnSources
	if len(sources) == 0 {
		// Default: use all available sources from connected nodes
		sources = tg.getSourcesFromNodes(ctx)
	}
	if len(sources) == 0 {
		log.Warn().Msg("task generator: no vuln sources available")
		return
	}

	payload := model.VulnCrawlPayload{
		Sources:      sources,
		PageLimit:    1,
		EnableGithub: false,
	}
	rawPayload, _ := json.Marshal(payload)

	taskID := uuid.New().String()
	t := &model.Task{
		TaskID:         taskID,
		Type:           model.TaskTypeVulnCrawl,
		Status:         model.TaskPending,
		Payload:        rawPayload,
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
		TimeoutSeconds: 1800, // 30 minutes timeout for vuln crawl
		MaxRetries:     3,
	}

	if err := tg.taskRepo.Create(ctx, t); err != nil {
		log.Error().Err(err).Msg("task generator: failed to create vuln crawl task")
		return
	}

	msg := &queue.TaskMessage{
		TaskID:  taskID,
		Type:    string(model.TaskTypeVulnCrawl),
		Payload: rawPayload,
	}
	if err := tg.queue.Enqueue(ctx, msg); err != nil {
		log.Error().Err(err).Str("task_id", taskID).Msg("task generator: failed to enqueue vuln crawl task")
		return
	}

	log.Info().
		Str("task_id", taskID).
		Strs("sources", sources).
		Msg("task generator: created vuln crawl task")
}

// generateArticleCrawlTask creates a new article crawl task.
func (tg *TaskGenerator) generateArticleCrawlTask(ctx context.Context) {
	sources := tg.articleSources
	if len(sources) == 0 {
		// Default article sources from articlegrabber
		sources = []string{"qianxin-weekly", "venustech", "freebuf", "securityweek", "hackernews", "xianzhi"}
	}

	payload := map[string]interface{}{
		"type":    "article_crawl",
		"sources": sources,
		"limit":   10,
	}
	rawPayload, _ := json.Marshal(payload)

	taskID := uuid.New().String()
	t := &model.Task{
		TaskID:         taskID,
		Type:           model.TaskTypeArticleCrawl,
		Status:         model.TaskPending,
		Payload:        rawPayload,
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
		TimeoutSeconds: 1800, // 30 minutes timeout for article crawl
		MaxRetries:     3,
	}

	if err := tg.taskRepo.Create(ctx, t); err != nil {
		log.Error().Err(err).Msg("task generator: failed to create article crawl task")
		return
	}

	msg := &queue.TaskMessage{
		TaskID:  taskID,
		Type:    string(model.TaskTypeArticleCrawl),
		Payload: rawPayload,
	}
	if err := tg.queue.Enqueue(ctx, msg); err != nil {
		log.Error().Err(err).Str("task_id", taskID).Msg("task generator: failed to enqueue article crawl task")
		return
	}

	log.Info().
		Str("task_id", taskID).
		Strs("sources", sources).
		Msg("task generator: created article crawl task")
}

// getSourcesFromNodes returns the union of all sources from connected nodes.
func (tg *TaskGenerator) getSourcesFromNodes(ctx context.Context) []string {
	nodes, err := tg.nodeRepo.List(ctx)
	if err != nil {
		return nil
	}

	sourceMap := make(map[string]bool)
	for _, node := range nodes {
		for _, src := range node.Sources {
			sourceMap[src] = true
		}
	}

	sources := make([]string, 0, len(sourceMap))
	for src := range sourceMap {
		sources = append(sources, src)
	}
	return sources
}

// CreateVulnCrawlTask creates a one-time vulnerability crawl task (for API use).
func (tg *TaskGenerator) CreateVulnCrawlTask(ctx context.Context, sources []string, pageLimit int) (string, error) {
	if len(sources) == 0 {
		sources = tg.vulnSources
	}

	payload := model.VulnCrawlPayload{
		Sources:      sources,
		PageLimit:    pageLimit,
		EnableGithub: false,
	}
	rawPayload, _ := json.Marshal(payload)

	taskID := uuid.New().String()
	t := &model.Task{
		TaskID:    taskID,
		Type:      model.TaskTypeVulnCrawl,
		Status:    model.TaskPending,
		Payload:   rawPayload,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	if err := tg.taskRepo.Create(ctx, t); err != nil {
		return "", err
	}

	msg := &queue.TaskMessage{
		TaskID:  taskID,
		Type:    string(model.TaskTypeVulnCrawl),
		Payload: rawPayload,
	}
	if err := tg.queue.Enqueue(ctx, msg); err != nil {
		return "", err
	}

	return taskID, nil
}

// CreateArticleCrawlTask creates a one-time article crawl task (for API use).
func (tg *TaskGenerator) CreateArticleCrawlTask(ctx context.Context, sources []string, limit int) (string, error) {
	if len(sources) == 0 {
		sources = tg.articleSources
	}

	payload := map[string]interface{}{
		"type":    "article_crawl",
		"sources": sources,
		"limit":   limit,
	}
	rawPayload, _ := json.Marshal(payload)

	taskID := uuid.New().String()
	t := &model.Task{
		TaskID:    taskID,
		Type:      model.TaskTypeArticleCrawl,
		Status:    model.TaskPending,
		Payload:   rawPayload,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	if err := tg.taskRepo.Create(ctx, t); err != nil {
		return "", err
	}

	msg := &queue.TaskMessage{
		TaskID:  taskID,
		Type:    string(model.TaskTypeArticleCrawl),
		Payload: rawPayload,
	}
	if err := tg.queue.Enqueue(ctx, msg); err != nil {
		return "", err
	}

	return taskID, nil
}
