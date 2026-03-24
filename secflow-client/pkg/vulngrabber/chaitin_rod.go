// Package grabber provides Chaitin crawler using go-rod for better WAF bypass.
package vulngrabber

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/kataras/golog"

	"github.com/secflow/client/pkg/rodutil"
)

// ChaitinCrawlerRod is the go-rod based implementation for Chaitin.
type ChaitinCrawlerRod struct {
	log *golog.Logger
}

// NewChaitinCrawlerRod creates a new Chaitin crawler using go-rod.
func NewChaitinCrawlerRod() Grabber {
	return &ChaitinCrawlerRod{
		log: golog.Child("[chaitin-rod]"),
	}
}

func (t *ChaitinCrawlerRod) ProviderInfo() *Provider {
	return &Provider{
		Name:        "chaitin",
		DisplayName: "长亭漏洞库",
		Link:        "https://stack.chaitin.com/vuldb/index",
	}
}

func (t *ChaitinCrawlerRod) IsValuable(info *VulnInfo) bool {
	if info.Severity != SeverityHigh && info.Severity != SeverityCritical {
		return false
	}

	if !ContainsChinese(info.Title) {
		return false
	}
	return true
}

func (t *ChaitinCrawlerRod) GetUpdate(ctx context.Context, pageLimit int) ([]*VulnInfo, error) {
	// Extend context for browser operations
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

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
		t.log.Warnf("apply bypass: %v", err)
	}

	var results []*VulnInfo
	// CT- 为长亭漏洞库的标识
	urlTpl := "https://stack.chaitin.com/api/v2/vuln/list/?limit=15&offset=%d&search=CT-"

	for i := 0; i < pageLimit; i++ {
		select {
		case <-ctx.Done():
			return results, ctx.Err()
		default:
		}

		t.log.Debugf("get vuln from chaitin page %d", i+1)
		u := fmt.Sprintf(urlTpl, i*15)

		// Use fetch API to get data
		fetchScript := fmt.Sprintf(`() => {
			return new Promise((resolve, reject) => {
				fetch('%s', {
					method: 'GET',
					headers: {
						'Accept': 'application/json, text/plain, */*',
						'Accept-Language': 'zh-CN,zh;q=0.9,en;q=0.8',
						'Referer': 'https://stack.chaitin.com/vuldb/index',
						'Origin': 'https://stack.chaitin.com'
					}
				})
				.then(response => {
					if (!response.ok) {
						throw new Error('HTTP error! status: ' + response.status);
					}
					return response.text();
				})
				.then(data => resolve(data))
				.catch(error => reject(error.message));
			});
		}`, u)

		result, err := page.Eval(fetchScript)
		if err != nil {
			t.log.Errorf("fetch page %d: %v", i+1, err)
			continue
		}

		var body ChaitinResp
		if err := json.Unmarshal([]byte(result.Value.String()), &body); err != nil {
			t.log.Errorf("unmarshal page %d: %v", i+1, err)
			continue
		}

		for _, d := range body.Data.List {
			severity := SeverityLow
			switch d.Severity {
			case "low":
				severity = SeverityLow
			case "medium":
				severity = SeverityMedium
			case "high":
				severity = SeverityHigh
			case "critical":
				severity = SeverityCritical
			}

			disclosureDate := d.CreatedAt.Format("2006-01-02")
			var refs []string
			if d.References != nil {
				refs = strings.Split(*d.References, "\n")
			}
			var cveId string
			if d.CveId != nil {
				cveId = *d.CveId
			}
			info := &VulnInfo{
				UniqueKey:   d.CtId,
				Title:       d.Title,
				Description: d.Summary,
				Severity:    severity,
				CVE:         cveId,
				Disclosure:  disclosureDate,
				References:  refs,
				From:        "https://stack.chaitin.com/vuldb/detail/" + d.Id,
				Creator:     t,
			}
			results = append(results, info)
		}
	}

	t.log.Debugf("got %d vulns from chaitin api", len(results))
	return results, nil
}
