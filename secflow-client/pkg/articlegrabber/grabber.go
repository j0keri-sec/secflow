// Package article implements grabbers for security news and articles.
package articlegrabber

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/imroc/req/v3"
)

// Article represents a security article or news item.
type Article struct {
	Title       string    `json:"title"`
	Summary     string    `json:"summary"`
	Content     string    `json:"content"`
	Author      string    `json:"author"`
	Source      string    `json:"source"`
	URL         string    `json:"url"`
	Image       string    `json:"image"` // cover image URL
	Tags        []string  `json:"tags"`
	PublishedAt time.Time `json:"published_at"`
}

// Grabber is the interface for article sources.
type Grabber interface {
	Name() string
	Fetch(ctx context.Context, limit int) ([]*Article, error)
}

// Registry holds all registered grabbers.
var registry = make(map[string]Grabber)

// Register adds a grabber to the registry.
func Register(g Grabber) {
	registry[g.Name()] = g
}

// Get returns a grabber by name.
func Get(name string) (Grabber, bool) {
	g, ok := registry[name]
	return g, ok
}

// List returns all registered grabber names.
func List() []string {
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	return names
}

// FreeBufGrabber fetches articles from FreeBuf (安全脉搏).
type FreeBufGrabber struct {
	client *req.Client
}

// NewFreeBufGrabber creates a new FreeBuf grabber.
func NewFreeBufGrabber() *FreeBufGrabber {
	return &FreeBufGrabber{
		client: req.NewClient().
			SetTimeout(30 * time.Second).
			SetUserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"),
	}
}

// Name returns the grabber name.
func (g *FreeBufGrabber) Name() string {
	return "freebuf"
}

// Fetch retrieves articles from FreeBuf.
func (g *FreeBufGrabber) Fetch(ctx context.Context, limit int) ([]*Article, error) {
	// FreeBuf 首页
	resp, err := g.client.R().SetContext(ctx).Get("https://www.freebuf.com/")
	if err != nil {
		return nil, fmt.Errorf("fetch freebuf: %w", err)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("parse freebuf: %w", err)
	}

	var articles []*Article
	doc.Find(".article-item").Each(func(i int, s *goquery.Selection) {
		if len(articles) >= limit {
			return
		}

		title := strings.TrimSpace(s.Find(".title").Text())
		link, _ := s.Find(".title a").Attr("href")
		summary := strings.TrimSpace(s.Find(".summary").Text())
		author := strings.TrimSpace(s.Find(".author").Text())
		timeStr := strings.TrimSpace(s.Find(".time").Text())

		if title == "" || link == "" {
			return
		}

		// Parse time
		publishedAt, _ := time.Parse("2006-01-02", timeStr)
		if publishedAt.IsZero() {
			publishedAt = time.Now()
		}

		// Ensure full URL
		if !strings.HasPrefix(link, "http") {
			link = "https://www.freebuf.com" + link
		}

		articles = append(articles, &Article{
			Title:       title,
			Summary:     summary,
			Source:      "FreeBuf",
			URL:         link,
			Author:      author,
			Tags:        []string{"安全资讯"},
			PublishedAt: publishedAt,
		})
	})

	return articles, nil
}

// SecurityWeekGrabber fetches articles from SecurityWeek.
type SecurityWeekGrabber struct {
	client *req.Client
}

// NewSecurityWeekGrabber creates a new SecurityWeek grabber.
func NewSecurityWeekGrabber() *SecurityWeekGrabber {
	return &SecurityWeekGrabber{
		client: req.NewClient().
			SetTimeout(30 * time.Second).
			SetUserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"),
	}
}

// Name returns the grabber name.
func (g *SecurityWeekGrabber) Name() string {
	return "securityweek"
}

