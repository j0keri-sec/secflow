package repository

import (
	"context"
	"strings"
	"time"

	"github.com/secflow/server/internal/model"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// ReportVulnFilter defines filter options for report vulnerability queries.
type ReportVulnFilter struct {
	DateFrom   time.Time
	DateTo     time.Time
	Page       int
	PageSize   int
	PushState  *bool
	Source     string
}

// GetStatsByDateRange returns vulnerability statistics for the given date range.
func (r *VulnRepo) GetStatsByDateRange(ctx context.Context, dateFrom, dateTo time.Time, source string) (*VulnStats, error) {
	filter := bson.M{}
	if !dateFrom.IsZero() || !dateTo.IsZero() {
		dateFilter := bson.M{}
		if !dateFrom.IsZero() {
			dateFilter["$gte"] = dateFrom
		}
		if !dateTo.IsZero() {
			dateFilter["$lte"] = dateTo
		}
		filter["created_at"] = dateFilter
	}

	// Filter by source if specified
	if source != "" {
		filter["source"] = source
	}

	coll := r.db.coll(model.CollVulnRecords)

	// Get total count
	total, err := coll.CountDocuments(ctx, filter)
	if err != nil {
		return nil, err
	}

	stats := &VulnStats{
		Total:      int(total),
		ByCategory: make(map[string]int),
		BySeverity: make(map[string]int),
	}

	// Get counts by category (using source field or first tag)
	pipeline := []bson.M{
		{"$match": filter},
		{"$group": bson.M{"_id": "$source", "count": bson.M{"$sum": 1}}},
	}

	cursor, err := coll.Aggregate(ctx, pipeline)
	if err != nil {
		return stats, nil // Non-critical, return partial data
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var result struct {
			ID    string `bson:"_id"`
			Count int    `bson:"count"`
		}
		if err := cursor.Decode(&result); err != nil {
			continue
		}
		category := result.ID
		if category == "" {
			category = "其他"
		}
		stats.ByCategory[category] = result.Count
	}

	return stats, nil
}

// VulnStats holds vulnerability statistics for reports.
type VulnStats struct {
	Total      int
	ByCategory map[string]int
	BySeverity map[string]int
}

// GetTopVulns returns top vulnerabilities for the given date range.
func (r *VulnRepo) GetTopVulns(ctx context.Context, dateFrom, dateTo time.Time, source string, limit int) ([]VulnItem, error) {
	filter := bson.M{}
	if !dateFrom.IsZero() || !dateTo.IsZero() {
		dateFilter := bson.M{}
		if !dateFrom.IsZero() {
			dateFilter["$gte"] = dateFrom
		}
		if !dateTo.IsZero() {
			dateFilter["$lte"] = dateTo
		}
		filter["created_at"] = dateFilter
	}

	// Filter by source if specified
	if source != "" {
		filter["source"] = source
	}

	coll := r.db.coll(model.CollVulnRecords)

	if limit <= 0 {
		limit = 10
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetLimit(int64(limit))

	cursor, err := coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var items []VulnItem
	num := 1
	for cursor.Next(ctx) {
		var v model.VulnRecord
		if err := cursor.Decode(&v); err != nil {
			continue
		}
		items = append(items, VulnItem{
			Number:     num,
			Name:       v.Title,
			CVE:        v.CVE,
			Severity:   string(v.Severity),
			Vendor:     extractVendor(v.Title),
			Product:    "",
			Description: v.Description,
			Solutions:  v.Solutions,
			Source:     v.Source,
		})
		num++
	}

	return items, nil
}

// VulnItem represents a vulnerability for reports.
type VulnItem struct {
	Number     int
	Name       string
	CVE        string
	Severity   string
	Vendor     string
	Product    string
	Description string
	Solutions  string
	Source     string
}

// ArticleListFilter defines filter options for listing articles.
type ArticleListFilter struct {
	DateFrom   time.Time
	DateTo     time.Time
	Page       int
	PageSize   int
	Source     string
	Keyword    string
}

// GetSecurityEvents returns security-related articles for the given date range.
func (r *ArticleRepository) GetSecurityEvents(ctx context.Context, dateFrom, dateTo time.Time, limit int) ([]EventItem, error) {
	filter := bson.M{}
	if !dateFrom.IsZero() || !dateTo.IsZero() {
		dateFilter := bson.M{}
		if !dateFrom.IsZero() {
			dateFilter["$gte"] = dateFrom
		}
		if !dateTo.IsZero() {
			dateFilter["$lte"] = dateTo
		}
		filter["published_at"] = dateFilter
	}

	coll := r.db.coll(model.CollArticles)

	opts := options.Find().
		SetSort(bson.D{{Key: "published_at", Value: -1}}).
		SetLimit(int64(limit))

	if limit <= 0 {
		limit = 5
	}

	cursor, err := coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var items []EventItem
	num := 1
	for cursor.Next(ctx) {
		var a model.Article
		if err := cursor.Decode(&a); err != nil {
			continue
		}
		items = append(items, EventItem{
			Number: num,
			Title:  a.Title,
			Source: a.Source,
		})
		num++
	}

	return items, nil
}

// EventItem represents a security event for reports.
type EventItem struct {
	Number int
	Title  string
	Source string
}

// extractVendor attempts to extract vendor name from vulnerability title.
func extractVendor(title string) string {
	vendors := []string{
		"Microsoft", "Google", "Adobe", "Apple", "Cisco", "Oracle", "IBM",
		"Intel", "NVIDIA", "Dell", "HP", "VMware", "Mozilla", "Apache",
		"nginx", "PHP", "MySQL", "PostgreSQL", "Redis", "Elastic",
		"Fortinet", "Palo Alto", "Juniper", "D-Link", "TP-Link", "NETGEAR",
	}

	titleLower := strings.ToLower(title)
	for _, v := range vendors {
		if strings.Contains(titleLower, strings.ToLower(v)) {
			return v
		}
	}
	return ""
}

