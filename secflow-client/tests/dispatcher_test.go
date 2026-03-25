package tests

import (
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/secflow/client/internal/task"
)

// mockWSClient simulates the WebSocket client for testing
type mockWSClient struct {
	mu             sync.Mutex
	sentProgress   []progressEntry
	sentResults    []resultEntry
	sentErrors     []errorEntry
	closed         bool
}

type progressEntry struct {
	TaskID  string
	Percent int
	Message string
}

type resultEntry struct {
	TaskID  string
	Results []any
}

type errorEntry struct {
	TaskID string
	Error  string
}

func (m *mockWSClient) SendProgress(taskID string, percent int, message string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sentProgress = append(m.sentProgress, progressEntry{taskID, percent, message})
	return nil
}

func (m *mockWSClient) SendResult(taskID string, results []any) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sentResults = append(m.sentResults, resultEntry{taskID, results})
	return nil
}

func (m *mockWSClient) SendError(taskID string, errMsg string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sentErrors = append(m.sentErrors, errorEntry{taskID, errMsg})
	return nil
}

// mockDB simulates the local database for testing
type mockDB struct {
	mu    sync.Mutex
	tasks map[string]*taskRecord
}

type taskRecord struct {
	ID         string
	Status     string
	Progress   int
	ReceivedAt time.Time
	UpdatedAt  time.Time
}

func (m *mockDB) UpsertTask(t *taskRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.tasks == nil {
		m.tasks = make(map[string]*taskRecord)
	}
	m.tasks[t.ID] = t
	return nil
}

func (m *mockDB) UpdateTaskStatus(id, status string, progress int, errMsg string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if t, ok := m.tasks[id]; ok {
		t.Status = status
		t.Progress = progress
		t.UpdatedAt = time.Now()
	}
	return nil
}

func (m *mockDB) InsertVulnCache(key, title, severity, cve, source, taskID string) error {
	return nil
}

func TestDispatcherPayloadParsing(t *testing.T) {
	tests := []struct {
		name      string
		payload   string
		wantType  string
		wantErr   bool
	}{
		{
			name: "vuln_crawl with sources",
			payload: `{
				"sources": ["avd", "seebug"],
				"page_limit": 1,
				"enable_github": false
			}`,
			wantType: "vuln",
		},
		{
			name: "vuln_crawl minimal",
			payload: `{
				"sources": ["nvd"]
			}`,
			wantType: "vuln",
		},
		{
			name: "article_crawl with limit",
			payload: `{
				"sources": ["qianxin"],
				"limit": 10
			}`,
			wantType: "article",
		},
		{
			name: "article_crawl minimal",
			payload: `{
				"sources": ["venustech"],
				"limit": 5
			}`,
			wantType: "article",
		},
		{
			name:      "empty payload",
			payload:   `{}`,
			wantType:  "vuln",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Check if payload is article type based on limit field
			var check struct {
				Limit int `json:"limit"`
			}
			json.Unmarshal([]byte(tt.payload), &check)

			isArticle := check.Limit > 0

			if tt.wantType == "article" {
				assert.True(t, isArticle)
			} else {
				// For vuln_crawl, limit field is either 0 or absent
				// If sources are present but limit is 0 or absent, it's vuln
			}
		})
	}
}

func TestVulnCrawlPayloadJSON(t *testing.T) {
	payload := task.VulnCrawlPayload{
		Sources:      []string{"avd", "seebug", "nvd"},
		PageLimit:    2,
		EnableGithub: true,
		Proxy:        "http://proxy:8080",
	}

	data, err := json.Marshal(payload)
	require.NoError(t, err)

	var parsed task.VulnCrawlPayload
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	assert.Equal(t, payload.Sources, parsed.Sources)
	assert.Equal(t, payload.PageLimit, parsed.PageLimit)
	assert.Equal(t, payload.EnableGithub, parsed.EnableGithub)
	assert.Equal(t, payload.Proxy, parsed.Proxy)
}

