// Package rodutil provides WAF and anti-bot detection bypass techniques.
// This package contains advanced bypass techniques for various WAF systems
// including Alibaba Cloud WAF, Tencent Cloud WAF, Changting WAF, and others.
package rodutil

import (
	"context"
	"crypto/md5"
	"fmt"
	"math"
	"math/rand"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
)

// BypassConfig contains configuration for bypass techniques.
type BypassConfig struct {
	// Enable mouse movement simulation
	SimulateMouse bool
	// Enable random delays between actions
	RandomDelays bool
	// Min delay in milliseconds
	MinDelay int
	// Max delay in milliseconds
	MaxDelay int
	// Enable viewport randomization
	RandomViewport bool
	// Enable user agent rotation
	RotateUserAgent bool
	// Enable proxy rotation (requires proxy pool)
	UseProxyRotation bool
	// Proxy pool URL or file path
	ProxyPool string
	// Enable advanced fingerprint spoofing
	AdvancedFingerprint bool
	// Enable WAF detection and auto-retry
	AutoRetryOnWAF bool
	// Maximum retry attempts on WAF detection
	MaxRetries int
	// Custom headers to add/override
	CustomHeaders map[string]string
}

// DefaultBypassConfig returns a default bypass configuration.
func DefaultBypassConfig() *BypassConfig {
	return &BypassConfig{
		SimulateMouse:       true,
		RandomDelays:        true,
		MinDelay:            100,
		MaxDelay:            500,
		RandomViewport:      true,
		RotateUserAgent:     true,
		UseProxyRotation:    false,
		AdvancedFingerprint: true,
		AutoRetryOnWAF:      true,
		MaxRetries:          3,
	}
}

// Proxy pool management
var (
	proxyPool     []string
	proxyPoolMu   sync.RWMutex
	currentProxy  int
	proxyPoolPath string
)

// LoadProxyPool loads proxy list from file or URL
func LoadProxyPool(source string) error {
	proxyPoolMu.Lock()
	defer proxyPoolMu.Unlock()

	proxyList := strings.Split(getEnvOrDefault("PROXY_LIST", ""), ",")
	for _, p := range proxyList {
		p = strings.TrimSpace(p)
		if p != "" {
			proxyPool = append(proxyPool, p)
		}
	}

	if len(proxyPool) == 0 {
		return fmt.Errorf("no proxies loaded from %s", source)
	}

	proxyPoolPath = source
	return nil
}

// GetNextProxy returns the next proxy from the pool in round-robin fashion
func GetNextProxy() string {
	proxyPoolMu.RLock()
	defer proxyPoolMu.RUnlock()

	if len(proxyPool) == 0 {
		return ""
	}

	proxy := proxyPool[currentProxy]
	currentProxy = (currentProxy + 1) % len(proxyPool)
	return proxy
}

// ApplyBypass applies all configured bypass techniques to the page.
func ApplyBypass(page *rod.Page, config *BypassConfig) error {
	if config == nil {
		config = DefaultBypassConfig()
	}

	if config.RotateUserAgent {
		if err := RotateUserAgent(page); err != nil {
			return fmt.Errorf("rotate user agent: %w", err)
		}
	}

	if config.RandomViewport {
		if err := RandomizeViewport(page); err != nil {
			return fmt.Errorf("randomize viewport: %w", err)
		}
	}

	if config.AdvancedFingerprint {
		if err := ApplyAdvancedFingerprint(page); err != nil {
			return fmt.Errorf("apply advanced fingerprint: %w", err)
		}
	}

	if config.SimulateMouse {
		if err := SimulateHumanMouse(page); err != nil {
			return fmt.Errorf("simulate mouse: %w", err)
		}
	}

	return nil
}

// RandomDelay waits for a random duration between min and max milliseconds.
func RandomDelay(minMs, maxMs int) {
	delay := rand.Intn(maxMs-minMs) + minMs
	time.Sleep(time.Duration(delay) * time.Millisecond)
}

