// Package grabber provides NVD (National Vulnerability Database) crawler using go-rod.
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

// NVDCrawlerRod is the go-rod based implementation for NVD.
type NVDCrawlerRod struct {
	log *golog.Logger
}

// NewNVDCrawlerRod creates a new NVD crawler using go-rod.
func NewNVDCrawlerRod() Grabber {
	return &NVDCrawlerRod{
		log: golog.Child("[NVD-rod]"),
	}
}

func (c *NVDCrawlerRod) ProviderInfo() *Provider {
	return &Provider{
		Name:        "NVD",
		DisplayName: "National Vulnerability Database",
		Link:        "https://nvd.nist.gov/",
	}
}

func (c *NVDCrawlerRod) IsValuable(info *VulnInfo) bool {
	return info.Severity == SeverityHigh || info.Severity == SeverityCritical
}

// CVSSv3SeverityToLevel converts CVSS v3 severity string to our severity level.
func CVSSv3SeverityToLevel(severity string) SeverityLevel {
	switch strings.ToUpper(severity) {
	case "CRITICAL":
		return SeverityCritical
	case "HIGH":
		return SeverityHigh
	case "MEDIUM":
		return SeverityMedium
	case "LOW":
		return SeverityLow
	default:
		return SeverityMedium
	}
}

func (c *NVDCrawlerRod) GetUpdate(ctx context.Context, pageLimit int) ([]*VulnInfo, error) {
	// NVD API 2.0 endpoint
	// Get recent vulnerabilities from the past 7 days
	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -7)

	// Build API URL with recent modified vulnerabilities
	apiURL := fmt.Sprintf(
		"https://services.nvd.nist.gov/rest/json/cves/2.0?pubStartDate=%s&resultsPerPage=20",
		startDate.Format(time.RFC3339),
	)

	c.log.Infof("Fetching NVD vulnerabilities from: %s", apiURL)

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
	script := fmt.Sprintf(`() => {
		return new Promise((resolve, reject) => {
			fetch('%s', {
				method: 'GET',
				headers: {
					'Accept': 'application/json',
					'Referer': 'https://nvd.nist.gov/'
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
	}`, apiURL)

	result, err := page.Eval(script)
	if err != nil {
		return nil, fmt.Errorf("failed to execute fetch script: %w", err)
	}

	// Parse JSON response
	jsonStr := result.Value.String()

	var nvdResp NVDResponse
	if err := json.Unmarshal([]byte(jsonStr), &nvdResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if nvdResp.TotalResults == 0 {
		c.log.Infof("No recent vulnerabilities found in NVD")
		return []*VulnInfo{}, nil
	}

	c.log.Infof("Fetched %d vulnerabilities from NVD (total: %d)", len(nvdResp.Vulnerabilities), nvdResp.TotalResults)

	// Process vulnerabilities
	var vulnInfos []*VulnInfo
	limit := pageLimit * 20
	if limit > len(nvdResp.Vulnerabilities) {
		limit = len(nvdResp.Vulnerabilities)
	}

	for i := 0; i < limit; i++ {
		vuln := nvdResp.Vulnerabilities[i]
		cveID := vuln.CVE.ID
		description := ""
		severity := SeverityMedium
		score := ""

		// Get description
		if len(vuln.CVE.Descriptions) > 0 {
			for _, desc := range vuln.CVE.Descriptions {
				if desc.Lang == "en" {
					description = desc.Value
					break
				}
			}
		}

		// Get CVSS score and severity - check if metrics exist and have data
		if len(vuln.CVE.Metrics.CvssMetricV31) > 0 {
			cvss := vuln.CVE.Metrics.CvssMetricV31[0].CvssData
			severity = CVSSv3SeverityToLevel(cvss.BaseSeverity)
			score = fmt.Sprintf("%.1f", cvss.BaseScore)
		} else if len(vuln.CVE.Metrics.CvssMetricV30) > 0 {
			cvss := vuln.CVE.Metrics.CvssMetricV30[0].CvssData
			severity = CVSSv3SeverityToLevel(cvss.BaseSeverity)
			score = fmt.Sprintf("%.1f", cvss.BaseScore)
		}

		// Get references
		var refs []string
		for _, ref := range vuln.CVE.References {
			refs = append(refs, ref.URL)
		}

		// Get published date
		published := vuln.CVE.Published
		if published == "" {
			published = time.Now().Format("2006-01-02")
		}

		vulnInfo := &VulnInfo{
			UniqueKey:   cveID + "_NVD",
			Title:       cveID,
			Description: description,
			Severity:    severity,
			CVE:         cveID,
			Disclosure:  published,
			References:  refs,
			From:        "https://nvd.nist.gov/vuln/detail/" + cveID,
			Creator:     c,
		}

		// Add CVSS score as tag if available
		if score != "" {
			vulnInfo.Tags = []string{"CVSS " + score}
		}

		vulnInfos = append(vulnInfos, vulnInfo)
	}

	return vulnInfos, nil
}

// NVDResponse represents the NVD API response structure.
type NVDResponse struct {
	ResultsPerPage int           `json:"resultsPerPage"`
	StartIndex    int           `json:"startIndex"`
	TotalResults  int           `json:"totalResults"`
	Vulnerabilities []struct {
		CVE struct {
			ID          string `json:"id"`
			Published   string `json:"published"`
			LastModified string `json:"lastModified"`
			Descriptions []struct {
				Lang  string `json:"lang"`
				Value string `json:"value"`
			} `json:"descriptions"`
			References []struct {
				URL       string `json:"url"`
				Source    string `json:"source"`
				Refsource string `json:"refsource"`
			} `json:"references"`
			Metrics struct {
				CvssMetricV31 []struct {
					CvssData struct {
						BaseScore     float64 `json:"baseScore"`
						BaseSeverity  string   `json:"baseSeverity"`
						AttackVector  string   `json:"attackVector"`
						AttackComplexity string `json:"attackComplexity"`
						PrivilegesRequired string `json:"privilegesRequired"`
						UserInteraction string `json:"userInteraction"`
						Scope          string `json:"scope"`
						ConfidentialityImpact string `json:"confidentialityImpact"`
						IntegrityImpact string `json:"integrityImpact"`
						AvailabilityImpact string `json:"availabilityImpact"`
					} `json:"cvssData"`
				} `json:"cvssMetricV31"`
				CvssMetricV30 []struct {
					CvssData struct {
						BaseScore     float64 `json:"baseScore"`
						BaseSeverity  string   `json:"baseSeverity"`
					} `json:"cvssData"`
				} `json:"cvssMetricV30"`
			} `json:"metrics"`
			Weaknesses []struct {
				Description []struct {
					Value string `json:"value"`
				} `json:"description"`
			} `json:"weaknesses"`
			Configurations []struct {
				Nodes []struct {
					CPEMatch []struct {
						Criteria     string `json:"criteria"`
						Vulnerable   bool   `json:"vulnerable"`
						CriteriaMatch bool `json:"criteriaMatch"`
					} `json:"cpeMatch"`
				} `json:"nodes"`
			} `json:"configurations"`
		} `json:"cve"`
	} `json:"vulnerabilities"`
}
