// Package middleware provides HTTP middleware for metrics collection.
package middleware

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/secflow/server/internal/metrics"
)

// Prometheus returns a Gin middleware that collects Prometheus metrics.
func Prometheus() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}

		c.Next()

		duration := time.Since(start).Seconds()
		status := strconv.Itoa(c.Writer.Status())

		metrics.RecordHTTPRequest(c.Request.Method, path, status, duration)
	}
}