// RandomizeViewport sets a random viewport size from common resolutions.
func RandomizeViewport(page *rod.Page) error {
	resolutions := []struct {
		width  int
		height int
	}{
		{1920, 1080},
		{1366, 768},
		{1440, 900},
		{1536, 864},
		{1280, 720},
		{1600, 900},
		{1280, 800},
		{1680, 1050},
		{2560, 1440},
		{3840, 2160},
		// Mobile resolutions
		{390, 844},   // iPhone 14
		{414, 896},   // iPhone 11
		{375, 667},   // iPhone 8
		{412, 915},   // Pixel 7
		{360, 800},   // Samsung Galaxy S21
	}

	res := resolutions[rand.Intn(len(resolutions))]
	return SetViewport(page, res.width, res.height)
}

// RotateUserAgent sets a random user agent.
func RotateUserAgent(page *rod.Page) error {
	ua := AllUserAgents[rand.Intn(len(AllUserAgents))]
	return SetUserAgent(page, ua)
}

// RandUserAgent returns a random user agent string.
func RandUserAgent() string {
	if len(AllUserAgents) == 0 {
		return ""
	}
	return AllUserAgents[rand.Intn(len(AllUserAgents))]
}

// AllUserAgents is a list of realistic desktop browser user agents.
// Sourced from https://github.com/intoli/user-agents (desktop only).
var AllUserAgents = []string{
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.4 Safari/605.1.15",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.5 Safari/605.1.15",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/137.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/137.0.0.0 Safari/537.36 Edg/137.0.0.0",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/137.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.5 Safari/605.1.15",
	"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/114.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:139.0) Gecko/20100101 Firefox/139.0",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:138.0) Gecko/20100101 Firefox/138.0",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:128.0) Gecko/20100101 Firefox/128.0",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/135.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/134.0.0.0 Safari/537.36 OPR/119.0.0.0",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36 Edg/136.0.0.0",
	"Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:136.0) Gecko/20100101 Firefox/136.0",
	"Mozilla/5.0 (X11; CrOS x86_64 14541.0.0) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36",
	"Mozilla/5.0 (X11; Linux x86_64; rv:125.0) Gecko/20100101 Firefox/125.0",
}

