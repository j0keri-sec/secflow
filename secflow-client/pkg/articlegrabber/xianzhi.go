// Package articlegrabber provides crawler for xianzhi community (先知社区).
package articlegrabber

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/kataras/golog"
	"github.com/imroc/req/v3"
	md "github.com/JohannesKaufmann/html-to-markdown"
)

// XianzhiGrabber fetches articles from xianzhi community (xz.aliyun.com).
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

// Fetch retrieves articles from xianzhi community.
func (g *XianzhiGrabber) Fetch(ctx context.Context, limit int) ([]*Article, error) {
	// Initialize session with rod to get cookies and bypass WAF
	if err := g.initSession(ctx); err != nil {
		g.LogWarn("Failed to init session: %v", err)
	}

	// Calculate pages needed (each page returns up to 20 items typically)
	pageSize := 20
	pageNo := 1
	totalFetched := 0

	articles := make([]*Article, 0)

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

		// Fetch detail content for each article using rod
		for i, item := range items {
			select {
			case <-ctx.Done():
				g.LogInfo("Context cancelled, stopping at %d articles", i)
				break
			default:
			}

			g.LogInfo("[%d/%d] Fetching detail for: %s", i+1, len(items), item.Title)

			fullContent, err := g.fetchArticleDetail(ctx, item.URL)
			if err != nil {
				g.LogWarn("Failed to fetch detail for %s: %v, using title as content", item.Title, err)
				fullContent = item.Title
			}

			// Process images in content - convert to base64
			fullContent = g.processContentImages(fullContent)

			item.Content = fullContent
			if item.Summary == "" || item.Summary == item.Title {
				item.Summary = extractSummary(fullContent)
			}

			articles = append(articles, item)

			// Rate limit between requests to avoid triggering WAF
			time.Sleep(500 * time.Millisecond)
		}

		totalFetched += len(items)

		if len(items) < pageSize {
			break
		}

		pageNo++
	}

	g.LogInfo("Successfully processed %d articles with full content", len(articles))
	return articles, nil
}

// initSession visits the main page with rod to establish a session and get cookies.
func (g *XianzhiGrabber) initSession(ctx context.Context) error {
	page, cleanup, err := g.CreatePage(ctx)
	if err != nil {
		return fmt.Errorf("create page: %w", err)
	}
	defer cleanup()

	baseURL := "https://xz.aliyun.com"
	g.LogInfo("Visiting page to establish session: %s", baseURL)

	// Navigate with bypass protection
	if err := g.Navigate(page, baseURL); err != nil {
		g.LogWarn("Navigation warning (may continue): %v", err)
	}

	// Wait for page to stabilize
	time.Sleep(2 * time.Second)

	g.LogDebug("Session initialized")
	return nil
}

// fetchArticleList fetches the article list for a given page using rod.
func (g *XianzhiGrabber) fetchArticleList(ctx context.Context, pageNo, pageSize int) ([]*Article, error) {
	page, cleanup, err := g.CreatePage(ctx)
	if err != nil {
		return nil, fmt.Errorf("create page: %w", err)
	}
	defer cleanup()

	// Build URL with pagination
	var url string
	if pageNo == 1 {
		url = "https://xz.aliyun.com"
	} else {
		url = fmt.Sprintf("https://xz.aliyun.com/page/%d", pageNo)
	}

	g.LogDebug("Navigating to list page: %s", url)

	if err := g.NavigateWithContext(ctx, page, url); err != nil {
		return nil, fmt.Errorf("navigate to list page: %w", err)
	}

	// Wait for content to load
	time.Sleep(2 * time.Second)

	// Extract article list using JavaScript
	script := `() => {
		const results = [];
		// Try multiple selectors for article items
		const selectors = [
			'.article-item',
			'.post-list .post-item',
			'.article-list .item',
			'.blog-list .blog-item',
			'article',
			'.article-item a',
			'.post-title a'
		];

		let items = [];
		for (const selector of selectors) {
			items = document.querySelectorAll(selector);
			if (items.length > 0) {
				break;
			}
		}

		items.forEach(item => {
			let title = '';
			let link = '';
			let author = '';
			let date = '';
			let summary = '';

			// Try to find title and link
			const titleLink = item.querySelector('a.title') || item.querySelector('h2 a') || item.querySelector('h3 a') || item.querySelector('a');
			if (titleLink) {
				title = titleLink.textContent.trim();
				link = titleLink.getAttribute('href');
			}

			// Try to find author
			const authorEl = item.querySelector('.author') || item.querySelector('.user-name') || item.querySelector('[class*="author"]');
			if (authorEl) {
				author = authorEl.textContent.trim();
			}

			// Try to find date
			const dateEl = item.querySelector('.date') || item.querySelector('.time') || item.querySelector('[class*="date"]') || item.querySelector('[class*="time"]');
			if (dateEl) {
				date = dateEl.textContent.trim();
			}

			// Try to find summary
			const summaryEl = item.querySelector('.summary') || item.querySelector('.desc') || item.querySelector('.excerpt');
			if (summaryEl) {
				summary = summaryEl.textContent.trim();
			}

			if (title && link) {
				results.push({
					title: title,
					url: link.startsWith('http') ? link : 'https://xz.aliyun.com' + link,
					author: author,
					date: date,
					summary: summary
				});
			}
		});

		return JSON.stringify(results);
	}`

	result, err := page.Eval(script)
	if err != nil {
		return nil, fmt.Errorf("execute script: %w", err)
	}

	var items []struct {
		Title   string `json:"title"`
		URL     string `json:"url"`
		Author  string `json:"author"`
		Date    string `json:"date"`
		Summary string `json:"summary"`
	}

	if err := json.Unmarshal([]byte(result.Value.String()), &items); err != nil {
		return nil, fmt.Errorf("parse article list: %w", err)
	}

	articles := make([]*Article, 0, len(items))
	for _, item := range items {
		publishedAt := g.parseDate(item.Date)

		articles = append(articles, &Article{
			Title:       item.Title,
			Summary:     item.Summary,
			Source:      "先知社区",
			URL:         item.URL,
			Author:      item.Author,
			Tags:        []string{"安全文章", "先知社区", "技术博客"},
			PublishedAt: publishedAt,
		})
	}

	return articles, nil
}

