// Package article implements grabbers for security news and articles.
package articlegrabber

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/kataras/golog"
	"github.com/imroc/req/v3"
	md "github.com/JohannesKaufmann/html-to-markdown"

	"github.com/secflow/client/pkg/rodutil"
)

// VenustechGrabber fetches security articles from Venustech (启明星辰).
type VenustechGrabber struct {
	*RodCrawler
	http *req.Client
	// Custom headers for Venustech to bypass bot detection
	customHeaders map[string]string
}

// NewVenustechGrabber creates a new Venustech article grabber.
func NewVenustechGrabber() *VenustechGrabber {
	g := &VenustechGrabber{
		RodCrawler: NewRodCrawler("venustech-article", golog.Child("[venustech-article]")),
		http: req.NewClient().
			SetTimeout(60 * time.Second),
		customHeaders: map[string]string{
			// Use Safari UA because Venustech blocks Chrome UA from non-browser clients (TLS fingerprint mismatch)
			"User-Agent":      "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.1 Safari/605.1.15",
			"Accept":          "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
			"Accept-Language": "zh-CN,zh;q=0.9,en;q=0.8",
			"Connection":      "keep-alive",
			"Upgrade-Insecure-Requests": "1",
			"Referer": "https://www.venustech.com.cn/",
		},
	}
	return g
}

// Name returns the grabber name.
func (g *VenustechGrabber) Name() string {
	return "venustech"
}

// CreatePage creates a new rod page with custom headers for Venustech.
func (g *VenustechGrabber) CreatePage(ctx context.Context) (*rod.Page, func(), error) {
	browser, err := rodutil.GetBrowser(nil)
	if err != nil {
		return nil, nil, fmt.Errorf("get browser: %w", err)
	}

	page, err := rodutil.NewPage(browser)
	if err != nil {
		return nil, nil, fmt.Errorf("create page: %w", err)
	}

	// Apply custom headers via CDP
	headers := make([]string, 0, len(g.customHeaders)*2)
	for key, value := range g.customHeaders {
		headers = append(headers, key, value)
	}
	if _, err := page.SetExtraHeaders(headers); err != nil {
		g.LogWarn("failed to set extra headers: %v", err)
	}

	// Apply bypass techniques
	if err := rodutil.ApplyBypass(page, g.config); err != nil {
		g.LogWarn("apply bypass: %v", err)
	}

	cleanup := func() {
		page.Close()
	}

	return page, cleanup, nil
}

// Fetch retrieves articles from Venustech security bulletin.
func (g *VenustechGrabber) Fetch(ctx context.Context, limit int) ([]*Article, error) {
	// Phase 1: Fetch article list using HTTP client (simpler, avoids bot detection)
	listItems, err := g.fetchArticleListHTTP(ctx, limit)
	if err != nil {
		g.LogWarn("HTTP list fetch failed: %v, trying rod", err)
		listItems, err = g.fetchArticleListRod(ctx, limit)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch article list: %w", err)
		}
	}

	g.LogInfo("Fetched %d article items from list", len(listItems))

	// Phase 2: Fetch detail content for each article using rod
	articles := make([]*Article, 0, len(listItems))
	for i, item := range listItems {
		select {
		case <-ctx.Done():
			g.LogInfo("Context cancelled, stopping at %d articles", i)
			break
		default:
		}

		g.LogInfo("[%d/%d] Fetching detail for: %s", i+1, len(listItems), item.Title)

		fullContent, err := g.fetchArticleDetail(ctx, item.URL)
		if err != nil {
			g.LogWarn("Failed to fetch detail for %s: %v, using title as content", item.Title, err)
			fullContent = item.Title
		}

		item.Content = fullContent
		if item.Summary == item.Title {
			item.Summary = extractSummary(fullContent)
		}
		articles = append(articles, item)

		// Rate limit between requests
		time.Sleep(500 * time.Millisecond)
	}

	g.LogInfo("Successfully processed %d articles with full content", len(articles))
	return articles, nil
}

