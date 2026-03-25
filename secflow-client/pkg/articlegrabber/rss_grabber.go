// Package articlegrabber provides a generic RSS-based article crawler.
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

// RSSGrabber is a generic RSS feed based article crawler.
type RSSGrabber struct {
	*RodCrawler
	client     *req.Client
	name       string
	sourceName string
	feedURL    string
}

// NewRSSGrabber creates a new generic RSS grabber.
func NewRSSGrabber(name, sourceName, feedURL string) *RSSGrabber {
	return &RSSGrabber{
		RodCrawler: NewRodCrawler(name, golog.Child("["+name+"]")),
		client: req.NewClient().
			SetTimeout(60 * time.Second).
			SetUserAgent("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/146.0.0.0 Safari/537.36"),
		name:       name,
		sourceName: sourceName,
		feedURL:    feedURL,
	}
}

// Name returns the grabber name.
func (g *RSSGrabber) Name() string {
	return g.name
}

// GenericRSSItem represents a generic RSS item.
type GenericRSSItem struct {
	Title   string `xml:"title"`
	Link    string `xml:"link"`
	Desc    string `xml:"description"`
	Content string `xml:"encoded"` // content:encoded
	PubDate string `xml:"pubDate"`
	Author  string `xml:"author"`
	Creator string `xml:"creator"`
}

// GenericRSSFeed represents a generic RSS feed.
type GenericRSSFeed struct {
	Channel GenericRSSChannel `xml:"channel"`
}

// GenericRSSChannel represents the RSS channel.
type GenericRSSChannel struct {
	Title string         `xml:"title"`
	Link  string         `xml:"link"`
	Items []GenericRSSItem `xml:"item"`
}

// Fetch retrieves articles from the RSS feed.
func (g *RSSGrabber) Fetch(ctx context.Context, limit int) ([]*Article, error) {
	articles := make([]*Article, 0)

	g.LogInfo("Fetching RSS feed from %s: %s", g.sourceName, g.feedURL)

	resp, err := g.client.R().SetContext(ctx).Get(g.feedURL)
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

	var feed GenericRSSFeed
	if err := xml.Unmarshal(body, &feed); err != nil {
		g.LogWarn("Failed to parse RSS feed XML: %v", err)
		return nil, fmt.Errorf("parse RSS feed XML: %w", err)
	}

	g.LogInfo("Successfully parsed RSS feed with %d items", len(feed.Channel.Items))

	count := 0
	for _, item := range feed.Channel.Items {
		if count >= limit {
			break
		}

		if item.Link == "" {
			continue
		}

		// Parse published date
		publishedAt := g.parseDate(item.PubDate)
		if publishedAt.IsZero() {
			publishedAt = time.Now()
		}

		// Get full content
		fullContent := item.Content
		if fullContent == "" {
			fullContent = item.Desc
		}

		// Process images
		fullContent = g.processImagesInHTML(fullContent)

		// Convert to markdown
		mdContent := g.htmlToMarkdown(fullContent)

		// Create summary
		summary := g.cleanText(item.Desc)
		if len(summary) > 300 {
			summary = summary[:300] + "..."
		}

		// Get author
		author := item.Author
		if author == "" {
			author = item.Creator
		}

		article := &Article{
			Title:       g.cleanText(item.Title),
			Summary:     summary,
			Content:     mdContent,
			Source:      g.sourceName,
			URL:         item.Link,
			Author:      author,
			Tags:        []string{"安全文章", g.sourceName},
			PublishedAt: publishedAt,
		}

		articles = append(articles, article)
		count++

		g.LogDebug("Parsed article: %s", article.Title)

		// Rate limit
		time.Sleep(500 * time.Millisecond)
	}

	g.LogInfo("Successfully processed %d articles from %s RSS feed", len(articles), g.sourceName)
	return articles, nil
}

