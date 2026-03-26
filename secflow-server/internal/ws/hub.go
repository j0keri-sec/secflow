// Package ws implements the WebSocket hub that manages persistent connections
// from client nodes.
package ws

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
	"github.com/secflow/server/internal/model"
)

const (
	// ProtocolVersion is the current WebSocket protocol version.
	ProtocolVersion = "1.0"
)

// MessageType is the discriminator field in all WebSocket messages.
type MessageType string

const (
	// Server → Client
	MsgTypeTask       MessageType = "task"        // dispatch a new task
	MsgTypeTaskCancel MessageType = "task_cancel"  // cancel a running task
	MsgTypePing       MessageType = "ping"         // keep-alive ping
	MsgTypeCommand    MessageType = "command"      // generic command (pause/resume/etc)

	// Client → Server
	MsgTypeRegister   MessageType = "register"    // node registration
	MsgTypeHeartbeat  MessageType = "heartbeat"   // periodic heartbeat with node info
	MsgTypeProgress   MessageType = "progress"    // task progress update
	MsgTypeResult     MessageType = "result"      // task result upload
	MsgTypePong       MessageType = "pong"        // ping response
)

// Message is the envelope exchanged over WebSocket connections.
type Message struct {
	Type    MessageType     `json:"type"`
	Version string          `json:"version"`        // protocol version
	Payload json.RawMessage `json:"payload,omitempty"`
}

// RegisterPayload is sent by a new client to identify itself.
type RegisterPayload struct {
	NodeID  string      `json:"node_id"`
	Token   string      `json:"token"`
	Name    string      `json:"name"`
	Info    interface{} `json:"info"`
	Sources []string    `json:"sources"`
}

// HeartbeatPayload carries the periodic node status update.
type HeartbeatPayload struct {
	NodeID  string      `json:"node_id"`
	Info    interface{} `json:"info"`
}

// ProgressPayload is sent by a client to report task progress.
type ProgressPayload struct {
	TaskID   string `json:"task_id"`
	Progress int    `json:"progress"` // 0–100
	Message  string `json:"message,omitempty"`
}

// ResultPayload carries the completed task result.
type ResultPayload struct {
	TaskID string          `json:"task_id"`
	Status string          `json:"status"` // done | failed
	Data   json.RawMessage `json:"data,omitempty"`
	Error  string          `json:"error,omitempty"`
}

// BatchTaskPayload represents multiple tasks sent to clients in one message.
type BatchTaskPayload struct {
	Tasks []struct {
		TaskID  string          `json:"task_id"`
		Type    string          `json:"type"`
		Payload json.RawMessage `json:"payload"`
	} `json:"tasks"` // Array of tasks
}

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 8 * 1024 * 1024 // 8 MB
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin: func(r *http.Request) bool {
		// Origin check is defense-in-depth; primary auth is via JWT token in WebSocket handshake
		// If no origins configured, accept all (auth will still filter)
		origin := r.Header.Get("Origin")
		if origin == "" {
			return true // No origin header, let auth handle it
		}

		// In production, validate origin against allowed origins
		// This prevents CSRF-style attacks on WebSocket connections
		if len(allowedOrigins) > 0 {
			for _, allowed := range allowedOrigins {
				if origin == allowed {
					return true
				}
			}
			return false
		}

		// No origins configured - accept all (but auth will still validate)
		return true
	},
	EnableCompression: true, // enable per-message-deflate compression
}

// allowedOrigins is set during hub creation from config
var allowedOrigins []string

// SetAllowedOrigins configures which origins are allowed for WebSocket connections
func SetAllowedOrigins(origins []string) {
	allowedOrigins = origins
}

// Client is a single WebSocket connection from a node.
type Client struct {
	hub    *Hub
	nodeID string
	conn   *websocket.Conn
	send   chan []byte
}

// Hub maintains all active node connections.
type Hub struct {
	mu      sync.RWMutex
	clients map[string]*Client // nodeID → client
	// taskCounts tracks the number of running tasks for each node (for load balancing)
	taskCounts map[string]int

	// register/unregister channels keep the clients map mutation single-threaded.
	registerCh   chan *Client
	unregisterCh chan *Client

	// onMessage is called for every inbound message from any client.
	onMessage func(nodeID string, msg *Message)
	// onConnect is called when a node connects.
	onConnect func(nodeID string)
	// onDisconnect is called when a node disconnects.
	onDisconnect func(nodeID string)
}

// NewHub creates and starts a Hub.
func NewHub(
	onMessage func(nodeID string, msg *Message),
	onConnect func(nodeID string),
	onDisconnect func(nodeID string),
) *Hub {
	h := &Hub{
		clients:      make(map[string]*Client),
		taskCounts:   make(map[string]int),
		registerCh:   make(chan *Client, 64),
		unregisterCh: make(chan *Client, 64),
		onMessage:    onMessage,
		onConnect:    onConnect,
		onDisconnect: onDisconnect,
	}
	go h.run()
	return h
}