// listItem represents an article from the list page
type listItem struct {
	Title string
	URL   string
	Date  string
}

// fetchArticleListHTTP fetches the article list using HTTP client (more reliable for static pages).
func (g *VenustechGrabber) fetchArticleListHTTP(ctx context.Context, limit int) ([]*Article, error) {
	pageSize := 10
	pageNo := 1
	var allItems []*listItem

	for len(allItems) < limit {
		var rawURL string
		if pageNo == 1 {
			rawURL = "https://www.venustech.com.cn/new_type/aqjx/"
		} else {
			rawURL = fmt.Sprintf("https://www.venustech.com.cn/new_type/aqjx/index_%d.html", pageNo)
		}

		g.LogInfo("Fetching list page via HTTP: %s", rawURL)

		req := g.http.R().SetContext(ctx)
		for key, value := range g.customHeaders {
			req.SetHeader(key, value)
		}
		resp, err := req.Get(rawURL)

		if err != nil {
			return nil, fmt.Errorf("HTTP request failed: %w", err)
		}
		defer resp.Body.Close()

		if !resp.IsSuccess() {
			return nil, fmt.Errorf("HTTP returned status %d", resp.StatusCode)
		}

		// Parse items from HTML
		items, err := g.parseListPageHTML(resp.Bytes())
		if err != nil {
			return nil, fmt.Errorf("parse HTML: %w", err)
		}

		if len(items) == 0 {
			break
		}

		allItems = append(allItems, items...)

		if len(items) < pageSize {
			break
		}

		pageNo++
	}

	// Convert to Articles
	articles := make([]*Article, 0, len(allItems))
	for _, item := range allItems {
		if len(articles) >= limit {
			break
		}

		publishedAt, _ := time.Parse("2006-01-02", item.Date)
		if publishedAt.IsZero() {
			publishedAt = time.Now()
		}

		// Ensure full URL
		url := item.URL
		if !strings.HasPrefix(url, "http") {
			url = "https://www.venustech.com.cn" + url
		}

		articles = append(articles, &Article{
			Title:       item.Title,
			Summary:     item.Title,
			Source:      "启明星辰",
			URL:         url,
			Tags:        []string{"安全简报", "启明星辰", "安全资讯"},
			PublishedAt: publishedAt,
		})
	}

	return articles, nil
}

// fetchArticleListRod fetches article list using rod (fallback).
func (g *VenustechGrabber) fetchArticleListRod(ctx context.Context, limit int) ([]*Article, error) {
	pageLimit := (limit + 9) / 10
	var results []*Article

	for i := 1; i <= pageLimit; i++ {
		select {
		case <-ctx.Done():
			return results, ctx.Err()
		default:
		}

		page, cleanup, err := g.CreatePage(ctx)
		if err != nil {
			g.LogWarn("create page: %v", err)
			continue
		}

		var rawURL string
		if i == 1 {
			rawURL = "https://www.venustech.com.cn/new_type/aqjx/"
		} else {
			rawURL = fmt.Sprintf("https://www.venustech.com.cn/new_type/aqjx/index_%d.html", i)
		}

		g.LogInfo("Fetching via rod: %s", rawURL)

		if err := g.Navigate(page, rawURL); err != nil {
			g.LogWarn("navigate: %v", err)
		}

		page.WaitLoad()
		time.Sleep(2 * time.Second)

		items, err := g.parseWithJS(page)
		if err != nil {
			g.LogWarn("JS parsing failed: %v", err)
		} else {
			results = append(results, items...)
		}

		cleanup()

		if len(results) >= limit {
			results = results[:limit]
			break
		}
	}

	return results, nil
}

