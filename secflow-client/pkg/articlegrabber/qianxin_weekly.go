// Package articlegrabber provides Qianxin weekly report crawler using go-rod + API.
package articlegrabber

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/kataras/golog"
	"github.com/imroc/req/v3"
	md "github.com/JohannesKaufmann/html-to-markdown"
)

// QianxinWeeklyGrabber fetches Qianxin weekly reports via API + rod detail fetching.
type QianxinWeeklyGrabber struct {
	*RodCrawler
	client  *req.Client
	apiURL  string
	cookies []*http.Cookie
}

// NewQianxinWeeklyGrabber creates a new Qianxin weekly report crawler.
func NewQianxinWeeklyGrabber() *QianxinWeeklyGrabber {
	return &QianxinWeeklyGrabber{
		RodCrawler: NewRodCrawler("qianxin-weekly", golog.Child("[qianxin-weekly]")),
		client: req.NewClient().
			SetTimeout(60 * time.Second).
			SetUserAgent("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/146.0.0.0 Safari/537.36"),
		apiURL: "https://ti.qianxin.com/alpha-api/v2/vuln/article-hotspot",
	}
}

// Name returns the grabber name.
func (g *QianxinWeeklyGrabber) Name() string {
	return "qianxin-weekly"
}

// Fetch retrieves Qianxin weekly security reports via API + rod detail fetching.
func (g *QianxinWeeklyGrabber) Fetch(ctx context.Context, limit int) ([]*Article, error) {
	// First, visit the page with rod to get cookies and apply bypass
	if err := g.initSession(ctx); err != nil {
		g.LogWarn("Failed to init session with rod: %v", err)
		// Continue anyway, sometimes API works without cookies
	}

	// Calculate pages needed (each page returns up to 10 items)
	pageSize := 10
	pageNo := 1
	totalFetched := 0

	// Phase 1: Fetch article list from API
	articleList := make([]*apiArticleItem, 0, limit)

	for totalFetched < limit {
		remaining := limit - totalFetched
		if remaining < pageSize {
			pageSize = remaining
		}

		g.LogInfo("Fetching article list page %d (page_size=%d)", pageNo, pageSize)

		items, err := g.fetchArticleList(ctx, pageNo, pageSize)
		if err != nil {
			g.LogWarn("Failed to fetch page %d: %v", pageNo, err)
			break
		}

		if len(items) == 0 {
			g.LogInfo("No more items on page %d", pageNo)
			break
		}

		articleList = append(articleList, items...)
		totalFetched += len(items)

		if len(items) < pageSize {
			break // No more pages
		}

		pageNo++
	}

	g.LogInfo("Fetched %d article items from API list", len(articleList))

	// Phase 2: Fetch detail content for each article using rod
	results := make([]*Article, 0, len(articleList))
	for i, item := range articleList {
		select {
		case <-ctx.Done():
			g.LogInfo("Context cancelled, stopping at %d articles", i)
			break
		default:
		}

		g.LogInfo("[%d/%d] Fetching article detail for: %s (ID: %d)", i+1, len(articleList), item.Title, item.ID)

		// Fetch full content using rod
		fullContent, err := g.fetchArticleDetail(ctx, item.ID)
		if err != nil {
			g.LogWarn("Failed to fetch detail for article %d: %v, using digest as fallback", item.ID, err)
			fullContent = item.Digest
		}

		article := g.convertToArticle(item, fullContent)
		if article != nil {
			results = append(results, article)
		}

		// Rate limit between requests
		time.Sleep(500 * time.Millisecond)
	}

	g.LogInfo("Successfully processed %d articles with full content", len(results))
	return results, nil
}

// initSession visits the main page with rod to establish a session and get cookies.
func (g *QianxinWeeklyGrabber) initSession(ctx context.Context) error {
	page, cleanup, err := g.CreatePage(ctx)
	if err != nil {
		return fmt.Errorf("create page: %w", err)
	}
	defer cleanup()

	baseURL := "https://ti.qianxin.com"
	url := baseURL + "/vulnerability/notice-list"

	g.LogInfo("Visiting page to establish session: %s", url)

	// Navigate with bypass protection
	if err := g.Navigate(page, url); err != nil {
		g.LogWarn("Navigation warning (may continue): %v", err)
	}

	// Wait for page to stabilize
	time.Sleep(2 * time.Second)

	// Try to extract cookies from the page
	g.cookies = g.extractCookies(page)

	g.LogDebug("Session initialized, got %d cookies", len(g.cookies))
	return nil
}

