// Package middleware contains Gin middleware used across all routes.
package middleware

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/secflow/server/pkg/auth"
)

const ctxKeyUserID    = "userID"
const ctxKeyUsername  = "username"
const ctxKeyRole      = "role"
const ctxKeyRequestID = "request_id"

// RateLimiter implements a simple in-memory rate limiter for auth endpoints.
// For production with multiple servers, use Redis-based rate limiting.
type RateLimiter struct {
	mu       sync.RWMutex
	requests map[string][]time.Time // IP -> timestamps of recent requests
	limit    int                   // max requests per window
	window   time.Duration         // time window
}

// NewRateLimiter creates a new rate limiter.
// limit: maximum requests allowed per window
// window: time window duration
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		requests: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
	}
	// Start cleanup goroutine
	go rl.cleanup()
	return rl
}

// Allow checks if a request from the given key (e.g., IP) should be allowed.
func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-rl.window)

	// Filter out old requests
	var recent []time.Time
	for _, t := range rl.requests[key] {
		if t.After(windowStart) {
			recent = append(recent, t)
		}
	}

	if len(recent) >= rl.limit {
		rl.requests[key] = recent
		return false
	}

	rl.requests[key] = append(recent, now)
	return true
}

// cleanup periodically removes old entries to prevent memory growth
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		windowStart := now.Add(-rl.window)
		for key, times := range rl.requests {
			var recent []time.Time
			for _, t := range times {
				if t.After(windowStart) {
					recent = append(recent, t)
				}
			}
			if len(recent) == 0 {
				delete(rl.requests, key)
			} else {
				rl.requests[key] = recent
			}
		}
		rl.mu.Unlock()
	}
}

// RateLimit returns a Gin middleware that rate limits requests by IP.
// Use this for sensitive endpoints like login, password reset, etc.
func RateLimit(limiter *RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		if !limiter.Allow(ip) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "rate limit exceeded, please try again later",
				"code":  429,
			})
			return
		}
		c.Next()
	}
}

// RequestID middleware generates a unique request ID for each request.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.NewString()
		}
		c.Set(ctxKeyRequestID, requestID)
		c.Header("X-Request-ID", requestID)
		c.Next()
	}
}

// JWTAuth returns a Gin middleware that validates Bearer tokens.
func JWTAuth(svc *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" || !strings.HasPrefix(header, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing or invalid authorization header"})
			return
		}
		token := strings.TrimPrefix(header, "Bearer ")
		claims, err := svc.ParseToken(token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}
		c.Set(ctxKeyUserID, claims.UserID)
		c.Set(ctxKeyUsername, claims.Username)
		c.Set(ctxKeyRole, claims.Role)
		c.Next()
	}
}

// RequireRole returns a Gin middleware that enforces a minimum role.
// Role hierarchy: admin > editor > viewer
func RequireRole(role string) gin.HandlerFunc {
	rankOf := map[string]int{"viewer": 1, "editor": 2, "admin": 3}
	return func(c *gin.Context) {
		userRole := c.GetString(ctxKeyRole)
		if rankOf[userRole] < rankOf[role] {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
			return
		}
		c.Next()
	}
}

// GetUserID extracts the user ID from the Gin context (set by JWTAuth).
func GetUserID(c *gin.Context) string   { return c.GetString(ctxKeyUserID) }
func GetUsername(c *gin.Context) string { return c.GetString(ctxKeyUsername) }
func GetRole(c *gin.Context) string     { return c.GetString(ctxKeyRole) }

// Logger returns a zerolog-based request logger middleware.
func Logger() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		if param.StatusCode >= 500 {
			return ""
		}
		return "" // zerolog handles request logging separately
	})
}

// Recovery returns a recovery middleware that returns JSON on panic.
func Recovery() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"error": "internal server error",
		})
	})
}

// CORS returns a CORS middleware based on configured allowed origins.
// If allowedOrigins is empty, it allows all origins (not recommended for production).
// If allowedOrigins contains values, only those origins are allowed.
func CORS(allowedOrigins []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")

		// Determine allowed origin
		allowedOrigin := ""
		if len(allowedOrigins) == 0 {
			// No origins configured - allow all (development mode)
			allowedOrigin = "*"
		} else {
			// Check if request origin is in whitelist
			for _, o := range allowedOrigins {
				if o == origin {
					allowedOrigin = origin
					break
				}
			}
			// If no match found, don't set Access-Control-Allow-Origin
			// This will cause browsers to block the response for cross-origin requests
		}

		if allowedOrigin != "" {
			c.Header("Access-Control-Allow-Origin", allowedOrigin)
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
			c.Header("Access-Control-Allow-Credentials", "true")
		}

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

// Timeout returns a middleware that adds a request timeout context.
// If a request takes longer than the specified duration, it aborts with 504 Gateway Timeout.
// This prevents slow requests from blocking the server indefinitely.
func Timeout(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()

		c.Request = c.Request.WithContext(ctx)

		done := make(chan struct{})
		go func() {
			c.Next()
			close(done)
		}()

		select {
		case <-done:
			// Request completed normally
			return
		case <-ctx.Done():
			c.AbortWithStatusJSON(http.StatusGatewayTimeout, gin.H{
				"error": "request timeout",
				"code":  504,
			})
		}
	}
}
