// Package grabber provides Seebug crawler using go-rod for better WAF bypass.
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

// SeebugCrawlerRod is the go-rod based implementation for Seebug.
type SeebugCrawlerRod struct {
	log *golog.Logger
}

// NewSeebugCrawlerRod creates a new Seebug crawler using go-rod.
func NewSeebugCrawlerRod() Grabber {
	return &SeebugCrawlerRod{
		log: golog.Child("[seebug-rod]"),
	}
}

func (s *SeebugCrawlerRod) ProviderInfo() *Provider {
	return &Provider{
		Name:        "seebug",
		DisplayName: "Seebug 漏洞平台",
		Link:        "https://www.seebug.org",
	}
}

func (s *SeebugCrawlerRod) IsValuable(info *VulnInfo) bool {
	return info.Severity == SeverityHigh || info.Severity == SeverityCritical
}

func (s *SeebugCrawlerRod) GetUpdate(ctx context.Context, pageLimit int) ([]*VulnInfo, error) {
	browser, err := rodutil.GetBrowser(nil)
	if err != nil {
		return nil, fmt.Errorf("get browser: %w", err)
	}

	// Get page count first
	pageCount, err := s.getPageCount(ctx, browser)
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

		pageResult, err := s.parsePage(ctx, browser, i)
		if err != nil {
			s.log.Errorf("parse page %d: %v", i, err)
			continue
		}

		s.log.Debugf("got %d vulns from page %d", len(pageResult), i)
		results = append(results, pageResult...)
	}

	return results, nil
}

func (s *SeebugCrawlerRod) getPageCount(ctx context.Context, browser *rod.Browser) (int, error) {
	page, err := rodutil.NewPage(browser)
	if err != nil {
		return 0, fmt.Errorf("create page: %w", err)
	}
	defer page.Close()

	// Apply bypass techniques
	if err := rodutil.ApplyBypass(page, &rodutil.BypassConfig{
		SimulateMouse:   false,
		RandomDelays:    false,
		RandomViewport:  true,
		RotateUserAgent: true,
	}); err != nil {
		s.log.Warnf("apply bypass: %v", err)
	}

	if err := page.Navigate("https://www.seebug.org/vuldb/vulnerabilities"); err != nil {
		return 0, fmt.Errorf("navigate: %w", err)
	}

	if err := page.WaitLoad(); err != nil {
		return 0, fmt.Errorf("wait load: %w", err)
	}

	// Wait for pagination to load
	if err := page.WaitElementsMoreThan("ul.pagination li", 0); err != nil {
		return 0, fmt.Errorf("wait for pagination: %w", err)
	}

	// Get page count using JavaScript
	script := `() => {
		const items = document.querySelectorAll('ul.pagination li');
		if (items.length < 3) return 0;
		const lastPage = items[items.length - 2];
		const text = lastPage.textContent.trim();
		return parseInt(text) || 0;
	}`

	result, err := page.Eval(script)
	if err != nil {
		return 0, fmt.Errorf("eval script: %w", err)
	}

	count := result.Value.Int()
	if count <= 0 {
		return 0, fmt.Errorf("invalid page count: %d", count)
	}

	return count, nil
}

func (s *SeebugCrawlerRod) parsePage(ctx context.Context, browser *rod.Browser, pageNum int) ([]*VulnInfo, error) {
	u := fmt.Sprintf("https://www.seebug.org/vuldb/vulnerabilities?page=%d", pageNum)

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
		s.log.Warnf("apply bypass: %v", err)
	}

	if err := page.Navigate(u); err != nil {
		return nil, fmt.Errorf("navigate: %w", err)
	}

	if err := page.WaitLoad(); err != nil {
		return nil, fmt.Errorf("wait load: %w", err)
	}

	// Wait for table to load
	if err := page.WaitElementsMoreThan(".sebug-table tbody tr", 0); err != nil {
		return nil, fmt.Errorf("wait for table: %w", err)
	}

	// Extract data using JavaScript
	script := `() => {
		const rows = document.querySelectorAll('.sebug-table tbody tr');
		const results = [];
		rows.forEach(row => {
			const tds = row.querySelectorAll('td');
			if (tds.length !== 6) return;
			
			const idLink = tds[0].querySelector('a');
			const href = idLink ? idLink.getAttribute('href') : '';
			const uniqueKey = idLink ? idLink.textContent.trim() : '';
			
			const disclosure = tds[1].textContent.trim();
			
			const severityDiv = tds[2].querySelector('div');
			const severityTitle = severityDiv ? severityDiv.getAttribute('data-original-title') : '';
			
			const title = tds[3].textContent.trim();
			
			const cveIcon = tds[4].querySelector('i.fa-id-card');
			const cveId = cveIcon ? cveIcon.getAttribute('data-original-title') : '';
			
			const detailIcon = tds[4].querySelector('i.fa-file-text-o');
			const hasDetail = detailIcon ? detailIcon.getAttribute('data-original-title') === '有详情' : false;
			
			results.push({
				href: href,
				uniqueKey: uniqueKey,
				disclosure: disclosure,
				severityTitle: severityTitle,
				title: title,
				cveId: cveId,
				hasDetail: hasDetail
			});
		});
		return JSON.stringify(results);
	}`

	result, err := page.Eval(script)
	if err != nil {
		return nil, fmt.Errorf("extract data: %w", err)
	}

	var rows []struct {
		Href          string `json:"href"`
		UniqueKey     string `json:"uniqueKey"`
		Disclosure    string `json:"disclosure"`
		SeverityTitle string `json:"severityTitle"`
		Title         string `json:"title"`
		CveId         string `json:"cveId"`
		HasDetail     bool   `json:"hasDetail"`
	}

	if err := json.Unmarshal([]byte(result.Value.String()), &rows); err != nil {
		return nil, fmt.Errorf("parse rows: %w", err)
	}

	if len(rows) == 0 {
		return nil, fmt.Errorf("no vulns found")
	}

	cvePattern := regexp.MustCompile(`^CVE-\d+-\d+$`)
	results := make([]*VulnInfo, 0, len(rows))

	for _, row := range rows {
		select {
		case <-ctx.Done():
			return results, ctx.Err()
		default:
		}

		var severity SeverityLevel
		switch strings.TrimSpace(row.SeverityTitle) {
		case "高危":
			severity = SeverityHigh
		case "中危":
			severity = SeverityMedium
		case "低危":
			severity = SeverityLow
		default:
			severity = SeverityLow
		}

		cveId := strings.TrimSpace(row.CveId)
		if strings.Contains(cveId, "、") {
			cveId = strings.Split(cveId, "、")[0]
		}
		if !cvePattern.MatchString(cveId) {
			cveId = ""
		}

		var tags []string
		if row.HasDetail {
			tags = append(tags, "有详情")
		}

		href := strings.TrimSpace(row.Href)
		if href != "" && !strings.HasPrefix(href, "http") {
			href = "https://www.seebug.org" + href
		}

		results = append(results, &VulnInfo{
			UniqueKey:   strings.TrimSpace(row.UniqueKey),
			Title:       strings.TrimSpace(row.Title),
			Description: "",
			Severity:    severity,
			CVE:         cveId,
			Disclosure:  strings.TrimSpace(row.Disclosure),
			References:  nil,
			Tags:        tags,
			Solutions:   "",
			From:        href,
			Creator:     s,
		})
	}

	return results, nil
}