// processImagesInHTML downloads images and embeds as base64.
func (g *RSSGrabber) processImagesInHTML(html string) string {
	if html == "" {
		return html
	}

	imgPattern := regexp.MustCompile(`<img[^>]+src=["']([^"']+)["'][^>]*>`)

	return imgPattern.ReplaceAllStringFunc(html, func(match string) string {
		srcParts := regexp.MustCompile(`src=["']([^"']+)["']`).FindStringSubmatch(match)
		if len(srcParts) == 2 {
			imgURL := srcParts[1]
			if strings.HasPrefix(imgURL, "data:") || imgURL == "" {
				return match
			}

			// Handle relative URLs
			if !strings.HasPrefix(imgURL, "http://") && !strings.HasPrefix(imgURL, "https://") {
				if strings.HasPrefix(imgURL, "//") {
					imgURL = "https:" + imgURL
				} else if strings.HasPrefix(imgURL, "/") {
					imgURL = "https://www.4hou.com" + imgURL
				} else {
					return match
				}
			}

			base64 := g.downloadImageAsBase64(imgURL)
			if base64 != "" {
				return regexp.MustCompile(`src=["'][^"']+["']`).ReplaceAllString(match, fmt.Sprintf(`src="%s"`, base64))
			}
		}
		return match
	})
}

// downloadImageAsBase64 downloads an image and returns base64.
func (g *RSSGrabber) downloadImageAsBase64(imageURL string) string {
	if imageURL == "" {
		return ""
	}

	// Skip tracking pixels and small icons
	if strings.Contains(imageURL, "pixel") || strings.Contains(imageURL, "tracking") || strings.Contains(imageURL, "1x1") {
		return ""
	}

	resp, err := g.client.R().Get(imageURL)
	if err != nil {
		g.LogDebug("Failed to download image: %v", err)
		return ""
	}
	defer resp.Body.Close()

	if !resp.IsSuccess() {
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

// htmlToMarkdown converts HTML to Markdown.
func (g *RSSGrabber) htmlToMarkdown(html string) string {
	if html == "" {
		return ""
	}

	converter := md.NewConverter("", true, nil)
	result, err := converter.ConvertString(html)
	if err != nil {
		g.LogWarn("Failed to convert HTML to markdown: %v", err)
		return g.cleanText(html)
	}

	return result
}

// cleanText removes HTML and cleans up text.
func (g *RSSGrabber) cleanText(text string) string {
	if text == "" {
		return ""
	}

	// Remove CDATA wrapper
	re := regexp.MustCompile(`<!\[CDATA\[(.*)\]\]>`,)
	text = re.ReplaceAllString(text, "$1")

	// Remove HTML tags
	re2 := regexp.MustCompile(`<[^>]*>`)
	text = re2.ReplaceAllString(text, "")

	// Decode entities
	text = strings.ReplaceAll(text, "&amp;", "&")
	text = strings.ReplaceAll(text, "&lt;", "<")
	text = strings.ReplaceAll(text, "&gt;", ">")
	text = strings.ReplaceAll(text, "&quot;", "\"")
	text = strings.ReplaceAll(text, "&#39;", "'")
	text = strings.ReplaceAll(text, "&nbsp;", " ")
	text = strings.ReplaceAll(text, "&#91;", "[")
	text = strings.ReplaceAll(text, "&#93;", "]")

	// Clean whitespace
	lines := strings.Split(text, "\n")
	var cleanLines []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			cleanLines = append(cleanLines, trimmed)
		}
	}

	return strings.Join(cleanLines, "\n")
}

// parseDate parses date string in various formats.
func (g *RSSGrabber) parseDate(dateStr string) time.Time {
	if dateStr == "" {
		return time.Time{}
	}

	layouts := []string{
		time.RFC1123,
		time.RFC3339,
		"Mon, 02 Jan 2006 15:04:05 +0000",
		"Mon, 02 Jan 2006 15:04:05 +0800",
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05+08:00",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}

	for _, layout := range layouts {
		if t, err := time.Parse(layout, dateStr); err == nil {
			return t
		}
	}

	return time.Time{}
}
