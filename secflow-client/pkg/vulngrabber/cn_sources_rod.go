// Package grabber provides Rod-based crawlers for Chinese security vendors.
// These crawlers use go-rod to bypass WAF protection commonly found on Chinese security sites.
package vulngrabber

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/go-rod/rod"
	"github.com/kataras/golog"

	"github.com/secflow/client/pkg/rodutil"
)

// WAF detection patterns for Chinese sites
var (
	cnBlockPatterns = []string{
		"访问被拒绝",
		"403 Forbidden",
		"安全验证",
		"请输入验证码",
		"系统繁忙",
		"请稍后再试",
		"您访问的频率过快",
		"访问频率超限",
	}
)

// ── CNVD (国家信息安全漏洞库) ──────────────────────────────────────────────

// CNVDCrawlerRod fetches vulnerabilities from CNVD using Rod for WAF bypass.
type CNVDCrawlerRod struct {
	log *golog.Logger
}

// NewCNVDCrawlerRod creates a new CNVD crawler using go-rod.
func NewCNVDCrawlerRod() Grabber {
	return &CNVDCrawlerRod{
		log: golog.Child("[cnvd-rod]"),
	}
}

func (c *CNVDCrawlerRod) ProviderInfo() *Provider {
	return &Provider{
		Name:        "cnvd-rod",
		DisplayName: "国家信息安全漏洞库",
		Link:        "https://www.cnvd.org.cn/",
	}
}

func (c *CNVDCrawlerRod) GetUpdate(ctx context.Context, pageLimit int) ([]*VulnInfo, error) {
	browser, err := rodutil.GetBrowser(nil)
	if err != nil {
		return nil, fmt.Errorf("get browser: %w", err)
	}

	page, err := rodutil.NewPage(browser)
	if err != nil {
		return nil, fmt.Errorf("create page: %w", err)
	}
	defer page.Close()

	// Apply bypass
	if err := rodutil.ApplyBypass(page, rodutil.DefaultBypassConfig()); err != nil {
		c.log.Warnf("apply bypass: %v", err)
	}

	var results []*VulnInfo
	for i := 1; i <= pageLimit; i++ {
		select {
		case <-ctx.Done():
			return results, ctx.Err()
		default:
		}

		pageResult, err := c.parsePage(ctx, page, i)
		if err != nil {
			c.log.Errorf("parse page %d: %v", i, err)
			continue
		}
		results = append(results, pageResult...)
	}

	return results, nil
}

func (c *CNVDCrawlerRod) parsePage(ctx context.Context, page *rod.Page, pageNum int) ([]*VulnInfo, error) {
	targetURL := fmt.Sprintf("https://www.cnvd.org.cn/shareData/flow?page=%d", pageNum)

	if err := rodutil.SafeNavigate(page, targetURL, rodutil.DefaultBypassConfig()); err != nil {
		return nil, fmt.Errorf("navigate: %w", err)
	}

	// Check for WAF block
	if rodutil.IsPageBlocked(page) {
		return nil, fmt.Errorf("waf blocked")
	}

	// Extract vulnerability data via JavaScript
	script := `
		() => {
			const vulns = [];
			const rows = document.querySelectorAll('table.tbl_vul_list tbody tr');
			rows.forEach(row => {
				const cells = row.querySelectorAll('td');
				if (cells.length >= 5) {
					const title = cells[0].textContent?.trim() || '';
					const cve = cells[1].textContent?.trim() || '';
					const level = cells[2].textContent?.trim() || '';
					const date = cells[4].textContent?.trim() || '';
					// Try to get the link from title cell
					const titleLink = cells[0].querySelector('a');
					const link = titleLink?.href || '';
					vulns.push({ title, cve, level, date, link });
				}
			});
			return JSON.stringify(vulns);
		}
	`

	result, err := page.Eval(script)
	if err != nil {
		return nil, fmt.Errorf("extract data: %w", err)
	}

	var data []struct {
		Title string `json:"title"`
		CVE   string `json:"cve"`
		Level string `json:"level"`
		Date  string `json:"date"`
		Link  string `json:"link"`
	}

	if err := json.Unmarshal([]byte(result.Value.String()), &data); err != nil {
		return nil, fmt.Errorf("parse data: %w", err)
	}

	var results []*VulnInfo
	for _, v := range data {
		severity := c.parseSeverity(v.Level)
		// Generate unique key from CVE or title
		uniqueKey := v.CVE
		if uniqueKey == "" {
			uniqueKey = fmt.Sprintf("cnvd_%s_%s", strings.TrimSpace(v.Title), strings.TrimSpace(v.Date))
		} else {
			uniqueKey = "CNVD_" + uniqueKey
		}
		// Normalize disclosure date
		disclosure := parseDisclosureDate(v.Date)

		results = append(results, &VulnInfo{
			UniqueKey:   uniqueKey,
			Title:       strings.TrimSpace(v.Title),
			CVE:         strings.TrimSpace(v.CVE),
			Severity:    severity,
			Disclosure:  disclosure,
			From:        v.Link,
			References:  []string{},
			Tags:        []string{"CNVD"},
			Solutions:   "",
			Creator:     c,
		})
	}

	return results, nil
}