// fetchArticleDetail fetches the full article content using rod.
func (g *XianzhiGrabber) fetchArticleDetail(ctx context.Context, articleURL string) (string, error) {
	page, cleanup, err := g.CreatePage(ctx)
	if err != nil {
		return "", fmt.Errorf("create page: %w", err)
	}
	defer cleanup()

	g.LogDebug("Navigating to article detail: %s", articleURL)

	if err := g.NavigateWithContext(ctx, page, articleURL); err != nil {
		return "", fmt.Errorf("navigate to detail page: %w", err)
	}

	// Wait for content to load
	time.Sleep(2 * time.Second)

	// Extract article content using JavaScript
	script := `() => {
		const selectors = [
			'.article-content',
			'.post-content',
			'.article-body',
			'.content-body',
			'.entry-content',
			'#article-content',
			'article .content',
			'.markdown-body',
			'.article-detail .content'
		];

		let contentEl = null;
		for (const selector of selectors) {
			contentEl = document.querySelector(selector);
			if (contentEl && contentEl.innerHTML.length > 200) {
				break;
			}
		}

		if (!contentEl) {
			return JSON.stringify({ title: '', html: '', author: '', date: '' });
		}

		// Get title
		const titleEl = document.querySelector('h1') || document.querySelector('.article-title') || document.querySelector('.post-title');
		const title = titleEl ? titleEl.textContent.trim() : '';

		// Get author
		const authorEl = document.querySelector('.author') || document.querySelector('.user-name') || document.querySelector('[class*="author"]');
		const author = authorEl ? authorEl.textContent.trim() : '';

		// Get date
		const dateEl = document.querySelector('.date') || document.querySelector('.time') || document.querySelector('[class*="date"]');
		const date = dateEl ? dateEl.textContent.trim() : '';

		const html = contentEl.innerHTML || '';
		const text = contentEl.textContent || '';

		return JSON.stringify({
			title: title,
			html: html,
			text: text.trim(),
			author: author,
			date: date
		});
	}`

	result, err := page.Eval(script)
	if err != nil {
		return "", fmt.Errorf("execute script: %w", err)
	}

	var detailData struct {
		Title  string `json:"title"`
		HTML   string `json:"html"`
		Text   string `json:"text"`
		Author string `json:"author"`
		Date   string `json:"date"`
	}

	if err := json.Unmarshal([]byte(result.Value.String()), &detailData); err != nil {
		return "", fmt.Errorf("parse detail data: %w", err)
	}

	content := detailData.HTML
	if content == "" || len(content) < 200 {
		content = detailData.Text
	}

	if content == "" {
		return "", fmt.Errorf("no content extracted from detail page")
	}

	// Convert HTML to Markdown
	mdContent := g.htmlToMarkdown(content)

	g.LogDebug("Extracted %d chars of content", len(mdContent))
	return mdContent, nil
}

