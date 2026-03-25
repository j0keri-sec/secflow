// Package rodutil provides helper functions for common scraping operations.
package rodutil

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
)

// WaitOptions contains options for wait operations.
type WaitOptions struct {
	Timeout       time.Duration
	RetryInterval time.Duration
}

// DefaultWaitOptions returns default wait options.
func DefaultWaitOptions() *WaitOptions {
	return &WaitOptions{
		Timeout:       30 * time.Second,
		RetryInterval: 500 * time.Millisecond,
	}
}

// WaitForElement waits for an element to appear on the page.
func WaitForElement(page *rod.Page, selector string, opts *WaitOptions) (*rod.Element, error) {
	if opts == nil {
		opts = DefaultWaitOptions()
	}

	element, err := page.Element(selector)
	if err != nil {
		return nil, fmt.Errorf("wait for element %s: %w", selector, err)
	}

	return element, nil
}

// WaitForElements waits for multiple elements to appear.
func WaitForElements(page *rod.Page, selector string, opts *WaitOptions) (rod.Elements, error) {
	if opts == nil {
		opts = DefaultWaitOptions()
	}

	elements, err := page.Elements(selector)
	if err != nil {
		return nil, fmt.Errorf("wait for elements %s: %w", selector, err)
	}

	return elements, nil
}

// ScrollToBottom scrolls the page to the bottom with human-like behavior.
func ScrollToBottom(page *rod.Page) error {
	heightScript := `() => document.body.scrollHeight`
	result, err := page.Eval(heightScript)
	if err != nil {
		return err
	}

	pageHeight := 2000
	if str := result.Value.String(); str != "" {
		var parsed int
		if _, parseErr := fmt.Sscanf(str, "%d", &parsed); parseErr == nil && parsed > 0 {
			pageHeight = parsed
		}
	}

	scrollStep := 300 + rand.Intn(200)
	currentPos := 0

	for currentPos < pageHeight {
		newPos := currentPos + scrollStep
		if newPos > pageHeight {
			newPos = pageHeight
		}

		_, err := page.Eval(fmt.Sprintf(`() => { window.scrollTo({ top: %d, behavior: 'smooth' }); }`, newPos))
		if err != nil {
			return err
		}

		currentPos = newPos
		RandomDelay(300, 800)

		if rand.Float64() > 0.7 {
			WaitRandom()
		}
	}

	return nil
}

// ScrollToElement scrolls to make an element visible.
func ScrollToElement(element *rod.Element) error {
	return element.ScrollIntoView()
}

// GetText gets the text content of an element.
func GetText(element *rod.Element) (string, error) {
	if element == nil {
		return "", nil
	}
	text, err := element.Text()
	if err != nil {
		return "", fmt.Errorf("get text: %w", err)
	}
	return strings.TrimSpace(text), nil
}

// GetAttribute gets an attribute value from an element.
func GetAttribute(element *rod.Element, attr string) (string, error) {
	if element == nil {
		return "", nil
	}
	val, err := element.Attribute(attr)
	if err != nil {
		return "", fmt.Errorf("get attribute %s: %w", attr, err)
	}
	if val == nil {
		return "", nil
	}
	return *val, nil
}

// GetHref gets the href attribute from an element.
func GetHref(element *rod.Element) (string, error) {
	return GetAttribute(element, "href")
}

// GetSrc gets the src attribute from an element.
func GetSrc(element *rod.Element) (string, error) {
	return GetAttribute(element, "src")
}

// Click clicks on an element and waits for navigation if needed.
func Click(element *rod.Element, waitForNav bool) error {
	RandomDelay(50, 150)

	if waitForNav {
		page := element.Page()
		wait := page.WaitNavigation(proto.PageLifecycleEventNameNetworkAlmostIdle)
		if err := element.Click(proto.InputMouseButtonLeft, 1); err != nil {
			return fmt.Errorf("click element: %w", err)
		}
		wait()
	} else {
		if err := element.Click(proto.InputMouseButtonLeft, 1); err != nil {
			return fmt.Errorf("click element: %w", err)
		}
	}

	RandomDelay(100, 300)
	return nil
}

