// Package article implements grabbers for security news and articles.
package articlegrabber

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/imroc/req/v3"
)

// Article represents a security article or news item.
type Article struct {
	Title       string    `json:"title"`
	Summary     string    `json:"summary"`
	Content     string    `json:"content"`
	Author      string    `json:"author"`
	Source      string    `json:"source"`
	URL         string    `json:"url"`
	Image       string    `json:"image"` // cover image URL
	Tags        []string  `json:"tags"`
	PublishedAt time.Time `json:"published_at"`
	FetchedAt   time.Time `json:"fetched_at"` // When we fetched this article
	IsNew       bool      `json:"is_new"`     // True if article was just fetched
}

// Grabber is the interface for article sources.
type Grabber interface {
	Name() string
	Fetch(ctx context.Context, limit int) ([]*Article, error)
}

// Registry holds all registered grabbers.
var registry = make(map[string]Grabber)

// Register adds a grabber to the registry.
func Register(g Grabber) {
	registry[g.Name()] = g
}

// Get returns a grabber by name.
func Get(name string) (Grabber, bool) {
	g, ok := registry[name]
	return g, ok
}

// List returns all registered grabber names.
func List() []string {
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	return names
}

// FreeBufGrabber fetches articles from FreeBuf (安全脉搏).
type FreeBufGrabber struct {
	client *req.Client
}

// NewFreeBufGrabber creates a new FreeBuf grabber.
func NewFreeBufGrabber() *FreeBufGrabber {
	return &FreeBufGrabber{
		client: req.NewClient().
			SetTimeout(30 * time.Second).
			SetUserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"),
	}
}

// Name returns the grabber name.
func (g *FreeBufGrabber) Name() string {
	return "freebuf"
}

// Fetch retrieves articles from FreeBuf.
func (g *FreeBufGrabber) Fetch(ctx context.Context, limit int) ([]*Article, error) {
	// FreeBuf 首页
	resp, err := g.client.R().SetContext(ctx).Get("https://www.freebuf.com/")
	if err != nil {
		return nil, fmt.Errorf("fetch freebuf: %w", err)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("parse freebuf: %w", err)
	}

	var articles []*Article
	doc.Find(".article-item").Each(func(i int, s *goquery.Selection) {
		if len(articles) >= limit {
			return
		}

		title := strings.TrimSpace(s.Find(".title").Text())
		link, _ := s.Find(".title a").Attr("href")
		summary := strings.TrimSpace(s.Find(".summary").Text())
		author := strings.TrimSpace(s.Find(".author").Text())
		timeStr := strings.TrimSpace(s.Find(".time").Text())

		if title == "" || link == "" {
			return
		}

		// Parse time
		publishedAt, _ := time.Parse("2006-01-02", timeStr)
		if publishedAt.IsZero() {
			publishedAt = time.Now()
		}

		// Ensure full URL
		if !strings.HasPrefix(link, "http") {
			link = "https://www.freebuf.com" + link
		}

		articles = append(articles, &Article{
			Title:       title,
			Summary:     summary,
			Source:      "FreeBuf",
			URL:         link,
			Author:      author,
			Tags:        []string{"安全资讯"},
			PublishedAt: publishedAt,
		})
	})

	return articles, nil
}

// SecurityWeekGrabber fetches articles from SecurityWeek.
type SecurityWeekGrabber struct {
	client *req.Client
}

// NewSecurityWeekGrabber creates a new SecurityWeek grabber.
func NewSecurityWeekGrabber() *SecurityWeekGrabber {
	return &SecurityWeekGrabber{
		client: req.NewClient().
			SetTimeout(30 * time.Second).
			SetUserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"),
	}
}

// Name returns the grabber name.
func (g *SecurityWeekGrabber) Name() string {
	return "securityweek"
}

// Fetch retrieves articles from SecurityWeek RSS feed.
func (g *SecurityWeekGrabber) Fetch(ctx context.Context, limit int) ([]*Article, error) {
	resp, err := g.client.R().SetContext(ctx).Get("https://www.securityweek.com/feed/")
	if err != nil {
		return nil, fmt.Errorf("fetch securityweek: %w", err)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("parse securityweek: %w", err)
	}

	var articles []*Article
	doc.Find("item").Each(func(i int, s *goquery.Selection) {
		if len(articles) >= limit {
			return
		}

		title := strings.TrimSpace(s.Find("title").Text())
		link := strings.TrimSpace(s.Find("link").Text())
		description := strings.TrimSpace(s.Find("description").Text())
		pubDate := strings.TrimSpace(s.Find("pubDate").Text())

		// Clean up title (remove CDATA)
		title = cleanCDATA(title)
		description = cleanCDATA(description)

		// Parse time
		publishedAt, _ := time.Parse(time.RFC1123, pubDate)
		if publishedAt.IsZero() {
			publishedAt = time.Now()
		}

		articles = append(articles, &Article{
			Title:       title,
			Summary:     description,
			Source:      "SecurityWeek",
			URL:         link,
			Tags:        []string{"国际安全资讯"},
			PublishedAt: publishedAt,
		})
	})

	return articles, nil
}

