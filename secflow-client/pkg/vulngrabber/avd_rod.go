// Package grabber provides AVD crawler using go-rod for better WAF bypass.
package vulngrabber

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/kataras/golog"

	"github.com/secflow/client/pkg/rodutil"
)

var avdCveIDRegexp = regexp.MustCompile(`^CVE-\d+-\d+$`)

// AVDCrawlerRod is the go-rod based implementation for AVD with WAF bypass.
type AVDCrawlerRod struct {
	log *golog.Logger
}

// NewAVDCrawlerRod creates a new AVD crawler using go-rod.
func NewAVDCrawlerRod() Grabber {
	return &AVDCrawlerRod{
		log: golog.Child("[aliyun-avd-rod]"),
	}
}

func (a *AVDCrawlerRod) ProviderInfo() *Provider {
	return &Provider{
		Name:        "avd",
		DisplayName: "阿里云漏洞库",
		Link:        "https://avd.aliyun.com/high-risk/list",
	}
}

func (a *AVDCrawlerRod) IsValuable(info *VulnInfo) bool {
	return info.Severity == SeverityHigh || info.Severity == SeverityCritical
}

func (a *AVDCrawlerRod) GetUpdate(ctx context.Context, pageLimit int) ([]*VulnInfo, error) {
	browser, err := rodutil.GetBrowser(nil)
	if err != nil {
		return nil, fmt.Errorf("get browser: %w", err)
	}

	var results []*VulnInfo
	for i := 1; i <= pageLimit; i++ {
		select {
		case <-ctx.Done():
			return results, ctx.Err()
		default:
		}

		pageResult, err := a.parsePage(ctx, browser, i)
		if err != nil {
			a.log.Errorf("parse page %d: %v", i, err)
			continue
		}

		a.log.Debugf("got %d vulns from page %d", len(pageResult), i)
		results = append(results, pageResult...)
	}

	return results, nil
}

func (a *AVDCrawlerRod) parsePage(ctx context.Context, browser *rod.Browser, pageNum int) ([]*VulnInfo, error) {
	u := fmt.Sprintf("https://avd.aliyun.com/high-risk/list?page=%d", pageNum)

	// Create page with bypass settings
	page, err := rodutil.NewPage(browser)
	if err != nil {
		return nil, fmt.Errorf("create page: %w", err)
	}
	defer page.Close()

	// Apply bypass techniques
	if err := rodutil.ApplyBypass(page, &rodutil.BypassConfig{
		SimulateMouse:   false,
		RandomDelays:    false,
		RandomViewport:  true,
		RotateUserAgent: true,
	}); err != nil {
		a.log.Warnf("apply bypass: %v", err)
	}

	// Navigate to list page
	if err := page.Navigate(u); err != nil {
		return nil, fmt.Errorf("navigate: %w", err)
	}

	if err := page.WaitLoad(); err != nil {
		return nil, fmt.Errorf("wait load: %w", err)
	}

	// Wait for table
	if err := page.WaitElementsMoreThan("tbody > tr", 0); err != nil {
		return nil, fmt.Errorf("wait for table: %w", err)
	}

	// Get all vulnerability links using JavaScript
	linksScript := `() => {
		const rows = document.querySelectorAll('tbody > tr');
		const links = [];
		rows.forEach(row => {
			const link = row.querySelector('td > a');
			if (link && link.href) {
				links.push(link.href);
			}
		});
		return JSON.stringify(links);
	}`

	result, err := page.Eval(linksScript)
	if err != nil {
		return nil, fmt.Errorf("get links: %w", err)
	}

	var links []string
	if err := json.Unmarshal([]byte(result.Value.String()), &links); err != nil {
		return nil, fmt.Errorf("parse links: %w", err)
	}

	if len(links) == 0 {
		return nil, fmt.Errorf("no links found")
	}

	a.log.Debugf("found %d vuln links", len(links))

	// Parse each vuln using the same page (navigate to detail and back)
	results := make([]*VulnInfo, 0, len(links))
	for _, vulnLink := range links {
		select {
		case <-ctx.Done():
			return results, ctx.Err()
		default:
		}

		avdInfo, err := a.parseSingleWithFetch(page, vulnLink)
		if err != nil {
			a.log.Errorf("parse %s: %v", vulnLink, err)
			continue
		}

		results = append(results, avdInfo)
	}

	return results, nil
}

// parseSingleWithFetch uses fetch API to get vuln details without creating new page
func (a *AVDCrawlerRod) parseSingleWithFetch(page *rod.Page, vulnLink string) (*VulnInfo, error) {
	a.log.Debugf("parsing vuln %s", vulnLink)

	// Parse AVD ID from URL
	u, _ := url.Parse(vulnLink)
	avd := strings.TrimSpace(u.Query().Get("id"))

	// Use fetch API to get the page content
	fetchScript := fmt.Sprintf(`() => {
		return new Promise((resolve, reject) => {
			fetch('%s', {
				method: 'GET',
				headers: {
					'Accept': 'text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8',
					'Accept-Language': 'zh-CN,zh;q=0.9,en;q=0.8',
					'Cache-Control': 'no-cache',
					'Pragma': 'no-cache'
				}
			})
			.then(response => {
				if (!response.ok) {
					throw new Error('HTTP error! status: ' + response.status);
				}
				return response.text();
			})
			.then(html => resolve(html))
			.catch(error => reject(error.message));
		});
	}`, vulnLink)

	result, err := page.Eval(fetchScript)
	if err != nil {
		return nil, fmt.Errorf("fetch vuln page: %w", err)
	}

	html := result.Value.String()
	return a.parseVulnHTML(page, vulnLink, avd, html)
}