func (c *CNVDCrawlerRod) parseSeverity(level string) SeverityLevel {
	switch {
	case strings.Contains(level, "高"):
		return SeverityHigh
	case strings.Contains(level, "中"):
		return SeverityMedium
	case strings.Contains(level, "低"):
		return SeverityLow
	default:
		return SeverityMedium
	}
}

func (c *CNVDCrawlerRod) IsValuable(info *VulnInfo) bool {
	return info.Severity == SeverityHigh || info.Severity == SeverityCritical
}

// ── CNNVD (国家信息安全漏洞共享平台) ────────────────────────────────────────

// CNNVDCrawlerRod fetches vulnerabilities from CNNVD using Rod.
type CNNVDCrawlerRod struct {
	log *golog.Logger
}

// NewCNNVDCrawlerRod creates a new CNNVD crawler using go-rod.
func NewCNNVDCrawlerRod() Grabber {
	return &CNNVDCrawlerRod{
		log: golog.Child("[cnnvd-rod]"),
	}
}

func (c *CNNVDCrawlerRod) ProviderInfo() *Provider {
	return &Provider{
		Name:        "cnnvd-rod",
		DisplayName: "国家信息安全漏洞共享平台",
		Link:        "http://www.cnnvd.org.cn/",
	}
}

func (c *CNNVDCrawlerRod) GetUpdate(ctx context.Context, pageLimit int) ([]*VulnInfo, error) {
	browser, err := rodutil.GetBrowser(nil)
	if err != nil {
		return nil, fmt.Errorf("get browser: %w", err)
	}

	page, err := rodutil.NewPage(browser)
	if err != nil {
		return nil, fmt.Errorf("create page: %w", err)
	}
	defer page.Close()

	var results []*VulnInfo
	for i := 1; i <= pageLimit; i++ {
		select {
		case <-ctx.Done():
			return results, ctx.Err()
		default:
		}

		pageResult, err := c.parsePage(ctx, page, i)
		if err != nil {
			c.log.Errorf("parse page %d: %v", i, err)
			continue
		}
		results = append(results, pageResult...)
	}

	return results, nil
}

func (c *CNNVDCrawlerRod) parsePage(ctx context.Context, page *rod.Page, pageNum int) ([]*VulnInfo, error) {
	targetURL := fmt.Sprintf("http://www.cnnvd.org.cn/web/vulnerability/querylist.tag?pageno=%d", pageNum)

	config := &rodutil.BypassConfig{
		RotateUserAgent:       true,
		RandomViewport:        true,
		AdvancedFingerprint:   true,
		SimulateMouse:         true,
		RandomDelays:          true,
		MinDelay:              500,
		MaxDelay:              1500,
	}

	if err := rodutil.SafeNavigate(page, targetURL, config); err != nil {
		return nil, fmt.Errorf("navigate: %w", err)
	}

	if rodutil.IsPageBlocked(page) {
		return nil, fmt.Errorf("waf blocked")
	}

	script := `
		() => {
			const vulns = [];
			const items = document.querySelectorAll('.vul_list ul li');
			items.forEach(item => {
				const link = item.querySelector('a');
				const title = link?.textContent?.trim() || '';
				const href = link?.href || '';
				const level = item.querySelector('.level')?.textContent?.trim() || '';
				const date = item.querySelector('.date')?.textContent?.trim() || '';
				vulns.push({ title, href, level, date });
			});
			return JSON.stringify(vulns);
		}
	`

	result, err := page.Eval(script)
	if err != nil {
		return nil, fmt.Errorf("extract data: %w", err)
	}

	var data []struct {
		Title string `json:"title"`
		Href  string `json:"href"`
		Level string `json:"level"`
		Date  string `json:"date"`
	}

	if err := json.Unmarshal([]byte(result.Value.String()), &data); err != nil {
		return nil, fmt.Errorf("parse data: %w", err)
	}

	var results []*VulnInfo
	for _, v := range data {
		severity := c.parseSeverity(v.Level)
		// Generate unique key from title and date
		uniqueKey := fmt.Sprintf("CNNVD_%s_%s", strings.TrimSpace(v.Title), strings.TrimSpace(v.Date))
		// Normalize disclosure date
		disclosure := parseDisclosureDate(v.Date)

		results = append(results, &VulnInfo{
			UniqueKey:   uniqueKey,
			Title:       strings.TrimSpace(v.Title),
			Severity:    severity,
			Disclosure:   disclosure,
			From:        v.Href,
			References:  []string{},
			Tags:        []string{"CNNVD"},
			Solutions:   "",
			Creator:     c,
		})
	}

	return results, nil
}

