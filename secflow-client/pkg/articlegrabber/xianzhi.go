// Package articlegrabber provides crawler for xianzhi community (先知社区).
package articlegrabber

import (
	"context"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"github.com/kataras/golog"
	"github.com/imroc/req/v3"
	md "github.com/JohannesKaufmann/html-to-markdown"
)

// XianzhiGrabber fetches articles from xianzhi community (xz.aliyun.com) via RSS feed + Rod detail.
type XianzhiGrabber struct {
	*RodCrawler
	client *req.Client
}

// NewXianzhiGrabber creates a new xianzhi community article crawler.
func NewXianzhiGrabber() *XianzhiGrabber {
	return &XianzhiGrabber{
		RodCrawler: NewRodCrawler("xianzhi", golog.Child("[xianzhi]")),
		client: req.NewClient().
			SetTimeout(60 * time.Second).
			SetUserAgent("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/146.0.0.0 Safari/537.36"),
	}
}

// Name returns the grabber name.
func (g *XianzhiGrabber) Name() string {
	return "xianzhi"
}

// AtomFeed represents the RSS/Atom feed structure from xianzhi.
type AtomFeed struct {
	Title   string `xml:"title"`
	Updated string `xml:"updated"`
	Entries []AtomEntry `xml:"entry"`
}

// AtomEntry represents a single entry in the Atom feed.
type AtomEntry struct {
	Title     string `xml:"title"`
	Link      string `xml:"link"` // Custom unmarshal needed for attribute
	Published string `xml:"published"`
	Updated   string `xml:"updated"`
	Summary   string `xml:"summary"`
	ID        string `xml:"id"`
}

// GetLink returns the link URL from the entry.
func (e *AtomEntry) GetLink() string {
	if e.Link != "" {
		return e.Link
	}
	return e.ID
}