// run is the hub event loop — it serialises all connection state changes.
func (h *Hub) run() {
	for {
		select {
		case c := <-h.registerCh:
			h.mu.Lock()
			h.clients[c.nodeID] = c
			h.mu.Unlock()
			if h.onConnect != nil {
				go h.onConnect(c.nodeID)
			}
			log.Info().Str("node_id", c.nodeID).Msg("ws: node connected")

		case c := <-h.unregisterCh:
			h.mu.Lock()
			if _, ok := h.clients[c.nodeID]; ok {
				delete(h.clients, c.nodeID)
				close(c.send)
			}
			h.mu.Unlock()
			if h.onDisconnect != nil {
				go h.onDisconnect(c.nodeID)
			}
			log.Info().Str("node_id", c.nodeID).Msg("ws: node disconnected")
		}
	}
}

// Send delivers a message to the specified node.
// Returns false when the node is not connected.
func (h *Hub) Send(nodeID string, msg *Message) bool {
	data, err := json.Marshal(msg)
	if err != nil {
		return false
	}
	h.mu.RLock()
	c, ok := h.clients[nodeID]
	h.mu.RUnlock()
	if !ok {
		return false
	}
	select {
	case c.send <- data:
		return true
	default:
		// client's buffer full — treat as disconnected
		h.unregisterCh <- c
		return false
	}
}

// SendToNode is an alias for Send, provided for API handler convenience.
func (h *Hub) SendToNode(nodeID string, msg *Message) bool {
	return h.Send(nodeID, msg)
}

// SendBatch delivers multiple task messages to the specified node in a single frame.
// Returns false when the node is not connected.
func (h *Hub) SendBatch(nodeID string, msgs []*Message) bool {
	// For batch sending, we create a single message with a batch payload
	if len(msgs) == 0 {
		return true // Nothing to send
	}
	
	// If only one message, use regular Send
	if len(msgs) == 1 {
		return h.Send(nodeID, msgs[0])
	}
	
// For multiple messages, create a batch task message
	var batchTasks []struct {
		TaskID  string          `json:"task_id"`
		Type    string          `json:"type"`
		Payload json.RawMessage `json:"payload"`
	}
	for _, msg := range msgs {
		// Parse the individual task payload
		var taskPayload struct {
			TaskID  string          `json:"task_id"`
			Type    string          `json:"type"`
			Payload json.RawMessage `json:"payload"`
		}
		if err := json.Unmarshal(msg.Payload, &taskPayload); err != nil {
			log.Warn().Err(err).Msg("hub: failed to unmarshal task for batch")
			continue
		}
		log.Debug().Str("task_id", taskPayload.TaskID).Int("payload_len", len(taskPayload.Payload)).Msg("hub: parsed task for batch")
		batchTasks = append(batchTasks, taskPayload)
	}

	if len(batchTasks) == 0 {
		return false
	}

	// Create batch payload
	batchPayload := BatchTaskPayload{Tasks: batchTasks}
	payloadBytes, err := json.Marshal(batchPayload)
	if err != nil {
		log.Error().Err(err).Msg("hub: failed to marshal batch payload")
		return false
	}
	
	// Send as a single message
	batchMsg := &Message{
		Type:    MsgTypeTask,
		Version: ProtocolVersion,
		Payload: payloadBytes,
	}
	
	return h.Send(nodeID, batchMsg)
}

// Broadcast delivers a message to all connected nodes.
func (h *Hub) Broadcast(msg *Message) {
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, c := range h.clients {
		select {
		case c.send <- data:
		default:
		}
	}
}

// ConnectedNodes returns a snapshot of currently connected node IDs.
func (h *Hub) ConnectedNodes() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	ids := make([]string, 0, len(h.clients))
	for id := range h.clients {
		ids = append(ids, id)
	}
	return ids
}

// Disconnect forcibly disconnects a node by its nodeID.
func (h *Hub) Disconnect(nodeID string) {
	h.mu.RLock()
	c, ok := h.clients[nodeID]
	h.mu.RUnlock()
	if ok {
		h.unregisterCh <- c
		_ = c.conn.Close()
	}
}

// ServeWS upgrades an HTTP connection to WebSocket and wires it to the hub.
// The caller must supply the nodeID after the initial RegisterPayload is received;
// for simplicity we accept it as a query param here (validated upstream).
func (h *Hub) ServeWS(w http.ResponseWriter, r *http.Request, nodeID string) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error().Err(err).Msg("ws: upgrade failed")
		return
	}
	c := &Client{
		hub:    h,
		nodeID: nodeID,
		conn:   conn,
		send:   make(chan []byte, 256),
	}
	h.registerCh <- c

	go c.writePump()
	go c.readPump()
}