func (c *CNNVDCrawlerRod) parseSeverity(level string) SeverityLevel {
	switch {
	case strings.Contains(level, "高危"):
		return SeverityHigh
	case strings.Contains(level, "中危"):
		return SeverityMedium
	case strings.Contains(level, "低危"):
		return SeverityLow
	case strings.Contains(level, "严重"):
		return SeverityCritical
	default:
		return SeverityMedium
	}
}

func (c *CNNVDCrawlerRod) IsValuable(info *VulnInfo) bool {
	return info.Severity == SeverityHigh || info.Severity == SeverityCritical
}

// ── NSFOCUS (绿盟科技) ─────────────────────────────────────────────────────

// NsfocusCrawlerRod fetches from NSFOCUS using Rod.
type NsfocusCrawlerRod struct {
	log *golog.Logger
}

// NewNsfocusCrawlerRod creates a new NSFOCUS crawler using go-rod.
func NewNsfocusCrawlerRod() Grabber {
	return &NsfocusCrawlerRod{
		log: golog.Child("[nsfocus-rod]"),
	}
}

func (c *NsfocusCrawlerRod) ProviderInfo() *Provider {
	return &Provider{
		Name:        "nsfocus-rod",
		DisplayName: "绿盟科技",
		Link:        "https://www.nsfocus.net/",
	}
}

func (c *NsfocusCrawlerRod) GetUpdate(ctx context.Context, pageLimit int) ([]*VulnInfo, error) {
	browser, err := rodutil.GetBrowser(nil)
	if err != nil {
		return nil, fmt.Errorf("get browser: %w", err)
	}

	page, err := rodutil.NewPage(browser)
	if err != nil {
		return nil, fmt.Errorf("create page: %w", err)
	}
	defer page.Close()

	// NSFOCUS vulnerability database
	targetURL := "https://www.nsfocus.net/vulndb/"

	config := &rodutil.BypassConfig{
		RotateUserAgent:       true,
		RandomViewport:        true,
		AdvancedFingerprint:   true,
		SimulateMouse:         true,
		RandomDelays:          true,
		MinDelay:              1000,
		MaxDelay:              3000,
		AutoRetryOnWAF:        true,
		MaxRetries:            3,
	}

	if err := rodutil.NavigateWithRetry(ctx, page, targetURL, config); err != nil {
		return nil, fmt.Errorf("navigate: %w", err)
	}

	if rodutil.IsPageBlocked(page) {
		return nil, fmt.Errorf("waf blocked after retries")
	}

	return []*VulnInfo{}, nil
}

func (c *NsfocusCrawlerRod) IsValuable(info *VulnInfo) bool {
	return info.Severity == SeverityHigh || info.Severity == SeverityCritical
}

// ── QIANXIN (奇安信CERT) ────────────────────────────────────────────────────

// QianxinCrawlerRod fetches from Qianxin CERT using Rod.
type QianxinCrawlerRod struct {
	log *golog.Logger
}

// NewQianxinCrawlerRod creates a new Qianxin crawler using go-rod.
func NewQianxinCrawlerRod() Grabber {
	return &QianxinCrawlerRod{
		log: golog.Child("[qianxin-rod]"),
	}
}

func (c *QianxinCrawlerRod) ProviderInfo() *Provider {
	return &Provider{
		Name:        "qianxin-rod",
		DisplayName: "奇安信CERT",
		Link:        "https://www.qianxin.com/",
	}
}