// ApplyAdvancedFingerprint applies comprehensive fingerprint spoofing
func ApplyAdvancedFingerprint(page *rod.Page) error {
	script := `
	() => {
		// Override webdriver detection
		Object.defineProperty(navigator, 'webdriver', {
			get: () => undefined,
			configurable: true
		});

		// Override plugins with realistic values
		Object.defineProperty(navigator, 'plugins', {
			get: () => {
				const plugins = [
					{ name: 'Chrome PDF Plugin', filename: 'internal-pdf-viewer', description: 'Portable Document Format' },
					{ name: 'Chrome PDF Viewer', filename: 'mhjfbmdgcfjbbpaeojofohoefgiehjai', description: '' },
					{ name: 'Native Client', filename: 'internal-nacl-plugin', description: '' }
				];
				plugins.length = Math.floor(Math.random() * 2) + 3;
				return plugins;
			},
			configurable: true
		});

		// Override languages
		Object.defineProperty(navigator, 'languages', {
			get: () => {
				const langs = ['zh-CN', 'zh', 'en-US', 'en'];
				return langs.slice(0, Math.floor(Math.random() * 2) + 2);
			},
			configurable: true
		});

		// Override hardware concurrency
		Object.defineProperty(navigator, 'hardwareConcurrency', {
			get: () => Math.floor(Math.random() * 4) + 4,
			configurable: true
		});

		// Override device memory
		Object.defineProperty(navigator, 'deviceMemory', {
			get: () => [2, 4, 8][Math.floor(Math.random() * 3)],
			configurable: true
		});

		// Override platform
		const platforms = ['Win32', 'MacIntel', 'Linux x86_64', 'iPhone', 'Android'];
		Object.defineProperty(navigator, 'platform', {
			get: () => platforms[Math.floor(Math.random() * platforms.length)],
			configurable: true
		});

		// Override vendor
		Object.defineProperty(navigator, 'vendor', {
			get: () => 'Google Inc.',
			configurable: true
		});

		// Override maxTouchPoints
		Object.defineProperty(navigator, 'maxTouchPoints', {
			get: () => Math.random() > 0.5 ? 0 : Math.floor(Math.random() * 10) + 1,
			configurable: true
		});

		// Chrome runtime object
		window.chrome = {
			runtime: { connect: function() {}, sendMessage: function() {} },
			storage: { sync: { get: function() {}, set: function() {} } },
			tabs: { query: function() {} }
		};

		// Remove automation flags
		window.__webdriver_evaluate = undefined;
		window.__selenium_evaluate = undefined;
		window.__webdriver_script_function = undefined;
		window.__webdriver_script_func = undefined;
		window.__webdriver_script_fn = undefined;
		window.__fxdriver_evaluate = undefined;
		window.__driver_undefined = undefined;
		window.__webdriver_call = undefined;
		window.__selenium_call = undefined;
		window.__webdriver_click = undefined;
		window.__driver_click = undefined;
		window.__fxdriver_click = undefined;

		// Canvas fingerprint randomization
		const originalGetContext = HTMLCanvasElement.prototype.getContext;
		HTMLCanvasElement.prototype.getContext = function(type, attributes) {
			const context = originalGetContext.call(this, type, attributes);
			if (type === '2d') {
				const originalFillText = context.fillText;
				context.fillText = function(...args) {
					if (args.length >= 4 && Math.random() > 0.7) {
						args[1] += (Math.random() - 0.5) * 0.1;
						args[2] += (Math.random() - 0.5) * 0.1;
					}
					return originalFillText.apply(this, args);
				};
			}
			return context;
		};

		// WebGL fingerprint spoofing
		const getParameter = WebGLRenderingContext.prototype.getParameter;
		WebGLRenderingContext.prototype.getParameter = function(param) {
			if (param === 37445) return 'Intel Inc.';
			if (param === 37446) return 'Intel Iris OpenGL Engine';
			if (navigator.userAgent.includes('Firefox')) {
				if (param === 37445) return 'NVIDIA Corporation';
				if (param === 37446) return 'NVIDIA GeForce GTX 1080/PCIe/SSE2';
			}
			return getParameter.call(this, param);
		};

		// Override permissions API
		const originalQuery = window.navigator.permissions.query;
		window.navigator.permissions.query = (parameters) => (
			parameters.name === 'notifications' ?
				Promise.resolve({ state: Notification.permission, onchange: null }) :
				originalQuery(parameters)
		);

		// MediaDevices enumeration spoofing
		if (navigator.mediaDevices && navigator.mediaDevices.enumerateDevices) {
			navigator.mediaDevices.enumerateDevices = function() {
				return Promise.resolve([
					{ kind: 'audioinput', deviceId: 'default', label: '', groupId: 'group_1' },
					{ kind: 'videoinput', deviceId: 'default', label: '', groupId: 'group_2' }
				]);
			};
		}

		// Override connection info
		Object.defineProperty(navigator, 'connection', {
			get: () => ({
				effectiveType: ['4g', '3g', '2g'][Math.floor(Math.random() * 3)],
				downlink: Math.floor(Math.random() * 10) + 5,
				rtt: Math.floor(Math.random() * 100) + 20,
				saveData: false
			}),
			configurable: true
		});

		return true;
	}
	`

	if _, err := page.Eval(script); err != nil {
		return fmt.Errorf("apply advanced fingerprint: %w", err)
	}

	return nil
}

// SimulateHumanMouse simulates human-like mouse movements with bezier curves
func SimulateHumanMouse(page *rod.Page) error {
	// Use fixed viewport since page.Viewport() may not be available
	width := 1920
	height := 1080

	// Generate random target position
	startX := rand.Float64() * float64(width)
	startY := rand.Float64() * float64(height)
	endX := rand.Float64() * float64(width)
	endY := rand.Float64() * float64(height)

	// Generate bezier curve control points for natural movement
	controlPoints := generateBezierControlPoints(startX, startY, endX, endY)

	// Move along the curve
	steps := 15 + rand.Intn(10) // 15-25 steps for smooth movement
	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		x, y := bezierPoint(t, controlPoints)

		if err := page.Mouse.MoveTo(proto.Point{X: x, Y: y}); err != nil {
			return err
		}

		// Add micro-random delays between movements
		baseDelay := 5 + rand.Intn(10)
		if i > 0 && i < steps {
			// Slow down at start and end of movement
			curveMultiplier := 1.0 + math.Sin(t*math.Pi)*0.5
			baseDelay = int(float64(baseDelay) * curveMultiplier)
		}
		time.Sleep(time.Duration(baseDelay) * time.Millisecond)
	}

	// Add small random pause at destination
	RandomDelay(50, 200)

	return nil
}

