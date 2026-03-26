package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestRateLimiter_Allow(t *testing.T) {
	// Create a limiter: 3 requests per second
	limiter := NewRateLimiter(3, time.Second)
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		key        string
		numAllowed int
	}{
		{
			name:       "first request allowed",
			key:        "ip1",
			numAllowed: 1,
		},
		{
			name:       "second request allowed",
			key:        "ip2",
			numAllowed: 2,
		},
		{
			name:       "third request allowed",
			key:        "ip3",
			numAllowed: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allowed := 0
			for i := 0; i < tt.numAllowed; i++ {
				if limiter.Allow(tt.key) {
					allowed++
				}
			}
			assert.Equal(t, tt.numAllowed, allowed, "should allow exact number of requests")
		})
	}
}

func TestRateLimiter_BlockExcess(t *testing.T) {
	limiter := NewRateLimiter(2, time.Second)
	gin.SetMode(gin.TestMode)

	key := "test-ip"

	// First two should be allowed
	assert.True(t, limiter.Allow(key), "first request should be allowed")
	assert.True(t, limiter.Allow(key), "second request should be allowed")

	// Third should be blocked
	assert.False(t, limiter.Allow(key), "third request should be blocked")
}

func TestRateLimiter_WindowExpiry(t *testing.T) {
	// Use a very short window for testing
	limiter := NewRateLimiter(2, 100*time.Millisecond)
	gin.SetMode(gin.TestMode)

	key := "test-ip"

	// Use up the limit
	assert.True(t, limiter.Allow(key))
	assert.True(t, limiter.Allow(key))
	assert.False(t, limiter.Allow(key))

	// Wait for window to expire
	time.Sleep(150 * time.Millisecond)

	// Should be allowed again
	assert.True(t, limiter.Allow(key), "request should be allowed after window expires")
}

func TestRateLimiter_DifferentKeys(t *testing.T) {
	limiter := NewRateLimiter(1, time.Second)
	gin.SetMode(gin.TestMode)

	// Different keys should have independent limits
	assert.True(t, limiter.Allow("ip1"))
	assert.True(t, limiter.Allow("ip2"))
	assert.True(t, limiter.Allow("ip3"))

	// Each should now be blocked
	assert.False(t, limiter.Allow("ip1"))
	assert.False(t, limiter.Allow("ip2"))
	assert.False(t, limiter.Allow("ip3"))
}

func TestRateLimitMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	limiter := NewRateLimiter(2, time.Second)

	r := gin.New()
	r.Use(RateLimit(limiter))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// First two requests should succeed
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest("GET", "/test", nil)
	req1.RemoteAddr = "192.168.1.1:1234"
	r.ServeHTTP(w1, req1)
	assert.Equal(t, 200, w1.Code)

	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/test", nil)
	req2.RemoteAddr = "192.168.1.1:1234"
	r.ServeHTTP(w2, req2)
	assert.Equal(t, 200, w2.Code)

	// Third request should be rate limited
	w3 := httptest.NewRecorder()
	req3, _ := http.NewRequest("GET", "/test", nil)
	req3.RemoteAddr = "192.168.1.1:1234"
	r.ServeHTTP(w3, req3)
	assert.Equal(t, 429, w3.Code)
}

func TestNewRateLimiter(t *testing.T) {
	limiter := NewRateLimiter(5, time.Minute)
	assert.NotNil(t, limiter)
	assert.Equal(t, 5, limiter.limit)
	assert.Equal(t, time.Minute, limiter.window)
}
