// Package articlegrabber provides crawler for 嘶吼 (4hou.com).
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

// SihouGrabber fetches articles from 嘶吼 (4hou.com) via RSS feed.
type SihouGrabber struct {
	*RodCrawler
	client *req.Client
}

// NewSihouGrabber creates a new 嘶吼 article crawler.
func NewSihouGrabber() *SihouGrabber {
	return &SihouGrabber{
		RodCrawler: NewRodCrawler("sihou", golog.Child("[sihou]")),
		client: req.NewClient().
			SetTimeout(60 * time.Second).
			SetUserAgent("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/146.0.0.0 Safari/537.36"),
	}
}

// Name returns the grabber name.
func (g *SihouGrabber) Name() string {
	return "sihou"
}

// RSSFeed represents the RSS feed structure from 4hou.
type RSSFeed struct {
	Channel RSSChannel `xml:"channel"`
}

// RSSChannel represents the channel element in RSS.
type RSSChannel struct {
	Title       string   `xml:"title"`
	Link        string   `xml:"link"`
	Description string   `xml:"description"`
	Items       []RSSItem `xml:"item"`
}

// RSSItem represents a single item in the RSS feed.
type RSSItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	Content     string `xml:"content:encoded"`
	PubDate     string `xml:"pubDate"`
	Author      string `xml:"author"`
	Category    string `xml:"category"`
}

// Fetch retrieves articles from 嘶吼 via RSS feed.
func (g *SihouGrabber) Fetch(ctx context.Context, limit int) ([]*Article, error) {
	articles := make([]*Article, 0)

	g.LogInfo("Fetching RSS feed from 嘶吼...")

	resp, err := g.client.R().SetContext(ctx).Get("https://www.4hou.com/feed")
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

	var feed RSSFeed
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

		// Get full content - prefer content:encoded over description
		fullContent := item.Content
		if fullContent == "" {
			fullContent = item.Description
		}

		// Process images and convert to base64
		fullContent = g.processImagesInHTML(fullContent)

		// Convert HTML to markdown
		mdContent := g.htmlToMarkdown(fullContent)

		// Clean description for summary
		summary := g.cleanHTML(item.Description)
		if len(summary) > 300 {
			summary = summary[:300] + "..."
		}

		// Clean title
		title := g.cleanHTML(item.Title)

		article := &Article{
			Title:       title,
			Summary:     summary,
			Content:     mdContent,
			Source:      "嘶吼",
			URL:         item.Link,
			Author:      item.Author,
			Tags:        g.extractTags(item.Category),
			PublishedAt: publishedAt,
		}

		articles = append(articles, article)
		count++

		g.LogDebug("Parsed article: %s", article.Title)

		// Rate limit between requests
		time.Sleep(500 * time.Millisecond)
	}

	g.LogInfo("Successfully processed %d articles from 嘶吼 RSS feed", len(articles))
	return articles, nil
}

// processImagesInHTML finds images in HTML and converts them to base64.
func (g *SihouGrabber) processImagesInHTML(html string) string {
	if html == "" {
		return html
	}

	imgPattern := regexp.MustCompile(`<img[^>]+src=["']([^"']+)["'][^>]*>`)

	return imgPattern.ReplaceAllStringFunc(html, func(match string) string {
		srcParts := regexp.MustCompile(`src=["']([^"']+)["']`).FindStringSubmatch(match)
		if len(srcParts) == 2 {
			url := srcParts[1]
			// Skip already base64 or empty
			if strings.HasPrefix(url, "data:") || url == "" {
				return match
			}

			// Skip external URLs that might not be accessible
			if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
				return match
			}

			base64 := g.downloadImageAsBase64(url)
			if base64 != "" {
				return regexp.MustCompile(`src=["'][^"']+["']`).ReplaceAllString(match, fmt.Sprintf(`src="%s"`, base64))
			}
		}
		return match
	})
}

// downloadImageAsBase64 downloads an image and returns it as base64 data URI.
func (g *SihouGrabber) downloadImageAsBase64(imageURL string) string {
	if imageURL == "" {
		return ""
	}

	// Skip certain image types that are likely icons/ads
	if strings.Contains(imageURL, "pixel") || strings.Contains(imageURL, "tracking") {
		return ""
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

// htmlToMarkdown converts HTML content to Markdown format.
func (g *SihouGrabber) htmlToMarkdown(html string) string {
	if html == "" {
		return ""
	}

	converter := md.NewConverter("", true, nil)
	mdContent, err := converter.ConvertString(html)
	if err != nil {
		g.LogWarn("Failed to convert HTML to markdown: %v", err)
		return g.cleanHTML(html)
	}

	return mdContent
}

// cleanHTML removes HTML tags and decodes entities.
func (g *SihouGrabber) cleanHTML(html string) string {
	if html == "" {
		return ""
	}

	// Remove content:encoded wrapper if present
	re := regexp.MustCompile(`<!\[CDATA\[(.*)\]\]>`, )
	html = re.ReplaceAllString(html, "$1")

	// Remove HTML tags
	re2 := regexp.MustCompile(`<[^>]*>`)
	result := re2.ReplaceAllString(html, "")

	// Decode HTML entities
	result = strings.ReplaceAll(result, "&amp;", "&")
	result = strings.ReplaceAll(result, "&lt;", "<")
	result = strings.ReplaceAll(result, "&gt;", ">")
	result = strings.ReplaceAll(result, "&quot;", "\"")
	result = strings.ReplaceAll(result, "&#39;", "'")
	result = strings.ReplaceAll(result, "&nbsp;", " ")
	result = strings.ReplaceAll(result, "&#91;", "[")
	result = strings.ReplaceAll(result, "&#93;", "]")

	// Clean up whitespace
	lines := strings.Split(result, "\n")
	var cleanLines []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			cleanLines = append(cleanLines, trimmed)
		}
	}

	return strings.Join(cleanLines, "\n")
}

// extractTags extracts tags from category string.
func (g *SihouGrabber) extractTags(category string) []string {
	if category == "" {
		return []string{"安全文章", "嘶吼"}
	}

	// Split by comma or other delimiter
	tags := strings.Split(category, ",")
	var cleanTags []string
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag != "" {
			cleanTags = append(cleanTags, tag)
		}
	}

	if len(cleanTags) == 0 {
		cleanTags = []string{"安全文章", "嘶吼"}
	}

	return cleanTags
}

// parseDate parses date string from RSS format.
func (g *SihouGrabber) parseDate(dateStr string) time.Time {
	if dateStr == "" {
		return time.Time{}
	}

	// RSS date format: "Wed, 25 Mar 2026 14:09:33 +0800"
	layouts := []string{
		time.RFC1123,
		"Mon, 02 Jan 2006 15:04:05 +0800",
		"Mon, 02 Jan 2006 15:04:05 +0000",
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

func init() {
	Register(NewSihouGrabber())
}