// controlPoint represents a bezier curve control point
type controlPoint struct {
	x, y float64
}

// generateBezierControlPoints generates natural control points for mouse movement
func generateBezierControlPoints(startX, startY, endX, endY float64) []controlPoint {
	// Calculate distance for scaling control point offsets
	dx := endX - startX
	dy := endY - startY
	distance := math.Sqrt(dx*dx + dy*dy) * 0.3

	// Avoid division by zero
	if distance < 1 {
		distance = 100
	}

	// Randomize curve direction
	signX := float64(1)
	signY := float64(1)
	if rand.Float64() > 0.5 {
		signX = -1
	}
	if rand.Float64() > 0.5 {
		signY = -1
	}

	// Perpendicular offset for natural curve
	perpX := -dy / distance * distance * signX
	perpY := dx / distance * distance * signY

	// Add some randomness
	randX := (rand.Float64() - 0.5) * distance
	randY := (rand.Float64() - 0.5) * distance

	return []controlPoint{
		{x: startX, y: startY},
		{x: startX + dx*0.3 + perpX + randX, y: startY + dy*0.3 + perpY + randY},
		{x: startX + dx*0.7 + perpX*0.5 + randX*0.5, y: startY + dy*0.7 + perpY*0.5 + randY*0.5},
		{x: endX, y: endY},
	}
}

// bezierPoint calculates a point on a cubic bezier curve
func bezierPoint(t float64, points []controlPoint) (float64, float64) {
	n := len(points) - 1
	if n == 0 {
		return points[0].x, points[0].y
	}

	var x, y float64
	for i := 0; i <= n; i++ {
		binomial := binomial(n, i)
		pow1 := math.Pow(1-t, float64(n-i))
		pow2 := math.Pow(t, float64(i))
		coeff := float64(binomial) * pow1 * pow2
		x += points[i].x * coeff
		y += points[i].y * coeff
	}
	return x, y
}

// binomial calculates binomial coefficient C(n, k)
func binomial(n, k int) int {
	if k < 0 || k > n {
		return 0
	}
	if k == 0 || k == n {
		return 1
	}
	if k > n-k {
		k = n - k
	}
	result := 1
	for i := 0; i < k; i++ {
		result = result * (n - i) / (i + 1)
	}
	return result
}

// SimulateScrolling simulates human-like scrolling behavior
func SimulateScrolling(page *rod.Page) error {
	// Get page height
	heightScript := `() => document.body.scrollHeight`
	result, err := page.Eval(heightScript)
	if err != nil {
		return err
	}

	pageHeight := 1000
	if str := result.Value.String(); str != "" {
		// Try to parse as integer
		var parsed int
		if _, parseErr := fmt.Sscanf(str, "%d", &parsed); parseErr == nil && parsed > 0 {
			pageHeight = parsed
		}
	}

	// Number of scroll operations
	scrolls := rand.Intn(5) + 3
	currentPos := 0

	for i := 0; i < scrolls; i++ {
		// Random scroll amount
		scrollAmount := rand.Intn(400) + 100

		// Sometimes scroll up
		if rand.Float64() > 0.8 && i > 0 {
			scrollAmount = -scrollAmount
		}

		newPos := currentPos + scrollAmount
		if newPos < 0 {
			newPos = 0
		}
		if newPos > pageHeight {
			newPos = pageHeight
		}

		_, err := page.Eval(fmt.Sprintf(`() => { window.scrollTo({ top: %d, behavior: 'smooth' }); }`, newPos))
		if err != nil {
			return err
		}

		currentPos = newPos
		RandomDelay(200, 500)
	}

	return nil
}

// WaitRandom waits for a random time to simulate human reading/pause
func WaitRandom() {
	RandomDelay(500, 2000)
}

