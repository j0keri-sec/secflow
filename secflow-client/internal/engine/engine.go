// Package engine wraps the grabber registry as a task Executor.
// It bridges the secflow-client task package with the grabber
// infrastructure to run vuln-crawl and article-crawl jobs on demand.
package engine

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/secflow/client/internal/task"
	"github.com/secflow/client/pkg/articlegrabber"
	grabpkg "github.com/secflow/client/pkg/vulngrabber"
)

// ProgressFn reports crawl progress with detailed status.
type ProgressFn func(percent int, sourceName string, details string)

// Engine implements task.Executor using the embedded grabber registry.
type Engine struct {
	proxy string
	log   *zap.Logger
}

// New creates an Engine with an optional proxy URL.
func New(proxy string, log *zap.Logger) *Engine {
	return &Engine{proxy: proxy, log: log}
}

// RunVulnCrawl satisfies the task.Executor interface.
// It invokes every requested grabber sequentially, reporting progress as it
// completes each source.
func (e *Engine) RunVulnCrawl(
	ctx context.Context,
	payload task.VulnCrawlPayload,
	progressFn func(int),
) ([]any, error) {
	return e.RunVulnCrawlWithProgress(ctx, payload, func(percent int, source, details string) {
		progressFn(percent)
	})
}

// RunVulnCrawlWithProgress is an enhanced version that provides detailed progress.
func (e *Engine) RunVulnCrawlWithProgress(
	ctx context.Context,
	payload task.VulnCrawlPayload,
	progressFn ProgressFn,
) ([]any, error) {
	sources := payload.Sources
	if len(sources) == 0 {
		sources = grabpkg.Available()
	}

	pageLimit := payload.PageLimit
	if pageLimit <= 0 {
		pageLimit = 1
	}

	var results []any
	var mu sync.Mutex
	seenKeys := make(map[string]struct{}) // Deduplication by key+CVE

	total := len(sources)
	successCount := 0
	failCount := 0
	totalVulns := 0

	for i, name := range sources {
		if ctx.Err() != nil {
			return results, ctx.Err()
		}

		g, err := grabpkg.ByName(name)
		if err != nil {
			e.log.Warn("unknown grabber, skipping", zap.String("source", name), zap.Error(err))
			progressFn((i+1)*100/total, name, "unknown grabber")
			failCount++
			continue
		}

		provider := g.ProviderInfo()
		displayName := name
		if provider != nil {
			displayName = provider.DisplayName
		}

		e.log.Info("running grabber",
			zap.String("source", name),
			zap.String("display_name", displayName),
			zap.Int("page_limit", pageLimit))

		progressFn((i)*100/total, name, "fetching...")

		vulns, err := g.GetUpdate(ctx, pageLimit)
		if err != nil {
			e.log.Warn("grabber error",
				zap.String("source", name),
				zap.String("display_name", displayName),
				zap.Error(err))
			progressFn((i+1)*100/total, name, "error: "+err.Error())
			failCount++
			continue
		}

		// Process and deduplicate results
		newCount := 0
		mu.Lock()
		for _, v := range vulns {
			// Generate deduplication key: prefer CVE, fall back to UniqueKey
			dedupKey := v.UniqueKey
			if v.CVE != "" {
				dedupKey = "cve:" + v.CVE
			}
			if v.UniqueKey != "" {
				dedupKey = v.UniqueKey
				if v.CVE != "" {
					dedupKey = "cve:" + v.CVE + "|" + v.UniqueKey
				}
			}

			// Skip duplicates
			if _, exists := seenKeys[dedupKey]; exists {
				continue
			}
			seenKeys[dedupKey] = struct{}{}

			// Convert VulnInfo to VulnRecord format for server compatibility
			record := vulnToServerFormat(v, name)
			results = append(results, record)
			newCount++
			totalVulns++
		}
		mu.Unlock()

		e.log.Info("grabber completed",
			zap.String("source", name),
			zap.String("display_name", displayName),
			zap.Int("vulns_found", len(vulns)),
			zap.Int("new_vulns", newCount))

		progressFn((i+1)*100/total, name, "✓ "+formatCount(newCount))
		successCount++
	}

	e.log.Info("vuln crawl completed",
		zap.Int("total_sources", total),
		zap.Int("success_count", successCount),
		zap.Int("fail_count", failCount),
		zap.Int("total_vulns", totalVulns))

	return results, nil
}

// formatCount formats a count number for display.
func formatCount(n int) string {
	switch {
	case n >= 1000:
		return "1k+"
	case n >= 100:
		return "99+"
	default:
		return strings.TrimLeft(fmt.Sprintf("%2d", n), " ")
	}
}