// readPump reads inbound messages and forwards them to the hub's onMessage.
func (c *Client) readPump() {
	defer func() {
		c.hub.unregisterCh <- c
		_ = c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	for {
		_, raw, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Warn().Err(err).Str("node_id", c.nodeID).Msg("ws: unexpected close")
			}
			return
		}
		var msg Message
		if err = json.Unmarshal(raw, &msg); err != nil {
			log.Warn().Err(err).Msg("ws: invalid message format")
			continue
		}
		if c.hub.onMessage != nil {
			c.hub.onMessage(c.nodeID, &msg)
		}
	}
}

// writePump drains the send channel and writes to the WebSocket connection.
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		_ = c.conn.Close()
	}()

	for {
		select {
		case msg, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}

		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// IncTaskCount increments the task count for a node (used for load balancing).
func (h *Hub) IncTaskCount(nodeID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.taskCounts[nodeID]++
}

// DecTaskCount decrements the task count for a node (used for load balancing).
func (h *Hub) DecTaskCount(nodeID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.taskCounts[nodeID] > 0 {
		h.taskCounts[nodeID]--
	}
}

// GetTaskCount returns the current task count for a node.
func (h *Hub) GetTaskCount(nodeID string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.taskCounts[nodeID]
}

// GetBestNode returns the node with the least number of running tasks.
// If multiple nodes have the same count, it selects one randomly among them.
func (h *Hub) GetBestNode() string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if len(h.clients) == 0 {
		return ""
	}

	// Find the minimum task count
	minCount := int(^uint(0) >> 1) // Max int
	for id := range h.clients {
		if h.taskCounts[id] < minCount {
			minCount = h.taskCounts[id]
		}
	}

	// Find all nodes with the minimum count
	var bestNodes []string
	for id := range h.clients {
		if h.taskCounts[id] == minCount {
			bestNodes = append(bestNodes, id)
		}
	}

	// Return the first one (simple approach)
	if len(bestNodes) > 0 {
		return bestNodes[0]
	}
	return ""
}

// GetBestNodeIntelligent returns the best node based on intelligent load balancing.
// It considers task count, CPU usage, memory usage, and historical success rate.
// Returns empty string if no nodes are available.
func (h *Hub) GetBestNodeIntelligent(getNodeStats func(nodeID string) (model.NodeTaskStats, error)) string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if len(h.clients) == 0 {
		return ""
	}

	bestNode := ""
	bestScore := -1.0

	// Evaluate each connected node
	for nodeID := range h.clients {
		// Get node statistics (from repository or cache)
		stats, err := getNodeStats(nodeID)
		if err != nil {
			// If we can't get stats, use current task count as fallback
			stats.CurrentTasks = h.taskCounts[nodeID]
		}

		// Calculate comprehensive score (higher is better)
		score := calculateNodeScore(stats)

		// Update best node
		if score > bestScore {
			bestScore = score
			bestNode = nodeID
		}
	}

	return bestNode
}

// calculateNodeScore computes a load balancing score for a node.
// Score is based on: available capacity, resource usage, and historical performance.
// Higher scores indicate better candidates for new tasks.
func calculateNodeScore(stats model.NodeTaskStats) float64 {
	score := 100.0 // Base score

	// 1. Task load factor (30 points)
	// Prefer nodes with fewer current tasks
	score -= float64(stats.CurrentTasks) * 10.0

	// 2. CPU usage penalty (20 points)
	// Penalize nodes with high CPU usage
	if stats.CPUPercent > 70.0 {
		cpuPenalty := (stats.CPUPercent - 70.0) * 0.5
		score -= cpuPenalty
	}

	// 3. Memory usage penalty (20 points)
	// Penalize nodes with high memory usage
	if stats.MemPercent > 80.0 {
		memPenalty := (stats.MemPercent - 80.0) * 0.5
		score -= memPenalty
	}

	// 4. Historical success rate bonus (30 points)
	// Prefer nodes with high success rates
	successRate := stats.SuccessRate()
	score += successRate * 30.0

	// 5. Average response time penalty (adjustment)
	// Penalize slow nodes slightly
	if stats.AvgResponseTime > 0 {
		// Convert to seconds and penalize if slower than 30 seconds
		avgSeconds := float64(stats.AvgResponseTime) / 1000.0
		if avgSeconds > 30.0 {
			score -= (avgSeconds - 30.0) * 0.1
		}
	}

	// Ensure score doesn't go negative
	if score < 0 {
		score = 0
	}

	return score
}
