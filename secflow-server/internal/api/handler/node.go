package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/secflow/server/internal/model"
	"github.com/secflow/server/internal/queue"
	"github.com/secflow/server/internal/repository"
	"github.com/secflow/server/internal/ws"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// defaultTimeout is the default timeout for background operations.
const defaultTimeout = 30 * time.Second

// NodeHandler manages client node registration and status.
type NodeHandler struct {
	nodeRepo     *repository.NodeRepo
	taskRepo     *repository.TaskRepo
	vulnRepo     *repository.VulnRepo
	articleRepo  *repository.ArticleRepository
	queue        *queue.Queue
	Hub          *ws.Hub // Exported to allow injection after creation
	nodeTokenKey string  // Shared secret key for node authentication
	scheduler    interface {
		HandleTaskFailure(ctx context.Context, taskID string, nodeID string, errorMsg string) error
	}
}

// NewNodeHandler creates a new NodeHandler without scheduler (for initialization).
func NewNodeHandler(nr *repository.NodeRepo, tr *repository.TaskRepo, vr *repository.VulnRepo, ar *repository.ArticleRepository, q *queue.Queue, h *ws.Hub, tokenKey string) *NodeHandler {
	return &NodeHandler{nodeRepo: nr, taskRepo: tr, vulnRepo: vr, articleRepo: ar, queue: q, Hub: h, nodeTokenKey: tokenKey}
}

// NewNodeHandlerWithScheduler creates a new NodeHandler with scheduler reference.
func NewNodeHandlerWithScheduler(nr *repository.NodeRepo, tr *repository.TaskRepo, vr *repository.VulnRepo, ar *repository.ArticleRepository, q *queue.Queue, h *ws.Hub, tokenKey string, scheduler interface {
	HandleTaskFailure(ctx context.Context, taskID string, nodeID string, errorMsg string) error
}) *NodeHandler {
	return &NodeHandler{
		nodeRepo:     nr,
		taskRepo:     tr,
		vulnRepo:     vr,
		articleRepo:  ar,
		queue:        q,
		Hub:          h,
		nodeTokenKey: tokenKey,
		scheduler:    scheduler,
	}
}

// List returns all registered nodes with their current status.
//
// GET /api/v1/nodes
func (h *NodeHandler) List(c *gin.Context) {
	nodes, err := h.nodeRepo.List(c)
	if err != nil {
		log.Error().Err(err).Msg("failed to list nodes")
		fail(c, http.StatusInternalServerError, "failed to retrieve nodes")
		return
	}
	// Annotate with real-time online status from the WebSocket hub.
	connected := make(map[string]bool)
	for _, id := range h.Hub.ConnectedNodes() {
		connected[id] = true
	}
	type nodeView struct {
		*model.Node
		Online bool `json:"online"`
	}
	views := make([]nodeView, 0, len(nodes))
	for _, n := range nodes {
		views = append(views, nodeView{Node: n, Online: connected[n.NodeID]})
	}
	// Return paginated response format to match frontend expectations
	okPage(c, int64(len(views)), 1, 100, views)
}

// ServeWS is the WebSocket endpoint for node connections.
// The node must pass ?token=<token_key>&node_id=<id>&name=<name> for auth.
// If node doesn't exist, it will be auto-registered.
//
// GET /api/v1/ws/node
func (h *NodeHandler) ServeWS(c *gin.Context) {
	nodeID := c.Query("node_id")
	token := c.Query("token")
	name := c.Query("name")

	if nodeID == "" || token == "" {
		fail(c, http.StatusBadRequest, "node_id and token are required")
		return
	}

	// Validate token against shared token key
	if token != h.nodeTokenKey {
		fail(c, http.StatusUnauthorized, "invalid token key")
		return
	}

	// Auto-register node if not exists
	node, err := h.nodeRepo.GetByNodeID(c, nodeID)
	if err != nil || node == nil {
		// Auto-create node
		if name == "" {
			name = "auto-" + nodeID[:8]
		}
		node = &model.Node{
			NodeID:       nodeID,
			Name:         name,
			Token:        "", // No individual token needed when using shared key
			Status:       model.NodeOffline,
			Sources:      []string{},
			RegisteredAt: time.Now().UTC(),
			LastSeenAt:   time.Now().UTC(),
		}
		if err := h.nodeRepo.Upsert(c, node); err != nil {
			fail(c, http.StatusInternalServerError, "failed to auto-register node")
			return
		}
		log.Info().Str("node_id", nodeID).Str("name", name).Msg("node auto-registered")
	}

	h.Hub.ServeWS(c.Writer, c.Request, nodeID)
}