func (c *QianxinCrawlerRod) GetUpdate(ctx context.Context, pageLimit int) ([]*VulnInfo, error) {
	browser, err := rodutil.GetBrowser(nil)
	if err != nil {
		return nil, fmt.Errorf("get browser: %w", err)
	}

	page, err := rodutil.NewPage(browser)
	if err != nil {
		return nil, fmt.Errorf("create page: %w", err)
	}
	defer page.Close()

	targetURL := "https://www.qianxin.com/col/channel/166"

	config := &rodutil.BypassConfig{
		RotateUserAgent:       true,
		RandomViewport:        true,
		AdvancedFingerprint:   true,
		SimulateMouse:         true,
		RandomDelays:          true,
		MinDelay:              500,
		MaxDelay:              1500,
		AutoRetryOnWAF:        true,
		MaxRetries:            3,
	}

	if err := rodutil.NavigateWithRetry(ctx, page, targetURL, config); err != nil {
		return nil, fmt.Errorf("navigate: %w", err)
	}

	if rodutil.IsPageBlocked(page) {
		return nil, fmt.Errorf("waf blocked")
	}

	// Extract vulnerability bulletins
	script := `
		() => {
			const vulns = [];
			const items = document.querySelectorAll('.news-list li, .vul-list li, article');
			items.forEach(item => {
				const title = item.querySelector('h3, .title, a')?.textContent?.trim() || '';
				const link = item.querySelector('a')?.href || '';
				const date = item.querySelector('.date, .time')?.textContent?.trim() || '';
				// Try to extract severity from class or text
				const severityEl = item.querySelector('[class*="level"], [class*="severity"]');
				const severity = severityEl?.textContent?.trim() || '';
				if (title && link) {
					vulns.push({ title, link, date, severity });
				}
			});
			return JSON.stringify(vulns);
		}
	`

	result, err := page.Eval(script)
	if err != nil {
		return nil, fmt.Errorf("extract data: %w", err)
	}

	var data []struct {
		Title   string `json:"title"`
		Link    string `json:"link"`
		Date    string `json:"date"`
		Severity string `json:"severity"`
	}

	if err := json.Unmarshal([]byte(result.Value.String()), &data); err != nil {
		return nil, fmt.Errorf("parse data: %w", err)
	}

	var results []*VulnInfo
	for _, v := range data {
		// Generate unique key
		uniqueKey := fmt.Sprintf("QIANXIN_%s_%s", strings.TrimSpace(v.Title), strings.TrimSpace(v.Date))
		// Parse disclosure date
		disclosure := parseDisclosureDate(v.Date)
		// Parse severity
		severity := c.parseSeverity(v.Severity)

		results = append(results, &VulnInfo{
			UniqueKey:   uniqueKey,
			Title:       strings.TrimSpace(v.Title),
			Severity:    severity,
			Disclosure:   disclosure,
			From:        v.Link,
			References:  []string{},
			Tags:        []string{"奇安信"},
			Solutions:   "",
			Creator:     c,
		})
	}

	return results, nil
}

func (c *QianxinCrawlerRod) parseSeverity(level string) SeverityLevel {
	switch {
	case strings.Contains(level, "高"):
		return SeverityHigh
	case strings.Contains(level, "中"):
		return SeverityMedium
	case strings.Contains(level, "低"):
		return SeverityLow
	case strings.Contains(level, "严重"):
		return SeverityCritical
	default:
		return SeverityMedium
	}
}

func (c *QianxinCrawlerRod) IsValuable(info *VulnInfo) bool {
	return info.Severity == SeverityHigh || info.Severity == SeverityCritical
}

// ── ANTIY (安天威胁情报) ────────────────────────────────────────────────────

// AntiyCrawlerRod fetches from Antiy using Rod.
type AntiyCrawlerRod struct {
	log *golog.Logger
}

// NewAntiyCrawlerRod creates a new Antiy crawler using go-rod.
func NewAntiyCrawlerRod() Grabber {
	return &AntiyCrawlerRod{
		log: golog.Child("[antiy-rod]"),
	}
}

func (c *AntiyCrawlerRod) ProviderInfo() *Provider {
	return &Provider{
		Name:        "antiy-rod",
		DisplayName: "安天威胁情报",
		Link:        "https://www.antiy.com/",
	}
}

func (c *AntiyCrawlerRod) GetUpdate(ctx context.Context, pageLimit int) ([]*VulnInfo, error) {
	browser, err := rodutil.GetBrowser(nil)
	if err != nil {
		return nil, fmt.Errorf("get browser: %w", err)
	}

	page, err := rodutil.NewPage(browser)
	if err != nil {
		return nil, fmt.Errorf("create page: %w", err)
	}
	defer page.Close()

	targetURL := "https://www.antiy.com/"
	// Antiy may have sub-pages for vulnerability intelligence

	config := &rodutil.BypassConfig{
		RotateUserAgent:       true,
		RandomViewport:        true,
		AdvancedFingerprint:   true,
		SimulateMouse:         true,
		RandomDelays:          true,
		MinDelay:              1000,
		MaxDelay:              2500,
		AutoRetryOnWAF:        true,
		MaxRetries:            3,
	}

	if err := rodutil.NavigateWithRetry(ctx, page, targetURL, config); err != nil {
		return nil, fmt.Errorf("navigate: %w", err)
	}

	if rodutil.IsPageBlocked(page) {
		return nil, fmt.Errorf("waf blocked")
	}

	return []*VulnInfo{}, nil
}