// HackernewsGrabber fetches top stories from Hacker News.
type HackernewsGrabber struct {
	client *http.Client
}

// NewHackernewsGrabber creates a new Hacker News grabber.
func NewHackernewsGrabber() *HackernewsGrabber {
	return &HackernewsGrabber{
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// Name returns the grabber name.
func (g *HackernewsGrabber) Name() string {
	return "hackernews"
}

// Fetch retrieves top stories from Hacker News.
func (g *HackernewsGrabber) Fetch(ctx context.Context, limit int) ([]*Article, error) {
	// Get top stories IDs
	resp, err := g.client.Get("https://hacker-news.firebaseio.com/v0/topstories.json")
	if err != nil {
		return nil, fmt.Errorf("fetch hackernews stories: %w", err)
	}
	defer resp.Body.Close()

	var storyIDs []int
	if err := json.NewDecoder(resp.Body).Decode(&storyIDs); err != nil {
		return nil, fmt.Errorf("decode story ids: %w", err)
	}

	if len(storyIDs) > limit {
		storyIDs = storyIDs[:limit]
	}

	var articles []*Article
	for _, id := range storyIDs {
		story, err := g.fetchStory(ctx, id)
		if err != nil {
			continue
		}
		// Filter for security-related stories
		if isSecurityRelated(story.Title) {
			articles = append(articles, story)
		}
	}

	return articles, nil
}

func (g *HackernewsGrabber) fetchStory(ctx context.Context, id int) (*Article, error) {
	url := fmt.Sprintf("https://hacker-news.firebaseio.com/v0/item/%d.json", id)
	resp, err := g.client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var data struct {
		Title string `json:"title"`
		URL   string `json:"url"`
		Time  int64  `json:"time"`
		By    string `json:"by"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	return &Article{
		Title:       data.Title,
		URL:         data.URL,
		Author:      data.By,
		Source:      "Hacker News",
		Tags:        []string{"技术资讯"},
		PublishedAt: time.Unix(data.Time, 0),
	}, nil
}

// Helper functions

func cleanCDATA(s string) string {
	re := regexp.MustCompile(`<!\[CDATA\[(.*?)\]\]>`)
	return re.ReplaceAllString(s, "$1")
}

func isSecurityRelated(title string) bool {
	securityKeywords := []string{
		"security", "vulnerability", "exploit", "cve", "hack",
		"breach", "attack", "malware", "ransomware", "phishing",
		"漏洞", "安全", "攻击", "恶意软件", "勒索软件",
	}
	titleLower := strings.ToLower(title)
	for _, kw := range securityKeywords {
		if strings.Contains(titleLower, kw) {
			return true
		}
	}
	return false
}

// Init registers all article grabbers.
func Init() {
	Register(NewFreeBufGrabber())
	Register(NewSecurityWeekGrabber())
	Register(NewHackernewsGrabber())
	Register(NewQianxinWeeklyGrabber())
	Register(NewVenustechGrabber())
	Register(NewXianzhiGrabber())
	Register(NewSihouGrabber())
}

// WorkerPool manages concurrent fetch workers.
type WorkerPool struct {
	workers int
	jobs    chan *FetchJob
	results chan *FetchResult
	wg      sync.WaitGroup
	sem     chan struct{}
}

// FetchJob represents a fetch task.
type FetchJob struct {
	Source string
	Limit  int
}

// FetchResult represents the result of a fetch task.
type FetchResult struct {
	Source  string
	Articles []*Article
	Error   error
}

// NewWorkerPool creates a new WorkerPool with the specified number of workers.
func NewWorkerPool(workers int) *WorkerPool {
	if workers <= 0 {
		workers = 4
	}
	return &WorkerPool{
		workers: workers,
		jobs:    make(chan *FetchJob, workers*2),
		results: make(chan *FetchResult, workers*2),
		sem:     make(chan struct{}, workers),
	}
}

// Run starts the worker pool and processes jobs.
func (p *WorkerPool) Run(ctx context.Context, grabbers map[string]Grabber, limiter *RateLimiter) {
	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		go p.worker(ctx, grabbers, limiter)
	}
}

// worker processes fetch jobs from the job channel.
func (p *WorkerPool) worker(ctx context.Context, grabbers map[string]Grabber, limiter *RateLimiter) {
	defer p.wg.Done()

	for job := range p.jobs {
		select {
		case <-ctx.Done():
			p.results <- &FetchResult{Source: job.Source, Error: ctx.Err()}
			return
		case p.sem <- struct{}{}:
			// Acquired semaphore
		}

		go func(j *FetchJob) {
			defer func() { <-p.sem }()

			// Apply rate limiting
			if limiter != nil {
				if err := limiter.Wait(ctx, j.Source); err != nil {
					p.results <- &FetchResult{Source: j.Source, Error: err}
					return
				}
			}

			grabber, ok := grabbers[j.Source]
			if !ok {
				p.results <- &FetchResult{Source: j.Source, Error: fmt.Errorf("grabber not found: %s", j.Source)}
				return
			}

			start := time.Now()
			articles, err := grabber.Fetch(ctx, j.Limit)
			_ = time.Since(start) // Latency tracked for future metrics

			if err != nil {
				if limiter != nil {
					limiter.RecordFailure(j.Source)
				}
				p.results <- &FetchResult{Source: j.Source, Articles: articles, Error: err}
				return
			}

			if limiter != nil {
				limiter.RecordSuccess(j.Source)
			}

			// Mark articles as fetched
			for _, a := range articles {
				a.FetchedAt = time.Now()
				a.IsNew = true
			}

			p.results <- &FetchResult{Source: j.Source, Articles: articles, Error: nil}
		}(job)
	}
}

// Submit adds a fetch job to the worker pool.
func (p *WorkerPool) Submit(job *FetchJob) {
	p.jobs <- job
}

// SubmitAndWait submits jobs and waits for all results.
func (p *WorkerPool) SubmitAndWait(ctx context.Context, jobs []*FetchJob) []*FetchResult {
	results := make([]*FetchResult, 0, len(jobs))

	// Submit all jobs
	for _, job := range jobs {
		p.Submit(job)
	}

	// Collect results
	for i := 0; i < len(jobs); i++ {
		select {
		case <-ctx.Done():
			return results
		case result := <-p.results:
			results = append(results, result)
		}
	}

	return results
}

// Shutdown gracefully shuts down the worker pool.
func (p *WorkerPool) Shutdown() {
	close(p.jobs)
	p.wg.Wait()
	close(p.results)
}

// Results returns the results channel.
func (p *WorkerPool) Results() <-chan *FetchResult {
	return p.results
}

// GrabberManager manages multiple grabbers with rate limiting and health monitoring.
type GrabberManager struct {
	grabbers   map[string]Grabber
	limiter    *RateLimiter
	pool       *WorkerPool
	dedup      *Deduplicator
	health     *HealthChecker
	syncStates map[string]*SyncState
	mu         sync.RWMutex
}

// SyncState tracks synchronization state for incremental syncs.
type SyncState struct {
	Source         string    `json:"source"`
	LastSync       time.Time `json:"last_sync"`
	LastArticleTime time.Time `json:"last_article_time"`
	ArticlesCount  int       `json:"articles_count"`
}

// NewGrabberManager creates a new GrabberManager with default settings.
func NewGrabberManager() *GrabberManager {
	return &GrabberManager{
		grabbers:   make(map[string]Grabber),
		limiter:    NewRateLimiter(),
		dedup:     NewDeduplicator(),
		health:    NewHealthChecker(),
		syncStates: make(map[string]*SyncState),
	}
}

// NewGrabberManagerWithConfig creates a GrabberManager with custom configuration.
func NewGrabberManagerWithConfig(workers int, maxRetries int, baseDelay, maxDelay time.Duration) *GrabberManager {
	m := &GrabberManager{
		grabbers:   make(map[string]Grabber),
		limiter:    NewRateLimiterWithConfig(maxRetries, baseDelay, maxDelay),
		dedup:     NewDeduplicator(),
		health:    NewHealthChecker(),
		syncStates: make(map[string]*SyncState),
	}
	m.pool = NewWorkerPool(workers)
	return m
}

// Register adds a grabber to the manager.
func (m *GrabberManager) Register(grabber Grabber) {
	m.mu.Lock()
	defer m.mu.Unlock()

	name := grabber.Name()
	m.grabbers[name] = grabber
	m.health.Register(name, grabber)
}

// RegisterAll registers all grabbers from the global registry.
func (m *GrabberManager) RegisterAll() {
	for name, grabber := range registry {
		m.mu.Lock()
		m.grabbers[name] = grabber
		m.health.Register(name, grabber)
		m.mu.Unlock()
	}
}

// FetchAll fetches articles from all registered grabbers concurrently.
func (m *GrabberManager) FetchAll(ctx context.Context, limit int) ([]*Article, error) {
	m.mu.RLock()
	jobs := make([]*FetchJob, 0, len(m.grabbers))
	for name := range m.grabbers {
		jobs = append(jobs, &FetchJob{Source: name, Limit: limit})
	}
	m.mu.RUnlock()

	results := m.pool.SubmitAndWait(ctx, jobs)

	allArticles := make([]*Article, 0)
	var lastErr error

	for _, result := range results {
		if result.Error != nil {
			lastErr = result.Error
			continue
		}

		// Apply deduplication
		uniqueArticles := m.dedup.BatchAdd(result.Articles)
		allArticles = append(allArticles, uniqueArticles...)
	}

	if len(allArticles) == 0 && lastErr != nil {
		return nil, lastErr
	}

	return allArticles, nil
}

// FetchWithTimeout fetches articles from a specific source with a timeout.
func (m *GrabberManager) FetchWithTimeout(ctx context.Context, source string, limit int) ([]*Article, error) {
	m.mu.RLock()
	grabber, ok := m.grabbers[source]
	m.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("grabber not found: %s", source)
	}

	// Apply rate limiting
	if err := m.limiter.Wait(ctx, source); err != nil {
		return nil, err
	}

	start := time.Now()
	articles, err := grabber.Fetch(ctx, limit)
	latency := time.Since(start)

	if err != nil {
		m.limiter.RecordFailure(source)
		m.health.RecordFailure(source, latency, err)
		return nil, err
	}

	m.limiter.RecordSuccess(source)
	m.health.RecordSuccess(source, latency)

	// Mark articles as fetched
	for _, a := range articles {
		a.FetchedAt = time.Now()
		a.IsNew = true
	}

	// Apply deduplication
	uniqueArticles := m.dedup.BatchAdd(articles)

	// Update sync state if we have articles
	if len(uniqueArticles) > 0 {
		m.UpdateSyncState(source, getLatestArticleTime(uniqueArticles))
	}

	return uniqueArticles, nil
}

// ShouldFetch determines if we should fetch from a source based on sync state.
// Returns true if no previous sync exists or if enough time has passed since last sync.
func (m *GrabberManager) ShouldFetch(source string, since time.Time) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	state, exists := m.syncStates[source]
	if !exists {
		return true
	}

	// Always fetch if never synced
	if state.LastSync.IsZero() {
		return true
	}

	// Fetch if last sync was before the specified time
	return state.LastSync.Before(since)
}

// UpdateSyncState updates the sync state for a source after fetching.
func (m *GrabberManager) UpdateSyncState(source string, latestArticleTime time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	state, exists := m.syncStates[source]
	if !exists {
		state = &SyncState{Source: source}
		m.syncStates[source] = state
	}

	state.LastSync = time.Now()
	state.LastArticleTime = latestArticleTime
	state.ArticlesCount++

	return nil
}

// GetSyncState returns the sync state for a source.
func (m *GrabberManager) GetSyncState(source string) *SyncState {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if state, exists := m.syncStates[source]; exists {
		copy := *state
		return &copy
	}
	return nil
}

// GetAllSyncStates returns all sync states.
func (m *GrabberManager) GetAllSyncStates() map[string]*SyncState {
	m.mu.RLock()
	defer m.mu.RUnlock()

	states := make(map[string]*SyncState, len(m.syncStates))
	for k, v := range m.syncStates {
		copy := *v
		states[k] = &copy
	}
	return states
}

// GetDeduplicator returns the deduplicator.
func (m *GrabberManager) GetDeduplicator() *Deduplicator {
	return m.dedup
}

// GetHealthChecker returns the health checker.
func (m *GrabberManager) GetHealthChecker() *HealthChecker {
	return m.health
}

// GetRateLimiter returns the rate limiter.
func (m *GrabberManager) GetRateLimiter() *RateLimiter {
	return m.limiter
}

// ResetDeduplicator clears the deduplicator state.
func (m *GrabberManager) ResetDeduplicator() {
	m.dedup.Clear()
}

// ResetSyncStates clears all sync states.
func (m *GrabberManager) ResetSyncStates() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.syncStates = make(map[string]*SyncState)
}

// getLatestArticleTime finds the latest article publish time from a list.
func getLatestArticleTime(articles []*Article) time.Time {
	if len(articles) == 0 {
		return time.Time{}
	}

	latest := articles[0].PublishedAt
	for _, a := range articles[1:] {
		if a.PublishedAt.After(latest) {
			latest = a.PublishedAt
		}
	}
	return latest
}
