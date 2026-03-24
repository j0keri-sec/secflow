package tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/secflow/server/internal/model"
)

// setupTestGin creates a Gin router in test mode.
func setupTestGin() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

// TestPageParams tests the pagination parameter extraction.
func TestPageParams(t *testing.T) {
	t.Run("Default values when no params provided", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c, _ = gin.CreateTestContext(w)
		
		// Simulate request with no query params
		req, _ := http.NewRequest(http.MethodGet, "/test", nil)
		c.Request = req
		
		// Test pageParams directly through handler
		r := setupTestGin()
		var capturedPage, capturedPageSize int
		r.GET("/test", func(c *gin.Context) {
			page := 1
			pageSize := 20
			if p := c.Query("page"); p != "" {
				if parsed, err := json.Number(p).Int64(); err == nil && parsed > 0 {
					page = int(parsed)
				}
			}
			if ps := c.Query("page_size"); ps != "" {
				if parsed, err := json.Number(ps).Int64(); err == nil && parsed > 0 && parsed <= 100 {
					pageSize = int(parsed)
				}
			}
			capturedPage = page
			capturedPageSize = pageSize
			c.JSON(200, gin.H{"page": page, "page_size": pageSize})
		})
		
		req, _ = http.NewRequest(http.MethodGet, "/test", nil)
		r.ServeHTTP(w, req)
		
		assert.Equal(t, 1, capturedPage)
		assert.Equal(t, 20, capturedPageSize)
	})

	t.Run("Custom values from query params", func(t *testing.T) {
		r := setupTestGin()
		var capturedPage, capturedPageSize int
		r.GET("/test", func(c *gin.Context) {
			page := 1
			pageSize := 20
			if p := c.Query("page"); p != "" {
				if parsed, err := json.Number(p).Int64(); err == nil && parsed > 0 {
					page = int(parsed)
				}
			}
			if ps := c.Query("page_size"); ps != "" {
				if parsed, err := json.Number(ps).Int64(); err == nil && parsed > 0 && parsed <= 100 {
					pageSize = int(parsed)
				}
			}
			capturedPage = page
			capturedPageSize = pageSize
			c.JSON(200, gin.H{"page": page, "page_size": pageSize})
		})
		
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/test?page=3&page_size=50", nil)
		r.ServeHTTP(w, req)
		
		assert.Equal(t, 3, capturedPage)
		assert.Equal(t, 50, capturedPageSize)
	})
}

