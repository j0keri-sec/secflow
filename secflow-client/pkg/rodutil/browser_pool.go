// Package rodutil provides go-rod based browser automation helpers for grabbers.
// It handles browser lifecycle, page pooling, and common scraping patterns.
package rodutil

import (
	"context"
	"sync"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
)

// PagePool manages a pool of reusable browser pages for improved performance.
// Instead of creating new pages for each request, pages are recycled.
type PagePool struct {
	browser     *rod.Browser
	pool        chan *rod.Page
	maxPoolSize int
	mu          sync.Mutex
	metrics     *PoolMetrics
}

// PoolMetrics tracks page pool usage statistics.
type PoolMetrics struct {
	mu            sync.RWMutex
	TotalAcq     int64
	TotalRel     int64
	TotalNew     int64
	TotalRecycled int64
	PoolHits    int64
	PoolMisses  int64
}

// NewPagePool creates a new page pool for the given browser.
func NewPagePool(browser *rod.Browser, maxSize int) *PagePool {
	return &PagePool{
		browser:     browser,
		pool:        make(chan *rod.Page, maxSize),
		maxPoolSize: maxSize,
		metrics:     &PoolMetrics{},
	}
}

// Acquire gets a page from the pool, or creates a new one if the pool is empty.
func (p *PagePool) Acquire(ctx context.Context) (*rod.Page, error) {
	p.metrics.mu.Lock()
	p.metrics.TotalAcq++
	p.metrics.mu.Unlock()

	// Try to get from pool first (non-blocking)
	select {
	case page := <-p.pool:
		p.metrics.mu.Lock()
		p.metrics.PoolHits++
		p.metrics.TotalRecycled++
		p.metrics.mu.Unlock()

		// Verify page is still usable
		if p.isPageUsable(page) {
			return page, nil
		}
		// Page is closed or broken, close it and create new
		page.Close()
		p.metrics.mu.Lock()
		p.metrics.TotalNew++
		p.metrics.mu.Unlock()

	default:
		p.metrics.mu.Lock()
		p.metrics.PoolMisses++
		p.metrics.TotalNew++
		p.metrics.mu.Unlock()
	}

	// Create new page
	return p.newPage(ctx)
}

// Release returns a page to the pool for reuse.
// If the pool is full, the page is closed instead.
func (p *PagePool) Release(page *rod.Page) {
	p.metrics.mu.Lock()
	p.metrics.TotalRel++
	p.metrics.mu.Unlock()

	// Check if page is still usable
	if !p.isPageUsable(page) {
		page.Close()
		return
	}

	// Try to return to pool (non-blocking)
	select {
	case p.pool <- page:
		return
	default:
		// Pool is full, close the page
		page.Close()
	}
}

// isPageUsable checks if a page is still connected and usable.
func (p *PagePool) isPageUsable(page *rod.Page) bool {
	// Simple check - try to navigate to about:blank as a health check
	err := page.Navigate("about:blank")
	if err != nil {
		return false
	}
	return true
}

// newPage creates a new page with stealth mode.
func (p *PagePool) newPage(_ context.Context) (*rod.Page, error) {
	// Create page
	newPage, err := p.browser.Page(proto.TargetCreateTarget{URL: "about:blank"})
	if err != nil {
		return nil, err
	}

	// Set viewport
	scale := 1.0
	if err := newPage.SetViewport(&proto.EmulationSetDeviceMetricsOverride{
		Width:  1920,
		Height: 1080,
		Scale:  &scale,
	}); err != nil {
		newPage.Close()
		return nil, err
	}

	// Enable stealth mode
	if err := p.enableStealth(newPage); err != nil {
		newPage.Close()
		return nil, err
	}

	return newPage, nil
}

// enableStealth applies stealth evasion measures to reduce automation fingerprinting.
// This is a basic implementation that removes common webdriver detection flags.
//
// Note: For advanced stealth features (Chrome DevTools Protocol stealth), consider
// using github.com/go-rod/stealth library. The current implementation provides
// basic fingerprinting reduction sufficient for non-adversarial targets.
func (p *PagePool) enableStealth(page *rod.Page) error {
	// Basic stealth: Execute script to remove webdriver flag
	// Note: page.Eval() runs immediately, not on page load, so this is limited
	// For full stealth, use rod-stealth library or CDP-level script injection
	stealthScript := `() => {
		try {
			Object.defineProperty(navigator, 'webdriver', {
				get: () => undefined,
				configurable: true,
				enumerable: true
			});
		} catch (e) {
			// Ignore errors - stealth is best-effort
		}
	}`
	_, err := page.Eval(stealthScript)
	return err
}

// Close closes all pages in the pool and clears it.
func (p *PagePool) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()

	close(p.pool)
	for page := range p.pool {
		page.Close()
	}
}

// Metrics returns current pool usage statistics.
func (p *PagePool) Metrics() PoolMetrics {
	p.metrics.mu.RLock()
	defer p.metrics.mu.RUnlock()
	return PoolMetrics{
		TotalAcq:     p.metrics.TotalAcq,
		TotalRel:     p.metrics.TotalRel,
		TotalNew:     p.metrics.TotalNew,
		TotalRecycled: p.metrics.TotalRecycled,
		PoolHits:    p.metrics.PoolHits,
		PoolMisses:  p.metrics.PoolMisses,
	}
}

// PoolSize returns the number of pages currently in the pool.
func (p *PagePool) PoolSize() int {
	return len(p.pool)
}

// MaxPoolSize returns the maximum pool size.
func (p *PagePool) MaxPoolSize() int {
	return p.maxPoolSize
}
