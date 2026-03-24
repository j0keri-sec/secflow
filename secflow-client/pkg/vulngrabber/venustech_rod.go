// Package grabber provides Venustech crawler using go-rod for better WAF bypass.
package vulngrabber

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"strings"

	"github.com/go-rod/rod"
	"github.com/kataras/golog"

	"github.com/secflow/client/pkg/rodutil"
)

// VenustechCrawlerRod is the go-rod based implementation for Venustech.
type VenustechCrawlerRod struct {
	log *golog.Logger
}

// NewVenustechCrawlerRod creates a new Venustech crawler using go-rod.
func NewVenustechCrawlerRod() Grabber {
	return &VenustechCrawlerRod{
		log: golog.Child("[venustech-rod]"),
	}
}

func (v *VenustechCrawlerRod) ProviderInfo() *Provider {
	return &Provider{
		Name:        "venustech",
		DisplayName: "启明星辰漏洞通告",
		Link:        "https://www.venustech.com.cn/new_type/aqtg/",
	}
}

func (v *VenustechCrawlerRod) IsValuable(info *VulnInfo) bool {
	return info.Severity == SeverityHigh || info.Severity == SeverityCritical
}

func (v *VenustechCrawlerRod) GetUpdate(ctx context.Context, pageLimit int) ([]*VulnInfo, error) {
	browser, err := rodutil.GetBrowser(nil)
	if err != nil {
		return nil, fmt.Errorf("get browser: %w", err)
	}

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
		v.log.Warnf("apply bypass: %v", err)
	}

	var results []*VulnInfo
	for i := 1; i <= pageLimit; i++ {
		select {
		case <-ctx.Done():
			return results, ctx.Err()
		default:
		}

		pageResult, err := v.parsePage(ctx, page, i)
		if err != nil {
			v.log.Errorf("parse page %d: %v", i, err)
			continue
		}
		v.log.Debugf("got %d vulns from page %d", len(pageResult), i)
		results = append(results, pageResult...)
	}

	return results, nil
}

