// Package rodutil provides go-rod based browser automation helpers for grabbers.
// It handles browser lifecycle, stealth mode, and common scraping patterns.
package rodutil

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)

var (
	// Global browser instance (lazy initialization)
	globalBrowser *rod.Browser
	browserOnce   sync.Once
	browserErr    error
	browserMutex  sync.Mutex
)

// BrowserConfig contains configuration for browser initialization.
type BrowserConfig struct {
	// Headless mode (default: true)
	Headless bool
	// Timeout for browser operations
	Timeout time.Duration
	// Proxy URL (optional)
	Proxy string
	// UserDataDir for persistent browser data (optional)
	UserDataDir string
	// Window size
	WindowWidth  int
	WindowHeight int
}

// DefaultConfig returns a default browser configuration.
func DefaultConfig() *BrowserConfig {
	return &BrowserConfig{
		Headless:     true,
		Timeout:      120 * time.Second, // Increased from 30s to 120s for stability
		WindowWidth:  1920,
		WindowHeight: 1080,
	}
}

// GetBrowser returns the global browser instance, creating it if necessary.
func GetBrowser(config *BrowserConfig) (*rod.Browser, error) {
	if config == nil {
		config = DefaultConfig()
	}

	browserMutex.Lock()
	defer browserMutex.Unlock()

	// Check if existing browser exists and is usable
	if globalBrowser != nil && browserErr == nil {
		// Verify browser is still connected/usable by checking pages
		pages, err := globalBrowser.Pages()
		if err == nil && len(pages) >= 0 {
			return globalBrowser, nil
		}
		// Browser is disconnected, close it and reinitialize
		_ = closeBrowserLocked()
	}

	// Browser doesn't exist or was in error state, reinitialize
	browserOnce = sync.Once{}
	return initBrowser(config)
}

// initBrowser creates a new browser instance.
func initBrowser(config *BrowserConfig) (*rod.Browser, error) {
	browserOnce.Do(func() {
		globalBrowser, browserErr = createBrowser(config)
	})
	if browserErr != nil {
		// Reset once so we can retry next time
		browserOnce = sync.Once{}
	}
	return globalBrowser, browserErr
}

// closeBrowserLocked closes the browser without unlocking mutex.
// Caller must hold browserMutex.
func closeBrowserLocked() error {
	if globalBrowser == nil {
		return nil
	}
	err := globalBrowser.Close()
	globalBrowser = nil
	// Reset once to allow re-initialization
	browserOnce = sync.Once{}
	return err
}

// createBrowser creates a new rod browser with stealth mode.
func createBrowser(config *BrowserConfig) (*rod.Browser, error) {
	// Check for system Chrome/Chromium
	l := launcher.New()

	// Try to find Chrome/Chromium on macOS
	chromePaths := []string{
		"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
		"/Applications/Chromium.app/Contents/MacOS/Chromium",
		"/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge",
	}
	for _, path := range chromePaths {
		if _, err := os.Stat(path); err == nil {
			l = l.Bin(path)
			break
		}
	}

	// Set headless mode
	if config.Headless {
		l.Headless(true)
	} else {
		l.Headless(false)
	}

	// Set proxy if provided
	if config.Proxy != "" {
		l.Proxy(config.Proxy)
	}

	// Set user data dir if provided
	if config.UserDataDir != "" {
		l.UserDataDir(config.UserDataDir)
	}

	// Common stealth arguments
	l.Set("--disable-blink-features", "AutomationControlled")
	l.Set("--disable-web-security")
	l.Set("--disable-features", "IsolateOrigins,site-per-process")
	l.Set("--window-size", fmt.Sprintf("%d,%d", config.WindowWidth, config.WindowHeight))
	l.Set("--no-sandbox")
	l.Set("--disable-setuid-sandbox")
	l.Set("--disable-dev-shm-usage")
	l.Set("--disable-accelerated-2d-canvas")
	l.Set("--disable-gpu")
	l.Set("--disable-features", "BlockInsecurePrivateNetworkRequests")

	// Launch browser
	controlURL, err := l.Launch()
	if err != nil {
		return nil, fmt.Errorf("launch browser: %w", err)
	}

	// Connect to browser
	browser := rod.New().ControlURL(controlURL).Timeout(config.Timeout)
	if err := browser.Connect(); err != nil {
		return nil, fmt.Errorf("connect to browser: %w", err)
	}

	return browser, nil
}

// NewPage creates a new page with stealth mode enabled.
func NewPage(browser *rod.Browser) (*rod.Page, error) {
	page, err := browser.Page(proto.TargetCreateTarget{URL: "about:blank"})
	if err != nil {
		return nil, fmt.Errorf("create page: %w", err)
	}

	// Set default viewport
	scale := 1.0
	if err := page.SetViewport(&proto.EmulationSetDeviceMetricsOverride{
		Width:  1920,
		Height: 1080,
		Scale:  &scale,
	}); err != nil {
		return nil, fmt.Errorf("set viewport: %w", err)
	}

	return page, nil
}

// Navigate navigates to a URL and waits for the page to load.
func Navigate(page *rod.Page, url string, waitFor string) error {
	if err := page.Navigate(url); err != nil {
		return fmt.Errorf("navigate to %s: %w", url, err)
	}

	if err := page.WaitLoad(); err != nil {
		return fmt.Errorf("wait for page load: %w", err)
	}

	// Wait for specific element if provided
	if waitFor != "" {
		if _, err := page.Element(waitFor); err != nil {
			return fmt.Errorf("wait for element %s: %w", waitFor, err)
		}
	}

	return nil
}

// NavigateWithContext navigates with context support for cancellation.
func NavigateWithContext(ctx context.Context, page *rod.Page, url string, waitFor string) error {
	done := make(chan error, 1)
	go func() {
		done <- Navigate(page, url, waitFor)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-done:
		return err
	}
}

// CloseBrowser closes the global browser instance.
func CloseBrowser() error {
	browserMutex.Lock()
	defer browserMutex.Unlock()
	return closeBrowserLocked()
}

// MustCloseBrowser closes the browser and panics on error.
func MustCloseBrowser() {
	if err := CloseBrowser(); err != nil {
		panic(err)
	}
}

// IsHeadlessEnv returns true if running in headless environment.
func IsHeadlessEnv() bool {
	return os.Getenv("DISPLAY") == "" || os.Getenv("SECFLOW_HEADLESS") != "false"
}

// GetBrowserPages returns all pages from the global browser instance.
func GetBrowserPages() ([]*rod.Page, error) {
	browserMutex.Lock()
	defer browserMutex.Unlock()

	if globalBrowser == nil {
		return nil, fmt.Errorf("browser not initialized")
	}

	return globalBrowser.Pages()
}