// onNodeMessage handles inbound WebSocket messages from a node.
func (h *NodeHandler) OnMessage(nodeID string, msg *ws.Message) {
	ctx := context.Background()
	switch msg.Type {
	case ws.MsgTypeRegister:
		// Register message is sent by client after connection, but authentication
		// is already done via URL token parameter. Just acknowledge it.
		log.Debug().Str("node_id", nodeID).Msg("ws: node registered")

	case ws.MsgTypeHeartbeat:
		var payload ws.HeartbeatPayload
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			return
		}
		_ = h.nodeRepo.SetStatus(ctx, nodeID, model.NodeOnline)
		_ = h.queue.Heartbeat(ctx, nodeID)

		// Update node info from heartbeat payload
		if payload.Info != nil {
			if infoMap, ok := payload.Info.(map[string]interface{}); ok {
				h.updateNodeInfoFromHeartbeat(ctx, nodeID, infoMap)
			}
		}

	case ws.MsgTypeProgress:
		var payload ws.ProgressPayload
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			return
		}
		_ = h.queue.SetProgress(ctx, payload.TaskID, payload.Progress)

	case ws.MsgTypeResult:
		var payload ws.ResultPayload
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			return
		}
		h.handleTaskResult(ctx, nodeID, &payload)

	default:
		log.Warn().Str("type", string(msg.Type)).Msg("ws: unknown message type")
	}
}

// OnConnect is called when a node connects.
func (h *NodeHandler) OnConnect(nodeID string) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()
	_ = h.nodeRepo.SetStatus(ctx, nodeID, model.NodeOnline)
	_ = h.queue.Heartbeat(ctx, nodeID)
}

// updateNodeInfoFromHeartbeat updates node info from heartbeat payload.
func (h *NodeHandler) updateNodeInfoFromHeartbeat(ctx context.Context, nodeID string, info map[string]interface{}) {
	node, err := h.nodeRepo.GetByNodeID(ctx, nodeID)
	if err != nil || node == nil {
		return
	}

	// Update string fields from heartbeat using helper
	setFieldIfValid(info, "ip", func(v string) { node.Info.IP = v })
	setFieldIfValid(info, "public_ip", func(v string) { node.Info.PublicIP = v })
	setFieldIfValid(info, "mac", func(v string) { node.Info.MAC = v })
	setFieldIfValid(info, "os", func(v string) { node.Info.OS = v })
	setFieldIfValid(info, "arch", func(v string) { node.Info.Arch = v })
	setFieldIfValid(info, "cpu_model", func(v string) { node.Info.CPUModel = v })

	// Update numeric fields
	if cpuCores, ok := info["cpu_cores"].(float64); ok {
		node.Info.CPUCores = int(cpuCores)
	}
	if cpuPct, ok := info["cpu_pct"].(float64); ok {
		node.Info.CPUPercent = cpuPct
	}
	if memTotal, ok := info["mem_total"].(float64); ok {
		node.Info.MemTotal = uint64(memTotal)
	}
	if memUsed, ok := info["mem_used"].(float64); ok {
		node.Info.MemUsed = uint64(memUsed)
	}
	if diskTotal, ok := info["disk_total"].(float64); ok {
		node.Info.DiskTotal = uint64(diskTotal)
	}
	if diskUsed, ok := info["disk_used"].(float64); ok {
		node.Info.DiskUsed = uint64(diskUsed)
	}

	// Update net_cards array
	if netCards, ok := info["net_cards"].([]interface{}); ok {
		cards := make([]string, 0, len(netCards))
		for _, card := range netCards {
			if s, ok := card.(string); ok {
				cards = append(cards, s)
			}
		}
		node.Info.NetCards = cards
	}

	// Save updated node info
	_ = h.nodeRepo.UpdateInfo(ctx, nodeID, node.Info)

	// Update task stats with current performance metrics
	if node.Info.MemTotal > 0 {
		memPercent := float64(node.Info.MemUsed) / float64(node.Info.MemTotal) * 100.0
		node.TaskStats.MemPercent = memPercent
	}
	node.TaskStats.CPUPercent = node.Info.CPUPercent
	node.TaskStats.CurrentTasks = h.Hub.GetTaskCount(nodeID)
	node.TaskStats.UpdatedAt = time.Now()

	// Update node stats in repository
	_ = h.nodeRepo.UpdateTaskStats(ctx, nodeID, node.TaskStats)
}