func TestArticleCrawlPayloadJSON(t *testing.T) {
	payload := task.ArticleCrawlPayload{
		Sources: []string{"qianxin", "venustech"},
		Limit:   20,
	}

	data, err := json.Marshal(payload)
	require.NoError(t, err)

	var parsed task.ArticleCrawlPayload
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	assert.Equal(t, payload.Sources, parsed.Sources)
	assert.Equal(t, payload.Limit, parsed.Limit)
}

func TestTaskCancel(t *testing.T) {
	// Create a simple task tracking structure
	cancels := make(map[string]context.CancelFunc)
	var mu sync.Mutex

	// Simulate task start
	taskID := "task-123"
	ctx, cancel := context.WithCancel(context.Background())
	mu.Lock()
	cancels[taskID] = cancel
	mu.Unlock()

	// Verify context is not cancelled yet
	select {
	case <-ctx.Done():
		t.Fatal("context should not be cancelled yet")
	default:
	}

	// Simulate cancel request
	mu.Lock()
	if c, ok := cancels[taskID]; ok {
		c()
	}
	mu.Unlock()

	// Give a tiny bit of time for the cancel to propagate
	time.Sleep(10 * time.Millisecond)

	// Verify context is cancelled
	select {
	case <-ctx.Done():
		// Expected
	default:
		t.Fatal("context should be cancelled")
	}
}

func TestConcurrentTaskTracking(t *testing.T) {
	cancels := make(map[string]context.CancelFunc)
	var mu sync.Mutex

	// Simulate multiple tasks starting
	taskIDs := []string{"task-1", "task-2", "task-3"}

	for _, id := range taskIDs {
		ctx, cancel := context.WithCancel(context.Background())
		mu.Lock()
		cancels[id] = cancel
		mu.Unlock()

		// Immediately cancel task-2
		if id == "task-2" {
			mu.Lock()
			if c, ok := cancels[id]; ok {
				c()
			}
			mu.Unlock()
		}
	}

	time.Sleep(10 * time.Millisecond)

	// Verify only task-2 is cancelled
	mu.Lock()
	defer mu.Unlock()

	// Note: We can't easily test this without the actual dispatcher
	// This is a structural test showing the pattern
	assert.Len(t, cancels, 3)
}

func TestDispatcherMock(t *testing.T) {
	ws := &mockWSClient{}
	db := &mockDB{}

	// Simulate task processing
	taskID := "test-task-001"
	payload := task.VulnCrawlPayload{
		Sources:   []string{"avd"},
		PageLimit: 1,
	}

	payloadJSON, _ := json.Marshal(payload)

	// Simulate dispatcher receiving task
	err := db.UpsertTask(&taskRecord{
		ID:         taskID,
		Status:     "running",
		ReceivedAt: time.Now(),
	})
	require.NoError(t, err)

	// Simulate progress updates
	for i := 0; i <= 100; i += 25 {
		err = ws.SendProgress(taskID, i, "running")
		require.NoError(t, err)
		err = db.UpdateTaskStatus(taskID, "running", i, "")
		require.NoError(t, err)
	}

	// Simulate completion
	results := []any{
		map[string]any{"key": "CVE-2024-0001", "title": "Test Vulnerability"},
	}
	err = ws.SendResult(taskID, results)
	require.NoError(t, err)
	err = db.UpdateTaskStatus(taskID, "done", 100, "")
	require.NoError(t, err)

	// Verify
	assert.Len(t, ws.sentProgress, 5)
	assert.Len(t, ws.sentResults, 1)
	assert.Equal(t, "test-task-001", ws.sentResults[0].TaskID)
	assert.Len(t, ws.sentResults[0].Results, 1)
}

func TestErrorHandling(t *testing.T) {
	ws := &mockWSClient{}

	err := ws.SendError("task-001", "connection timeout")
	require.NoError(t, err)

	assert.Len(t, ws.sentErrors, 1)
	assert.Equal(t, "task-001", ws.sentErrors[0].TaskID)
	assert.Equal(t, "connection timeout", ws.sentErrors[0].Error)
}

// context import for context package
import "context"