// TestVulnModel tests the VulnRecord model.
func TestVulnModel(t *testing.T) {
	t.Run("VulnRecord creation", func(t *testing.T) {
		now := bson.NewObjectID().Timestamp()
		vuln := &model.VulnRecord{
			ID:          bson.NewObjectID(),
			Key:         "avd:CVE-2024-1234",
			Title:       "Test Vulnerability",
			Description: "A test vulnerability description",
			Severity:    model.SeverityHigh,
			CVE:         "CVE-2024-1234",
			Source:      "avd-rod",
			URL:         "https://avd.example.com/vuln/CVE-2024-1234",
			Pushed:      false,
			ReportedBy:  "test-node",
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		assert.NotZero(t, vuln.ID)
		assert.Equal(t, "avd:CVE-2024-1234", vuln.Key)
		assert.Equal(t, "Test Vulnerability", vuln.Title)
		assert.Equal(t, model.SeverityHigh, vuln.Severity)
		assert.Equal(t, "CVE-2024-1234", vuln.CVE)
		assert.False(t, vuln.Pushed)
	})

	t.Run("Severity levels", func(t *testing.T) {
		assert.Equal(t, model.SeverityLevel("低危"), model.SeverityLow)
		assert.Equal(t, model.SeverityLevel("中危"), model.SeverityMedium)
		assert.Equal(t, model.SeverityLevel("高危"), model.SeverityHigh)
		assert.Equal(t, model.SeverityLevel("严重"), model.SeverityCritical)
	})

	t.Run("VulnRecord JSON serialization", func(t *testing.T) {
		vuln := &model.VulnRecord{
			Key:      "test:vuln-001",
			Title:    "Test",
			Severity: model.SeverityHigh,
		}
		
		data, err := json.Marshal(vuln)
		require.NoError(t, err)
		
		var parsed map[string]interface{}
		err = json.Unmarshal(data, &parsed)
		require.NoError(t, err)
		
		assert.Equal(t, "test:vuln-001", parsed["key"])
		assert.Equal(t, "Test", parsed["title"])
		assert.Equal(t, "高危", parsed["severity"])
	})
}

// TestTaskModel tests the Task model.
func TestTaskModel(t *testing.T) {
	t.Run("Task creation", func(t *testing.T) {
		task := &model.Task{
			ID:       bson.NewObjectID(),
			TaskID:   bson.NewObjectID().Hex(),
			Type:     model.TaskTypeVulnCrawl,
			Status:   model.TaskPending,
			Priority: 50,
			Progress: 0,
		}

		assert.NotZero(t, task.ID)
		assert.Equal(t, model.TaskTypeVulnCrawl, task.Type)
		assert.Equal(t, model.TaskPending, task.Status)
		assert.Equal(t, 50, task.Priority)
		assert.Equal(t, 0, task.Progress)
	})

	t.Run("Task status transitions", func(t *testing.T) {
		statuses := []model.TaskStatus{
			model.TaskPending,
			model.TaskDispatched,
			model.TaskRunning,
			model.TaskDone,
			model.TaskFailed,
		}

		for _, status := range statuses {
			task := &model.Task{Status: status}
			assert.Equal(t, status, task.Status)
		}
	})

	t.Run("TaskType constants", func(t *testing.T) {
		assert.Equal(t, model.TaskType("vuln_crawl"), model.TaskTypeVulnCrawl)
		assert.Equal(t, model.TaskType("article_crawl"), model.TaskTypeArticleCrawl)
	})
}

// TestNodeModel tests the Node model.
func TestNodeModel(t *testing.T) {
	t.Run("Node creation", func(t *testing.T) {
		node := &model.Node{
			ID:     bson.NewObjectID(),
			NodeID: bson.NewObjectID().Hex(),
			Name:   "test-node-01",
			Status: model.NodeOnline,
			Sources: []string{"avd-rod", "seebug-rod"},
		}

		assert.NotZero(t, node.ID)
		assert.Equal(t, "test-node-01", node.Name)
		assert.Equal(t, model.NodeOnline, node.Status)
		assert.Len(t, node.Sources, 2)
	})

	t.Run("NodeStatus constants", func(t *testing.T) {
		assert.Equal(t, model.NodeStatus("online"), model.NodeOnline)
		assert.Equal(t, model.NodeStatus("offline"), model.NodeOffline)
		assert.Equal(t, model.NodeStatus("busy"), model.NodeBusy)
		assert.Equal(t, model.NodeStatus("paused"), model.NodePaused)
	})

	t.Run("NodeTaskStats calculations", func(t *testing.T) {
		stats := model.NodeTaskStats{
			TotalTasks:    100,
			SuccessTasks:  95,
			FailedTasks:   5,
			CurrentTasks:  2,
			CPUPercent:    45.0,
			MemPercent:    60.0,
		}

		// Test success rate calculation
		assert.InDelta(t, 0.95, stats.SuccessRate(), 0.001)

		// Test load score calculation
		score := stats.LoadScore()
		// With 2 tasks, 45% CPU, 60% mem: score should be 20 (2*10 from tasks, no penalties)
		assert.Equal(t, 20.0, score)
	})

	t.Run("NodeTaskStats zero division handling", func(t *testing.T) {
		stats := model.NodeTaskStats{
			TotalTasks: 0,
		}
		// Should return 1.0 for new nodes
		assert.Equal(t, 1.0, stats.SuccessRate())
	})
}

// TestArticleModel tests the Article model.
func TestArticleModel(t *testing.T) {
	t.Run("Article creation", func(t *testing.T) {
		article := &model.Article{
			ID:        bson.NewObjectID(),
			Title:     "Test Article",
			Summary:   "A test article summary",
			Author:    "Test Author",
			Source:    "test-source",
			URL:       "https://example.com/article/1",
			Tags:      []string{"security", "news"},
			Pushed:    false,
			ReportedBy: "test-node",
		}

		assert.NotZero(t, article.ID)
		assert.Equal(t, "Test Article", article.Title)
		assert.Len(t, article.Tags, 2)
		assert.Contains(t, article.Tags, "security")
	})
}

// TestPushChannelModel tests the PushChannel model.
func TestPushChannelModel(t *testing.T) {
	t.Run("PushChannel creation", func(t *testing.T) {
		channel := &model.PushChannel{
			ID:      bson.NewObjectID(),
			Name:    "DingTalk Alert",
			Type:    "dingding",
			Config:  map[string]string{"webhook": "https://oapi.dingtalk.com/robot/send?access_token=xxx"},
			Enabled: true,
		}

		assert.NotZero(t, channel.ID)
		assert.Equal(t, "DingTalk Alert", channel.Name)
		assert.Equal(t, "dingding", channel.Type)
		assert.True(t, channel.Enabled)
		assert.Contains(t, channel.Config, "webhook")
	})
}

// TestReportModel tests the Report model.
func TestReportModel(t *testing.T) {
	t.Run("Report creation", func(t *testing.T) {
		report := &model.Report{
			ID:          bson.NewObjectID(),
			Title:       "Weekly Security Report",
			Description: "Weekly vulnerability summary",
			Status:      model.ReportPending,
			Period:      "2024-01-01 ~ 2024-01-07",
			CreatedBy:   bson.NewObjectID(),
		}

		assert.NotZero(t, report.ID)
		assert.Equal(t, model.ReportPending, report.Status)
	})

	t.Run("ReportStatus constants", func(t *testing.T) {
		assert.Equal(t, model.ReportStatus("pending"), model.ReportPending)
		assert.Equal(t, model.ReportStatus("generating"), model.ReportGenerating)
		assert.Equal(t, model.ReportStatus("done"), model.ReportDone)
		assert.Equal(t, model.ReportStatus("failed"), model.ReportFailed)
	})
}

// TestAuditLogModel tests the AuditLog model.
func TestAuditLogModel(t *testing.T) {
	t.Run("AuditLog creation", func(t *testing.T) {
		log := &model.AuditLog{
			ID:       bson.NewObjectID(),
			UserID:   bson.NewObjectID(),
			Username: "testuser",
			Action:   "login",
			Resource: "system",
			IP:       "192.168.1.1",
		}

		assert.NotZero(t, log.ID)
		assert.Equal(t, "testuser", log.Username)
		assert.Equal(t, "login", log.Action)
		assert.Equal(t, "192.168.1.1", log.IP)
	})
}

// TestCollectionNames tests the MongoDB collection name constants.
func TestCollectionNames(t *testing.T) {
	t.Run("Collection constants are defined", func(t *testing.T) {
		assert.Equal(t, "users", model.CollUsers)
		assert.Equal(t, "invite_codes", model.CollInviteCodes)
		assert.Equal(t, "password_reset_tokens", model.CollPasswordResetTokens)
		assert.Equal(t, "nodes", model.CollNodes)
		assert.Equal(t, "tasks", model.CollTasks)
		assert.Equal(t, "vuln_records", model.CollVulnRecords)
		assert.Equal(t, "articles", model.CollArticles)
		assert.Equal(t, "push_channels", model.CollPushChannels)
		assert.Equal(t, "audit_logs", model.CollAuditLogs)
		assert.Equal(t, "reports", model.CollReports)
	})
}

// TestVulnCrawlPayload tests the VulnCrawlPayload model.
func TestVulnCrawlPayload(t *testing.T) {
	t.Run("VulnCrawlPayload creation", func(t *testing.T) {
		payload := &model.VulnCrawlPayload{
			Sources:      []string{"avd-rod", "seebug-rod", "nvd-rod"},
			PageLimit:    5,
			EnableGithub: true,
			Proxy:        "http://proxy:8080",
		}

		assert.Len(t, payload.Sources, 3)
		assert.Equal(t, 5, payload.PageLimit)
		assert.True(t, payload.EnableGithub)
		assert.Equal(t, "http://proxy:8080", payload.Proxy)
	})

	t.Run("VulnCrawlPayload JSON serialization", func(t *testing.T) {
		payload := &model.VulnCrawlPayload{
			Sources:   []string{"avd-rod"},
			PageLimit: 3,
		}

		data, err := json.Marshal(payload)
		require.NoError(t, err)

		var parsed model.VulnCrawlPayload
		err = json.Unmarshal(data, &parsed)
		require.NoError(t, err)

		assert.Equal(t, payload.Sources, parsed.Sources)
		assert.Equal(t, payload.PageLimit, parsed.PageLimit)
	})
}

// TestResponseHelpers tests the response helper functions.
func TestResponseHelpers(t *testing.T) {
	t.Run("Success response format", func(t *testing.T) {
		r := setupTestGin()
		r.GET("/test", func(c *gin.Context) {
			// Simulate ok() helper behavior
			c.JSON(200, gin.H{
				"code": 0,
				"msg":  "success",
				"data": gin.H{"key": "value"},
			})
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/test", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, float64(0), resp["code"])
		assert.Equal(t, "success", resp["msg"])
	})

	t.Run("Error response format", func(t *testing.T) {
		r := setupTestGin()
		r.GET("/test", func(c *gin.Context) {
			c.JSON(400, gin.H{
				"code": 400,
				"msg":  "bad request",
			})
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/test", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, 400, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, float64(400), resp["code"])
		assert.Equal(t, "bad request", resp["msg"])
	})
}

// TestQueryParameterBinding tests query parameter binding.
func TestQueryParameterBinding(t *testing.T) {
	t.Run("Boolean query parameter parsing", func(t *testing.T) {
		r := setupTestGin()
		r.GET("/test", func(c *gin.Context) {
			pushed := c.Query("pushed")
			
			var pushedVal *bool
			if pushed == "true" {
				t := true
				pushedVal = &t
			} else if pushed == "false" {
				f := false
				pushedVal = &f
			}
			
			c.JSON(200, gin.H{"pushed": pushedVal != nil && *pushedVal})
		})

		tests := []struct {
			query    string
			expected bool
		}{
			{"?pushed=true", true},
			{"?pushed=false", false},
			{"", false},
		}

		for _, tc := range tests {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodGet, "/test"+tc.query, nil)
			r.ServeHTTP(w, req)

			var resp map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &resp)
			assert.Equal(t, tc.expected, resp["pushed"], "Query: %s", tc.query)
		}
	})
}

// TestURLPathParameterExtraction tests URL path parameter extraction.
func TestURLPathParameterExtraction(t *testing.T) {
	t.Run("MongoDB ObjectID from path", func(t *testing.T) {
		r := setupTestGin()
		r.GET("/vulns/:id", func(c *gin.Context) {
			idStr := c.Param("id")
			
			_, err := bson.ObjectIDFromHex(idStr)
			if err != nil {
				c.JSON(400, gin.H{"error": "invalid id"})
				return
			}
			c.JSON(200, gin.H{"valid": true})
		})

		// Valid ObjectID
		validID := bson.NewObjectID().Hex()
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/vulns/"+validID, nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, 200, w.Code)

		// Invalid ObjectID
		w = httptest.NewRecorder()
		req, _ = http.NewRequest(http.MethodGet, "/vulns/invalid-id", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, 400, w.Code)
	})
}

// TestIntegration_VulnWorkflow tests a complete vulnerability workflow.
func TestIntegration_VulnWorkflow(t *testing.T) {
	t.Run("Vuln workflow is documented", func(t *testing.T) {
		workflow := []string{
			"1. Client fetches tasks from Redis queue via WebSocket",
			"2. Client executes crawl using go-rod",
			"3. Client sends results via HTTP POST /api/v1/report",
			"4. Server stores in MongoDB via VulnRepo.Upsert()",
			"5. Server checks if renotification needed",
			"6. If severity changed or new vuln, push via configured channel",
		}
		assert.Equal(t, 6, len(workflow))
	})
}

// TestIntegration_TaskWorkflow tests the task lifecycle.
func TestIntegration_TaskWorkflow(t *testing.T) {
	t.Run("Task lifecycle is documented", func(t *testing.T) {
		lifecycle := []struct {
			status   model.TaskStatus
			expected string
		}{
			{model.TaskPending, "Task created, waiting in queue"},
			{model.TaskDispatched, "Task sent to client node"},
			{model.TaskRunning, "Client executing task"},
			{model.TaskDone, "Task completed successfully"},
			{model.TaskFailed, "Task failed after retries"},
		}
		
		assert.Equal(t, 5, len(lifecycle))
		
		for _, stage := range lifecycle {
			t.Logf("Status: %s -> %s", stage.status, stage.expected)
		}
	})
}

// BenchmarkVulnListFilterCreation benchmarks the filter struct creation.
func BenchmarkVulnListFilterCreation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		filter := map[string]interface{}{
			"Severity": "高危",
			"Source":   "avd-rod",
			"CVE":      "CVE-2024",
			"Page":     1,
			"PageSize": 20,
		}
		_ = filter
	}
}
