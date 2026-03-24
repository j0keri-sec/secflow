// Package task dispatches incoming task assignments to the appropriate executor.
// Supports "vuln_crawl" and "article_crawl" tasks.
package task

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/secflow/client/internal/db"
	iws "github.com/secflow/client/internal/ws"
)

// ArticleCrawlPayload is the JSON payload for an article_crawl task.
type ArticleCrawlPayload struct {
	Sources []string `json:"sources"`
	Limit   int      `json:"limit"`
}

// VulnCrawlPayload is the JSON payload for a vuln_crawl task.
type VulnCrawlPayload struct {
	Sources      []string `json:"sources"`
	PageLimit    int      `json:"page_limit"`
	EnableGithub bool     `json:"enable_github"`
	Proxy        string   `json:"proxy"`
}

// Executor is the interface that must be implemented by crawl backends.
type Executor interface {
	// RunVulnCrawl executes a vulnerability crawl for the given sources.
	RunVulnCrawl(ctx context.Context, payload VulnCrawlPayload, progressFn func(int)) ([]any, error)
	// RunArticleCrawl executes an article crawl for the given sources.
	RunArticleCrawl(ctx context.Context, payload ArticleCrawlPayload, progressFn func(int)) ([]any, error)
}

// Dispatcher manages in-flight tasks and routes assignments to executors.
type Dispatcher struct {
	exec   Executor
	ws     *iws.Client
	db     *db.DB
	log    *zap.Logger
	nodeID string

	mu      sync.Mutex
	cancels map[string]context.CancelFunc
}

// New creates a Dispatcher.
func New(exec Executor, ws *iws.Client, db *db.DB, nodeID string, log *zap.Logger) *Dispatcher {
	return &Dispatcher{
		exec:    exec,
		ws:      ws,
		db:      db,
		log:     log,
		nodeID:  nodeID,
		cancels: make(map[string]context.CancelFunc),
	}
}

// OnTaskAssign is called by the WS client when a task_assign frame arrives.
func (d *Dispatcher) OnTaskAssign(msg *iws.TaskAssignMsg) {
	// Determine task type from payload structure
	var base struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(msg.Payload, &base); err != nil {
		// Try to determine from payload content
		if isArticlePayload(msg.Payload) {
			d.handleArticleCrawl(msg)
		} else {
			d.handleVulnCrawl(msg)
		}
		return
	}

	switch base.Type {
	case "article_crawl":
		d.handleArticleCrawl(msg)
	case "vuln_crawl", "":
		d.handleVulnCrawl(msg)
	default:
		d.handleVulnCrawl(msg)
	}
}

// isArticlePayload checks if payload is for article crawl based on fields
func isArticlePayload(payload json.RawMessage) bool {
	var check struct {
		Sources []string `json:"sources"`
		Limit   int      `json:"limit"`
	}
	json.Unmarshal(payload, &check)
	// If it has "limit" field, it's likely article crawl
	return check.Limit > 0 && len(check.Sources) > 0
}