// vulnToServerFormat converts a grabber VulnInfo to server-compatible VulnRecord format.
func vulnToServerFormat(v *grabpkg.VulnInfo, sourceName string) *serverVulnRecord {
	// Use From field for both from and url since VulnInfo only has From
	fromURL := v.From
	if fromURL == "" && v.UniqueKey != "" {
		fromURL = v.UniqueKey
	}

	record := &serverVulnRecord{
		Key:           v.UniqueKey,
		Title:         v.Title,
		Description:   v.Description,
		Severity:      string(v.Severity),
		CVE:           v.CVE,
		Disclosure:    v.Disclosure,
		Solutions:     v.Solutions,
		References:    v.References,
		Tags:          v.Tags,
		GithubSearch:  v.GithubSearch,
		From:          v.From,
		URL:           fromURL,
		Source:        sourceName,
		Pushed:        false,
	}
	return record
}

// serverVulnRecord is a simplified version of the server's VulnRecord model for compatibility.
type serverVulnRecord struct {
	Key           string   `json:"key"`
	Title         string   `json:"title"`
	Description   string   `json:"description"`
	Severity      string   `json:"severity"`
	CVE           string   `json:"cve,omitempty"`
	Disclosure    string   `json:"disclosure"`
	Solutions     string   `json:"solutions"`
	References    []string `json:"references"`
	Tags          []string `json:"tags"`
	GithubSearch  []string `json:"github_search,omitempty"`
	From          string   `json:"from"`
	URL           string   `json:"url,omitempty"`
	Source        string   `json:"source"`
	Pushed        bool     `json:"pushed"`
}

// VulnInfoAccessor provides duck-typing interface for backward compatibility.
type VulnInfoAccessor interface {
	GetKey() string
	GetTitle() string
	GetSeverity() string
	GetCVE() string
	GetSource() string
	GetDescription() string
	GetDisclosure() string
	GetTags() []string
	GetReferences() []string
	GetGithubSearch() []string
}

// Ensure grabber.VulnInfo implements VulnInfoAccessor
var _ VulnInfoAccessor = (*grabpkg.VulnInfo)(nil)

// RunArticleCrawl satisfies the task.Executor interface for article crawling.
func (e *Engine) RunArticleCrawl(
	ctx context.Context,
	payload task.ArticleCrawlPayload,
	progressFn func(int),
) ([]any, error) {
	sources := payload.Sources
	if len(sources) == 0 {
		sources = articlegrabber.List()
	}

	limit := payload.Limit
	if limit <= 0 {
		limit = 10
	}

	var results []any
	total := len(sources)

	for i, name := range sources {
		if ctx.Err() != nil {
			return results, ctx.Err()
		}

		g, ok := articlegrabber.Get(name)
		if !ok {
			e.log.Warn("unknown article grabber, skipping", zap.String("source", name))
			progressFn((i + 1) * 100 / total)
			continue
		}

		e.log.Info("running article grabber", zap.String("source", name))
		articles, err := g.Fetch(ctx, limit)
		if err != nil {
			e.log.Warn("article grabber error", zap.String("source", name), zap.Error(err))
		} else {
			for _, a := range articles {
				results = append(results, articleToServerFormat(a))
			}
		}

		progressFn((i + 1) * 100 / total)
	}

	return results, nil
}

// articleToServerFormat converts an articlegrabber.Article to server-compatible Article format.
func articleToServerFormat(a *articlegrabber.Article) *serverArticle {
	return &serverArticle{
		Title:       a.Title,
		Summary:     a.Summary,
		Content:     a.Content,
		Author:      a.Author,
		Source:      a.Source,
		URL:         a.URL,
		Image:       a.Image,
		Tags:        a.Tags,
		PublishedAt: a.PublishedAt,
	}
}

// serverArticle is a simplified version of the server's Article model for compatibility.
type serverArticle struct {
	Title       string    `json:"title"`
	Summary     string    `json:"summary"`
	Content     string    `json:"content"`
	Author      string    `json:"author"`
	Source      string    `json:"source"`
	URL         string    `json:"url"`
	Image       string    `json:"image"`
	Tags        []string  `json:"tags"`
	PublishedAt time.Time `json:"published_at"`
}

// GetAvailableSources returns the list of available grabber sources.
func GetAvailableSources() []string {
	return grabpkg.Available()
}

// GetAvailableArticleSources returns the list of available article sources.
func GetAvailableArticleSources() []string {
	return articlegrabber.List()
}

// GetGrabber returns a grabber by name.
func GetGrabber(name string) (grabpkg.Grabber, error) {
	return grabpkg.ByName(name)
}
