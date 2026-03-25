package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/secflow/server/pkg/auth"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestRequestID(t *testing.T) {
	r := gin.New()
	r.Use(RequestID())
	r.GET("/test", func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		c.String(http.StatusOK, requestID)
	})

	// Test without providing request ID
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)
	assert.NotEmpty(t, w.Header().Get("X-Request-ID"))

	// Test with provided request ID
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/test", nil)
	req2.Header.Set("X-Request-ID", "custom-request-id")
	r.ServeHTTP(w2, req2)
	assert.Equal(t, "custom-request-id", w2.Header().Get("X-Request-ID"))
}

func TestJWTAuth(t *testing.T) {
	authSvc := auth.New("test-secret", 24*time.Hour)
	r := gin.New()
	r.Use(JWTAuth(authSvc))
	r.GET("/test", func(c *gin.Context) {
		userID := GetUserID(c)
		c.String(http.StatusOK, userID)
	})

	// Generate a valid token
	token, _ := authSvc.GenerateToken("user123", "testuser", "admin")

	// Test with valid token
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Test without token
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusUnauthorized, w2.Code)

	// Test with invalid token
	w3 := httptest.NewRecorder()
	req3, _ := http.NewRequest("GET", "/test", nil)
	req3.Header.Set("Authorization", "Bearer invalid-token")
	r.ServeHTTP(w3, req3)
	assert.Equal(t, http.StatusUnauthorized, w3.Code)
}

func TestRequireRole(t *testing.T) {
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("role", "editor")
		c.Next()
	})
	r.Use(RequireRole("viewer"))
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	// Editor should access viewer endpoint
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Test with insufficient role
	r2 := gin.New()
	r2.Use(func(c *gin.Context) {
		c.Set("role", "viewer")
		c.Next()
	})
	r2.Use(RequireRole("admin"))
	r2.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/test", nil)
	r2.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusForbidden, w2.Code)
}

func TestCORS(t *testing.T) {
	// Test with no allowed origins (allow all)
	r := gin.New()
	r.Use(CORS([]string{}))
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "http://example.com")
	r.ServeHTTP(w, req)
	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))

	// Test with specific allowed origins
	r2 := gin.New()
	r2.Use(CORS([]string{"http://allowed.com"}))
	r2.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/test", nil)
	req2.Header.Set("Origin", "http://allowed.com")
	r2.ServeHTTP(w2, req2)
	assert.Equal(t, "http://allowed.com", w2.Header().Get("Access-Control-Allow-Origin"))

	// Test with disallowed origin
	w3 := httptest.NewRecorder()
	req3, _ := http.NewRequest("GET", "/test", nil)
	req3.Header.Set("Origin", "http://disallowed.com")
	r2.ServeHTTP(w3, req3)
	assert.Empty(t, w3.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORS_Preflight(t *testing.T) {
	r := gin.New()
	r.Use(CORS([]string{"http://allowed.com"}))
	r.OPTIONS("/test", func(c *gin.Context) {})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("OPTIONS", "/test", nil)
	req.Header.Set("Origin", "http://allowed.com")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestRecovery(t *testing.T) {
	r := gin.New()
	r.Use(Recovery())
	r.GET("/panic", func(c *gin.Context) {
		panic("test panic")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/panic", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestTimeout(t *testing.T) {
	r := gin.New()
	r.Use(Timeout(100 * time.Millisecond))
	r.GET("/slow", func(c *gin.Context) {
		time.Sleep(200 * time.Millisecond)
		c.String(http.StatusOK, "OK")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/slow", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusGatewayTimeout, w.Code)
}
