package handler

import (
	"encoding/csv"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/secflow/server/internal/repository"
)

// VulnHandler handles all vuln_records endpoints.
type VulnHandler struct {
	vulnRepo *repository.VulnRepo
}

func NewVulnHandler(vr *repository.VulnRepo) *VulnHandler {
	return &VulnHandler{vulnRepo: vr}
}

// List returns a paginated list of vulnerability records.
//
// GET /api/v1/vulns
func (h *VulnHandler) List(c *gin.Context) {
	page, pageSize := pageParams(c)
	filter := repository.VulnListFilter{
		Severity: c.Query("severity"),
		Source:   c.Query("source"),
		CVE:      c.Query("cve"),
		Keywords: c.Query("keyword"),
		Page:     page,
		PageSize: pageSize,
	}
	if p := c.Query("pushed"); p == "true" {
		t := true
		filter.Pushed = &t
	} else if p == "false" {
		f := false
		filter.Pushed = &f
	}

	items, total, err := h.vulnRepo.List(c, filter)
	if err != nil {
		log.Error().Err(err).Msg("failed to list vulnerabilities")
		fail(c, http.StatusInternalServerError, "failed to retrieve vulnerabilities")
		return
	}
	okPage(c, total, page, pageSize, items)
}

// Get returns a single vulnerability by its MongoDB ID or unique key.
//
// GET /api/v1/vulns/:id
func (h *VulnHandler) Get(c *gin.Context) {
	key := c.Param("id")
	v, err := h.vulnRepo.GetByKey(c, key)
	if err != nil {
		log.Error().Err(err).Str("key", key).Msg("failed to get vulnerability")
		fail(c, http.StatusInternalServerError, "failed to retrieve vulnerability")
		return
	}
	if v == nil {
		fail(c, http.StatusNotFound, "not found")
		return
	}
	ok(c, v)
}

// Delete removes a vulnerability by MongoDB ObjectID.
//
// DELETE /api/v1/vulns/:id
func (h *VulnHandler) Delete(c *gin.Context) {
	id, err := bson.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	if err = h.vulnRepo.Delete(c, id); err != nil {
		log.Error().Err(err).Str("id", id.Hex()).Msg("failed to delete vulnerability")
		fail(c, http.StatusInternalServerError, "failed to delete vulnerability")
		return
	}
	ok(c, nil)
}

// Stats returns aggregated counts used by the dashboard.
//
// GET /api/v1/vulns/stats
func (h *VulnHandler) Stats(c *gin.Context) {
	total, _ := h.vulnRepo.Count(c)
	severities := []string{"严重", "高危", "中危", "低危"}
	bySeverity := make(map[string]int64, 4)
	for _, s := range severities {
		_, cnt, err := h.vulnRepo.List(c, repository.VulnListFilter{
			Severity: s,
			PageSize: 1,
		})
		if err == nil {
			bySeverity[s] = cnt
		}
	}
	ok(c, gin.H{
		"total":       total,
		"by_severity": bySeverity,
	})
}

// Export streams vulnerability records as a CSV download.
//
// GET /api/v1/vulns/export
func (h *VulnHandler) Export(c *gin.Context) {
	filter := repository.VulnListFilter{
		Severity: c.Query("severity"),
		Source:   c.Query("source"),
		CVE:      c.Query("cve"),
		Keywords: c.Query("keyword"),
		Page:     1,
		PageSize: 10000, // cap at 10 k for safety
	}

	items, _, err := h.vulnRepo.List(c, filter)
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="vulns_%d.csv"`, len(items)))
	c.Header("Content-Type", "text/csv; charset=utf-8")

	w := csv.NewWriter(c.Writer)
	defer w.Flush()

	_ = w.Write([]string{"标题", "CVE", "严重程度", "数据源", "披露日期", "推送", "创建时间"})
	for _, v := range items {
		pushed := "否"
		if v.Pushed {
			pushed = "是"
		}
		_ = w.Write([]string{
			v.Title, v.CVE, string(v.Severity),
			v.Source, v.Disclosure, pushed,
			v.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
}
