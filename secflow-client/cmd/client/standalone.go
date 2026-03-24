// Package main provides standalone mode functionality for SecFlow client.
//
// In standalone mode, the client runs independently without connecting to
// a SecFlow server. It executes scheduled crawling tasks based on local
// configuration and stores all data in the local SQLite database.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/secflow/client/internal/config"
	"github.com/secflow/client/internal/db"
	"github.com/secflow/client/internal/engine"
	"github.com/secflow/client/internal/task"
	grabpkg "github.com/secflow/client/pkg/vulngrabber"
)

// StandaloneRunner manages scheduled crawling tasks in standalone mode.
type StandaloneRunner struct {
	cfg     *config.Config
	eng     *engine.Engine
	db      *db.DB
	log     *zap.Logger
	sources []string
}

// NewStandaloneRunner creates a new standalone task runner.
func NewStandaloneRunner(cfg *config.Config, eng *engine.Engine, store *db.DB, log *zap.Logger) *StandaloneRunner {
	allSources := grabpkg.Available()
	sources := cfg.GetEnabledSources(allSources)

	// If scheduler.sources is configured, use those instead
	if len(cfg.Scheduler.Sources) > 0 {
		sources = cfg.Scheduler.Sources
	}

	return &StandaloneRunner{
		cfg:     cfg,
		eng:     eng,
		db:      store,
		log:     log,
		sources: sources,
	}
}

// Start begins the standalone scheduling loop.
func (r *StandaloneRunner) Start(ctx context.Context) error {
	if !r.cfg.Scheduler.Enabled {
		r.log.Info("scheduler is disabled, running once and exiting")
		return r.runOnce(ctx)
	}

	interval := r.cfg.Scheduler.Interval
	if interval <= 0 {
		interval = 1 * time.Hour // default to 1 hour
	}

	r.log.Info("starting standalone scheduler",
		zap.Duration("interval", interval),
		zap.Strings("sources", r.sources),
	)

	// Run immediately on start
	if err := r.runOnce(ctx); err != nil {
		r.log.Error("initial crawl failed", zap.Error(err))
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := r.runOnce(ctx); err != nil {
				r.log.Error("scheduled crawl failed", zap.Error(err))
			}
		}
	}
}

// runOnce executes a single crawl cycle for all configured sources.
func (r *StandaloneRunner) runOnce(ctx context.Context) error {
	r.log.Info("starting scheduled crawl", zap.Strings("sources", r.sources))

	if len(r.sources) == 0 {
		r.log.Warn("no sources configured, skipping crawl")
		return nil
	}

	// Create task ID for this run
	taskID := fmt.Sprintf("standalone-%d", time.Now().Unix())

	// Build payload
	payload := task.VulnCrawlPayload{
		Sources:      r.sources,
		PageLimit:    r.cfg.Task.DefaultPageLimit,
		EnableGithub: r.cfg.Task.EnableGithubSearch,
		Proxy:        r.cfg.Proxy,
	}

	// Record task start
	rawPayload, _ := json.Marshal(payload)
	_ = r.db.UpsertTask(&db.TaskRecord{
		ID:         taskID,
		TaskID:     taskID,
		Type:       "vuln_crawl",
		Status:     "running",
		Payload:    string(rawPayload),
		ReceivedAt: time.Now(),
		UpdatedAt:  time.Now(),
	})

	// Progress callback
	progressFn := func(pct int) {
		r.log.Debug("crawl progress", zap.Int("percent", pct))
		_ = r.db.UpdateTaskStatus(taskID, "running", pct, "")
	}

	// Execute crawl
	startTime := time.Now()
	vulns, err := r.eng.RunVulnCrawl(ctx, payload, progressFn)
	if err != nil {
		r.log.Error("crawl failed", zap.Error(err))
		_ = r.db.UpdateTaskStatus(taskID, "failed", 0, err.Error())
		return fmt.Errorf("crawl execution: %w", err)
	}

	// Store results locally
	for _, v := range vulns {
		if m, ok := v.(map[string]any); ok {
			key, _ := m["key"].(string)
			title, _ := m["title"].(string)
			sev, _ := m["severity"].(string)
			cve, _ := m["cve"].(string)
			src, _ := m["source"].(string)
			_ = r.db.InsertVulnCache(key, title, sev, cve, src, taskID)
		}
	}

	// Mark task complete
	_ = r.db.UpdateTaskStatus(taskID, "done", 100, "")

	r.log.Info("scheduled crawl completed",
		zap.Duration("duration", time.Since(startTime)),
		zap.Int("vulns_found", len(vulns)),
	)

	return nil
}