// parseListPageHTML parses article list from HTML content.
func (g *VenustechGrabber) parseListPageHTML(htmlBytes []byte) ([]*listItem, error) {
	html := string(htmlBytes)

	// Pattern to match: <a class="left safety-tit" href="/new_type/aqjx/...">Title</a>
	linkPattern := regexp.MustCompile(`href="(/new_type/aqjx/[^"]+)"[^>]*>([^<]+)</a>`)
	timePattern := regexp.MustCompile(`<span[^>]*class="[^"]*safety-time[^"]*"[^>]*>([^<]+)</span>`)

	links := linkPattern.FindAllStringSubmatch(html, -1)
	times := timePattern.FindAllStringSubmatch(html, -1)

	items := make([]*listItem, 0)

	for i, link := range links {
		if len(link) != 3 {
			continue
		}

		url := link[1]
		title := strings.TrimSpace(link[2])

		if title == "" {
			continue
		}

		date := ""
		if i < len(times) && len(times[i]) == 2 {
			date = strings.TrimSpace(times[i][1])
		}

		items = append(items, &listItem{
			Title: title,
			URL:   url,
			Date:  date,
		})
	}

	return items, nil
}

// parseWithJS parses article list using JavaScript evaluation.
func (g *VenustechGrabber) parseWithJS(page *rod.Page) ([]*Article, error) {
	parseScript := `() => {
		const results = [];
		const items = document.querySelectorAll('li.safetyItem');

		items.forEach(item => {
			const linkEl = item.querySelector('a.safety-tit');
			const timeEl = item.querySelector('span.safety-time');

			if (linkEl) {
				const href = linkEl.getAttribute('href');
				const title = linkEl.textContent.trim();
				const date = timeEl ? timeEl.textContent.trim() : '';

				if (href && title) {
					results.push({
						url: href.startsWith('http') ? href : 'https://www.venustech.com.cn' + href,
						title: title,
						date: date
					});
				}
			}
		});

		return JSON.stringify(results);
	}`

	result, err := page.Eval(parseScript)
	if err != nil {
		return nil, fmt.Errorf("eval: %w", err)
	}

	var items []struct {
		Title string `json:"title"`
		URL   string `json:"url"`
		Date  string `json:"date"`
	}

	if err := result.Value.Unmarshal(&items); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}

	articles := make([]*Article, 0, len(items))
	for _, item := range items {
		publishedAt, _ := time.Parse("2006-01-02", item.Date)
		if publishedAt.IsZero() {
			publishedAt = time.Now()
		}

		articles = append(articles, &Article{
			Title:       item.Title,
			Summary:     item.Title,
			Source:      "启明星辰",
			URL:         item.URL,
			Tags:        []string{"安全简报", "启明星辰", "安全资讯"},
			PublishedAt: publishedAt,
		})
	}

	return articles, nil
}

// fetchArticleDetail fetches the full article content.
func (g *VenustechGrabber) fetchArticleDetail(ctx context.Context, articleURL string) (string, error) {
	// Try rod first, fall back to HTTP if needed
	content, err := g.fetchArticleDetailRod(ctx, articleURL)
	if err != nil {
		g.LogWarn("Rod fetch failed for %s: %v, trying HTTP", articleURL, err)
		content, err = g.fetchArticleDetailHTTP(ctx, articleURL)
		if err != nil {
			return "", fmt.Errorf("all fetch methods failed: %w", err)
		}
	}
	return content, nil
}