// handleVulnCrawl processes vulnerability crawl tasks
func (d *Dispatcher) handleVulnCrawl(msg *iws.TaskAssignMsg) {
	log := d.log.With(zap.String("task_id", msg.TaskID), zap.String("type", "vuln_crawl"))

	var payload VulnCrawlPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		log.Error("invalid task payload", zap.Error(err))
		_ = d.ws.SendError(msg.TaskID, fmt.Sprintf("invalid payload: %v", err))
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	d.mu.Lock()
	d.cancels[msg.TaskID] = cancel
	d.mu.Unlock()

	rawPayload, _ := json.Marshal(payload)
	_ = d.db.UpsertTask(&db.TaskRecord{
		ID:         msg.TaskID,
		TaskID:     msg.TaskID,
		Type:       "vuln_crawl",
		Status:     "running",
		Payload:    string(rawPayload),
		ReceivedAt: time.Now(),
		UpdatedAt:  time.Now(),
	})

	go func() {
		defer func() {
			d.mu.Lock()
			delete(d.cancels, msg.TaskID)
			d.mu.Unlock()
			cancel()
		}()

		log.Info("starting vuln_crawl task", zap.Strings("sources", payload.Sources))

		progressFn := func(pct int) {
			if err := d.ws.SendProgress(msg.TaskID, pct, ""); err != nil {
				log.Warn("send progress failed", zap.Error(err))
			}
			_ = d.db.UpdateTaskStatus(msg.TaskID, "running", pct, "")
		}

		vulns, err := d.exec.RunVulnCrawl(ctx, payload, progressFn)
		if err != nil {
			if ctx.Err() != nil {
				log.Info("task cancelled")
				_ = d.db.UpdateTaskStatus(msg.TaskID, "failed", 0, "cancelled")
				return
			}
			log.Error("task failed", zap.Error(err))
			_ = d.ws.SendError(msg.TaskID, err.Error())
			_ = d.db.UpdateTaskStatus(msg.TaskID, "failed", 0, err.Error())
			return
		}

		for _, v := range vulns {
			if m, ok := v.(map[string]any); ok {
				key, _ := m["key"].(string)
				title, _ := m["title"].(string)
				sev, _ := m["severity"].(string)
				cve, _ := m["cve"].(string)
				src, _ := m["source"].(string)
				_ = d.db.InsertVulnCache(key, title, sev, cve, src, msg.TaskID)
			}
		}

		if err := d.ws.SendResult(msg.TaskID, vulns); err != nil {
			log.Error("upload result failed", zap.Error(err))
			_ = d.db.UpdateTaskStatus(msg.TaskID, "failed", 100, err.Error())
			return
		}

		_ = d.db.UpdateTaskStatus(msg.TaskID, "done", 100, "")
		log.Info("task done", zap.Int("vulns", len(vulns)))
	}()
}

// handleArticleCrawl processes article crawl tasks
func (d *Dispatcher) handleArticleCrawl(msg *iws.TaskAssignMsg) {
	log := d.log.With(zap.String("task_id", msg.TaskID), zap.String("type", "article_crawl"))

	var payload ArticleCrawlPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		log.Error("invalid task payload", zap.Error(err))
		_ = d.ws.SendError(msg.TaskID, fmt.Sprintf("invalid payload: %v", err))
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	d.mu.Lock()
	d.cancels[msg.TaskID] = cancel
	d.mu.Unlock()

	rawPayload, _ := json.Marshal(payload)
	_ = d.db.UpsertTask(&db.TaskRecord{
		ID:         msg.TaskID,
		TaskID:     msg.TaskID,
		Type:       "article_crawl",
		Status:     "running",
		Payload:    string(rawPayload),
		ReceivedAt: time.Now(),
		UpdatedAt:  time.Now(),
	})

	go func() {
		defer func() {
			d.mu.Lock()
			delete(d.cancels, msg.TaskID)
			d.mu.Unlock()
			cancel()
		}()

		log.Info("starting article_crawl task", zap.Strings("sources", payload.Sources))

		progressFn := func(pct int) {
			if err := d.ws.SendProgress(msg.TaskID, pct, ""); err != nil {
				log.Warn("send progress failed", zap.Error(err))
			}
			_ = d.db.UpdateTaskStatus(msg.TaskID, "running", pct, "")
		}

		articles, err := d.exec.RunArticleCrawl(ctx, payload, progressFn)
		if err != nil {
			if ctx.Err() != nil {
				log.Info("task cancelled")
				_ = d.db.UpdateTaskStatus(msg.TaskID, "failed", 0, "cancelled")
				return
			}
			log.Error("task failed", zap.Error(err))
			_ = d.ws.SendError(msg.TaskID, err.Error())
			_ = d.db.UpdateTaskStatus(msg.TaskID, "failed", 0, err.Error())
			return
		}

		if err := d.ws.SendResult(msg.TaskID, articles); err != nil {
			log.Error("upload result failed", zap.Error(err))
			_ = d.db.UpdateTaskStatus(msg.TaskID, "failed", 100, err.Error())
			return
		}

		_ = d.db.UpdateTaskStatus(msg.TaskID, "done", 100, "")
		log.Info("task done", zap.Int("articles", len(articles)))
	}()
}

// OnTaskCancel is called when the server requests cancellation of a task.
func (d *Dispatcher) OnTaskCancel(taskID string) {
	d.mu.Lock()
	cancel, ok := d.cancels[taskID]
	d.mu.Unlock()
	if ok {
		cancel()
		d.log.Info("task cancelled by server", zap.String("task_id", taskID))
	}
}
