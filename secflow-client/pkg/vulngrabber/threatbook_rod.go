// Package grabber provides ThreatBook crawler using go-rod for better WAF bypass.
package vulngrabber

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/kataras/golog"
	"github.com/pkg/errors"

	"github.com/secflow/client/pkg/rodutil"
)

// ThreatBookCrawlerRod is the go-rod based implementation for ThreatBook.
type ThreatBookCrawlerRod struct {
	log *golog.Logger
}

// NewThreatBookCrawlerRod creates a new ThreatBook crawler using go-rod.
func NewThreatBookCrawlerRod() Grabber {
	return &ThreatBookCrawlerRod{
		log: golog.Child("[threatbook-rod]"),
	}
}

func (t *ThreatBookCrawlerRod) ProviderInfo() *Provider {
	return &Provider{
		Name:        "threatbook",
		DisplayName: "微步在线研究响应中心-漏洞通告",
		Link:        "https://x.threatbook.com/v5/vul/",
	}
}

func (t *ThreatBookCrawlerRod) GetUpdate(ctx context.Context, pageLimit int) ([]*VulnInfo, error) {
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

	// First visit the main page to get CSRF token
	page.Navigate("https://x.threatbook.com/v5/vulIntelligence")
	page.WaitLoad()

	// Get CSRF token from cookies
	csrfScript := `() => {
		let match = document.cookie.match(/csrfToken=([^;]+)/);
		return match ? match[1] : '';
	}`
	csrfResult, err := page.Eval(csrfScript)
	if err != nil {
		return nil, fmt.Errorf("get CSRF token: %w", err)
	}
	csrf := csrfResult.Value.String()

	// Fetch data using fetch API with CSRF token
	fetchScript := fmt.Sprintf(`() => {
		return new Promise((resolve, reject) => {
			fetch('https://x.threatbook.com/v5/node/vul_module/homePage', {
				method: 'GET',
				headers: {
					'Accept': 'application/json, text/plain, */*',
					'Accept-Language': 'zh-CN,zh;q=0.9,en;q=0.8,en-GB;q=0.7,en-US;q=0.6',
					'Referer': 'https://x.threatbook.com/v5/vulIntelligence',
					'x-csrf-token': '%s',
					'cookie': 'csrfToken=' + '%s'
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
	}`, csrf, csrf)

	result, err := page.Eval(fetchScript)
	if err != nil {
		return nil, fmt.Errorf("fetch data: %w", err)
	}

	var body threatBookHomepage
	if err := json.Unmarshal([]byte(result.Value.String()), &body); err != nil {
		return nil, fmt.Errorf("unmarshal data: %w", err)
	}

	t.log.Debugf("got %d vulns", len(body.Data.HighRisk))

	var results []*VulnInfo
	for _, v := range body.Data.HighRisk {
		disclosure := v.VulnPublishTime
		if disclosure == "" {
			disclosure = v.VulnUpdateTime
		}

		// Build affected products description
		var affectedProducts []string
		var productTags []string
		for _, affect := range v.Affects {
			// Format: "厂商>产品" - split and extract product name
			parts := strings.Split(affect, ">")
			if len(parts) >= 2 {
				product := strings.TrimSpace(parts[1])
				affectedProducts = append(affectedProducts, product)
				productTags = append(productTags, product)
			} else {
				affectedProducts = append(affectedProducts, affect)
				productTags = append(productTags, affect)
			}
		}

		// Build description with affected products
		description := "受影响产品: " + strings.Join(affectedProducts, ", ")

		// Map riskLevel to Severity
		severity := SeverityMedium
		switch v.RiskLevel {
		case "严重":
			severity = SeverityCritical
		case "高风险", "高危":
			severity = SeverityHigh
		case "中风险", "中危":
			severity = SeverityMedium
		case "低风险", "低危":
			severity = SeverityLow
		}

		var tags []string
		if v.Is0Day {
			tags = append(tags, "0day")
		}
		if v.PocExist {
			tags = append(tags, "有Poc")
		}
		if v.Premium {
			tags = append(tags, "有漏洞分析")
		}
		if v.Solution {
			tags = append(tags, "有修复方案")
		}
		// Add product tags
		for _, pt := range productTags {
			if pt != "" {
				tags = append(tags, pt)
			}
		}

		solutions := "暂无修复方案"
		if v.Solution {
			solutions = "官方已提供修复方案"
		}

		vuln := &VulnInfo{
			UniqueKey:   v.Id,
			Title:       v.VulnNameZh,
			Description: description,
			Severity:    severity,
			Disclosure:  disclosure,
			Solutions:   solutions,
			References:  nil,
			Tags:        tags,
			From:        t.ProviderInfo().Link + v.Id,
			Creator:     t,
		}
		results = append(results, vuln)
	}
	t.log.Debugf("got %d vulns", len(results))

	return results, nil
}

func (t *ThreatBookCrawlerRod) IsValuable(info *VulnInfo) bool {
	// ThreatBook homepage API 不返回 "有漏洞分析" 标签，过滤规则放宽一些
	var hasPoc bool
	for _, tag := range info.Tags {
		if tag == "有Poc" {
			hasPoc = true
			break
		}
	}
	// 只要有 POC 就推送
	if !hasPoc {
		return false
	}
	if info.Disclosure == "" {
		return false
	}
	// 2024-04-29 format
	dis, err := time.Parse("2006-01-02", info.Disclosure)
	if err != nil {
		t.log.Error(errors.Wrap(err, "parse disclosure time"))
		return false
	}
	// 只看两周内的，古董漏洞就别推了
	if time.Since(dis) > 14*24*time.Hour {
		return false
	}

	return true
}