// Fetch retrieves articles from SecurityWeek RSS feed.
func (g *SecurityWeekGrabber) Fetch(ctx context.Context, limit int) ([]*Article, error) {
	resp, err := g.client.R().SetContext(ctx).Get("https://www.securityweek.com/feed/")
	if err != nil {
		return nil, fmt.Errorf("fetch securityweek: %w", err)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("parse securityweek: %w", err)
	}

	var articles []*Article
	doc.Find("item").Each(func(i int, s *goquery.Selection) {
		if len(articles) >= limit {
			return
		}

		title := strings.TrimSpace(s.Find("title").Text())
		link := strings.TrimSpace(s.Find("link").Text())
		description := strings.TrimSpace(s.Find("description").Text())
		pubDate := strings.TrimSpace(s.Find("pubDate").Text())

		// Clean up title (remove CDATA)
		title = cleanCDATA(title)
		description = cleanCDATA(description)

		// Parse time
		publishedAt, _ := time.Parse(time.RFC1123, pubDate)
		if publishedAt.IsZero() {
			publishedAt = time.Now()
		}

		articles = append(articles, &Article{
			Title:       title,
			Summary:     description,
			Source:      "SecurityWeek",
			URL:         link,
			Tags:        []string{"国际安全资讯"},
			PublishedAt: publishedAt,
		})
	})

	return articles, nil
}

// HackernewsGrabber fetches top stories from Hacker News.
type HackernewsGrabber struct {
	client *http.Client
}

// NewHackernewsGrabber creates a new Hacker News grabber.
func NewHackernewsGrabber() *HackernewsGrabber {
	return &HackernewsGrabber{
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// Name returns the grabber name.
func (g *HackernewsGrabber) Name() string {
	return "hackernews"
}

// Fetch retrieves top stories from Hacker News.
func (g *HackernewsGrabber) Fetch(ctx context.Context, limit int) ([]*Article, error) {
	// Get top stories IDs
	resp, err := g.client.Get("https://hacker-news.firebaseio.com/v0/topstories.json")
	if err != nil {
		return nil, fmt.Errorf("fetch hackernews stories: %w", err)
	}
	defer resp.Body.Close()

	var storyIDs []int
	if err := json.NewDecoder(resp.Body).Decode(&storyIDs); err != nil {
		return nil, fmt.Errorf("decode story ids: %w", err)
	}

	if len(storyIDs) > limit {
		storyIDs = storyIDs[:limit]
	}

	var articles []*Article
	for _, id := range storyIDs {
		story, err := g.fetchStory(ctx, id)
		if err != nil {
			continue
		}
		// Filter for security-related stories
		if isSecurityRelated(story.Title) {
			articles = append(articles, story)
		}
	}

	return articles, nil
}

func (g *HackernewsGrabber) fetchStory(ctx context.Context, id int) (*Article, error) {
	url := fmt.Sprintf("https://hacker-news.firebaseio.com/v0/item/%d.json", id)
	resp, err := g.client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var data struct {
		Title string `json:"title"`
		URL   string `json:"url"`
		Time  int64  `json:"time"`
		By    string `json:"by"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	return &Article{
		Title:       data.Title,
		URL:         data.URL,
		Author:      data.By,
		Source:      "Hacker News",
		Tags:        []string{"技术资讯"},
		PublishedAt: time.Unix(data.Time, 0),
	}, nil
}

// Helper functions

func cleanCDATA(s string) string {
	re := regexp.MustCompile(`<!\[CDATA\[(.*?)\]\]>`)
	return re.ReplaceAllString(s, "$1")
}

func isSecurityRelated(title string) bool {
	securityKeywords := []string{
		"security", "vulnerability", "exploit", "cve", "hack",
		"breach", "attack", "malware", "ransomware", "phishing",
		"漏洞", "安全", "攻击", "恶意软件", "勒索软件",
	}
	titleLower := strings.ToLower(title)
	for _, kw := range securityKeywords {
		if strings.Contains(titleLower, kw) {
			return true
		}
	}
	return false
}

// Init registers all article grabbers.
func Init() {
	Register(NewFreeBufGrabber())
	Register(NewSecurityWeekGrabber())
	Register(NewHackernewsGrabber())
	Register(NewQianxinWeeklyGrabber())
	Register(NewVenustechGrabber())
	Register(NewXianzhiGrabber())
}
