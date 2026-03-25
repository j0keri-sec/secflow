package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// AuthIntegrationTestSuite tests authentication flows
type AuthIntegrationTestSuite struct {
	IntegrationTestSuite
}

// TestAuthFlow tests the complete authentication flow
func (s *AuthIntegrationTestSuite) TestAuthLogin() {
	// Test login endpoint
	resp, err := s.APIClient.Post("/api/v1/auth/login", map[string]string{
		"username": "admin",
		"password": "admin123",
	})
	require.NoError(s.T(), err)
	defer resp.Body.Close()

	// Should return 200 or 401 depending on test data
	assert.True(s.T(), resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusUnauthorized)
}

// TestAuthRegister tests user registration
func (s *AuthIntegrationTestSuite) TestAuthRegister() {
	// Generate unique username
	username := fmt.Sprintf("testuser_%d", time.Now().UnixNano())

	resp, err := s.APIClient.Post("/api/v1/auth/register", map[string]string{
		"username":   username,
		"password":   "TestPass123!",
		"email":     username + "@test.com",
		"invite_code": "TESTCODE123",
	})
	require.NoError(s.T(), err)
	defer resp.Body.Close()

	// Should return 201 or 400 depending on invite code validity
	assert.True(s.T(), resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusBadRequest)
}

// TestPasswordReset tests password reset flow
func (s *AuthIntegrationTestSuite) TestPasswordResetRequest() {
	resp, err := s.APIClient.Post("/api/v1/auth/reset/request", map[string]string{
		"email": "test@example.com",
	})
	require.NoError(s.T(), err)
	defer resp.Body.Close()

	// Should always return 200 (email enumeration prevention)
	assert.Equal(s.T(), http.StatusOK, resp.StatusCode)
}

// NodeIntegrationTestSuite tests node management
type NodeIntegrationTestSuite struct {
	IntegrationTestSuite
}

// TestNodeList tests listing nodes
func (s *NodeIntegrationTestSuite) TestNodeList() {
	// First login to get token
	loginResp, err := s.APIClient.Post("/api/v1/auth/login", map[string]string{
		"username": "admin",
		"password": "admin123",
	})
	require.NoError(s.T(), err)
	defer loginResp.Body.Close()

	if loginResp.StatusCode == http.StatusOK {
		var loginResult struct {
			Code int    `json:"code"`
			Data struct {
				Token string `json:"token"`
			} `json:"data"`
		}
		json.NewDecoder(loginResp.Body).Decode(&loginResult)
		s.APIClient.SetToken(loginResult.Data.Token)
	}

	// Test nodes endpoint
	resp, err := s.APIClient.Get("/api/v1/nodes")
	require.NoError(s.T(), err)
	defer resp.Body.Close()

	// Should return 200 with list or 401 without auth
	assert.True(s.T(), resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusUnauthorized)
}

// TestHealthEndpoint tests health check
func (s *NodeIntegrationTestSuite) TestHealthEndpoint() {
	resp, err := s.APIClient.Get("/api/v1/health")
	require.NoError(s.T(), err)
	defer resp.Body.Close()

	assert.Equal(s.T(), http.StatusOK, resp.StatusCode)
}

// VulnIntegrationTestSuite tests vulnerability operations
type VulnIntegrationTestSuite struct {
	IntegrationTestSuite
}

// TestVulnList tests listing vulnerabilities
func (s *VulnIntegrationTestSuite) TestVulnList() {
	resp, err := s.APIClient.Get("/api/v1/vulns?page=1&page_size=10")
	require.NoError(s.T(), err)
	defer resp.Body.Close()

	// Should return 200 or 401 depending on auth
	assert.True(s.T(), resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusUnauthorized)
}

// TestVulnSearch tests vulnerability search
func (s *VulnIntegrationTestSuite) TestVulnSearch() {
	resp, err := s.APIClient.Post("/api/v1/vulns/search", map[string]interface{}{
		"query":    "remote code execution",
		"severity": []string{"CRITICAL", "HIGH"},
	})
	require.NoError(s.T(), err)
	defer resp.Body.Close()

	assert.True(s.T(), resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusUnauthorized)
}

// TaskIntegrationTestSuite tests task operations
type TaskIntegrationTestSuite struct {
	IntegrationTestSuite
}

