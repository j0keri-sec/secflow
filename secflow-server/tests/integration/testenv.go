package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// TestEnvironment holds the test server and dependencies
type TestEnvironment struct {
	Server *httptest.Server
	DB     *TestDB
	Queue  *TestQueue
}

// TestDB is a wrapper for test database operations
type TestDB struct {
	URI        string
	Database   string
	CleanupFn func()
}

// TestQueue is a wrapper for test queue operations
type TestQueue struct {
	URL       string
	RedisAddr string
}

// SetupTestEnv creates a test environment for integration tests
func SetupTestEnv() (*TestEnvironment, error) {
	gin.SetMode(gin.TestMode)

	env := &TestEnvironment{
		DB: &TestDB{
			URI:      getEnvOrDefault("SECFLOW_MONGO_URI", "mongodb://localhost:27017/secflow_test"),
			Database: "secflow_test",
		},
		Queue: &TestQueue{
			URL:       getEnvOrDefault("SECFLOW_REDIS_URL", "redis://localhost:6379/0"),
			RedisAddr: getEnvOrDefault("REDIS_ADDR", "localhost:6379"),
		},
	}

	return env, nil
}

// Cleanup cleans up the test environment
func (e *TestEnvironment) Cleanup() {
	if e.Server != nil {
		e.Server.Close()
	}
	if e.DB != nil && e.DB.CleanupFn != nil {
		e.DB.CleanupFn()
	}
}

// WaitForServices waits for dependent services to be ready
func WaitForServices(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Check MongoDB
	mongoOK := make(chan error, 1)
	go func() {
		// In real implementation, would ping MongoDB
		time.Sleep(100 * time.Millisecond)
		mongoOK <- nil
	}()

	// Check Redis
	redisOK := make(chan error, 1)
	go func() {
		// In real implementation, would ping Redis
		time.Sleep(100 * time.Millisecond)
		redisOK <- nil
	}()

	select {
	case <-ctx.Done():
		return fmt.Errorf("timeout waiting for services")
	case err := <-mongoOK:
		if err != nil {
			return fmt.Errorf("mongodb not ready: %w", err)
		}
	case err := <-redisOK:
		if err != nil {
			return fmt.Errorf("redis not ready: %w", err)
		}
	}

	return nil
}

// TestAPIClient is a helper for making API requests in tests
type TestAPIClient struct {
	BaseURL    string
	HTTPClient *http.Client
	Token      string
}

// NewTestAPIClient creates a new API test client
func NewTestAPIClient(baseURL string) *TestAPIClient {
	return &TestAPIClient{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// SetToken sets the JWT token for authenticated requests
func (c *TestAPIClient) SetToken(token string) {
	c.Token = token
}

// Get performs a GET request
func (c *TestAPIClient) Get(path string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, c.BaseURL+path, nil)
	if err != nil {
		return nil, err
	}
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	return c.HTTPClient.Do(req)
}

// Post performs a POST request
func (c *TestAPIClient) Post(path string, body interface{}) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodPost, c.BaseURL+path, marshalJSON(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	return c.HTTPClient.Do(req)
}

// Delete performs a DELETE request
func (c *TestAPIClient) Delete(path string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodDelete, c.BaseURL+path, nil)
	if err != nil {
		return nil, err
	}
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	return c.HTTPClient.Do(req)
}

// AssertResponseStatus is a helper to assert HTTP status codes
func AssertResponseStatus(t require.Assertions, resp *http.Response, expected int) {
	t.Equal(expected, resp.StatusCode, "Expected status %d but got %d", expected, resp.StatusCode)
}

// IntegrationTestSuite is a base test suite for integration tests
type IntegrationTestSuite struct {
	suite.Suite
	Env      *TestEnvironment
	APIClient *TestAPIClient
}

// SetupSuite runs once before all tests
func (s *IntegrationTestSuite) SetupSuite() {
	env, err := SetupTestEnv()
	s.Require().NoError(err)
	s.Env = env

	serverURL := getEnvOrDefault("SECFLOW_SERVER_URL", "http://localhost:8080")
	s.APIClient = NewTestAPIClient(serverURL)
}

// TearDownSuite runs once after all tests
func (s *IntegrationTestSuite) TearDownSuite() {
	if s.Env != nil {
		s.Env.Cleanup()
	}
}

// getEnvOrDefault returns environment variable or default
func getEnvOrDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

// marshalJSON marshals body to JSON and returns a reader
func marshalJSON(body interface{}) *bytes.Reader {
	if body == nil {
		return bytes.NewReader(nil)
	}
	data, err := json.Marshal(body)
	if err != nil {
		return bytes.NewReader(nil)
	}
	return bytes.NewReader(data)
}
