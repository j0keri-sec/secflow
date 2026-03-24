// Package vulngrabber defines the interfaces and common types for vulnerability
// data source crawlers. Each data source implements the Grabber interface,
// which can also be used as a standalone library.
//
// This package is adapted from secflow-agent/pkg/grabber to work with
// secflow-server's VulnRecord format.
package vulngrabber

import (
	"context"
	"fmt"
	"regexp"
)

// SeverityLevel represents the severity of a vulnerability.
type SeverityLevel string

const (
	SeverityLow      SeverityLevel = "低危"
	SeverityMedium   SeverityLevel = "中危"
	SeverityHigh     SeverityLevel = "高危"
	SeverityCritical SeverityLevel = "严重"
)

// Change reason constants for vulnerability updates.
const (
	ReasonNewCreated      = "漏洞创建"
	ReasonTagUpdated      = "标签更新"
	ReasonSeverityUpdated = "等级更新"
)

// VulnInfo is the normalized vulnerability information shared across all grabbers.
// This struct is compatible with secflow-server's model.VulnRecord.
type VulnInfo struct {
	// UniqueKey is the unique identifier for this vulnerability within its source.
	// Maps to VulnRecord.Key on the server.
	UniqueKey string `json:"unique_key" bson:"unique_key"`
	// Title is the human-readable title of the vulnerability.
	Title string `json:"title" bson:"title"`
	// Description is the detailed description.
	Description string `json:"description" bson:"description"`
	// Severity is the severity level.
	Severity SeverityLevel `json:"severity" bson:"severity"`
	// CVE is the CVE identifier, if available.
	CVE string `json:"cve" bson:"cve"`
	// Disclosure is the public disclosure date in "2006-01-02" format.
	Disclosure string `json:"disclosure" bson:"disclosure"`
	// Solutions contains the remediation guidance.
	Solutions string `json:"solutions" bson:"solutions"`
	// GithubSearch contains related GitHub URLs discovered via search.
	GithubSearch []string `json:"github_search" bson:"github_search"`
	// References contains external reference URLs.
	References []string `json:"references" bson:"references"`
	// Tags contains labels describing the vulnerability characteristics.
	Tags []string `json:"tags" bson:"tags"`
	// From is the source URL of this vulnerability record.
	// Maps to VulnRecord.From on the server.
	From string `json:"from" bson:"from"`
	// Reason describes why this vulnerability is being pushed.
	Reason []string `json:"reason" bson:"reason"`

	// Creator is the grabber that produced this vulnerability.
	// Not serialized — only used during the current collection cycle.
	Creator Grabber `json:"-" bson:"-"`
}

// String returns a short human-readable representation.
func (v *VulnInfo) String() string {
	return fmt.Sprintf("%s (%s)", v.Title, v.From)
}

// Provider contains metadata about a vulnerability data source.
type Provider struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Link        string `json:"link"`
}

// Grabber is the interface that every vulnerability data source must implement.
type Grabber interface {
	// ProviderInfo returns metadata about this data source.
	ProviderInfo() *Provider
	// GetUpdate fetches recent vulnerabilities, up to pageLimit pages.
	GetUpdate(ctx context.Context, pageLimit int) ([]*VulnInfo, error)
	// IsValuable returns true if the vulnerability is considered high-value
	// and should be pushed even without explicit whitelist matching.
	IsValuable(info *VulnInfo) bool
}

// MergeUniqueStrings merges two string slices, de-duplicating elements.
func MergeUniqueStrings(s1, s2 []string) []string {
	seen := make(map[string]struct{}, len(s1)+len(s2))
	for _, s := range s1 {
		seen[s] = struct{}{}
	}
	for _, s := range s2 {
		seen[s] = struct{}{}
	}
	result := make([]string, 0, len(seen))
	for k := range seen {
		result = append(result, k)
	}
	return result
}

// ToVulnRecord converts VulnInfo to a map matching server's VulnRecord format.
// This is used when sending results to the server.
func (v *VulnInfo) ToVulnRecord(sourceName string) map[string]interface{} {
	return map[string]interface{}{
		"key":           v.UniqueKey,
		"title":         v.Title,
		"description":   v.Description,
		"severity":      v.Severity,
		"cve":           v.CVE,
		"disclosure":    v.Disclosure,
		"solutions":     v.Solutions,
		"references":    v.References,
		"tags":          v.Tags,
		"github_search": v.GithubSearch,
		"from":          v.From,
		"source":        sourceName, // The grabber name
		"pushed":        false,
		// reported_by is set by the server based on the node ID
	}
}

// Accessor methods for duck-typing compatibility with engine.vulnToMap.
func (v *VulnInfo) GetKey() string           { return v.UniqueKey }
func (v *VulnInfo) GetTitle() string         { return v.Title }
func (v *VulnInfo) GetSeverity() string      { return string(v.Severity) }
func (v *VulnInfo) GetCVE() string           { return v.CVE }
func (v *VulnInfo) GetSource() string        { return "" } // Set by engine based on grabber name
func (v *VulnInfo) GetDescription() string   { return v.Description }
func (v *VulnInfo) GetDisclosure() string    { return v.Disclosure }
func (v *VulnInfo) GetTags() []string        { return v.Tags }
func (v *VulnInfo) GetReferences() []string  { return v.References }
func (v *VulnInfo) GetGithubSearch() []string { return v.GithubSearch }
func (v *VulnInfo) GetSolutions() string     { return v.Solutions }
func (v *VulnInfo) GetFrom() string          { return v.From }
func (v *VulnInfo) GetUniqueKey() string     { return v.UniqueKey }

// scriptRegexp is used by WAF bypass implementations in multiple grabbers.
// Defined here to avoid duplicate definitions across files.
var scriptRegexp = regexp.MustCompile(`(?m)<script>(.*?)</script>`)