// setFieldIfValid extracts a string value from info map and calls setter if valid.
func setFieldIfValid(info map[string]interface{}, key string, setter func(string)) {
	if v, ok := info[key].(string); ok && v != "" {
		setter(v)
	}
}

// OnDisconnect is called when a node disconnects.
func (h *NodeHandler) OnDisconnect(nodeID string) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()
	_ = h.nodeRepo.SetStatus(ctx, nodeID, model.NodeOffline)
}

// handleTaskResult processes the completed task result from a client.
func (h *NodeHandler) handleTaskResult(ctx context.Context, nodeID string, payload *ws.ResultPayload) {
	// Check if task was already stopped/failed (race with Stop API)
	if t, err := h.taskRepo.GetByTaskID(ctx, payload.TaskID); err == nil && t != nil {
		if t.Status == model.TaskFailed || t.Status == model.TaskDone {
			log.Info().Str("task_id", payload.TaskID).Str("status", string(t.Status)).Msg("task result ignored: already processed")
			return
		}
	}

	if payload.Status == "failed" {
		log.Error().Str("task_id", payload.TaskID).Str("node_id", nodeID).Str("error", payload.Error).Msg("task failed")

		// Update node task stats for failed task
		if h.nodeRepo != nil {
			if node, err := h.nodeRepo.GetByNodeID(ctx, nodeID); err == nil && node != nil {
				node.TaskStats.TotalTasks++
				node.TaskStats.FailedTasks++
				node.TaskStats.CurrentTasks = h.Hub.GetTaskCount(nodeID)
				node.TaskStats.UpdatedAt = time.Now()
				_ = h.nodeRepo.UpdateTaskStats(ctx, nodeID, node.TaskStats)
			}
		}

		// Handle task failure with retry logic
		if h.scheduler != nil {
			if err := h.scheduler.HandleTaskFailure(ctx, payload.TaskID, nodeID, payload.Error); err != nil {
				log.Error().Err(err).Str("task_id", payload.TaskID).Msg("failed to handle task failure")
			}
		} else {
			// Fallback: just decrement task count
			if h.Hub != nil {
				h.Hub.DecTaskCount(nodeID)
			}
		}
		return
	}

	// First try to parse as a generic array of maps
	var rawData []map[string]any
	if err := json.Unmarshal(payload.Data, &rawData); err != nil {
		// Data parse error - still complete the task but log warning
		log.Warn().Str("task_id", payload.TaskID).Err(err).Msg("task result parse error: completing task anyway")
		h.decrementTaskCount(nodeID, payload.TaskID)
		return
	}

	log.Info().Str("task_id", payload.TaskID).Int("raw_count", len(rawData)).Msg("parsed result data")

	// Empty result is still a valid completion
	if len(rawData) == 0 {
		log.Info().Str("task_id", payload.TaskID).Msg("task completed with no results")
		h.decrementTaskCount(nodeID, payload.TaskID)
		return
	}

	// Determine the type based on the first record's fields
	firstRecord := rawData[0]
	log.Info().Str("task_id", payload.TaskID).Interface("first_record_keys", firstRecord).Msg("first record fields")
	if _, hasKey := firstRecord["key"]; hasKey {
		// This is a vuln record
		log.Info().Str("task_id", payload.TaskID).Msg("processing as vuln records")
		h.processVulnRecords(ctx, nodeID, payload.TaskID, rawData)
	} else if _, hasTitle := firstRecord["title"]; hasTitle {
		// This is an article
		log.Info().Str("task_id", payload.TaskID).Msg("processing as article records")
		h.processArticleRecords(ctx, nodeID, payload.TaskID, rawData)
	} else {
		log.Warn().Str("task_id", payload.TaskID).Msg("unknown task result format: unrecognized structure")
		h.decrementTaskCount(nodeID, payload.TaskID)
	}
}