// extractCookies extracts cookies from the rod page session.
func (g *QianxinWeeklyGrabber) extractCookies(page *rod.Page) []*http.Cookie {
	// Get cookies via JavaScript execution
	script := `() => {
		return document.cookie.split(';').map(c => {
			const [name, ...valueParts] = c.trim().split('=');
			return { name: name.trim(), value: valueParts.join('=').trim() };
		});
	}`

	result, err := page.Eval(script)
	if err != nil {
		g.LogDebug("Failed to get cookies: %v", err)
		return nil
	}

	var cookieStrs []map[string]string
	if err := json.Unmarshal([]byte(result.Value.String()), &cookieStrs); err != nil {
		g.LogDebug("Failed to parse cookies: %v", err)
		return nil
	}

	cookies := make([]*http.Cookie, 0, len(cookieStrs))
	for _, cs := range cookieStrs {
		if name, ok := cs["name"]; ok {
			cookies = append(cookies, &http.Cookie{
				Name:  name,
				Value: cs["value"],
			})
		}
	}

	return cookies
}

// fetchArticleList fetches article list from the API.
func (g *QianxinWeeklyGrabber) fetchArticleList(ctx context.Context, pageNo, pageSize int) ([]*apiArticleItem, error) {
	// Build request body
	reqBody := map[string]interface{}{
		"page_no":   pageNo,
		"page_size": pageSize,
		"category":  "热点周报",
	}

	resp, err := g.client.R().
		SetContext(ctx).
		SetHeader("Accept", "application/json, text/plain, */*").
		SetHeader("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8").
		SetHeader("Origin", "https://ti.qianxin.com").
		SetHeader("Referer", "https://ti.qianxin.com/vulnerability/notice-list").
		SetHeader("Content-Type", "application/json").
		SetBody(reqBody).
		Post(g.apiURL)

	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	if !resp.IsSuccess() {
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(resp.Bytes()))
	}

	var apiResp apiResponse
	if err := json.Unmarshal(resp.Bytes(), &apiResp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	if apiResp.Status != 10000 {
		return nil, fmt.Errorf("API returned status %d: %s", apiResp.Status, apiResp.Message)
	}

	return apiResp.Data.Data, nil
}