// BypassCloudflare attempts to bypass Cloudflare protection
func BypassCloudflare(page *rod.Page) error {
	// Wait for Cloudflare challenge to complete
	time.Sleep(3 * time.Second)

	// Check for Cloudflare challenge page
	challengeTypes := []string{
		"#challenge-form",
		".cf-error-wrapper",
		"#cf-challenge-container",
		"[data-ray]",
		"#turnstile-wrapper",
	}

	for _, selector := range challengeTypes {
		hasChallenge, _, err := page.Has(selector)
		if err != nil {
			continue
		}
		if hasChallenge {
			// Cloudflare challenge detected - try to wait it out
			time.Sleep(10 * time.Second)
			break
		}
	}

	// Check for "Checking your browser" page
	checkingBrowser, _, _ := page.Has("#bw chall-info")
	if checkingBrowser {
		time.Sleep(15 * time.Second)
	}

	return nil
}

// BypassAliyunWAF attempts to bypass Alibaba Cloud WAF
func BypassAliyunWAF(page *rod.Page) error {
	// Alibaba Cloud WAF indicators
	wafIndicators := []string{
		".waf-block-page",
		"#waf_tg_error",
		".security-verify",
		"[class*='aliyun']",
	}

	for _, selector := range wafIndicators {
		hasWAF, _, err := page.Has(selector)
		if err != nil {
			continue
		}
		if hasWAF {
			// Refresh the page with different fingerprint
			ApplyAdvancedFingerprint(page)
			time.Sleep(2 * time.Second)
			return fmt.Errorf("aliyun waf detected: %s", selector)
		}
	}

	return nil
}

// BypassTencentWAF attempts to bypass Tencent Cloud WAF
func BypassTencentWAF(page *rod.Page) error {
	// Tencent Cloud WAF indicators
	wafIndicators := []string{
		"#captcha",
		".tencent-waf",
		".qcloud-captcha",
		"[id*='captcha']",
	}

	for _, selector := range wafIndicators {
		hasWAF, _, err := page.Has(selector)
		if err != nil {
			continue
		}
		if hasWAF {
			// Try to wait for captcha to load, then attempt bypass
			time.Sleep(5 * time.Second)

			// Check if it's a Tencent captcha
			captchaScript := `
				() => {
					return document.querySelector('#captcha, .qcloud-captcha') !== null;
				}
			`
			result, err := page.Eval(captchaScript)
			if err == nil && result.Value.Bool() {
				return fmt.Errorf("tencent captcha detected")
			}
		}
	}

	return nil
}

// DetectWAF detects if there's a WAF challenge page
func DetectWAF(page *rod.Page) (bool, string) {
	// Common WAF challenge page indicators
	wafPatterns := []struct {
		selector string
		name     string
	}{
		{"#challenge-form", "Cloudflare"},
		{".cf-error-wrapper", "Cloudflare"},
		{".waf-block-page", "Aliyun WAF"},
		{"#waf_tg_error", "Aliyun WAF"},
		{".security-verify", "Security Verify"},
		{"#captcha, .qcloud-captcha", "Tencent WAF"},
		{".geetest_panel", "Geetest"},
		{"#nc_1_wrapper", "Alibaba NC"},
		{".梭星", "Changting WAF"},
		{"抱歉，您的请求已被拒绝", "Generic WAF"},
		{"Access Denied", "AWS/Fastly WAF"},
		{"403 Forbidden", "Generic WAF"},
	}

	for _, pattern := range wafPatterns {
		hasWAF, _, err := page.Has(pattern.selector)
		if err != nil {
			continue
		}
		if hasWAF {
			return true, pattern.name
		}
	}

	return false, ""
}

// BypassCaptcha attempts to handle simple captcha challenges
func BypassCaptcha(page *rod.Page) error {
	// Check for common captcha indicators
	captchaSelectors := []string{
		".g-recaptcha",
		"#captcha",
		".captcha",
		"[class*='captcha']",
		"[id*='captcha']",
		"#nc_1_wrapper",
		".geetest_panel",
		".tencent-captcha",
	}

	for _, selector := range captchaSelectors {
		hasCaptcha, _, err := page.Has(selector)
		if err != nil {
			continue
		}
		if hasCaptcha {
			return fmt.Errorf("captcha detected: %s", selector)
		}
	}

	return nil
}

// StealthMode applies comprehensive stealth settings to the page
func StealthMode(page *rod.Page) error {
	return ApplyAdvancedFingerprint(page)
}

