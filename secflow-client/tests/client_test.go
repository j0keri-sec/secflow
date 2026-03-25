package tests

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/secflow/client/internal/db"
)

func TestDBOpen(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	database, err := db.Open(dbPath)
	require.NoError(t, err)
	require.NotNil(t, database)

	err = database.Close()
	require.NoError(t, err)
}

func TestDBInit(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	database, err := db.Open(dbPath)
	require.NoError(t, err)
	defer database.Close()

	// Test inserting vuln cache
	err = database.InsertVulnCache("CVE-2024-0001", "Test Vuln", "HIGH", "CVE-2024-0001", "test", "task-001")
	require.NoError(t, err)

	// Test inserting task record
	task := &db.TaskRecord{
		ID:         "task-001",
		TaskID:     "task-001",
		Type:       "vuln_crawl",
		Status:     "running",
		ReceivedAt: database.Now(),
		UpdatedAt:  database.Now(),
	}
	err = database.UpsertTask(task)
	require.NoError(t, err)

	// Test updating task status
	err = database.UpdateTaskStatus("task-001", "done", 100, "")
	require.NoError(t, err)
}

func TestDBTaskLifecycle(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	database, err := db.Open(dbPath)
	require.NoError(t, err)
	defer database.Close()

	taskID := "task-lifecycle-test"

	// Create task
	task := &db.TaskRecord{
		ID:         taskID,
		TaskID:     taskID,
		Type:       "vuln_crawl",
		Status:     "pending",
		ReceivedAt: database.Now(),
		UpdatedAt:  database.Now(),
	}
	err = database.UpsertTask(task)
	require.NoError(t, err)

	// Update to running
	err = database.UpdateTaskStatus(taskID, "running", 0, "")
	require.NoError(t, err)

	// Update progress
	err = database.UpdateTaskStatus(taskID, "running", 50, "")
	require.NoError(t, err)

	// Update to done
	err = database.UpdateTaskStatus(taskID, "done", 100, "")
	require.NoError(t, err)
}

func TestDBVulnCache(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	database, err := db.Open(dbPath)
	require.NoError(t, err)
	defer database.Close()

	// Insert multiple vulns
	vulns := []struct {
		key      string
		title    string
		severity string
		cve      string
		source   string
		taskID   string
	}{
		{"CVE-2024-0001", "Vuln 1", "HIGH", "CVE-2024-0001", "avd", "task-1"},
		{"CVE-2024-0002", "Vuln 2", "MEDIUM", "CVE-2024-0002", "seebug", "task-1"},
		{"CVE-2024-0003", "Vuln 3", "LOW", "CVE-2024-0003", "nvd", "task-2"},
	}

	for _, v := range vulns {
		err = database.InsertVulnCache(v.key, v.title, v.severity, v.cve, v.source, v.taskID)
		require.NoError(t, err)
	}

	// Verify we can insert more without error (dedup handled by table schema)
	err = database.InsertVulnCache("CVE-2024-0004", "Vuln 4", "CRITICAL", "CVE-2024-0004", "kev", "task-3")
	require.NoError(t, err)
}

func TestDBNonexistentTask(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	database, err := db.Open(dbPath)
	require.NoError(t, err)
	defer database.Close()

	// Update non-existent task should not error
	err = database.UpdateTaskStatus("nonexistent-task", "done", 100, "")
	require.NoError(t, err)
}

func TestDBFileCreation(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "secflow_client.db")

	// File should not exist before Open
	_, err := os.Stat(dbPath)
	assert.True(t, os.IsNotExist(err))

	database, err := db.Open(dbPath)
	require.NoError(t, err)
	defer database.Close()

	// File should exist after Open
	_, err = os.Stat(dbPath)
	assert.NoError(t, err)
}

func TestDBInvalidPath(t *testing.T) {
	// Try to open database in non-existent directory
	_, err := db.Open("/nonexistent/path/test.db")
	assert.Error(t, err)
}

func TestDBConcurrentAccess(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	database, err := db.Open(dbPath)
	require.NoError(t, err)
	defer database.Close()

	// Run concurrent inserts
	done := make(chan bool)
	for i := 0; i < 5; i++ {
		go func(id int) {
			for j := 0; j < 10; j++ {
				taskID := "task-concurrent"
				task := &db.TaskRecord{
					ID:         taskID,
					TaskID:     taskID,
					Type:       "vuln_crawl",
					Status:     "running",
					ReceivedAt: database.Now(),
					UpdatedAt:  database.Now(),
				}
				_ = database.UpsertTask(task)
				_ = database.UpdateTaskStatus(taskID, "running", id*10+j, "")
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 5; i++ {
		<-done
	}
}