// fetchArticleDetail fetches the full article content using rod.
// This navigates to the detail page and extracts the complete article content.
func (g *QianxinWeeklyGrabber) fetchArticleDetail(ctx context.Context, articleID int) (string, error) {
	// Create a dedicated page for detail fetching
	page, cleanup, err := g.CreatePage(ctx)
	if err != nil {
		return "", fmt.Errorf("create page: %w", err)
	}
	defer cleanup()

	// Build detail URL
	baseURL := "https://ti.qianxin.com"
	detailURL := fmt.Sprintf("%s/vulnerability/notice-detail/%d?type=hot-week", baseURL, articleID)

	g.LogDebug("Navigating to article detail: %s", detailURL)

	// Navigate to detail page with bypass
	if err := g.NavigateWithContext(ctx, page, detailURL); err != nil {
		return "", fmt.Errorf("navigate to detail page: %w", err)
	}

	// Wait for content to load
	time.Sleep(2 * time.Second)

	// Extract article content using JavaScript
	// Priority: innerHTML (preserves structure) > textContent (plain text)
	script := `() => {
		// Most specific selectors for article content first
		const selectors = [
			'#poc-preview',
			'#poc-preview .content',
			'.article .content',
			'.article-content',
			'.notice-detail .content',
			'.detail-container .article'
		];

		let html = '';
		let text = '';

		for (const selector of selectors) {
			const el = document.querySelector(selector);
			if (el) {
				const innerHTML = el.innerHTML || '';
				const textContent = el.textContent || '';
				if (innerHTML.length > 500) {
					// Check if it has structured content (h1, h2, p tags)
					if (innerHTML.includes('<h1') || innerHTML.includes('<h2') || innerHTML.includes('<p>')) {
						html = innerHTML;
						text = textContent.trim();
						break;
					}
				}
			}
		}

		// Fallback: look for div with data-v-* containing article content
		if (!html || html.length < 200) {
			const allDivs = document.querySelectorAll('div[data-v-]');
			for (const div of allDivs) {
				const innerHTML = div.innerHTML || '';
				if (innerHTML.length > 1000 && (innerHTML.includes('<h1>') || innerHTML.includes('<h2>'))) {
					html = innerHTML;
					text = div.textContent.trim();
					break;
				}
			}
		}

		// Get title
		const titleEl = document.querySelector('.detail-title, .notice-detail h1, article h1, .article h1');
		const title = titleEl ? titleEl.textContent.trim() : '';

		return JSON.stringify({
			title: title,
			html: html,
			text: text
		});
	}`

	result, err := page.Eval(script)
	if err != nil {
		return "", fmt.Errorf("execute script: %w", err)
	}

	var detailData struct {
		Title string `json:"title"`
		HTML  string `json:"html"`
		Text  string `json:"text"`
	}

	if err := json.Unmarshal([]byte(result.Value.String()), &detailData); err != nil {
		return "", fmt.Errorf("parse detail data: %w", err)
	}

	// Prefer HTML if available and has substantial content, otherwise use text
	content := detailData.HTML
	if content == "" || len(content) < 200 {
		content = detailData.Text
	}

	if content == "" {
		return "", fmt.Errorf("no content extracted from detail page")
	}

	// Convert HTML to Markdown
	md := g.htmlToMarkdown(content)

	g.LogDebug("Extracted %d chars of content for article %d", len(md), articleID)
	return md, nil
}

// htmlToMarkdown converts HTML content to Markdown format.
func (g *QianxinWeeklyGrabber) htmlToMarkdown(html string) string {
	if html == "" {
		return ""
	}

	// Clean up Vue data-v-* attributes
	html = cleanVueAttributes(html)

	// Use html-to-markdown library
	converter := md.NewConverter("", true, nil)
	mdContent, err := converter.ConvertString(html)
	if err != nil {
		g.LogWarn("Failed to convert HTML to markdown: %v", err)
		return g.stripHTML(html)
	}

	return mdContent
}

// cleanVueAttributes removes Vue-specific data-v-* attributes and unwanted tags from HTML.
func cleanVueAttributes(html string) string {
	// Remove data-v-xxxxx attributes from tags
	re := regexp.MustCompile(`\s+data-v-[a-zA-Z0-9]+=""`)
	html = re.ReplaceAllString(html, "")

	// Remove data-v-xxxxx="..." patterns more broadly
	re2 := regexp.MustCompile(`\s+data-v-[a-zA-Z0-9]+="[^"]*"`)
	html = re2.ReplaceAllString(html, "")

	// Remove span tags with inline styles (subscription notices, etc.)
	// Pattern matches <span style="...">...</span>
	re3 := regexp.MustCompile(`<span\s+style="[^"]*">[^<]*</span>`)
	html = re3.ReplaceAllString(html, "")

	// Also remove empty paragraph tags and nbsp paragraphs
	re4 := regexp.MustCompile(`<p>&nbsp;</p>`)
	html = re4.ReplaceAllString(html, "")

	re5 := regexp.MustCompile(`<p>\s*</p>`)
	html = re5.ReplaceAllString(html, "")

	return html
}

// stripHTML removes HTML tags from content.
func (g *QianxinWeeklyGrabber) stripHTML(html string) string {
	// Simple HTML tag removal
	re := strings.NewReplacer(
		"<br>", "\n",
		"<br/>", "\n",
		"<br />", "\n",
		"</p>", "\n",
		"</div>", "\n",
		"</li>", "\n",
		"</h1>", "\n",
		"</h2>", "\n",
		"</h3>", "\n",
		"</h4>", "\n",
	)

	result := re.Replace(html)

	// Remove all remaining HTML tags
	inTag := false
	var clean strings.Builder
	for _, r := range result {
		if r == '<' {
			inTag = true
			continue
		}
		if r == '>' {
			inTag = false
			continue
		}
		if !inTag {
			clean.WriteRune(r)
		}
	}

	// Clean up whitespace
	lines := strings.Split(clean.String(), "\n")
	var finalLines []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			finalLines = append(finalLines, trimmed)
		}
	}

	return strings.Join(finalLines, "\n\n")
}