// ClickWithMove clicks on an element after moving mouse there naturally.
func ClickWithMove(element *rod.Element, page *rod.Page, waitForNav bool) error {
	// Use javascript to get element position since element.Box() may not be available
	script := `
		() => {
			const el = document.querySelector ? document.querySelector(arguments[0]) : null;
			if (!el) return null;
			const rect = el.getBoundingClientRect();
			return {
				x: rect.x + rect.width / 2,
				y: rect.y + rect.height / 2,
				width: rect.width,
				height: rect.height
			};
		}
	`

	result, err := page.Eval(fmt.Sprintf(`%s`, script))
	if err != nil {
		// Fallback: just click the element
		return element.Click(proto.InputMouseButtonLeft, 1)
	}
	
	var pos struct {
		X      float64 `json:"x"`
		Y      float64 `json:"y"`
		Width  float64 `json:"width"`
		Height float64 `json:"height"`
	}
	
	if err := json.Unmarshal([]byte(result.Value.String()), &pos); err != nil {
		return element.Click(proto.InputMouseButtonLeft, 1)
	}

	// Calculate click position with small random offset
	clickX := pos.X + (rand.Float64()-0.5)*10
	clickY := pos.Y + (rand.Float64()-0.5)*10

	// Move mouse with bezier curve
	startX := rand.Float64() * 800
	startY := rand.Float64() * 600

	points := generateBezierControlPoints(startX, startY, clickX, clickY)
	steps := 10 + rand.Intn(5)

	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		x, y := bezierPoint(t, points)
		if err := page.Mouse.MoveTo(proto.Point{X: x, Y: y}); err != nil {
			return err
		}
		RandomDelay(5, 15)
	}

	RandomDelay(50, 150)
	return element.Click(proto.InputMouseButtonLeft, 1)
}

// Fill fills an input element with human-like typing behavior.
func Fill(element *rod.Element, text string) error {
	if err := element.SelectAllText(); err != nil {
		if _, err := element.Eval(`(el) => { el.value = ''; }`); err != nil {
			return fmt.Errorf("clear input: %w", err)
		}
	}

	for i, r := range text {
		char := string(r)
		if err := element.Input(char); err != nil {
			return fmt.Errorf("type character: %w", err)
		}

		baseDelay := 30 + rand.Intn(120)
		if rand.Float64() > 0.95 {
			baseDelay += 200 + rand.Intn(300)
		}
		if len(text) > 50 {
			baseDelay = int(float64(baseDelay) * 0.7)
		}
		if i < len(text)-1 && (text[i:i+1] == " " || text[i:i+1] == ".") {
			baseDelay += 50 + rand.Intn(100)
		}

		time.Sleep(time.Duration(baseDelay) * time.Millisecond)
	}

	RandomDelay(100, 300)
	return nil
}

// Clear clears an input element.
func Clear(element *rod.Element) error {
	if err := element.SelectAllText(); err != nil {
		if _, err := element.Eval(`(el) => { el.value = ''; }`); err != nil {
			return fmt.Errorf("clear input: %w", err)
		}
	}
	if err := element.Input(""); err != nil {
		return fmt.Errorf("clear input: %w", err)
	}
	return nil
}

// WaitForNetworkIdle waits for network to be idle.
func WaitForNetworkIdle(page *rod.Page, idleTime time.Duration) error {
	return page.WaitIdle(idleTime)
}

// GetPageSource gets the current page HTML source.
func GetPageSource(page *rod.Page) (string, error) {
	html, err := page.HTML()
	if err != nil {
		return "", fmt.Errorf("get page source: %w", err)
	}
	return html, nil
}

// ExecuteScript executes JavaScript on the page.
func ExecuteScript(page *rod.Page, script string) (interface{}, error) {
	result, err := page.Eval(script)
	if err != nil {
		return nil, fmt.Errorf("execute script: %w", err)
	}
	return result.Value, nil
}

// ExecuteScriptWithResult executes JavaScript and unmarshals the result.
func ExecuteScriptWithResult(page *rod.Page, script string, dest interface{}) error {
	result, err := page.Eval(script)
	if err != nil {
		return fmt.Errorf("execute script: %w", err)
	}

	jsonData, err := json.Marshal(result.Value)
	if err != nil {
		return fmt.Errorf("marshal result: %w", err)
	}

	if err := json.Unmarshal(jsonData, dest); err != nil {
		return fmt.Errorf("unmarshal result: %w", err)
	}

	return nil
}

