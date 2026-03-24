// Package grabber provides OSCS crawler using go-rod for better WAF bypass.
package vulngrabber

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/kataras/golog"

	"github.com/secflow/client/pkg/rodutil"
)

const OSCSRodPageSize = 10

// OSCSCrawlerRod is the go-rod based implementation for OSCS.
type OSCSCrawlerRod struct {
	log *golog.Logger
}

// NewOSCSCrawlerRod creates a new OSCS crawler using go-rod.
func NewOSCSCrawlerRod() Grabber {
	return &OSCSCrawlerRod{
		log: golog.Child("[oscs-rod]"),
	}
}

func (t *OSCSCrawlerRod) ProviderInfo() *Provider {
	return &Provider{
		Name:        "oscs",
		DisplayName: "OSCS开源安全情报预警",
		Link:        "https://www.oscs1024.com/cm",
	}
}

func (t *OSCSCrawlerRod) GetUpdate(ctx context.Context, pageLimit int) ([]*VulnInfo, error) {
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

	pageCount, err := t.getPageCount(ctx, page)
	if err != nil {
		return nil, fmt.Errorf("get page count: %w", err)
	}
	if pageCount == 0 {
		return nil, fmt.Errorf("invalid page count")
	}
	if pageCount > pageLimit {
		pageCount = pageLimit
	}

	var results []*VulnInfo
	for i := 1; i <= pageCount; i++ {
		select {
		case <-ctx.Done():
			return results, ctx.Err()
		default:
		}
		pageResult, err := t.parsePage(ctx, page, i)
		if err != nil {
			t.log.Errorf("parse page %d: %v", i, err)
			continue
		}
		t.log.Debugf("got %d vulns from page %d", len(pageResult), i)
		results = append(results, pageResult...)
	}
	return results, nil
}

func (t *OSCSCrawlerRod) getPageCount(ctx context.Context, page *rod.Page) (int, error) {
	body, err := t.fetchList(page, 1, 10)
	if err != nil {
		return 0, err
	}

	if body.Code != 200 || !body.Success {
		return 0, fmt.Errorf("response error: %s", body.Info)
	}

	total := body.Data.Total
	if total <= 0 {
		return 0, fmt.Errorf("invalid total count: %d", total)
	}
	pageCount := total / OSCSRodPageSize
	if pageCount == 0 {
		return 1, nil
	}
	if total%pageCount != 0 {
		pageCount += 1
	}
	return pageCount, nil
}

func (t *OSCSCrawlerRod) parsePage(ctx context.Context, page *rod.Page, pageNum int) ([]*VulnInfo, error) {
	body, err := t.fetchList(page, pageNum, OSCSRodPageSize)
	if err != nil {
		return nil, err
	}

	results := make([]*VulnInfo, 0, len(body.Data.Data))
	for _, d := range body.Data.Data {
		select {
		case <-ctx.Done():
			return results, nil
		default:
		}

		var tags []string
		if d.IsPush == 1 {
			tags = append(tags, "发布预警")
		}
		eventType := "公开漏洞"
		switch d.IntelligenceType {
		case 1:
			eventType = "公开漏洞"
		case 2:
			eventType = "墨菲安全独家"
		case 3:
			eventType = "投毒情报"
		}
		tags = append(tags, eventType)

		info, err := t.parseSingleVuln(page, d.Mps)
		if err != nil {
			t.log.Errorf("failed to parse %s: %v", d.Url, err)
			continue
		}
		info.Tags = tags
		results = append(results, info)
	}
	return results, nil
}

func (t *OSCSCrawlerRod) fetchList(page *rod.Page, pageNum, size int) (*oscsListResp, error) {
	bodyJSON, _ := json.Marshal(map[string]interface{}{
		"page":     pageNum,
		"per_page": size,
	})

	fetchScript := fmt.Sprintf(`() => {
		return new Promise((resolve, reject) => {
			fetch('https://www.oscs1024.com/oscs/v1/intelligence/list', {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json',
					'Accept': 'application/json, text/plain, */*',
					'Accept-Language': 'zh-CN,zh;q=0.9,en;q=0.8',
					'Referer': 'https://www.oscs1024.com/cm',
					'Origin': 'https://www.oscs1024.com'
				},
				body: %s
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
	}`, fmt.Sprintf("`%s`", string(bodyJSON)))

	result, err := page.Eval(fetchScript)
	if err != nil {
		return nil, fmt.Errorf("fetch list: %w", err)
	}

	var body oscsListResp
	if err := json.Unmarshal([]byte(result.Value.String()), &body); err != nil {
		return nil, fmt.Errorf("unmarshal list: %w", err)
	}

	return &body, nil
}

func (t *OSCSCrawlerRod) parseSingleVuln(page *rod.Page, mps string) (*VulnInfo, error) {
	bodyJSON := fmt.Sprintf(`{"vuln_no":"%s"}`, mps)

	fetchScript := fmt.Sprintf(`() => {
		return new Promise((resolve, reject) => {
			fetch('https://www.oscs1024.com/oscs/v1/vdb/info', {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json',
					'Accept': 'application/json, text/plain, */*',
					'Accept-Language': 'zh-CN,zh;q=0.9,en;q=0.8',
					'Referer': 'https://www.oscs1024.com/cm',
					'Origin': 'https://www.oscs1024.com'
				},
				body: %s
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
	}`, fmt.Sprintf("`%s`", bodyJSON))

	result, err := page.Eval(fetchScript)
	if err != nil {
		return nil, fmt.Errorf("fetch detail: %w", err)
	}

	var respBody oscsDetailResp
	if err := json.Unmarshal([]byte(result.Value.String()), &respBody); err != nil {
		return nil, fmt.Errorf("unmarshal detail: %w", err)
	}

	if respBody.Code != 200 || !respBody.Success || len(respBody.Data) == 0 {
		return nil, fmt.Errorf("response error: %s", respBody.Info)
	}

	data := respBody.Data[0]
	severity := SeverityLow
	switch data.Level {
	case "SeverityCritical":
		severity = SeverityCritical
	case "SeverityHigh":
		severity = SeverityHigh
	case "SeverityMedium":
		severity = SeverityMedium
	case "SeverityLow":
		severity = SeverityLow
	}

	disclosure := time.UnixMilli(int64(data.PublishTime)).Format("2006-01-02")
	refs := make([]string, 0, len(data.References))
	for _, ref := range data.References {
		refs = append(refs, ref.Url)
	}

	return &VulnInfo{
		UniqueKey:   data.VulnNo,
		Title:       data.VulnTitle,
		Description: data.Description,
		Severity:    severity,
		CVE:         data.CveId,
		Disclosure:  disclosure,
		References:  refs,
		Tags:        nil,
		Solutions:   t.buildSolution(data.SoulutionData),
		From:        "https://www.oscs1024.com/hd/" + data.VulnNo,
		Creator:     t,
	}, nil
}

func (t *OSCSCrawlerRod) IsValuable(info *VulnInfo) bool {
	// 仅有预警的 或高危严重的
	if info.Severity != SeverityCritical && info.Severity != SeverityHigh {
		return false
	}
	for _, tag := range info.Tags {
		if tag == "发布预警" {
			return true
		}
	}
	return false
}

func (t *OSCSCrawlerRod) buildSolution(solution []string) string {
	var builder strings.Builder
	for i, s := range solution {
		builder.WriteString(fmt.Sprintf("%d. %s\n", i+1, s))
	}
	return builder.String()
}
