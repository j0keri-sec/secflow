// Package grabber provides Apache Struts2 Security Bulletins crawler using go-rod.
package vulngrabber

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/go-rod/rod"
	"github.com/kataras/golog"

	"github.com/secflow/client/pkg/rodutil"
)

const Struts2UrlRod = "https://cwiki.apache.org/confluence/display/WW/Security+Bulletins"

// Struts2CrawlerRod is the go-rod based implementation for Apache Struts2 Security Bulletins.
type Struts2CrawlerRod struct {
	log *golog.Logger
}

// NewStruts2CrawlerRod creates a new Struts2 crawler using go-rod.
func NewStruts2CrawlerRod() Grabber {
	return &Struts2CrawlerRod{
		log: golog.Child("[Struts2-Security-rod]"),
	}
}

func (c *Struts2CrawlerRod) ProviderInfo() *Provider {
	return &Provider{
		Name:        "Struts2",
		DisplayName: "Apache Struts2 Security Bulletins",
		Link:        Struts2UrlRod,
	}
}

func (c *Struts2CrawlerRod) IsValuable(info *VulnInfo) bool {
	return info.Severity == SeverityHigh || info.Severity == SeverityCritical
}

func (c *Struts2CrawlerRod) GetUpdate(ctx context.Context, vulnLimit int) ([]*VulnInfo, error) {
	// Get browser instance
	browser, err := rodutil.GetBrowser(nil)
	if err != nil {
		return nil, fmt.Errorf("get browser: %w", err)
	}

	// Create page with stealth mode
	page, err := rodutil.NewPage(browser)
	if err != nil {
		return nil, fmt.Errorf("create page: %w", err)
	}
	defer page.Close()

	// Use JavaScript fetch to get the page content
	script := `() => {
		return new Promise((resolve, reject) => {
			fetch('https://cwiki.apache.org/confluence/display/WW/Security+Bulletins')
				.then(response => response.text())
				.then(html => resolve(html))
				.catch(error => reject(error.message));
		});
	}`

	result, err := page.Eval(script)
	if err != nil {
		return nil, fmt.Errorf("fetch page: %w", err)
	}

	html := result.Value.String()

	// Parse the HTML to extract vulnerability list
	vulnInfos, err := c.parseBulletinList(ctx, page, html, vulnLimit)
	if err != nil {
		return nil, err
	}

	c.log.Debugf("got %d vulns", len(vulnInfos))
	return vulnInfos, nil
}

func (c *Struts2CrawlerRod) parseBulletinList(ctx context.Context, page *rod.Page, html string, vulnLimit int) ([]*VulnInfo, error) {
	var vulnInfos []*VulnInfo

	// Use JavaScript to extract the vulnerability list
	// First, set the HTML content to a new document for parsing
	script := `(htmlContent, limit) => {
		// Create a temporary div to parse HTML
		const tempDiv = document.createElement('div');
		tempDiv.innerHTML = htmlContent;
		
		// Find all list items in the main content area
		const mainContent = tempDiv.querySelector('#main-content');
		if (!mainContent) {
			return JSON.stringify({error: 'main-content not found'});
		}
		
		const lists = mainContent.querySelectorAll('ul');
		const results = [];
		
		// Process each ul list
		for (const ul of lists) {
			const items = ul.querySelectorAll(':scope > li');
			const startIndex = Math.max(0, items.length - limit);
			
			for (let i = startIndex; i < items.length; i++) {
				const item = items[i];
				const link = item.querySelector('a');
				if (link) {
					const href = link.getAttribute('href');
					// Only include links to S2 bulletins
					if (href && href.includes('S2-')) {
						results.push({
							title: link.textContent.trim(),
							href: href
						});
					}
				}
			}
		}
		
		return JSON.stringify(results.slice(-limit));
	}`

	result, err := page.Eval(script, html, vulnLimit)
	if err != nil {
		return nil, fmt.Errorf("extract bulletin list: %w", err)
	}

	// Parse the result
	jsonStr := result.Value.String()
	
	// Check for error
	var errorCheck struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal([]byte(jsonStr), &errorCheck); err == nil && errorCheck.Error != "" {
		return nil, fmt.Errorf("parse error: %s", errorCheck.Error)
	}
	
	var bulletins []struct {
		Title string `json:"title"`
		Href  string `json:"href"`
	}
	if err := json.Unmarshal([]byte(jsonStr), &bulletins); err != nil {
		return nil, fmt.Errorf("parse bulletin list: %w", err)
	}

	c.log.Infof("Found %d bulletins", len(bulletins))

	s2Id := regexp.MustCompile(`S2-\d{3}`)

	// Process each bulletin
	for _, bulletin := range bulletins {
		fullLink := "https://cwiki.apache.org" + bulletin.Href
		
		vuln, err := c.getVulnInfoFromURL(ctx, page, fullLink)
		if err != nil {
			c.log.Error(err)
			continue
		}
		
		vuln.Title = bulletin.Title
		vuln.UniqueKey = s2Id.FindString(bulletin.Title)
		if vuln.UniqueKey == "" {
			c.log.Warnf("can not find unique key from %s", bulletin.Title)
			continue
		}
		vuln.Creator = c
		vulnInfos = append(vulnInfos, vuln)
	}

	return vulnInfos, nil
}

