// Package grabber provides KEV (CISA Known Exploited Vulnerabilities) crawler using go-rod.
package vulngrabber

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/kataras/golog"

	"github.com/secflow/client/pkg/rodutil"
)

const KEVUrlRod = "https://www.cisa.gov/sites/default/files/feeds/known_exploited_vulnerabilities.json"

const KEVPageSizeRod = 10 // KEV每次都是返回全量数据，所以这里自己定义一下pagesize匹配原来的爬取逻辑

// KEVCrawlerRod is the go-rod based implementation for CISA KEV.
type KEVCrawlerRod struct {
	log *golog.Logger
}

// NewKEVCrawlerRod creates a new KEV crawler using go-rod.
func NewKEVCrawlerRod() Grabber {
	return &KEVCrawlerRod{
		log: golog.Child("[KEV-rod]"),
	}
}

func (c *KEVCrawlerRod) ProviderInfo() *Provider {
	return &Provider{
		Name:        "KEV",
		DisplayName: "Known Exploited Vulnerabilities Catalog",
		Link:        "https://www.cisa.gov/known-exploited-vulnerabilities-catalog",
	}
}

func (c *KEVCrawlerRod) IsValuable(info *VulnInfo) bool {
	return info.Severity == SeverityHigh || info.Severity == SeverityCritical
}

func (c *KEVCrawlerRod) GetUpdate(ctx context.Context, pageLimit int) ([]*VulnInfo, error) {
	// Create browser config
	config := rodutil.DefaultConfig()
	config.Headless = true

	// Get browser instance
	browser, err := rodutil.GetBrowser(config)
	if err != nil {
		return nil, fmt.Errorf("failed to get browser: %w", err)
	}

	// Create page with stealth mode
	page, err := rodutil.NewPage(browser)
	if err != nil {
		return nil, fmt.Errorf("failed to create page: %w", err)
	}
	defer page.Close()

	// Use JavaScript fetch API to get JSON data directly
	script := `() => {
		return new Promise((resolve, reject) => {
			fetch('https://www.cisa.gov/sites/default/files/feeds/known_exploited_vulnerabilities.json', {
				method: 'GET',
				headers: {
					'Accept': 'application/json',
					'Referer': 'https://www.cisa.gov/known-exploited-vulnerabilities-catalog'
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
	}`

	result, err := page.Eval(script)
	if err != nil {
		return nil, fmt.Errorf("failed to execute fetch script: %w", err)
	}

	// Parse JSON response
	jsonStr := result.Value.String()
	
	var body kevRespRod
	if err := json.Unmarshal([]byte(jsonStr), &body); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Check for errors
	if body.Title == "" && body.Count == 0 {
		return nil, fmt.Errorf("empty response from KEV API")
	}

	c.log.Infof("Fetched %d vulnerabilities from KEV", len(body.Vulnerabilities))

	// Process vulnerabilities
	var vulnInfos []*VulnInfo
	itemLimit := pageLimit * KEVPageSizeRod
	var maxCount = len(body.Vulnerabilities)
	if pageLimit*KEVPageSizeRod > maxCount {
		itemLimit = maxCount
	}

	// Sort by date added (newest first)
	sort.Slice(body.Vulnerabilities, func(i, j int) bool {
		return body.Vulnerabilities[i].DateAdded > body.Vulnerabilities[j].DateAdded
	})

	for i := 0; i < itemLimit; i++ {
		vuln := body.Vulnerabilities[i]
		vulnInfo := &VulnInfo{
			UniqueKey:   vuln.CveID + "_KEV",
			Title:       strings.TrimSpace(vuln.VulnerabilityName),
			Description: strings.TrimSpace(vuln.ShortDescription),
			Severity:    SeverityCritical, // 数据源本身无该字段，因为有在野利用直接提成SeverityCritical
			CVE:         strings.TrimSpace(vuln.CveID),
			Solutions:   strings.TrimSpace(vuln.RequiredAction),
			Disclosure:  strings.TrimSpace(vuln.DateAdded),
			From:        "https://www.cisa.gov/known-exploited-vulnerabilities-catalog",
			Creator:     c,
		}

		// Parse references from notes
		if vuln.Notes != "" {
			refs := strings.Split(vuln.Notes, ";")
			for _, ref := range refs {
				if ref = strings.TrimSpace(ref); ref != "" {
					vulnInfo.References = append(vulnInfo.References, ref)
				}
			}
		}

		// Set tags
		vulnInfo.Tags = []string{
			strings.TrimSpace(vuln.VendorProject),
			strings.TrimSpace(vuln.Product),
			"在野利用",
		}

		vulnInfos = append(vulnInfos, vulnInfo)
	}

	return vulnInfos, nil
}

type kevRespRod struct {
	Title           string    `json:"title"`
	CatalogVersion  string    `json:"catalogVersion"`
	DateReleased    string    `json:"dateReleased"`
	Count           int       `json:"count"`
	Vulnerabilities []struct {
		CveID                      string `json:"cveID"`
		VendorProject              string `json:"vendorProject"`
		Product                    string `json:"product"`
		VulnerabilityName          string `json:"vulnerabilityName"`
		DateAdded                  string `json:"dateAdded"`
		ShortDescription           string `json:"shortDescription"`
		RequiredAction             string `json:"requiredAction"`
		DueDate                    string `json:"dueDate"`
		KnownRansomwareCampaignUse string `json:"knownRansomwareCampaignUse"`
		Notes                      string `json:"notes"`
	} `json:"vulnerabilities"`
}