// processVulnRecords converts raw maps to VulnRecord and saves them.
func (h *NodeHandler) processVulnRecords(ctx context.Context, nodeID, taskID string, rawData []map[string]any) {
	log.Debug().Int("raw_count", len(rawData)).Msg("processing vuln records")

	for i, item := range rawData {
		v := &model.VulnRecord{}

		// Map string fields
		v.Key = getString(item, "key")
		if v.Key == "" {
			log.Debug().Int("index", i).Interface("item", item).Msg("skipping vuln with empty key")
			continue
		}
		v.Title = getString(item, "title")
		v.Description = getString(item, "description")
		v.CVE = getString(item, "cve")
		v.Disclosure = getString(item, "disclosure")
		v.Solutions = getString(item, "solutions")
		v.From = getString(item, "from")
		v.Source = getString(item, "source")
		v.URL = getString(item, "url")
		
		// Handle Severity - convert to SeverityLevel type
		if sev := getString(item, "severity"); sev != "" {
			v.Severity = model.SeverityLevel(sev)
		}
		
		// Handle array fields
		if refs, ok := item["references"].([]any); ok {
			for _, r := range refs {
				if s, ok := r.(string); ok {
					v.References = append(v.References, s)
				}
			}
		}
		if tags, ok := item["tags"].([]any); ok {
			for _, t := range tags {
				if s, ok := t.(string); ok {
					v.Tags = append(v.Tags, s)
				}
			}
		}
		if github, ok := item["github_search"].([]any); ok {
			for _, g := range github {
				if s, ok := g.(string); ok {
					v.GithubSearch = append(v.GithubSearch, s)
				}
			}
		}
		
		v.ReportedBy = nodeID
		
		if _, err := h.vulnRepo.Upsert(ctx, v); err != nil {
			log.Error().Err(err).Str("key", v.Key).Msg("failed to upsert vuln from node")
		}
	}
	
	log.Info().Int("count", len(rawData)).Str("node_id", nodeID).Str("task_id", taskID).Msg("vuln task result processed")
	h.decrementTaskCount(nodeID, taskID)
}

// processArticleRecords converts raw maps to Article and saves them.
func (h *NodeHandler) processArticleRecords(ctx context.Context, nodeID, taskID string, rawData []map[string]any) {
	articles := make([]*model.Article, 0, len(rawData))
	for _, item := range rawData {
		a := &model.Article{
			Title:       getString(item, "title"),
			Summary:     getString(item, "summary"),
			Content:     getString(item, "content"),
			Author:      getString(item, "author"),
			Source:      getString(item, "source"),
			URL:         getString(item, "url"),
			Image:       getString(item, "image"),
			Tags:        []string{},
			ReportedBy:  nodeID,
		}
		
		if tags, ok := item["tags"].([]any); ok {
			for _, t := range tags {
				if s, ok := t.(string); ok {
					a.Tags = append(a.Tags, s)
				}
			}
		}
		
		if pubAt := getString(item, "published_at"); pubAt != "" {
			if t, err := time.Parse(time.RFC3339, pubAt); err == nil {
				a.PublishedAt = t
			} else if t, err := time.Parse("2006-01-02T15:04:05Z", pubAt); err == nil {
				a.PublishedAt = t
			} else if t, err := time.Parse("2006-01-02", pubAt); err == nil {
				a.PublishedAt = t
			}
		}
		
		articles = append(articles, a)
	}
	
	// Bulk upsert articles
	if h.articleRepo != nil && len(articles) > 0 {
		upserted, err := h.articleRepo.BulkUpsert(ctx, articles)
		if err != nil {
			log.Error().Err(err).Str("node_id", nodeID).Msg("failed to bulk upsert articles")
		} else {
			log.Info().Int("upserted", upserted).Int("received", len(articles)).Str("node_id", nodeID).Str("task_id", taskID).Msg("articles upserted successfully")
		}
	}
	
	log.Info().Int("count", len(articles)).Str("node_id", nodeID).Str("task_id", taskID).Msg("article task result processed")
	h.decrementTaskCount(nodeID, taskID)
}