// BlockResources blocks common tracking and ad resources.
func BlockResources(page *rod.Page, patterns ...string) error {
	defaultPatterns := []string{
		"*.png", "*.jpg", "*.jpeg", "*.gif", "*.webp",
		"*.css", "*.woff", "*.woff2", "*.ttf",
		"*google-analytics.com*", "*googletagmanager.com*",
		"*doubleclick.net*", "*facebook.com*",
		"*baidu.com*", "*cnzz.com*", "*51.la*",
	}

	if len(patterns) > 0 {
		defaultPatterns = patterns
	}

	router := page.HijackRequests()
	for _, pattern := range defaultPatterns {
		router.MustAdd(pattern, func(ctx *rod.Hijack) {
			ctx.Response.Fail(proto.NetworkErrorReasonBlockedByClient)
		})
	}

	go router.Run()
	return nil
}

// SetUserAgent sets a custom user agent for the page.
func SetUserAgent(page *rod.Page, userAgent string) error {
	if err := page.SetUserAgent(&proto.NetworkSetUserAgentOverride{
		UserAgent: userAgent,
	}); err != nil {
		return fmt.Errorf("set user agent: %w", err)
	}
	return nil
}

// SetViewport sets the viewport size.
func SetViewport(page *rod.Page, width, height int) error {
	if err := page.SetViewport(&proto.EmulationSetDeviceMetricsOverride{
		Width:  width,
		Height: height,
	}); err != nil {
		return fmt.Errorf("set viewport: %w", err)
	}
	return nil
}

// WaitForElementWithContext waits for an element with context cancellation.
func WaitForElementWithContext(ctx context.Context, page *rod.Page, selector string) (*rod.Element, error) {
	done := make(chan struct {
		elem *rod.Element
		err  error
	}, 1)

	go func() {
		elem, err := page.Element(selector)
		done <- struct {
			elem *rod.Element
			err  error
		}{elem, err}
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case result := <-done:
		return result.elem, result.err
	}
}

// IsElementVisible checks if an element is visible.
func IsElementVisible(element *rod.Element) bool {
	if element == nil {
		return false
	}
	visible, err := element.Visible()
	if err != nil {
		return false
	}
	return visible
}

// GetElementCount returns the number of elements matching the selector.
func GetElementCount(page *rod.Page, selector string) (int, error) {
	elements, err := page.Elements(selector)
	if err != nil {
		return 0, fmt.Errorf("get elements: %w", err)
	}
	return len(elements), nil
}

// Hover performs a human-like hover over an element.
func Hover(element *rod.Element, page *rod.Page) error {
	script := `
		() => {
			const el = this;
			const rect = el.getBoundingClientRect();
			return { x: rect.x + rect.width / 2, y: rect.y + rect.height / 2 };
		}
	`
	result, err := element.Eval(script)
	if err != nil {
		return err
	}

	var pos struct {
		X float64 `json:"x"`
		Y float64 `json:"y"`
	}
	if err := json.Unmarshal([]byte(result.Value.String()), &pos); err != nil {
		return err
	}

	// Move with bezier curve
	startX := rand.Float64() * 500
	startY := rand.Float64() * 500

	points := generateBezierControlPoints(startX, startY, pos.X, pos.Y)
	steps := 8 + rand.Intn(5)

	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		x, y := bezierPoint(t, points)
		if err := page.Mouse.MoveTo(proto.Point{X: x, Y: y}); err != nil {
			return err
		}
		RandomDelay(5, 12)
	}

	RandomDelay(200, 500)

	return nil
}

// TypeText types text with human-like timing variations.
func TypeText(element *rod.Element, text string) error {
	return Fill(element, text)
}

// HoverAndClick performs hover then click with human-like behavior.
func HoverAndClick(element *rod.Element, page *rod.Page, waitForNav bool) error {
	if err := Hover(element, page); err != nil {
		return err
	}
	return element.Click(proto.InputMouseButtonLeft, 1)
}

// IsPageBlocked checks if the page shows a WAF/blocking message.
func IsPageBlocked(page *rod.Page) bool {
	blockPatterns := []string{
		"403",
		"403 Forbidden",
		"Access Denied",
		"被拒绝",
		"访问被拒绝",
		"安全验证",
		"captcha",
		"验证码",
		"请输入验证码",
		"Too Many Requests",
		"请稍后再试",
		"waf",
		"blocked",
		"blocked by",
		"security check",
	}

	html, err := page.HTML()
	if err != nil {
		return false
	}

	for _, pattern := range blockPatterns {
		if strings.Contains(strings.ToLower(html), strings.ToLower(pattern)) {
			return true
		}
	}

	return false
}

