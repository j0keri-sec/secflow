// Package grabber provides PacketStorm crawler using go-rod.
package vulngrabber

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/kataras/golog"

	"github.com/secflow/client/pkg/rodutil"
)

// PacketStormCrawlerRod is the go-rod based implementation for PacketStorm.
type PacketStormCrawlerRod struct {
	log *golog.Logger
}

// NewPacketStormCrawlerRod creates a new PacketStorm crawler using go-rod.
func NewPacketStormCrawlerRod() Grabber {
	return &PacketStormCrawlerRod{
		log: golog.Child("[PacketStorm-rod]"),
	}
}

func (c *PacketStormCrawlerRod) ProviderInfo() *Provider {
	return &Provider{
		Name:        "PacketStorm",
		DisplayName: "PacketStorm Security",
		Link:        "https://packetstormsecurity.com/",
	}
}

func (c *PacketStormCrawlerRod) IsValuable(info *VulnInfo) bool {
	return info.Severity == SeverityHigh || info.Severity == SeverityCritical
}

func (c *PacketStormCrawlerRod) GetUpdate(ctx context.Context, pageLimit int) ([]*VulnInfo, error) {
	// PacketStorm recent vulnerabilities feed
	baseURL := "https://packetstormsecurity.com/"

	c.log.Infof("Fetching PacketStorm vulnerabilities from: %s", baseURL)

	browser, err := rodutil.GetBrowser(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get browser: %w", err)
	}

	page, err := rodutil.NewPage(browser)
	if err != nil {
		return nil, fmt.Errorf("failed to create page: %w", err)
	}
	defer page.Close()

	// Apply bypass config
	bypassConfig := rodutil.DefaultBypassConfig()
	if err := rodutil.ApplyBypass(page, bypassConfig); err != nil {
		c.log.Warnf("apply bypass: %v", err)
	}

	// Navigate to the page
	if err := rodutil.SafeNavigate(page, baseURL, bypassConfig); err != nil {
		return nil, fmt.Errorf("failed to navigate: %w", err)
	}

	// Wait for content to load
	page.WaitLoad()

	// Wait a bit for dynamic content
	time.Sleep(2 * time.Second)

	// Get page content using HTML() method
	content, err := page.HTML()
	if err != nil {
		return nil, fmt.Errorf("failed to get page content: %w", err)
	}

	// Parse vulnerabilities from HTML
	return c.parsePage(content, pageLimit)
}

func (c *PacketStormCrawlerRod) parsePage(content string, pageLimit int) ([]*VulnInfo, error) {
	// Pattern to extract vulnerability entries
	// PacketStorm uses different formats, try common patterns
	pattern := regexp.MustCompile(`<li[^>]*class="[^"]*file[^"]*"[^>]*>.*?<a[^>]*href="(/files/[^"]+)"[^>]*>([^<]+)</a>.*?(?:CVE-[\d-]+)?.*?(?:<dd[^>]*>([^<]*)|</li>)`)

	matches := pattern.FindAllStringSubmatch(content, -1)

	if len(matches) == 0 {
		// Try alternative pattern
		pattern2 := regexp.MustCompile(`<dt><a href="(/files/\d+/[^"]+)">([^<]+)</a></dt>`)
		matches = pattern2.FindAllStringSubmatch(content, -1)
	}

	if len(matches) == 0 {
		c.log.Warnf("No vulnerability entries found in page")
		return []*VulnInfo{}, nil
	}

	limit := pageLimit
	if limit > len(matches) {
		limit = len(matches)
	}

	var vulnInfos []*VulnInfo

	for i := 0; i < limit; i++ {
		match := matches[i]

		if len(match) < 3 {
			continue
		}

		url := "https://packetstormsecurity.com" + match[1]
		title := strings.TrimSpace(match[2])

		// Extract CVE from title if present
		cvePattern := regexp.MustCompile(`(CVE-[\d-]+)`)
		cveMatch := cvePattern.FindStringSubmatch(title)
		cve := ""
		if len(cveMatch) > 1 {
			cve = cveMatch[1]
		}

		// Determine severity from title keywords
		severity := SeverityMedium
		lowerTitle := strings.ToLower(title)
		if strings.Contains(lowerTitle, "remote code execution") ||
			strings.Contains(lowerTitle, "rce") ||
			strings.Contains(lowerTitle, "sql injection") ||
			strings.Contains(lowerTitle, "xss") ||
			strings.Contains(lowerTitle, "csrf") ||
			strings.Contains(lowerTitle, "path traversal") {
			severity = SeverityHigh
		}

		vulnInfo := &VulnInfo{
			UniqueKey:   fmt.Sprintf("PACKETSTORM_%d", i),
			Title:       title,
			Description: title,
			Severity:    severity,
			CVE:         cve,
			Disclosure:  time.Now().Format("2006-01-02"),
			From:        url,
			Tags:        []string{"exploit", "packetstorm"},
			Creator:     c,
		}

		vulnInfos = append(vulnInfos, vulnInfo)
	}

	c.log.Infof("Parsed %d vulnerabilities from PacketStorm", len(vulnInfos))

	return vulnInfos, nil
}