// getString safely extracts a string field from a map.
func getString(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

// decrementTaskCount decreases the task count for a node and marks the task as done.
func (h *NodeHandler) decrementTaskCount(nodeID string, taskID string) {
	if h.Hub != nil {
		h.Hub.DecTaskCount(nodeID)
	}

	// Update node task stats for successful task completion
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()
	if h.nodeRepo != nil {
		if node, err := h.nodeRepo.GetByNodeID(ctx, nodeID); err == nil && node != nil {
			node.TaskStats.TotalTasks++
			node.TaskStats.SuccessTasks++
			node.TaskStats.CurrentTasks = h.Hub.GetTaskCount(nodeID)
			node.TaskStats.UpdatedAt = time.Now()
			_ = h.nodeRepo.UpdateTaskStats(ctx, nodeID, node.TaskStats)
		}
	}

	// Mark task as done
	h.completeTask(ctx, taskID)
}

// completeTask marks a task as completed.
func (h *NodeHandler) completeTask(ctx context.Context, taskID string) {
	if h.taskRepo != nil {
		// Use UpdateStatus with empty assignedTo to mark as done
		if err := h.taskRepo.UpdateStatus(ctx, taskID, model.TaskDone, ""); err != nil {
			log.Error().Err(err).Str("task_id", taskID).Msg("failed to mark task as completed")
		}
	}
}

// CreateNodeRequest is the payload for registering a new node.
type CreateNodeRequest struct {
	Name string   `json:"name" binding:"required"`
	IP   string   `json:"ip"`
	Tags []string `json:"tags"`
}

// Create registers a new client node and returns its credentials.
//
// POST /api/v1/nodes
func (h *NodeHandler) Create(c *gin.Context) {
	var req CreateNodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}

	node := &model.Node{
		NodeID:       uuid.New().String(),
		Name:         req.Name,
		Token:        uuid.New().String(), // Generate a secure token
		Status:       model.NodeOffline,
		Sources:      []string{},
		RegisteredAt: time.Now().UTC(),
		LastSeenAt:   time.Now().UTC(),
	}

	if req.IP != "" {
		node.Info.IP = req.IP
	}

	if err := h.nodeRepo.Upsert(c, node); err != nil {
		log.Error().Err(err).Str("node_id", node.NodeID).Msg("failed to create node")
		fail(c, http.StatusInternalServerError, "failed to register node")
		return
	}

	// Return node credentials (token is only shown once)
	ok(c, gin.H{
		"node_id": node.NodeID,
		"token":   node.Token,
		"name":    node.Name,
		"status":  node.Status,
	})
}

// Delete removes a node by its MongoDB ObjectID.
//
// DELETE /api/v1/nodes/:id
func (h *NodeHandler) Delete(c *gin.Context) {
	id, err := bson.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		fail(c, http.StatusBadRequest, "invalid id")
		return
	}

	// Get node to check if it exists and disconnect if online
	node, err := h.nodeRepo.GetByID(c, id)
	if err != nil {
		fail(c, http.StatusNotFound, "node not found")
		return
	}

	// Disconnect if online
	if h.Hub != nil {
		h.Hub.Disconnect(node.NodeID)
	}

	if err := h.nodeRepo.Delete(c, id); err != nil {
		log.Error().Err(err).Str("node_id", id.Hex()).Msg("failed to delete node")
		fail(c, http.StatusInternalServerError, "failed to delete node")
		return
	}

	ok(c, nil)
}

// RegenerateToken generates a new token for a node.
//
// POST /api/v1/nodes/:id/regenerate-token
func (h *NodeHandler) RegenerateToken(c *gin.Context) {
	id, err := bson.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		fail(c, http.StatusBadRequest, "invalid id")
		return
	}

	newToken := uuid.New().String()
	if err := h.nodeRepo.UpdateToken(c, id, newToken); err != nil {
		log.Error().Err(err).Str("node_id", id.Hex()).Msg("failed to regenerate token")
		fail(c, http.StatusInternalServerError, "failed to regenerate token")
		return
	}

	// Disconnect the node to force reconnection with new token
	node, _ := h.nodeRepo.GetByID(c, id)
	if node != nil && h.Hub != nil {
		h.Hub.Disconnect(node.NodeID)
	}

	ok(c, gin.H{"token": newToken})
}