// Fetch retrieves articles from xianzhi community via RSS + detail scraping.
func (g *XianzhiGrabber) Fetch(ctx context.Context, limit int) ([]*Article, error) {
	articles := make([]*Article, 0)

	// Step 1: Fetch RSS feed for article list
	g.LogInfo("Fetching RSS feed from xianzhi community...")
	
	httpReq := g.client.R().SetContext(ctx)
	resp, err := httpReq.Get("https://xz.aliyun.com/feed")
	if err != nil {
		g.LogWarn("Failed to fetch RSS feed: %v", err)
		return nil, fmt.Errorf("fetch RSS feed: %w", err)
	}
	defer resp.Body.Close()

	if !resp.IsSuccess() {
		g.LogWarn("RSS feed request failed with status: %d", resp.StatusCode)
		return nil, fmt.Errorf("RSS feed request failed: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		g.LogWarn("Failed to read RSS feed body: %v", err)
		return nil, fmt.Errorf("read RSS feed body: %w", err)
	}

	var feed AtomFeed
	if err := xml.Unmarshal(body, &feed); err != nil {
		g.LogWarn("Failed to parse RSS feed XML: %v", err)
		return nil, fmt.Errorf("parse RSS feed XML: %w", err)
	}

	g.LogInfo("Successfully parsed RSS feed with %d entries", len(feed.Entries))

	// Step 2: Fetch detail for each article (if possible)
	count := 0
	for _, entry := range feed.Entries {
		if count >= limit {
			break
		}

		url := entry.GetLink()
		if url == "" {
			url = entry.ID
		}
		if url == "" {
			continue
		}

		publishedAt := g.parseDate(entry.Published)
		if publishedAt.IsZero() {
			publishedAt = g.parseDate(entry.Updated)
		}
		if publishedAt.IsZero() {
			publishedAt = time.Now()
		}

		// Try to fetch full content with Rod
		fullContent := ""
		contentFetched := false

		if err := g.CheckContext(ctx); err == nil {
			fullContent, contentFetched = g.fetchDetailWithRod(ctx, url)
		}

		// Use full content or fall back to summary
		summary := g.cleanSummary(entry.Summary)
		if summary == "" {
			summary = entry.Title
		}

		if !contentFetched || fullContent == "" {
			fullContent = summary
		} else {
			// Convert HTML to markdown and process images
			fullContent = g.htmlToMarkdownWithImages(fullContent)
		}

		article := &Article{
			Title:       g.cleanTitle(entry.Title),
			Summary:     summary,
			Content:     fullContent,
			Source:      "先知社区",
			URL:         url,
			Author:      "",
			Tags:        []string{"安全文章", "先知社区", "技术博客"},
			PublishedAt: publishedAt,
		}

		articles = append(articles, article)
		count++

		g.LogDebug("Parsed article: %s (content: %d chars)", article.Title, len(fullContent))

		// Rate limit between requests
		time.Sleep(1 * time.Second)
	}

	g.LogInfo("Successfully processed %d articles from RSS feed", len(articles))
	return articles, nil
}

// fetchDetailWithRod tries to fetch article detail page using Rod browser.
// Returns content and whether it was successfully fetched.
func (g *XianzhiGrabber) fetchDetailWithRod(ctx context.Context, url string) (string, bool) {
	page, cleanup, err := g.CreatePage(ctx)
	if err != nil {
		g.LogDebug("Failed to create page for detail: %v", err)
		return "", false
	}
	defer cleanup()

	g.LogDebug("Navigating to article detail with Rod: %s", url)

	// Use longer timeout for WAF challenge
	if err := g.NavigateWithContext(ctx, page, url); err != nil {
		g.LogDebug("Failed to navigate: %v", err)
		return "", false
	}

	// Wait for WAF challenge
	g.LogDebug("Waiting for WAF challenge...")
	time.Sleep(8 * time.Second)

	// Check if blocked by WAF
	script := `() => {
		const hasWaf = document.body.innerHTML.includes('aliyun_waf') || 
		               document.body.innerHTML.includes('WAF') ||
		               document.body.innerHTML.includes('Access Denied') ||
		               document.body.innerHTML.includes('captcha');
		return { blocked: hasWaf, htmlLength: document.body.innerHTML.length };
	}`

	result, err := page.Eval(script)
	if err != nil {
		g.LogDebug("WAF check failed: %v", err)
		return "", false
	}

	wafResult := result.Value.String()
	if strings.Contains(wafResult, "blocked:true") {
		g.LogDebug("Article blocked by WAF: %s", url)
		return "", false
	}

	// Extract content - return just the HTML part
	contentScript := `() => {
		const selectors = [
			'.article-content',
			'.post-content',
			'.article-body',
			'.content-body',
			'.news-content',
			'.news-detail',
			'article',
			'.container'
		];
		
		for (const sel of selectors) {
			const el = document.querySelector(sel);
			if (el && el.innerText.length > 200) {
				return el.innerHTML;
			}
		}
		
		// Fallback to any substantial content
		const bodyText = document.body.innerText;
		if (bodyText.length > 500) {
			return document.body.innerHTML;
		}
		
		return '';
	}`

	contentResult, err := page.Eval(contentScript)
	if err != nil {
		g.LogDebug("Content extraction failed: %v", err)
		return "", false
	}

	htmlContent := contentResult.Value.String()
	if htmlContent == "" {
		g.LogDebug("No content extracted")
		return "", false
	}

	g.LogDebug("Successfully fetched content via Rod (%d chars)", len(htmlContent))
	return htmlContent, true
}

// htmlToMarkdownWithImages converts HTML to Markdown and embeds images as base64.
func (g *XianzhiGrabber) htmlToMarkdownWithImages(htmlContent string) string {
	if htmlContent == "" {
		return ""
	}

	// Extract and replace images with base64
	content := g.processImagesInHTML(htmlContent)

	// Convert HTML to Markdown
	converter := md.NewConverter("", true, nil)
	mdContent, err := converter.ConvertString(content)
	if err != nil {
		g.LogWarn("Failed to convert HTML to markdown: %v", err)
		return g.stripHTML(content)
	}

	return mdContent
}

// processImagesInHTML finds images in HTML and converts them to base64.
func (g *XianzhiGrabber) processImagesInHTML(html string) string {
	if html == "" {
		return html
	}

	// Find all image URLs
	imgPattern := regexp.MustCompile(`<img[^>]+src=["']([^"']+)["'][^>]*>`)
	
	return imgPattern.ReplaceAllStringFunc(html, func(match string) string {
		srcParts := regexp.MustCompile(`src=["']([^"']+)["']`).FindStringSubmatch(match)
		if len(srcParts) == 2 {
			url := srcParts[1]
			base64 := g.downloadImageAsBase64(url)
			if base64 != "" {
				return regexp.MustCompile(`src=["'][^"']+["']`).ReplaceAllString(match, fmt.Sprintf(`src="%s"`, base64))
			}
		}
		return match
	})
}

// downloadImageAsBase64 downloads an image and returns it as base64 data URI.
func (g *XianzhiGrabber) downloadImageAsBase64(imageURL string) string {
	if imageURL == "" || strings.HasPrefix(imageURL, "data:") {
		return imageURL
	}

	resp, err := g.client.R().Get(imageURL)
	if err != nil {
		g.LogDebug("Failed to download image: %v", err)
		return ""
	}
	defer resp.Body.Close()

	if !resp.IsSuccess() {
		g.LogDebug("Image download failed with status: %d", resp.StatusCode)
		return ""
	}

	contentType := resp.GetHeader("Content-Type")
	if contentType == "" {
		contentType = "image/jpeg"
	}

	data := resp.Bytes()
	base64Data := base64.StdEncoding.EncodeToString(data)
	return fmt.Sprintf("data:%s;base64,%s", contentType, base64Data)
}

// stripHTML removes HTML tags from content.
func (g *XianzhiGrabber) stripHTML(html string) string {
	re := regexp.MustCompile(`<[^>]*>`)
	result := re.ReplaceAllString(html, "")
	
	result = strings.ReplaceAll(result, "&amp;", "&")
	result = strings.ReplaceAll(result, "&lt;", "<")
	result = strings.ReplaceAll(result, "&gt;", ">")
	result = strings.ReplaceAll(result, "&quot;", "\"")
	result = strings.ReplaceAll(result, "&#39;", "'")
	result = strings.ReplaceAll(result, "&nbsp;", " ")
	
	return strings.TrimSpace(result)
}

// cleanTitle removes HTML tags from title.
func (g *XianzhiGrabber) cleanTitle(title string) string {
	re := regexp.MustCompile(`<[^>]*>`)
	clean := re.ReplaceAllString(title, "")
	
	clean = strings.ReplaceAll(clean, "&amp;", "&")
	clean = strings.ReplaceAll(clean, "&lt;", "<")
	clean = strings.ReplaceAll(clean, "&gt;", ">")
	clean = strings.ReplaceAll(clean, "&quot;", "\"")
	clean = strings.ReplaceAll(clean, "&#39;", "'")
	clean = strings.ReplaceAll(clean, "&nbsp;", " ")

	return strings.TrimSpace(clean)
}

// cleanSummary removes HTML formatting from summary.
func (g *XianzhiGrabber) cleanSummary(summary string) string {
	if summary == "" {
		return ""
	}

	re := regexp.MustCompile(`<br\s*/?>`)
	summary = re.ReplaceAllString(summary, "\n")

	re2 := regexp.MustCompile(`<p[^>]*>`)
	summary = re2.ReplaceAllString(summary, "")

	re3 := regexp.MustCompile(`</p>`)
	summary = re3.ReplaceAllString(summary, "\n")

	re4 := regexp.MustCompile(`<[^>]*>`)
	summary = re4.ReplaceAllString(summary, "")

	summary = strings.ReplaceAll(summary, "&amp;", "&")
	summary = strings.ReplaceAll(summary, "&lt;", "<")
	summary = strings.ReplaceAll(summary, "&gt;", ">")
	summary = strings.ReplaceAll(summary, "&quot;", "\"")
	summary = strings.ReplaceAll(summary, "&#39;", "'")
	summary = strings.ReplaceAll(summary, "&nbsp;", " ")

	lines := strings.Split(summary, "\n")
	var cleanLines []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			cleanLines = append(cleanLines, trimmed)
		}
	}

	return strings.Join(cleanLines, "\n\n")
}

// parseDate parses date string from various formats.
func (g *XianzhiGrabber) parseDate(dateStr string) time.Time {
	if dateStr == "" {
		return time.Time{}
	}

	layouts := []string{
		time.RFC3339,
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05+08:00",
		"2006-01-02 15:04:05",
		"2006-01-02",
		"2006年01月02日",
		"Jan 02, 2006",
		"02 Jan 2006",
	}

	for _, layout := range layouts {
		if t, err := time.Parse(layout, dateStr); err == nil {
			return t
		}
	}

	return time.Time{}
}

func init() {
	Register(NewXianzhiGrabber())
}
