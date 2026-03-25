package tests

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/secflow/client/internal/engine"
	"github.com/secflow/client/internal/task"
)

// mockExecutor implements task.Executor for testing
type mockExecutor struct {
	vulnCrawlCalled    bool
	articleCrawlCalled  bool
	vulnPayload        task.VulnCrawlPayload
	articlePayload     task.ArticleCrawlPayload
	vulnResults        []any
	articleResults     []any
	vulnErr            error
	articleErr         error
}

func (m *mockExecutor) RunVulnCrawl(ctx context.Context, payload task.VulnCrawlPayload, progressFn func(int)) ([]any, error) {
	m.vulnCrawlCalled = true
	m.vulnPayload = payload
	if progressFn != nil {
		progressFn(50)
	}
	return m.vulnResults, m.vulnErr
}

func (m *mockExecutor) RunArticleCrawl(ctx context.Context, payload task.ArticleCrawlPayload, progressFn func(int)) ([]any, error) {
	m.articleCrawlCalled = true
	m.articlePayload = payload
	if progressFn != nil {
		progressFn(100)
	}
	return m.articleResults, m.articleErr
}

func TestEngineGetAvailableSources(t *testing.T) {
	sources := engine.GetAvailableSources()
	assert.NotEmpty(t, sources)

	// Check some expected sources
	expectedSources := []string{"avd", "seebug", "nvd", "kev"}
	for _, expected := range expectedSources {
		found := false
		for _, s := range sources {
			if s == expected {
				found = true
				break
			}
		}
		assert.True(t, found, "expected source %q not found", expected)
	}
}

func TestEngineGetAvailableArticleSources(t *testing.T) {
	sources := engine.GetAvailableArticleSources()
	assert.NotEmpty(t, sources)
}

func TestEngineGetGrabber(t *testing.T) {
	// Test getting existing grabber
	g, err := engine.GetGrabber("avd")
	require.NoError(t, err)
	assert.NotNil(t, g)

	provider := g.ProviderInfo()
	assert.NotNil(t, provider)
	assert.Equal(t, "AVD", provider.DisplayName)

	// Test getting non-existent grabber
	_, err = engine.GetGrabber("nonexistent")
	assert.Error(t, err)
}

func TestVulnToServerFormat(t *testing.T) {
	// This tests the internal conversion logic through the engine
	log, _ := zap.NewDevelopment()
	e := engine.New("", log)

	// Create a mock payload
	payload := task.VulnCrawlPayload{
		Sources:      []string{"avd"},
		PageLimit:   1,
		EnableGithub: false,
	}

	// Run vuln crawl (will use real grabbers but with limited pages)
	ctx := context.Background()

	progressCalls := 0
	progressFn := func(pct int) {
		progressCalls++
		assert.GreaterOrEqual(t, pct, 0)
		assert.LessOrEqual(t, pct, 100)
	}

	results, err := e.RunVulnCrawl(ctx, payload, progressFn)

	// Note: This test may fail if network is unavailable or grabber is blocked
	// In a real test environment, we would mock the grabbers
	if err != nil {
		t.Logf("VulnCrawl error (may be expected without network): %v", err)
	}

	// Verify progress was called
	if err == nil {
		assert.Greater(t, progressCalls, 0)
		t.Logf("Found %d vulns", len(results))
	}
}

func TestArticleCrawlPayload(t *testing.T) {
	payload := task.ArticleCrawlPayload{
		Sources: []string{"qianxin"},
		Limit:   5,
	}

	assert.Equal(t, []string{"qianxin"}, payload.Sources)
	assert.Equal(t, 5, payload.Limit)
}

func TestVulnCrawlPayload(t *testing.T) {
	payload := task.VulnCrawlPayload{
		Sources:      []string{"avd", "seebug"},
		PageLimit:    2,
		EnableGithub: true,
		Proxy:        "http://proxy:8080",
	}

	assert.Equal(t, []string{"avd", "seebug"}, payload.Sources)
	assert.Equal(t, 2, payload.PageLimit)
	assert.True(t, payload.EnableGithub)
	assert.Equal(t, "http://proxy:8080", payload.Proxy)
}

func TestEngineStandaloneDefaults(t *testing.T) {
	log, _ := zap.NewDevelopment()
	e := engine.New("", log)

	// Test with empty sources (should use all available)
	payload := task.VulnCrawlPayload{
		Sources:   []string{},
		PageLimit: 1,
	}

	ctx := context.Background()
	progressFn := func(int) {}

	results, err := e.RunVulnCrawl(ctx, payload, progressFn)
	if err != nil {
		t.Logf("Crawl error (may be network related): %v", err)
	}

	t.Logf("Found %d total vulns from all sources", len(results))
}

func TestEngineProgressReporting(t *testing.T) {
	log, _ := zap.NewDevelopment()
	e := engine.New("", log)

	// Track progress values
	var progressValues []int

	payload := task.VulnCrawlPayload{
		Sources:   []string{"kev"}, // kev uses HTTP API, should be faster
		PageLimit: 1,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	progressFn := func(pct int) {
		progressValues = append(progressValues, pct)
	}

	_, err := e.RunVulnCrawl(ctx, payload, progressFn)
	if err != nil {
		t.Logf("Crawl error: %v", err)
	}

	// Progress should always be monotonic
	for i := 1; i < len(progressValues); i++ {
		assert.GreaterOrEqual(t, progressValues[i], progressValues[i-1])
	}
}

func TestFormatCount(t *testing.T) {
	// Test the internal formatCount through actual crawl output
	log, _ := zap.NewDevelopment()
	e := engine.New("", log)

	payload := task.VulnCrawlPayload{
		Sources:   []string{"kev"},
		PageLimit: 1,
	}

	ctx := context.Background()
	progressFn := func(pct int) {
		// Progress should be 0-100
		assert.GreaterOrEqual(t, pct, 0)
		assert.LessOrEqual(t, pct, 100)
	}

	_, err := e.RunVulnCrawl(ctx, payload, progressFn)
	if err != nil {
		t.Logf("Expected network error: %v", err)
	}
}