// htmlToMarkdown converts HTML content to Markdown format.
func (g *XianzhiGrabber) htmlToMarkdown(html string) string {
	if html == "" {
		return ""
	}

	// Clean up unwanted attributes
	html = cleanXianzhiHTML(html)

	// Use html-to-markdown library
	converter := md.NewConverter("", true, nil)
	mdContent, err := converter.ConvertString(html)
	if err != nil {
		g.LogWarn("Failed to convert HTML to markdown: %v", err)
		return g.stripHTML(html)
	}

	return mdContent
}

// cleanXianzhiHTML removes unwanted elements from xianzhi HTML content.
func cleanXianzhiHTML(html string) string {
	// Remove Vue data-v-* attributes
	re := regexp.MustCompile(`\s+data-v-[a-zA-Z0-9]+=""`)
	html = re.ReplaceAllString(html, "")

	re2 := regexp.MustCompile(`\s+data-v-[a-zA-Z0-9]+="[^"]*"`)
	html = re2.ReplaceAllString(html, "")

	// Remove style attributes
	re3 := regexp.MustCompile(`\s*style="[^"]*"`)
	html = re3.ReplaceAllString(html, "")

	// Remove empty paragraph tags
	re4 := regexp.MustCompile(`<p>\s*</p>`)
	html = re4.ReplaceAllString(html, "")

	re5 := regexp.MustCompile(`<p>&nbsp;</p>`)
	html = re5.ReplaceAllString(html, "")

	return html
}

// stripHTML removes HTML tags from content.
func (g *XianzhiGrabber) stripHTML(html string) string {
	re := strings.NewReplacer(
		"<br>", "\n",
		"<br/>", "\n",
		"<br />", "\n",
		"</p>", "\n",
		"</div>", "\n",
		"</li>", "\n",
	)

	result := re.Replace(html)

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

// processContentImages finds images in content and converts them to base64.
func (g *XianzhiGrabber) processContentImages(content string) string {
	if content == "" {
		return content
	}

	// Find all image URLs in the content (Markdown and HTML formats)
	// Markdown: ![alt](url)
	// HTML: <img src="url" />

	// Replace Markdown images
	mdImgPattern := regexp.MustCompile(`!\[([^\]]*)\]\(([^)]+)\)`)
	content = mdImgPattern.ReplaceAllStringFunc(content, func(match string) string {
		parts := mdImgPattern.FindStringSubmatch(match)
		if len(parts) == 3 {
			alt := parts[1]
			url := parts[2]
			base64 := g.downloadImageAsBase64(url)
			if base64 != "" {
				return fmt.Sprintf("![%s](%s)", alt, base64)
			}
			return match // Keep original if download fails
		}
		return match
	})

	// Replace HTML images
	htmlImgPattern := regexp.MustCompile(`<img[^>]+src="([^"]+)"[^>]*>`)
	content = htmlImgPattern.ReplaceAllStringFunc(content, func(match string) string {
		srcParts := regexp.MustCompile(`src="([^"]+)"`).FindStringSubmatch(match)
		if len(srcParts) == 2 {
			url := srcParts[1]
			base64 := g.downloadImageAsBase64(url)
			if base64 != "" {
				// Replace src with base64, keep other attributes
				return regexp.MustCompile(`src="[^"]*"`).ReplaceAllString(match, fmt.Sprintf(`src="%s"`, base64))
			}
		}
		return match
	})

	return content
}

// downloadImageAsBase64 downloads an image from URL and returns it as base64-encoded data URI.
// Returns empty string on failure.
func (g *XianzhiGrabber) downloadImageAsBase64(imageURL string) string {
	if imageURL == "" {
		return ""
	}

	// Skip data URIs (already processed)
	if strings.HasPrefix(imageURL, "data:") {
		return imageURL
	}

	resp, err := g.client.R().
		SetHeader("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36").
		SetHeader("Accept", "image/webp,image/apng,image/*,*/*;q=0.8").
		SetHeader("Referer", "https://xz.aliyun.com/").
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

// parseDate parses date string from various formats.
func (g *XianzhiGrabber) parseDate(dateStr string) time.Time {
	if dateStr == "" {
		return time.Now()
	}

	layouts := []string{
		"2006-01-02",
		"2006-01-02 15:04:05",
		"2006年01月02日",
		"Jan 02, 2006",
		"02 Jan 2006",
	}

	for _, layout := range layouts {
		if t, err := time.Parse(layout, dateStr); err == nil {
			return t
		}
	}

	return time.Now()
}

func init() {
	Register(NewXianzhiGrabber())
}