// WaitForPageReady waits for page to be fully loaded with human-like behavior.
func WaitForPageReady(page *rod.Page) error {
	if err := page.WaitLoad(); err != nil {
		return err
	}

	RandomDelay(500, 1500)

	if err := SimulateScrolling(page); err != nil {
		return err
	}

	RandomDelay(300, 800)

	return nil
}

// ExtractLinks extracts all links from the page with optional filtering.
func ExtractLinks(page *rod.Page, filter func(url string) bool) ([]string, error) {
	script := `
		() => {
			const links = [];
			document.querySelectorAll('a[href]').forEach(a => {
				const href = a.href;
				if (href && href.startsWith('http')) {
					links.push(href);
				}
			});
			return links;
		}
	`

	result, err := page.Eval(script)
	if err != nil {
		return nil, fmt.Errorf("extract links: %w", err)
	}

	var links []string
	if err := json.Unmarshal([]byte(result.Value.String()), &links); err != nil {
		return nil, fmt.Errorf("parse links: %w", err)
	}

	if filter != nil {
		filtered := make([]string, 0)
		for _, link := range links {
			if filter(link) {
				filtered = append(filtered, link)
			}
		}
		return filtered, nil
	}

	return links, nil
}

// WaitForPagination waits for pagination to load with retry.
func WaitForPagination(page *rod.Page, containerSelector string, maxRetries int) (bool, error) {
	for i := 0; i < maxRetries; i++ {
		count, err := GetElementCount(page, containerSelector)
		if err == nil && count > 0 {
			return true, nil
		}
		RandomDelay(500, 1000)
	}
	return false, fmt.Errorf("pagination not found after %d retries", maxRetries)
}

// SimulateRead simulates reading behavior by pausing and scrolling.
func SimulateRead(page *rod.Page, readTimeSeconds int) error {
	totalPauses := readTimeSeconds / 3
	if totalPauses < 1 {
		totalPauses = 1
	}

	for i := 0; i < totalPauses; i++ {
		_, err := page.Eval(fmt.Sprintf(`() => { window.scrollBy(0, %d); }`, 100+rand.Intn(200)))
		if err != nil {
			return err
		}

		readDuration := 2000 + rand.Intn(3000)
		time.Sleep(time.Duration(readDuration) * time.Millisecond)
	}

	return nil
}

// GetCookies returns all cookies from the page.
func GetCookies(page *rod.Page) ([]*proto.NetworkCookie, error) {
	cookies, err := page.Cookies(nil)
	if err != nil {
		return nil, fmt.Errorf("get cookies: %w", err)
	}
	return cookies, nil
}

// SetCookies sets cookies on the page.
func SetCookies(page *rod.Page, cookies []*proto.NetworkCookie) error {
	// Convert NetworkCookie to NetworkCookieParam for SetCookies
	params := make([]*proto.NetworkCookieParam, len(cookies))
	for i, c := range cookies {
		params[i] = &proto.NetworkCookieParam{
			Name:     c.Name,
			Value:    c.Value,
			Domain:   c.Domain,
			Path:     c.Path,
			Secure:   c.Secure,
			HTTPOnly: c.HTTPOnly,
			SameSite: c.SameSite,
			Expires:  c.Expires,
		}
	}
	if err := page.SetCookies(params); err != nil {
		return fmt.Errorf("set cookies: %w", err)
	}
	return nil
}

// ClearCookies clears all cookies from the page.
func ClearCookies(page *rod.Page) error {
	script := `
		() => {
			document.cookie.split(";").forEach(function(c) {
				document.cookie = c.replace(/^ +/, "").replace(/=.*/, "=;expires=" + new Date().toUTCString() + ";path=/");
			});
		}
	`
	_, err := page.Eval(script)
	return err
}

// distance calculates the distance between two points
func distance(x1, y1, x2, y2 float64) float64 {
	dx := x2 - x1
	dy := y2 - y1
	return math.Sqrt(dx*dx + dy*dy)
}
