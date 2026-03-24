// Package grabber provides VulHub crawler using go-rod.
package vulngrabber

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/go-rod/rod"
	"github.com/kataras/golog"

	"github.com/secflow/client/pkg/rodutil"
)

// VulHubCrawlerRod is the go-rod based implementation for VulHub.
type VulHubCrawlerRod struct {
	log *golog.Logger
}

// NewVulHubCrawlerRod creates a new VulHub crawler using go-rod.
func NewVulHubCrawlerRod() Grabber {
	return &VulHubCrawlerRod{
		log: golog.Child("[VulHub-rod]"),
	}
}

func (c *VulHubCrawlerRod) ProviderInfo() *Provider {
	return &Provider{
		Name:        "VulHub",
		DisplayName: "VulHub 漏洞库",
		Link:        "https://www.vulhub.org.cn/",
	}
}

func (c *VulHubCrawlerRod) IsValuable(info *VulnInfo) bool {
	return info.Severity == SeverityHigh || info.Severity == SeverityCritical
}

func (c *VulHubCrawlerRod) GetUpdate(ctx context.Context, pageLimit int) ([]*VulnInfo, error) {
	// VulHub API endpoint
	apiURL := "https://www.vulhub.org.cn/api/v1/vuln/list?page=1&page_size=20&order=latest"

	c.log.Infof("Fetching VulHub vulnerabilities from: %s", apiURL)

	config := rodutil.DefaultConfig()
	config.Headless = true

	browser, err := rodutil.GetBrowser(config)
	if err != nil {
		return nil, fmt.Errorf("failed to get browser: %w", err)
	}

	page, err := rodutil.NewPage(browser)
	if err != nil {
		return nil, fmt.Errorf("failed to create page: %w", err)
	}
	defer page.Close()

	// Use JavaScript fetch to get data
	script := fmt.Sprintf(`() => {
		return new Promise((resolve, reject) => {
			fetch('%s', {
				method: 'GET',
				headers: {
					'Accept': 'application/json',
					'Referer': 'https://www.vulhub.org.cn/'
				}
			})
			.then(response => {
				if (!response.ok) {
					throw new Error('HTTP error! status: ' + response.status);
				}
				return response.json();
			})
			.then(data => resolve(JSON.stringify(data)))
			.catch(error => reject(error.message));
		});
	}`, apiURL)

	result, err := page.Eval(script)
	if err != nil {
		// If API fails, try to crawl the main page
		c.log.Warnf("API fetch failed, trying main page: %v", err)
		return c.crawlFromPage(ctx, page, pageLimit)
	}

	jsonStr := result.Value.String()
	var vulhubResp VulHubResponse
	if err := json.Unmarshal([]byte(jsonStr), &vulhubResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if vulhubResp.Code != 0 || len(vulhubResp.Data.Items) == 0 {
		c.log.Warnf("API returned no data, trying main page")
		return c.crawlFromPage(ctx, page, pageLimit)
	}

	c.log.Infof("Fetched %d vulnerabilities from VulHub", len(vulhubResp.Data.Items))

	return c.parseItems(vulhubResp.Data.Items, pageLimit)
}

func (c *VulHubCrawlerRod) crawlFromPage(ctx context.Context, page *rod.Page, pageLimit int) ([]*VulnInfo, error) {
	// Fallback: crawl from the main vulnerability listing page
	mainURL := "https://www.vulhub.org.cn/vuldb"

	c.log.Infof("Crawling VulHub from: %s", mainURL)

	if err := rodutil.SafeNavigate(page, mainURL, rodutil.DefaultBypassConfig()); err != nil {
		return nil, fmt.Errorf("failed to navigate: %w", err)
	}

	// Wait for content to load
	page.WaitLoad()

	// Get page content using HTML() method
	content, err := page.HTML()
	if err != nil {
		return nil, fmt.Errorf("failed to get page content: %w", err)
	}

	// Parse vulnerabilities from HTML
	// Try to find JSON data embedded in the page
	startIdx := strings.Index(content, `"vulnList":`)
	if startIdx == -1 {
		startIdx = strings.Index(content, `"items":`)
	}
	if startIdx == -1 {
		c.log.Warnf("No vulnerability data found in page")
		return []*VulnInfo{}, nil
	}

	// Find the end of the JSON object
	endIdx := strings.Index(content[startIdx:], "</script>")
	if endIdx == -1 {
		endIdx = len(content)
	}
	jsonStr := content[startIdx : startIdx+endIdx]

	// Try to parse as JSON
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		c.log.Warnf("Failed to parse embedded JSON: %v", err)
		return []*VulnInfo{}, nil
	}

	// Extract items
	items, ok := data["items"].([]interface{})
	if !ok {
		items, ok = data["vulnList"].([]interface{})
	}
	if !ok {
		return []*VulnInfo{}, nil
	}

	return c.parseInterfaceItems(items, pageLimit)
}