// fetchArticleDetailRod fetches article content using rod.
func (g *VenustechGrabber) fetchArticleDetailRod(ctx context.Context, articleURL string) (string, error) {
	page, cleanup, err := g.CreatePage(ctx)
	if err != nil {
		return "", fmt.Errorf("create page: %w", err)
	}
	defer cleanup()

	if err := g.NavigateWithContext(ctx, page, articleURL); err != nil {
		return "", fmt.Errorf("navigate: %w", err)
	}

	page.WaitLoad()
	time.Sleep(2 * time.Second)

	script := `() => {
		const contentEl = document.querySelector('div.news_text');
		if (!contentEl) {
			return JSON.stringify({ title: '', html: '', text: '' });
		}

		const titleEl = document.querySelector('h3.news-title');
		const title = titleEl ? titleEl.textContent.trim() : '';

		const html = contentEl.innerHTML || '';
		const text = contentEl.textContent || '';

		return JSON.stringify({
			title: title,
			html: html,
			text: text.trim()
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

	if err := result.Value.Unmarshal(&detailData); err != nil {
		return "", fmt.Errorf("parse detail data: %w", err)
	}

	content := detailData.HTML
	if content == "" || len(content) < 200 {
		content = detailData.Text
	}

	if content == "" {
		return "", fmt.Errorf("no content extracted")
	}

	return g.htmlToMarkdown(content), nil
}

// fetchArticleDetailHTTP fetches article content using HTTP client.
func (g *VenustechGrabber) fetchArticleDetailHTTP(ctx context.Context, articleURL string) (string, error) {
	req := g.http.R().SetContext(ctx)
	// Apply custom headers
	for key, value := range g.customHeaders {
		req.SetHeader(key, value)
	}
	resp, err := req.Get(articleURL)

	if err != nil {
		return "", fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if !resp.IsSuccess() {
		return "", fmt.Errorf("HTTP returned status %d", resp.StatusCode)
	}

	html := string(resp.Bytes())

	// The content is in div.news_text, followed by newsPage div
	// Use non-greedy match to stop at the first </div> that closes news_text
	// but we need to be careful because of nested divs
	contentMatch := regexp.MustCompile(`<div class="news_text"[^>]*>([\s\S]*?)</div>`).FindStringSubmatch(html)
	if len(contentMatch) < 2 {
		return "", fmt.Errorf("content div not found")
	}

	content := contentMatch[1]

	// Clean up: remove any footer/navigation content that might have been captured
	// Split by common footer markers and take only the first part
	if strings.Contains(content, "上一篇") {
		content = strings.Split(content, "上一篇")[0]
	}

	// Also extract title if available
	titleMatch := regexp.MustCompile(`<h3 class="news-title"[^>]*>([^<]+)</h3>`).FindStringSubmatch(html)
	if len(titleMatch) >= 2 {
		g.LogDebug("Extracted title: %s", titleMatch[1])
	}

	if content == "" {
		return "", fmt.Errorf("empty content")
	}

	return g.htmlToMarkdown(content), nil
}

// htmlToMarkdown converts HTML content to Markdown format.
func (g *VenustechGrabber) htmlToMarkdown(html string) string {
	if html == "" {
		return ""
	}

	// Clean up unwanted attributes and tags
	html = cleanVenustechHTML(html)

	// Use html-to-markdown library
	converter := md.NewConverter("", true, nil)
	mdContent, err := converter.ConvertString(html)
	if err != nil {
		g.LogWarn("Failed to convert HTML to markdown: %v", err)
		return g.stripHTML(html)
	}

	return mdContent
}

// cleanVenustechHTML removes unwanted elements from Venustech HTML content.
func cleanVenustechHTML(html string) string {
	// Remove style attributes
	re := regexp.MustCompile(`\s*style="[^"]*"`)
	html = re.ReplaceAllString(html, "")

	// Remove visibility styles (br tags used for spacing)
	re2 := regexp.MustCompile(`<br\s*style="visibility:\s*visible;"\s*/?>`)
	html = re2.ReplaceAllString(html, "\n")

	// Remove empty paragraph tags and nbsp paragraphs
	re3 := regexp.MustCompile(`<p>\s*</p>`)
	html = re3.ReplaceAllString(html, "")

	re4 := regexp.MustCompile(`<p>&nbsp;</p>`)
	html = re4.ReplaceAllString(html, "")

	re5 := regexp.MustCompile(`<p><br\s*/?></p>`)
	html = re5.ReplaceAllString(html, "\n")

	// Remove div with newsPage (navigation between articles) - use [\s\S] to match newlines
	re6 := regexp.MustCompile(`<div class="newsPage"[^>]*>[\s\S]*?</div>`)
	html = re6.ReplaceAllString(html, "")

	// Remove footer content: "上一篇", "关于我们", "解决方案", "安全研究", "联系我们" sections
	// These typically come after the main article content
	re7 := regexp.MustCompile(`<div[^>]*class="[^"]*footer[^"]*"[^>]*>[\s\S]*?</div>`)
	html = re7.ReplaceAllString(html, "")

	re8 := regexp.MustCompile(`<li[^>]*>[\s\S]*?<a[^>]*关于我们[\s\S]*?</li>`)
	html = re8.ReplaceAllString(html, "")

	re9 := regexp.MustCompile(`<li[^>]*>[\s\S]*?<a[^>]*解决方案[\s\S]*?</li>`)
	html = re9.ReplaceAllString(html, "")

	re10 := regexp.MustCompile(`<li[^>]*>[\s\S]*?<a[^>]*安全研究[\s\S]*?</li>`)
	html = re10.ReplaceAllString(html, "")

	re11 := regexp.MustCompile(`<li[^>]*>[\s\S]*?<a[^>]*联系我们[\s\S]*?</li>`)
	html = re11.ReplaceAllString(html, "")

	re12 := regexp.MustCompile(`<a[^>]*href="/new_type/[^"]*"[^>]*>[^<]*关于我们[^<]*</a>`)
	html = re12.ReplaceAllString(html, "")

	re13 := regexp.MustCompile(`<a[^>]*href="/new_type/[^"]*"[^>]*>[^<]*解决方案[^<]*</a>`)
	html = re13.ReplaceAllString(html, "")

	re14 := regexp.MustCompile(`<a[^>]*href="/new_type/[^"]*"[^>]*>[^<]*安全研究[^<]*</a>`)
	html = re14.ReplaceAllString(html, "")

	re15 := regexp.MustCompile(`<a[^>]*href="/new_type/[^"]*"[^>]*>[^<]*联系我们[^<]*</a>`)
	html = re15.ReplaceAllString(html, "")

	re16 := regexp.MustCompile(`<span[^>]*>[\s\S]*?关于我们[\s\S]*?</span>`)
	html = re16.ReplaceAllString(html, "")

	re17 := regexp.MustCompile(`<span[^>]*>[\s\S]*?公司介绍[\s\S]*?</span>`)
	html = re17.ReplaceAllString(html, "")

	// Remove copyright and legal info
	re18 := regexp.MustCompile(`Copyright[\s\S]*?</div>`)
	html = re18.ReplaceAllString(html, "")

	re19 := regexp.MustCompile(`法律声明[\s\S]*?</div>`)
	html = re19.ReplaceAllString(html, "")

	// Remove hotlines and contact info
	re20 := regexp.MustCompile(`7\*24小时服务热线[\s\S]*?</div>`)
	html = re20.ReplaceAllString(html, "")

	// Clean up extra whitespace
	re21 := regexp.MustCompile(`\s+`)
	html = re21.ReplaceAllString(html, " ")

	return strings.TrimSpace(html)
}

// stripHTML removes HTML tags from content.
func (g *VenustechGrabber) stripHTML(html string) string {
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

// extractSummary extracts a brief summary from article content.
func extractSummary(content string) string {
	if content == "" {
		return ""
	}

	lines := strings.Split(content, "\n")
	var summaryLines []string
	charCount := 0
	maxChars := 200

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if charCount >= maxChars {
			break
		}
		summaryLines = append(summaryLines, trimmed)
		charCount += len(trimmed)
	}

	summary := strings.Join(summaryLines, " ")
	if len(summary) > maxChars {
		summary = summary[:maxChars] + "..."
	}

	return summary
}

func init() {
	Register(NewVenustechGrabber())
}