func (c *AntiyCrawlerRod) IsValuable(info *VulnInfo) bool {
	return info.Severity == SeverityHigh || info.Severity == SeverityCritical
}

// ── DBAppSecurity (安恒信息) ────────────────────────────────────────────────

// DbappsecurityCrawlerRod fetches from DBAppSecurity using Rod.
type DbappsecurityCrawlerRod struct {
	log *golog.Logger
}

// NewDbappsecurityCrawlerRod creates a new DBAppSecurity crawler using go-rod.
func NewDbappsecurityCrawlerRod() Grabber {
	return &DbappsecurityCrawlerRod{
		log: golog.Child("[dbappsecurity-rod]"),
	}
}

func (c *DbappsecurityCrawlerRod) ProviderInfo() *Provider {
	return &Provider{
		Name:        "dbappsecurity-rod",
		DisplayName: "安恒信息",
		Link:        "https://www.dbappsecurity.com.cn/",
	}
}

func (c *DbappsecurityCrawlerRod) GetUpdate(ctx context.Context, pageLimit int) ([]*VulnInfo, error) {
	browser, err := rodutil.GetBrowser(nil)
	if err != nil {
		return nil, fmt.Errorf("get browser: %w", err)
	}

	page, err := rodutil.NewPage(browser)
	if err != nil {
		return nil, fmt.Errorf("create page: %w", err)
	}
	defer page.Close()

	targetURL := "https://www.dbappsecurity.com.cn/notify/"

	config := &rodutil.BypassConfig{
		RotateUserAgent:       true,
		RandomViewport:        true,
		AdvancedFingerprint:   true,
		SimulateMouse:         true,
		RandomDelays:          true,
		MinDelay:              500,
		MaxDelay:              1500,
		AutoRetryOnWAF:        true,
		MaxRetries:            3,
	}

	if err := rodutil.NavigateWithRetry(ctx, page, targetURL, config); err != nil {
		return nil, fmt.Errorf("navigate: %w", err)
	}

	if rodutil.IsPageBlocked(page) {
		return nil, fmt.Errorf("waf blocked")
	}

	return []*VulnInfo{}, nil
}

func (c *DbappsecurityCrawlerRod) IsValuable(info *VulnInfo) bool {
	return info.Severity == SeverityHigh || info.Severity == SeverityCritical
}

// ── Utility Functions ────────────────────────────────────────────────────────

// parseCVEID extracts CVE ID from text using regex
func parseCVEID(text string) string {
	cveRegex := regexp.MustCompile(`CVE-\d{4}-\d{4,}`)
	match := cveRegex.FindString(text)
	return strings.TrimSpace(match)
}

// parseDisclosureDate parses Chinese date format to standard format
func parseDisclosureDate(dateStr string) string {
	// Handle common Chinese date formats
	dateStr = strings.TrimSpace(dateStr)

	// Format: YYYY-MM-DD
	if matched, _ := regexp.MatchString(`\d{4}-\d{2}-\d{2}`, dateStr); matched {
		return dateStr
	}

	// Format: YYYY/MM/DD
	if matched, _ := regexp.MatchString(`\d{4}/\d{2}/\d{2}`, dateStr); matched {
		return strings.ReplaceAll(dateStr, "/", "-")
	}

	// Format: YYYY年MM月DD日
	dateRegex := regexp.MustCompile(`(\d{4})年(\d{1,2})月(\d{1,2})日`)
	matches := dateRegex.FindStringSubmatch(dateStr)
	if len(matches) == 4 {
		month := matches[2]
		day := matches[3]
		if len(month) == 1 {
			month = "0" + month
		}
		if len(day) == 1 {
			day = "0" + day
		}
		return fmt.Sprintf("%s-%s-%s", matches[1], month, day)
	}

	return ""
}

// isValidURL checks if a string is a valid URL
func isValidURL(str string) bool {
	u, err := url.Parse(str)
	return err == nil && u.Scheme != "" && u.Host != ""
}

// containsBlockPattern checks if text contains any WAF block patterns
func containsBlockPattern(text string) bool {
	lowerText := strings.ToLower(text)
	for _, pattern := range cnBlockPatterns {
		if strings.Contains(lowerText, strings.ToLower(pattern)) {
			return true
		}
	}
	return false
}