func (c *VulHubCrawlerRod) parseItems(items []VulHubItem, pageLimit int) ([]*VulnInfo, error) {
	limit := pageLimit
	if limit > len(items) {
		limit = len(items)
	}

	var vulnInfos []*VulnInfo

	for i := 0; i < limit; i++ {
		item := items[i]

		// Parse severity
		severity := SeverityMedium
		severityStr := strings.ToLower(item.Severity)
		if strings.Contains(severityStr, "critical") || strings.Contains(severityStr, "严重") {
			severity = SeverityCritical
		} else if strings.Contains(severityStr, "high") || strings.Contains(severityStr, "高") {
			severity = SeverityHigh
		} else if strings.Contains(severityStr, "medium") || strings.Contains(severityStr, "中") {
			severity = SeverityMedium
		} else if strings.Contains(severityStr, "low") || strings.Contains(severityStr, "低") {
			severity = SeverityLow
		}

		vulnInfo := &VulnInfo{
			UniqueKey:   fmt.Sprintf("VULHUB_%d", item.ID),
			Title:       item.Title,
			Description: item.Description,
			Severity:    severity,
			CVE:         item.CVE,
			Disclosure:  item.Published,
			Solutions:   item.Solution,
			References:  item.References,
			Tags:        []string{item.Category},
			From:        fmt.Sprintf("https://www.vulhub.org.cn/vuldb/%d", item.ID),
			Creator:     c,
		}

		vulnInfos = append(vulnInfos, vulnInfo)
	}

	return vulnInfos, nil
}

func (c *VulHubCrawlerRod) parseInterfaceItems(items []interface{}, pageLimit int) ([]*VulnInfo, error) {
	limit := pageLimit
	if limit > len(items) {
		limit = len(items)
	}

	var vulnInfos []*VulnInfo

	for i := 0; i < limit; i++ {
		item, ok := items[i].(map[string]interface{})
		if !ok {
			continue
		}

		id := int64(0)
		if idVal, ok := item["id"].(float64); ok {
			id = int64(idVal)
		}

		title := ""
		if titleVal, ok := item["title"].(string); ok {
			title = titleVal
		}

		description := ""
		if descVal, ok := item["description"].(string); ok {
			description = descVal
		}

		cve := ""
		if cveVal, ok := item["cve"].(string); ok {
			cve = cveVal
		}

		published := ""
		if pubVal, ok := item["published"].(string); ok {
			published = pubVal
		}

		vulnInfo := &VulnInfo{
			UniqueKey:   fmt.Sprintf("VULHUB_%d", id),
			Title:       title,
			Description: description,
			Severity:    SeverityMedium,
			CVE:         cve,
			Disclosure:  published,
			From:        fmt.Sprintf("https://www.vulhub.org.cn/vuldb/%d", id),
			Creator:     c,
		}

		vulnInfos = append(vulnInfos, vulnInfo)
	}

	return vulnInfos, nil
}

// VulHubResponse represents the VulHub API response.
type VulHubResponse struct {
	Code int `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		Items []VulHubItem `json:"items"`
		Total int `json:"total"`
	} `json:"data"`
}

// VulHubItem represents a single vulnerability item.
type VulHubItem struct {
	ID          int      `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Severity    string   `json:"severity"`
	CVE         string   `json:"cve"`
	Published   string   `json:"published"`
	Solution    string   `json:"solution"`
	References  []string `json:"references"`
	Category    string   `json:"category"`
}