func (c *Struts2CrawlerRod) getVulnInfoFromURL(ctx context.Context, page *rod.Page, url string) (*VulnInfo, error) {
	// Use fetch API to get the page content
	script := `(url) => {
		return new Promise((resolve, reject) => {
			fetch(url)
				.then(response => response.text())
				.then(html => resolve(html))
				.catch(error => reject(error.message));
		});
	}`

	result, err := page.Eval(script, url)
	if err != nil {
		return &VulnInfo{From: url}, nil
	}

	html := result.Value.String()

	// Extract vulnerability details using JavaScript on the HTML
	extractScript := `(htmlContent) => {
		const tempDiv = document.createElement('div');
		tempDiv.innerHTML = htmlContent;
		
		const result = {};
		
		// Extract severity - look for table cells
		const ths = tempDiv.querySelectorAll('th');
		for (const th of ths) {
			const text = th.textContent.trim();
			const nextTd = th.nextElementSibling;
			if (nextTd) {
				if (text.includes('Maximum security rating')) {
					result.severity = nextTd.textContent.trim();
				} else if (text.includes('CVE Identifier')) {
					result.cve = nextTd.textContent.trim();
				} else if (text.includes('Impact of vulnerability')) {
					result.impact = nextTd.textContent.trim();
				}
			}
		}
		
		// Extract description - look for Problem section
		const h2s = tempDiv.querySelectorAll('h2');
		for (const h2 of h2s) {
			const id = h2.getAttribute('id') || '';
			if (id.endsWith('-Problem')) {
				let nextEl = h2.nextElementSibling;
				if (nextEl) {
					result.description = nextEl.textContent.trim();
				}
			} else if (id.endsWith('-Solution')) {
				let nextEl = h2.nextElementSibling;
				if (nextEl) {
					result.solution = nextEl.textContent.trim();
				}
			}
		}
		
		return JSON.stringify(result);
	}`

	extractResult, err := page.Eval(extractScript, html)
	if err != nil {
		return &VulnInfo{From: url}, nil
	}

	var details struct {
		Severity    string `json:"severity"`
		CVE         string `json:"cve"`
		Description string `json:"description"`
		Solution    string `json:"solution"`
		Impact      string `json:"impact"`
	}
	if err := json.Unmarshal([]byte(extractResult.Value.String()), &details); err != nil {
		return &VulnInfo{From: url}, nil
	}

	vuln := &VulnInfo{
		Severity:    c.getSeverityFromString(details.Severity),
		CVE:         details.CVE,
		Description: details.Description,
		Solutions:   details.Solution,
		Tags:        []string{details.Impact},
		From:        url,
	}

	return vuln, nil
}

func (c *Struts2CrawlerRod) getVulnInfoSimple(page *rod.Page, url string) (*VulnInfo, error) {
	// Simpler extraction using page text
	script := `() => {
		const text = document.body.innerText;
		return JSON.stringify({text: text});
	}`

	result, err := page.Eval(script)
	if err != nil {
		return &VulnInfo{From: url}, nil
	}

	var data struct {
		Text string `json:"text"`
	}
	json.Unmarshal([]byte(result.Value.String()), &data)

	// Try to extract CVE from text
	cveRegex := regexp.MustCompile(`CVE-\d{4}-\d+`)
	cve := ""
	if matches := cveRegex.FindStringSubmatch(data.Text); len(matches) > 0 {
		cve = matches[0]
	}

	return &VulnInfo{
		CVE:  cve,
		From: url,
	}, nil
}

func (c *Struts2CrawlerRod) getSeverityFromString(severityText string) SeverityLevel {
	switch strings.ToLower(strings.TrimSpace(severityText)) {
	case "critical":
		return SeverityCritical
	case "important":
		return SeverityHigh
	case "moderate":
		return SeverityMedium
	case "low":
		return SeverityLow
	default:
		return SeverityLow
	}
}