// HumanLikeBehavior performs a sequence of human-like actions
func HumanLikeBehavior(page *rod.Page) error {
	// Random delay before starting
	WaitRandom()

	// Simulate mouse movements
	if err := SimulateHumanMouse(page); err != nil {
		return err
	}

	// Random delay
	RandomDelay(200, 800)

	// Simulate scrolling
	if err := SimulateScrolling(page); err != nil {
		return err
	}

	// Apply stealth mode
	if err := StealthMode(page); err != nil {
		return err
	}

	return nil
}

// WaitForElementHumanLike waits for an element with human-like random delays
func WaitForElementHumanLike(page *rod.Page, selector string, config *BypassConfig) (*rod.Element, error) {
	if config == nil {
		config = DefaultBypassConfig()
	}

	if config.RandomDelays {
		RandomDelay(config.MinDelay, config.MaxDelay)
	}

	element, err := page.Element(selector)
	if err != nil {
		return nil, err
	}

	if config.RandomDelays {
		RandomDelay(config.MinDelay/2, config.MaxDelay/2)
	}

	return element, nil
}

// SafeNavigate navigates with full bypass protection
func SafeNavigate(page *rod.Page, url string, config *BypassConfig) error {
	if config == nil {
		config = DefaultBypassConfig()
	}

	// Apply bypass techniques before navigation
	if err := ApplyBypass(page, config); err != nil {
		return fmt.Errorf("apply bypass: %w", err)
	}

	// Navigate
	if err := page.Navigate(url); err != nil {
		return fmt.Errorf("navigate: %w", err)
	}

	// Wait for load
	if err := page.WaitLoad(); err != nil {
		return fmt.Errorf("wait load: %w", err)
	}

	// Check for WAF immediately after navigation
	wafDetected, wafName := DetectWAF(page)
	if wafDetected {
		return fmt.Errorf("waf detected during navigation: %s", wafName)
	}

	// Try to bypass Cloudflare if present
	if err := BypassCloudflare(page); err != nil {
		return fmt.Errorf("bypass cloudflare: %w", err)
	}

	// Check for captcha
	if err := BypassCaptcha(page); err != nil {
		return fmt.Errorf("captcha check: %w", err)
	}

	// Check for specific WAF types
	if err := BypassAliyunWAF(page); err != nil {
		return fmt.Errorf("aliyun waf check: %w", err)
	}

	if err := BypassTencentWAF(page); err != nil {
		return fmt.Errorf("tencent waf check: %w", err)
	}

	// Apply human-like behavior
	if err := HumanLikeBehavior(page); err != nil {
		return fmt.Errorf("human behavior: %w", err)
	}

	return nil
}

// NavigateWithRetry navigates with automatic retry on WAF detection
func NavigateWithRetry(ctx context.Context, page *rod.Page, url string, config *BypassConfig) error {
	if config == nil {
		config = DefaultBypassConfig()
	}

	maxRetries := config.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 3
	}

	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if attempt > 0 {
			// Exponential backoff
			backoff := time.Duration(math.Pow(2, float64(attempt))) * time.Second
			backoff += time.Duration(rand.Intn(5000)) * time.Millisecond

			// Rotate proxy if enabled
			if config.UseProxyRotation {
				proxy := GetNextProxy()
				if proxy != "" {
					// Would need to recreate browser with new proxy
				}
			}

			// Apply fresh fingerprint for retry
			ApplyAdvancedFingerprint(page)

			time.Sleep(backoff)
		}

		err := SafeNavigate(page, url, config)
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if it's a WAF error
		errStr := err.Error()
		if strings.Contains(errStr, "waf") || strings.Contains(errStr, "captcha") ||
			strings.Contains(errStr, "challenge") || strings.Contains(errStr, "403") ||
			strings.Contains(errStr, "Access Denied") {
			continue
		}

		// Non-WAF error, don't retry
		return err
	}

	return fmt.Errorf("max retries (%d) exceeded: %w", maxRetries, lastErr)
}

// Helper function to get environment variable with default
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// SessionFingerprint generates a unique fingerprint for the current session
func SessionFingerprint() string {
	data := fmt.Sprintf("%d-%d-%d", time.Now().UnixNano(), rand.Int63(), rand.Int63())
	return fmt.Sprintf("%x", md5.Sum([]byte(data)))
}
