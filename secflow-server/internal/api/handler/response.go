// Package handler contains all Gin HTTP handler functions.
package handler

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// ProtocolVersion is the current API protocol version.
const ProtocolVersion = "1.0"

// Standardized error codes for API responses.
const (
	ErrCodeOK            = 0
	ErrCodeBadRequest    = 400
	ErrCodeUnauthorized  = 401
	ErrCodeForbidden     = 403
	ErrCodeNotFound      = 404
	ErrCodeConflict      = 409
	ErrCodeInternalError = 500
)

// resp is the unified API response envelope.
type resp struct {
	Code      int         `json:"code"`
	Message   string      `json:"message"`
	RequestID string      `json:"request_id"`
	Timestamp int64       `json:"timestamp"`
	Version   string      `json:"version"`
	Data      interface{} `json:"data,omitempty"`
}

// pageData wraps a paginated result set.
type pageData struct {
	Total    int64       `json:"total"`
	Page     int64       `json:"page"`
	PageSize int64       `json:"page_size"`
	Items    interface{} `json:"items"`
}

// buildResp creates a response envelope with tracing fields.
func buildResp(c *gin.Context, code int, message string, data interface{}) resp {
	return resp{
		Code:      code,
		Message:   message,
		RequestID: c.GetString("request_id"),
		Timestamp: time.Now().UnixMilli(),
		Version:   ProtocolVersion,
		Data:      data,
	}
}

// ok sends a 200 JSON response. Alias: OK
func ok(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, buildResp(c, 0, "ok", data))
}

// OK is a public alias for ok.
func OK(c *gin.Context, data interface{}) { ok(c, data) }

// okPage sends a 200 paginated JSON response.
func okPage(c *gin.Context, total int64, page, pageSize int64, items interface{}) {
	c.JSON(http.StatusOK, buildResp(c, 0, "ok", pageData{Total: total, Page: page, PageSize: pageSize, Items: items}))
}

// PageResult builds a pageData value for use with OK().
func PageResult(items interface{}, total int64, page, pageSize int) interface{} {
	return pageData{
		Total:    total,
		Page:     int64(page),
		PageSize: int64(pageSize),
		Items:    items,
	}
}

// fail sends an error JSON response. Alias: Err
func fail(c *gin.Context, status int, msg string) {
	c.JSON(status, buildResp(c, status, msg, nil))
}

// Err is a public alias for fail.
func Err(c *gin.Context, status int, msg string) { fail(c, status, msg) }

// pageParams extracts pagination query params with defaults.
func pageParams(c *gin.Context) (page, pageSize int64) {
	page = int64(maxInt(c.GetInt("page"), 1))
	pageSize = int64(c.GetInt("page_size"))
	if pageSize == 0 {
		pageSize = 20
	}
	if p := c.Query("page"); p != "" {
		var v int64
		if _, err := intFromStr(p, &v); err == nil && v > 0 {
			page = v
		}
	}
	if ps := c.Query("page_size"); ps != "" {
		var v int64
		if _, err := intFromStr(ps, &v); err == nil && v > 0 {
			pageSize = v
		}
	}
	return
}

// paginate is a convenience wrapper returning (page, pageSize) as int.
func paginate(c *gin.Context) (int, int) {
	p, ps := pageParams(c)
	return int(p), int(ps)
}

func intFromStr(s string, v *int64) (int64, error) {
	n, err := parseI64(s)
	if err != nil {
		return 0, err
	}
	*v = n
	return n, nil
}

func parseI64(s string) (int64, error) {
	var n int64
	_, err := fmt.Sscanf(s, "%d", &n)
	return n, err
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

