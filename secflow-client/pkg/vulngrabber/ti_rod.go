// Package grabber provides TI (Qianxin) crawler using go-rod for better compatibility.
package vulngrabber

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/kataras/golog"

	"github.com/secflow/client/pkg/rodutil"
)

// TiCrawlerRod is the go-rod based implementation for TI (Qianxin).
type TiCrawlerRod struct {
	log *golog.Logger
}

// NewTiCrawlerRod creates a new TI crawler using go-rod.
func NewTiCrawlerRod() Grabber {
	return &TiCrawlerRod{
		log: golog.Child("[qianxin-ti-rod]"),
	}
}

func (t *TiCrawlerRod) ProviderInfo() *Provider {
	return &Provider{
		Name:        "qianxin-ti",
		DisplayName: "奇安信威胁情报中心",
		Link:        "https://ti.qianxin.com/",
	}
}

func (t *TiCrawlerRod) IsValuable(info *VulnInfo) bool {
	if info.Severity != SeverityHigh && info.Severity != SeverityCritical {
		return false
	}
	for _, tag := range info.Tags {
		if tag == "奇安信CERT验证" ||
			tag == "POC公开" ||
			tag == "EXP公开" ||
			tag == "技术细节公布" {
			return true
		}
	}
	return false
}

func (t *TiCrawlerRod) GetUpdate(ctx context.Context, _ int) ([]*VulnInfo, error) {
	browser, err := rodutil.GetBrowser(nil)
	if err != nil {
		return nil, fmt.Errorf("get browser: %w", err)
	}

	page, err := rodutil.NewPage(browser)
	if err != nil {
		return nil, fmt.Errorf("create page: %w", err)
	}
	defer page.Close()

	// Apply bypass techniques
	if err := rodutil.ApplyBypass(page, nil); err != nil {
		t.log.Warnf("apply bypass: %v", err)
	}

	// Navigate to the vulnerability page first to set up context
	if err := rodutil.SafeNavigate(page, "https://ti.qianxin.com/vulnerability", nil); err != nil {
		return nil, fmt.Errorf("navigate: %w", err)
	}

	// Wait for page to be ready
	if err := page.WaitLoad(); err != nil {
		return nil, fmt.Errorf("wait for page load: %w", err)
	}

	// Use JavaScript to make the API request
	script := `() => {
		return new Promise((resolve, reject) => {
			fetch('https://ti.qianxin.com/alpha-api/v2/vuln/one-day', {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json',
					'Referer': 'https://ti.qianxin.com/',
					'Origin': 'https://ti.qianxin.com'
				},
				credentials: 'include'
			})
			.then(response => response.json())
			.then(data => resolve(JSON.stringify(data)))
			.catch(error => reject(error.message));
		});
	}`

	result, err := page.Eval(script)
	if err != nil {
		return nil, fmt.Errorf("fetch api: %w", err)
	}

	// Parse the JSON response
	jsonStr := result.Value.String()
	var body tiOneDayResp
	if err := json.Unmarshal([]byte(jsonStr), &body); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	// Check response status - Qianxin API returns status 10000 with message "success" for success
	if body.Status != 200 && body.Status != 10000 {
		return nil, fmt.Errorf("api error: status=%d, message=%s", body.Status, body.Message)
	}

	// Process the results
	var results []*VulnInfo
	for _, d := range body.Data.KeyVulnAdd {
		select {
		case <-ctx.Done():
			return results, ctx.Err()
		default:
		}

		tags := make([]string, 0, len(d.Tag))
		for _, tag := range d.Tag {
			tags = append(tags, strings.TrimSpace(tag.Name))
		}

		severity := SeverityLow
		switch d.RatingLevel {
		case "低危":
			severity = SeverityLow
		case "中危":
			severity = SeverityMedium
		case "高危":
			severity = SeverityHigh
		case "极危":
			severity = SeverityCritical
		}

		info := &VulnInfo{
			UniqueKey:   d.QvdCode,
			Title:       d.VulnName,
			Description: d.Description,
			Severity:    severity,
			CVE:         d.CveCode,
			Disclosure:  d.PublishTime,
			References:  nil,
			Tags:        tags,
			Solutions:   "",
			From:        "https://ti.qianxin.com/vulnerability/detail/" + strconv.Itoa(d.Id),
			Creator:     t,
		}
		results = append(results, info)
	}

	// Deduplicate based on UniqueKey
	uniqResults := make(map[string]*VulnInfo)
	for _, info := range results {
		uniqResults[info.UniqueKey] = info
	}

	// Keep order
	newResults := make([]*VulnInfo, 0, len(uniqResults))
	for _, info := range results {
		if uniqResults[info.UniqueKey] == nil {
			continue
		}
		newResults = append(newResults, info)
		uniqResults[info.UniqueKey] = nil
	}

	t.log.Debugf("got %d vulns from oneday api", len(newResults))
	return newResults, nil
}