// convertToArticle converts an API article item + full content to our Article model.
func (g *QianxinWeeklyGrabber) convertToArticle(item *apiArticleItem, fullContent string) *Article {
	if item == nil || item.Title == "" {
		return nil
	}

	// Build detail URL from article ID
	baseURL := "https://ti.qianxin.com"
	url := fmt.Sprintf("%s/vulnerability/notice-detail/%d?type=hot-week", baseURL, item.ID)

	// Parse published date
	publishedAt := parseAPIDate(item.UpdateTime)

	// Download cover image and convert to base64
	var image string
	if item.Cover != "" {
		image = g.downloadImageAsBase64(item.Cover)
		if image == "" {
			image = item.Cover // Fall back to original URL
		}
	}

	// Use full content if available, otherwise fallback to digest
	content := fullContent
	if content == "" {
		content = item.Digest
	}

	// Format digest as summary (bullet points)
	summary := item.Digest
	if summary != "" {
		// Replace bullet points with newlines for readability
		summary = strings.ReplaceAll(summary, "•", "\n•")
	}

	return &Article{
		Title:       item.Title,
		Summary:     summary,
		Content:     content,
		Author:      item.Author,
		Source:      "奇安信热点周报",
		URL:         url,
		Image:       image,
		Tags:        []string{"热点周报", "奇安信", "安全资讯"},
		PublishedAt: publishedAt,
	}
}

// downloadImageAsBase64 downloads an image from URL and returns it as base64-encoded data URI.
// Returns empty string on failure.
func (g *QianxinWeeklyGrabber) downloadImageAsBase64(imageURL string) string {
	if imageURL == "" {
		return ""
	}

	resp, err := g.client.R().
		SetHeader("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36").
		SetHeader("Accept", "image/webp,image/apng,image/*,*/*;q=0.8").
		SetHeader("Referer", "https://ti.qianxin.com/").
		Get(imageURL)

	if err != nil {
		g.LogDebug("Failed to download image from %s: %v", imageURL, err)
		return ""
	}
	defer resp.Body.Close()

	if !resp.IsSuccess() {
		g.LogDebug("Image download failed with status %d: %s", resp.StatusCode, imageURL)
		return ""
	}

	// Determine content type
	contentType := resp.GetHeader("Content-Type")
	if contentType == "" {
		contentType = "image/jpeg"
	}

	// Convert to base64 with data URI prefix
	base64Data := base64.StdEncoding.EncodeToString(resp.Bytes())
	dataURI := fmt.Sprintf("data:%s;base64,%s", contentType, base64Data)

	return dataURI
}

// parseAPIDate parses the date string from API response.
func parseAPIDate(dateStr string) time.Time {
	if dateStr == "" {
		return time.Now()
	}

	// API returns: "2025-12-22 10:56:29"
	layouts := []string{
		"2006-01-02 15:04:05",
		"2006-01-02",
	}

	for _, layout := range layouts {
		if t, err := time.Parse(layout, dateStr); err == nil {
			return t
		}
	}

	return time.Now()
}

// API response structures

type apiResponse struct {
	Status  int             `json:"status"`
	Message string          `json:"message"`
	Data    apiResponseData `json:"data"`
}

type apiResponseData struct {
	Total int               `json:"total"`
	Data  []*apiArticleItem `json:"data"`
}

type apiArticleItem struct {
	ID         int    `json:"id"`
	Title      string `json:"title"`
	Author     string `json:"author"`
	Category   string `json:"category"`
	Digest     string `json:"digest"`
	Cover      string `json:"cover"`
	ReadNum    int    `json:"read_num"`
	UpdateTime string `json:"update_time"`
}

func init() {
	Register(NewQianxinWeeklyGrabber())
}