func (v *VenustechCrawlerRod) parsePage(ctx context.Context, page *rod.Page, pageNum int) ([]*VulnInfo, error) {
	rawURL := "https://www.venustech.com.cn/new_type/aqtg/"
	if pageNum > 1 {
		rawURL = fmt.Sprintf("%sindex_%d.html", rawURL, pageNum)
	}

	// Fetch list page HTML
	fetchScript := fmt.Sprintf(`() => {
		return new Promise((resolve, reject) => {
			fetch('%s', {
				method: 'GET',
				headers: {
					'Accept': 'text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8',
					'Accept-Language': 'zh-CN,zh;q=0.9,en;q=0.8'
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
	}`, rawURL)

	result, err := page.Eval(fetchScript)
	if err != nil {
		return nil, fmt.Errorf("fetch list page: %w", err)
	}

	// Parse list page to get links
	linksScript := fmt.Sprintf(`() => {
		const parser = new DOMParser();
		const doc = parser.parseFromString(`+"`"+`%s`+"`"+`, 'text/html');
		const items = doc.querySelectorAll('body > div > div.wrapper.clearfloat > div.right.main-content > div > div.main-inner-bt > ul > li > a');
		const links = [];
		items.forEach(item => {
			const href = item.getAttribute('href');
			const text = item.textContent.trim();
			if (href && !text.includes('多个安全漏洞')) {
				links.push({
					href: href.startsWith('http') ? href : 'https://www.venustech.com.cn' + href,
					text: text
				});
			}
		});
		return JSON.stringify(links);
	}`, strings.ReplaceAll(strings.ReplaceAll(result.Value.String(), "\\", "\\\\"), "`", "\\`"))

	linksResult, err := page.Eval(linksScript)
	if err != nil {
		return nil, fmt.Errorf("parse list page: %w", err)
	}

	var links []struct {
		Href string `json:"href"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal([]byte(linksResult.Value.String()), &links); err != nil {
		return nil, fmt.Errorf("unmarshal links: %w", err)
	}

	if len(links) == 0 {
		return nil, fmt.Errorf("no vulns found")
	}

	// Parse each vuln
	results := make([]*VulnInfo, 0, len(links))
	for _, link := range links {
		select {
		case <-ctx.Done():
			return results, ctx.Err()
		default:
		}

		vulnInfo, err := v.parseSingle(ctx, page, link.Href)
		if err != nil {
			v.log.Errorf("parse %s: %v", link.Href, err)
			continue
		}
		results = append(results, vulnInfo)
	}

	return results, nil
}

func (v *VenustechCrawlerRod) parseSingle(ctx context.Context, page *rod.Page, vulnURL string) (*VulnInfo, error) {
	v.log.Debugf("parsing vuln %s", vulnURL)

	// Fetch detail page HTML
	fetchScript := fmt.Sprintf(`() => {
		return new Promise((resolve, reject) => {
			fetch('%s', {
				method: 'GET',
				headers: {
					'Accept': 'text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8',
					'Accept-Language': 'zh-CN,zh;q=0.9,en;q=0.8'
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
	}`, vulnURL)

	result, err := page.Eval(fetchScript)
	if err != nil {
		return nil, fmt.Errorf("fetch detail page: %w", err)
	}

	html := result.Value.String()

	// Parse vuln data from HTML
	return v.parseVulnHTML(page, vulnURL, html)
}

func (v *VenustechCrawlerRod) parseVulnHTML(page *rod.Page, vulnURL, html string) (*VulnInfo, error) {
	// Escape HTML for JavaScript
	escapedHTML := strings.ReplaceAll(html, "\\", "\\\\")
	escapedHTML = strings.ReplaceAll(escapedHTML, "`", "\\`")
	escapedHTML = strings.ReplaceAll(escapedHTML, "${", "\\${")

	parseScript := fmt.Sprintf(`() => {
		const parser = new DOMParser();
		const doc = parser.parseFromString(`+"`"+`%s`+"`"+`, 'text/html');
		
		const result = {
			title: "",
			description: "",
			severity: "低危",
			cve: "",
			disclosure: "",
			refs: []
		};

		const contentSel = doc.querySelector('body > div > div.wrapper.clearfloat > div.right.main-content > div > div > div.news-content.ctn');
		if (!contentSel) return JSON.stringify(result);

		const vulnTableSel = contentSel.querySelector('div > table');
		if (vulnTableSel) {
			const vulnDataSel = vulnTableSel.querySelectorAll('tbody > tr > td');
			for (let i = 0; i < vulnDataSel.length; i += 2) {
				const keyText = vulnDataSel[i].textContent.replace(/\s/g, '').replace(/\u00A0/g, '');
				const valueText = vulnDataSel[i + 1]?.textContent?.trim() || '';

				if (keyText === '漏洞名称') {
					result.title = valueText;
				} else if (keyText === 'CVEID') {
					if (valueText.includes('CVE')) {
						result.cve = valueText.includes('、') ? valueText.split('、')[0] : valueText;
					}
				} else if (keyText === '发现时间') {
					if (/^\d{4}-\d{2}-\d{2}$/.test(valueText)) {
						result.disclosure = valueText;
					}
				} else if (keyText === '漏洞等级' || keyText === '等级') {
					result.severity = valueText;
				}
			}
		}

		// Extract title from h3 if not found
		if (!result.title) {
			const h3 = contentSel.querySelector('h3');
			if (h3) {
				result.title = h3.textContent.trim().replace(/^【漏洞通告】/, '');
			}
		}

		// Extract description - use NextUntil logic to get content between table and next h2/h3
		// This mimics the goquery NextUntil behavior
		const h2Data = (() => {
			if (!vulnTableSel) return '';
			const parts = [];
			let sibling = vulnTableSel.nextElementSibling;
			while (sibling && sibling.tagName !== 'H2') {
				const text = sibling.textContent?.trim();
				if (text) parts.push(text);
				sibling = sibling.nextElementSibling;
			}
			return parts.join(' ');
		})();
		
		const h3Data = (() => {
			if (!vulnTableSel) return '';
			const parts = [];
			let sibling = vulnTableSel.nextElementSibling;
			while (sibling && sibling.tagName !== 'H3') {
				const text = sibling.textContent?.trim();
				if (text) parts.push(text);
				sibling = sibling.nextElementSibling;
			}
			return parts.join(' ');
		})();
		
		if (h2Data && h3Data) {
			result.description = h2Data.length < h3Data.length ? h2Data : h3Data;
		} else if (h2Data) {
			result.description = h2Data;
		} else {
			result.description = h3Data;
		}

		// Extract references
		const h3Elements = contentSel.querySelectorAll('div > h3');
		h3Elements.forEach(h3 => {
			if (h3.textContent.includes('参考链接')) {
				let sibling = h3.nextElementSibling;
				while (sibling && sibling.tagName !== 'H2') {
					if (sibling.tagName === 'SECTION') {
						const ref = sibling.textContent.trim();
						if (ref) result.refs.push(ref);
					}
					sibling = sibling.nextElementSibling;
				}
			}
		});

		return JSON.stringify(result);
	}`, escapedHTML)

	result, err := page.Eval(parseScript)
	if err != nil {
		return nil, fmt.Errorf("parse html: %w", err)
	}

	var data struct {
		Title       string   `json:"title"`
		Description string   `json:"description"`
		Severity    string   `json:"severity"`
		CVE         string   `json:"cve"`
		Disclosure  string   `json:"disclosure"`
		Refs        []string `json:"refs"`
	}
	if err := json.Unmarshal([]byte(result.Value.String()), &data); err != nil {
		return nil, fmt.Errorf("unmarshal result: %w", err)
	}

	// Parse severity
	severity := SeverityLow
	switch data.Severity {
	case "高危":
		severity = SeverityHigh
	case "中危":
		severity = SeverityMedium
	case "低危":
		severity = SeverityLow
	}

	// Use filename as UniqueKey
	filename := path.Base(vulnURL)
	ext := path.Ext(filename)
	uniqueKey := strings.TrimSuffix(filename, ext) + "_venustech"

	return &VulnInfo{
		UniqueKey:   uniqueKey,
		Title:       data.Title,
		Description: data.Description,
		Severity:    severity,
		CVE:         data.CVE,
		Disclosure:  data.Disclosure,
		References:  data.Refs,
		From:        vulnURL,
		Creator:     v,
	}, nil
}