// Pause pauses a node (prevents it from receiving new tasks).
//
// POST /api/v1/nodes/:id/pause
func (h *NodeHandler) Pause(c *gin.Context) {
	id, err := bson.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		fail(c, http.StatusBadRequest, "invalid id")
		return
	}

	node, err := h.nodeRepo.GetByID(c, id)
	if err != nil {
		fail(c, http.StatusNotFound, "node not found")
		return
	}

	// Update node status to paused
	if err := h.nodeRepo.SetStatus(c, node.NodeID, model.NodePaused); err != nil {
		fail(c, http.StatusInternalServerError, "failed to pause node")
		return
	}

	// Notify node via WebSocket if connected
	if h.Hub != nil {
		h.Hub.SendToNode(node.NodeID, &ws.Message{
			Type:    ws.MsgTypeCommand,
			Version: ws.ProtocolVersion,
			Payload: []byte(`{"action":"pause"}`),
		})
	}

	ok(c, gin.H{"node_id": node.NodeID, "status": "paused"})
}

// Resume resumes a paused node.
//
// POST /api/v1/nodes/:id/resume
func (h *NodeHandler) Resume(c *gin.Context) {
	id, err := bson.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		fail(c, http.StatusBadRequest, "invalid id")
		return
	}

	node, err := h.nodeRepo.GetByID(c, id)
	if err != nil {
		fail(c, http.StatusNotFound, "node not found")
		return
	}

	// Update node status back to online
	if err := h.nodeRepo.SetStatus(c, node.NodeID, model.NodeOnline); err != nil {
		fail(c, http.StatusInternalServerError, "failed to resume node")
		return
	}

	// Notify node via WebSocket if connected
	if h.Hub != nil {
		h.Hub.SendToNode(node.NodeID, &ws.Message{
			Type:    ws.MsgTypeCommand,
			Version: ws.ProtocolVersion,
			Payload: []byte(`{"action":"resume"}`),
		})
	}

	ok(c, gin.H{"node_id": node.NodeID, "status": "online"})
}

// GetLogs retrieves logs for a node.
//
// GET /api/v1/nodes/:id/logs
func (h *NodeHandler) GetLogs(c *gin.Context) {
	id, err := bson.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		fail(c, http.StatusBadRequest, "invalid id")
		return
	}

	node, err := h.nodeRepo.GetByID(c, id)
	if err != nil {
		fail(c, http.StatusNotFound, "node not found")
		return
	}

	// Get logs from Redis queue
	ctx := c.Request.Context()
	logs, err := h.queue.GetNodeLogs(ctx, node.NodeID)
	if err != nil {
		fail(c, http.StatusInternalServerError, "failed to get logs")
		return
	}

	ok(c, gin.H{
		"node_id": node.NodeID,
		"logs":    logs,
		"count":   len(logs),
	})
}

// Disconnect forces a node to disconnect.
//
// POST /api/v1/nodes/:id/disconnect
func (h *NodeHandler) Disconnect(c *gin.Context) {
	id, err := bson.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		fail(c, http.StatusBadRequest, "invalid id")
		return
	}

	node, err := h.nodeRepo.GetByID(c, id)
	if err != nil {
		fail(c, http.StatusNotFound, "node not found")
		return
	}

	// Disconnect the node
	if h.Hub != nil {
		h.Hub.Disconnect(node.NodeID)
	}

	// Update status to offline
	_ = h.nodeRepo.SetStatus(c, node.NodeID, model.NodeOffline)

	ok(c, gin.H{"node_id": node.NodeID, "status": "offline"})
}

// objectIDFromHex is a shared helper across handlers.
func objectIDFromHex(hex string) (bson.ObjectID, error) {
	return bson.ObjectIDFromHex(hex)
}
