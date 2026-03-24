// Package articlegrabber provides crawlers for security news and articles.
package articlegrabber

import (
	"context"
	"fmt"

	"github.com/go-rod/rod"
	"github.com/kataras/golog"

	"github.com/secflow/client/pkg/rodutil"
)

// RodCrawler provides a base implementation for go-rod based article crawlers.
type RodCrawler struct {
	log    *golog.Logger
	name   string
	config *rodutil.BypassConfig
}

// NewRodCrawler creates a new base rod crawler.
func NewRodCrawler(name string, log *golog.Logger) *RodCrawler {
	return &RodCrawler{
		log:    log,
		name:   name,
		config: rodutil.DefaultBypassConfig(),
	}
}

// WithConfig sets custom bypass config.
func (r *RodCrawler) WithConfig(config *rodutil.BypassConfig) *RodCrawler {
	r.config = config
	return r
}

// CreatePage creates a new rod page with bypass applied.
func (r *RodCrawler) CreatePage(ctx context.Context) (*rod.Page, func(), error) {
	browser, err := rodutil.GetBrowser(nil)
	if err != nil {
		return nil, nil, fmt.Errorf("get browser: %w", err)
	}

	page, err := rodutil.NewPage(browser)
	if err != nil {
		return nil, nil, fmt.Errorf("create page: %w", err)
	}

	// Apply bypass techniques
	if err := rodutil.ApplyBypass(page, r.config); err != nil {
		r.log.Warnf("apply bypass: %v", err)
	}

	cleanup := func() {
		page.Close()
	}

	return page, cleanup, nil
}

// Navigate navigates to a URL with bypass protection.
func (r *RodCrawler) Navigate(page *rod.Page, url string) error {
	return rodutil.SafeNavigate(page, url, r.config)
}

// NavigateWithContext navigates with context support.
func (r *RodCrawler) NavigateWithContext(ctx context.Context, page *rod.Page, url string) error {
	return rodutil.NavigateWithRetry(ctx, page, url, r.config)
}

// LogDebug logs debug message.
func (r *RodCrawler) LogDebug(format string, args ...interface{}) {
	r.log.Debugf(format, args...)
}

// LogInfo logs info message.
func (r *RodCrawler) LogInfo(format string, args ...interface{}) {
	r.log.Infof(format, args...)
}

// LogWarn logs warning message.
func (r *RodCrawler) LogWarn(format string, args ...interface{}) {
	r.log.Warnf(format, args...)
}

// LogError logs error message.
func (r *RodCrawler) LogError(format string, args ...interface{}) {
	r.log.Errorf(format, args...)
}

// CheckContext checks if context is cancelled.
func (r *RodCrawler) CheckContext(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}