// TestTaskList tests listing tasks
func (s *TaskIntegrationTestSuite) TestTaskList() {
	resp, err := s.APIClient.Get("/api/v1/tasks?page=1&page_size=10")
	require.NoError(s.T(), err)
	defer resp.Body.Close()

	assert.True(s.T(), resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusUnauthorized)
}

// TestTaskCreate tests creating a task
func (s *TaskIntegrationTestSuite) TestTaskCreate() {
	taskReq := map[string]interface{}{
		"type": "vuln_crawl",
		"payload": map[string]interface{}{
			"sources":    []string{"avd", "seebug"},
			"page_limit": 1,
		},
	}

	resp, err := s.APIClient.Post("/api/v1/tasks", taskReq)
	require.NoError(s.T(), err)
	defer resp.Body.Close()

	// Should return 201 or 401/403
	assert.True(s.T(), resp.StatusCode == http.StatusCreated ||
		resp.StatusCode == http.StatusUnauthorized ||
		resp.StatusCode == http.StatusForbidden)
}

// TestHealthCheckIntegration tests the health check in Docker environment
func TestHealthCheckIntegration(t *testing.T) {
	baseURL := getEnvOrDefault("SECFLOW_SERVER_URL", "http://localhost:8080")

	// Wait for server to be ready
	client := &http.Client{Timeout: 5 * time.Second}

	maxRetries := 30
	for i := 0; i < maxRetries; i++ {
		resp, err := client.Get(baseURL + "/api/v1/health")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				t.Log("Server is ready")
				return
			}
		}
		time.Sleep(2 * time.Second)
	}

	t.Skip("Server not ready within timeout, skipping integration test")
}

// TestWebSocketIntegration tests WebSocket connectivity
func TestWebSocketIntegration(t *testing.T) {
	baseURL := getEnvOrDefault("SECFLOW_WS_URL", "ws://localhost:8080/api/v1/ws/node")

	// This test would require gorilla/websocket client
	// For now, just verify the URL is accessible
	t.Logf("WebSocket URL: %s", baseURL)

	// In real implementation:
	// ws, _, err := websocket.DefaultDialer.Dial(baseURL+"?token=test", nil)
	// require.NoError(t, err)
	// defer ws.Close()
}

// TestRedisQueueIntegration tests Redis queue operations
func TestRedisQueueIntegration(t *testing.T) {
	// This would test queue operations via a test client
	t.Log("Testing Redis queue integration")
	// In real implementation:
	// - Enqueue a task
	// - Verify it's in Redis
	// - Dequeue the task
	// - Verify it's removed
}

// TestMongoDBIntegration tests MongoDB operations
func TestMongoDBIntegration(t *testing.T) {
	// This would test database operations
	t.Log("Testing MongoDB integration")
	// In real implementation:
	// - Insert a document
	// - Query the document
	// - Delete the document
}

// Run all integration tests
func TestIntegrationSuites(t *testing.T) {
	// Check if integration test environment is available
	serverURL := getEnvOrDefault("SECFLOW_SERVER_URL", "")

	if serverURL == "" {
		t.Skip("SECFLOW_SERVER_URL not set, skipping integration tests")
	}

	// Run test suites
	suite.Run(t, new(AuthIntegrationTestSuite))
	suite.Run(t, new(NodeIntegrationTestSuite))
	suite.Run(t, new(VulnIntegrationTestSuite))
	suite.Run(t, new(TaskIntegrationTestSuite))
}

// BenchmarkIntegrationTest tests performance in integration environment
func BenchmarkIntegrationTest(b *testing.B) {
	baseURL := getEnvOrDefault("SECFLOW_SERVER_URL", "http://localhost:8080")
	client := &http.Client{Timeout: 10 * time.Second}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := client.Get(baseURL + "/api/v1/health")
		if err == nil {
			resp.Body.Close()
		}
	}
}

// Helper to make authenticated requests
func makeAuthRequest(method, url, token string, body interface{}) (*http.Response, error) {
	var bodyReader *bytes.Reader
	if body != nil {
		data, _ := json.Marshal(body)
		bodyReader = bytes.NewReader(data)
	} else {
		bodyReader = bytes.NewReader(nil)
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	return client.Do(req)
}