func (a *AVDCrawlerRod) parseVulnHTML(page *rod.Page, vulnLink, avdID, html string) (*VulnInfo, error) {
	// Escape backticks and backslashes in HTML for JavaScript template literal
	escapedHTML := strings.ReplaceAll(html, "\\", "\\\\")
	escapedHTML = strings.ReplaceAll(escapedHTML, "`", "\\`")
	escapedHTML = strings.ReplaceAll(escapedHTML, "${", "\\${")

	// Create a temporary DOM to parse the HTML
	script := fmt.Sprintf(`() => {
		const parser = new DOMParser();
		const doc = parser.parseFromString(`+"`"+`%s`+"`"+`, 'text/html');
		
		const result = {
			title: "",
			description: "",
			fixSteps: "",
			level: "",
			cveID: "",
			disclosure: "",
			refs: [],
			tags: []
		};

		// Parse metrics
		doc.querySelectorAll('.metric').forEach(m => {
			const label = m.querySelector('.metric-label')?.textContent?.trim() || '';
			const value = m.querySelector('.metric-value')?.textContent?.trim() || '';

			if (label.startsWith('CVE')) {
				result.cveID = value;
			} else if (label.includes('利用情况') && value !== '暂无') {
				result.tags.push(value.replace(/\s/g, ''));
			} else if (label.includes('披露时间')) {
				result.disclosure = value;
			}
		});

		// Parse title and level
		const header = doc.querySelector('h5.header__title');
		if (header) {
			result.level = header.querySelector('.badge')?.textContent?.trim() || '';
			result.title = header.querySelector('.header__title__text')?.textContent?.trim() || '';
		}

		// Parse main content
		const mainContent = doc.querySelector('div.py-4.pl-4.pr-4.px-2.bg-white.rounded.shadow-sm');
		if (mainContent) {
			const children = mainContent.children;
			for (let i = 0; i < children.length; i++) {
				const sentinel = children[i].textContent?.trim() || '';

				if (sentinel === '漏洞描述' && i + 1 < children.length) {
					result.description = children[i + 1].querySelector('div')?.textContent?.trim() || '';
					i++;
				} else if (sentinel === '解决建议' && i + 1 < children.length) {
					const fixDiv = children[i + 1];
					const textNodes = [];
					for (const node of fixDiv.childNodes) {
						if (node.nodeType === Node.TEXT_NODE) {
							const text = node.textContent?.trim();
							if (text) textNodes.push(text);
						}
					}
					result.fixSteps = textNodes.join('\n').replace(/、/g, '. ');
					i++;
				}
			}
		}

		// Parse references
		doc.querySelectorAll('div.reference tbody > tr a').forEach(a => {
			const href = a.getAttribute('href');
			if (href && href.startsWith('http')) {
				result.refs.push(href);
			}
		});

		return JSON.stringify(result);
	}`, escapedHTML)

	result, err := page.Eval(script)
	if err != nil {
		return nil, fmt.Errorf("parse html: %w", err)
	}

	// Parse JSON result
	jsonStr := result.Value.String()

	var data struct {
		Title       string   `json:"title"`
		Description string   `json:"description"`
		FixSteps    string   `json:"fixSteps"`
		Level       string   `json:"level"`
		CveID       string   `json:"cveID"`
		Disclosure  string   `json:"disclosure"`
		Refs        []string `json:"refs"`
		Tags        []string `json:"tags"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return nil, fmt.Errorf("parse result json: %w", err)
	}

	// Validate CVE
	if !avdCveIDRegexp.MatchString(data.CveID) {
		data.CveID = ""
	}

	// Validate disclosure date
	if _, err := time.Parse("2006-01-02", data.Disclosure); err != nil {
		data.Disclosure = ""
	}

	if data.CveID == "" && data.Disclosure == "" {
		return nil, fmt.Errorf("invalid vuln data: no CVE or disclosure date")
	}

	// Parse severity
	severity := SeverityLow
	switch data.Level {
	case "低危":
		severity = SeverityLow
	case "中危":
		severity = SeverityMedium
	case "高危":
		severity = SeverityHigh
	case "严重":
		severity = SeverityCritical
	}

	return &VulnInfo{
		UniqueKey:   avdID,
		Title:       data.Title,
		Description: data.Description,
		Severity:    severity,
		CVE:         data.CveID,
		Disclosure:  data.Disclosure,
		References:  data.Refs,
		Solutions:   data.FixSteps,
		From:        vulnLink,
		Tags:        data.Tags,
		Creator:     a,
	}, nil
}
