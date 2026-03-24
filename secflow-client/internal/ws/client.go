// Package ws implements the persistent WebSocket connection to the SecFlow server.
//
// Protocol (JSON frames using envelope format {type, payload}):
//
//	Server → Client:
//	  {"type":"ping"}
//	  {"type":"task","payload":{"task_id":"...","payload":{...}}}
//	  {"type":"task_cancel","payload":{"task_id":"..."}}
//
//	Client → Server:
//	  {"type":"pong"}
//	  {"type":"register","payload":{"node_id":"...","token":"...","name":"...","info":{...},"sources":[]}}
//	  {"type":"heartbeat","payload":{"node_id":"...","info":{...}}}
//	  {"type":"progress","payload":{"task_id":"...","progress":42,"message":"..."}}
//	  {"type":"result","payload":{"task_id":"...","status":"done","data":{...}}}
//	  {"type":"result","payload":{"task_id":"...","status":"failed","error":"..."}}
package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

var dialer = websocket.Dialer{
	EnableCompression: true,
}

// Handler is the callback interface the engine registers with the WS client.
type Handler interface {
	OnTaskAssign(msg *TaskAssignMsg)
	OnTaskCancel(taskID string)
}

// ── Message Types (matching server protocol) ───────────────────────────────

type MessageType string

const (
	// Server → Client
	MsgTypeTask       MessageType = "task"
	MsgTypeTaskCancel MessageType = "task_cancel"
	MsgTypePing       MessageType = "ping"

	// Client → Server
	MsgTypeRegister  MessageType = "register"
	MsgTypeHeartbeat MessageType = "heartbeat"
	MsgTypeProgress  MessageType = "progress"
	MsgTypeResult    MessageType = "result"
	MsgTypePong      MessageType = "pong"
)

// ProtocolVersion is the current WebSocket protocol version.
const ProtocolVersion = "1.0"

// Message is the envelope format matching the server's ws.Message.
type Message struct {
	Type    MessageType     `json:"type"`
	Version string          `json:"version,omitempty"`
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
	NodeID string      `json:"node_id"`
	Info   interface{} `json:"info"`
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

// TaskPayload is the inner payload for task assignment.
type TaskPayload struct {
	TaskID  string          `json:"task_id"`
	Type    string          `json:"type,omitempty"`
	Payload json.RawMessage `json:"payload"`
}

// BatchTaskPayload represents multiple tasks sent in one message.
type BatchTaskPayload struct {
	Tasks []TaskPayload `json:"tasks"`
}

// TaskCancelPayload is the inner payload for task cancellation.
type TaskCancelPayload struct {
	TaskID string `json:"task_id"`
}

// TaskAssignMsg is received when the server dispatches a new task.
type TaskAssignMsg struct {
	TaskID  string          `json:"task_id"`
	Payload json.RawMessage `json:"payload"`
}

// NodeInfo carries hardware/network details reported in heartbeats.
type NodeInfo struct {
	IP        string   `json:"ip"`
	PublicIP  string   `json:"public_ip"`
	MAC       string   `json:"mac"`
	OS        string   `json:"os"`
	Arch      string   `json:"arch"`
	CPUModel  string   `json:"cpu_model"`
	CPUCores  int      `json:"cpu_cores"`
	CPUPct    float64  `json:"cpu_pct"`
	MemTotal  uint64   `json:"mem_total"`
	MemUsed   uint64   `json:"mem_used"`
	DiskTotal uint64   `json:"disk_total"`
	DiskUsed  uint64   `json:"disk_used"`
	NetCards  []string `json:"net_cards"`
}

// Client manages a single gorilla WebSocket connection with auto-reconnect.
type Client struct {
	wsURL   string
	token   string
	nodeID  string
	name    string
	sources []string
	handler Handler
	log     *zap.Logger

	mu   sync.Mutex
	conn *websocket.Conn
}

// New creates a Client (does not connect yet; call Run).
func New(wsURL, token, nodeID, name string, sources []string, handler Handler, log *zap.Logger) *Client {
	return &Client{
		wsURL:   wsURL,
		token:   token,
		nodeID:  nodeID,
		name:    name,
		sources: sources,
		handler: handler,
		log:     log,
	}
}

// Run connects and keeps the connection alive until ctx is cancelled.
func (c *Client) Run(ctx context.Context) {
	for {
		if err := c.connect(ctx); err != nil {
			if ctx.Err() != nil {
				return
			}
			c.log.Warn("ws disconnected, reconnecting in 5s", zap.Error(err))
			select {
			case <-time.After(5 * time.Second):
			case <-ctx.Done():
				return
			}
		}
	}
}

func (c *Client) connect(ctx context.Context) error {
	// Build WebSocket URL with query parameters for authentication.
	// Server expects: /api/v1/ws/node?token=<token_key>&node_id=<node_id>&name=<name>
	u, err := url.Parse(c.wsURL)
	if err != nil {
		return fmt.Errorf("parse ws url: %w", err)
	}
	q := u.Query()
	q.Set("token", c.token)
	q.Set("node_id", c.nodeID)
	q.Set("name", c.name)
	u.RawQuery = q.Encode()

	conn, _, err := dialer.DialContext(ctx, u.String(), nil)
	if err != nil {
		return fmt.Errorf("dial %s: %w", u.String(), err)
	}
	c.mu.Lock()
	c.conn = conn
	c.mu.Unlock()

	c.log.Info("ws connected", zap.String("url", c.wsURL))

	// Send registration frame immediately after connecting.
	regPayload := RegisterPayload{
		NodeID:  c.nodeID,
		Token:   c.token,
		Name:    c.name,
		Info:    nil, // Will be filled in first heartbeat
		Sources: c.sources,
	}
	if err := c.sendMessage(MsgTypeRegister, regPayload); err != nil {
		return err
	}

	defer func() {
		conn.Close()
		c.mu.Lock()
		c.conn = nil
		c.mu.Unlock()
	}()

	return c.readLoop(ctx)
}

func (c *Client) readLoop(ctx context.Context) error {
	for {
		_, raw, err := c.conn.ReadMessage()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return fmt.Errorf("read: %w", err)
		}

		var msg Message
		if err := json.Unmarshal(raw, &msg); err != nil {
			c.log.Warn("failed to unmarshal message", zap.Error(err), zap.String("raw", string(raw)))
			continue
		}

		switch msg.Type {
		case MsgTypePing:
			_ = c.sendMessage(MsgTypePong, nil)

		case MsgTypeTask:
			// First try to parse as a batch task (most common case from server)
			var batchPayload BatchTaskPayload
			if err := json.Unmarshal(msg.Payload, &batchPayload); err == nil && len(batchPayload.Tasks) > 0 {
				// Handle batch tasks
				if c.handler != nil {
					for _, task := range batchPayload.Tasks {
						go c.handler.OnTaskAssign(&TaskAssignMsg{
							TaskID:  task.TaskID,
							Payload: task.Payload,
						})
					}
				}
			} else {
				// Try to parse as a single task
				var taskPayload TaskPayload
				if err := json.Unmarshal(msg.Payload, &taskPayload); err == nil && taskPayload.TaskID != "" {
					if c.handler != nil {
						go c.handler.OnTaskAssign(&TaskAssignMsg{
							TaskID:  taskPayload.TaskID,
							Payload: taskPayload.Payload,
						})
					}
				} else {
					c.log.Warn("failed to unmarshal task payload (single or batch)")
				}
			}

		case MsgTypeTaskCancel:
			var cancelPayload TaskCancelPayload
			if err := json.Unmarshal(msg.Payload, &cancelPayload); err != nil {
				c.log.Warn("failed to unmarshal task cancel payload", zap.Error(err))
				continue
			}
			if c.handler != nil {
				go c.handler.OnTaskCancel(cancelPayload.TaskID)
			}
		}
	}
}

// sendMessage sends a message with the envelope format {type, payload}.
func (c *Client) sendMessage(msgType MessageType, payload interface{}) error {
	c.mu.Lock()
	conn := c.conn
	c.mu.Unlock()
	if conn == nil {
		return fmt.Errorf("not connected")
	}

	var payloadBytes json.RawMessage
	var err error
	if payload != nil {
		payloadBytes, err = json.Marshal(payload)
		if err != nil {
			return err
		}
	}

	msg := Message{
		Type:    msgType,
		Version: ProtocolVersion,
		Payload: payloadBytes,
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return conn.WriteMessage(websocket.TextMessage, data)
}

// SendProgress notifies the server of task progress.
func (c *Client) SendProgress(taskID string, progress int, message string) error {
	return c.sendMessage(MsgTypeProgress, ProgressPayload{
		TaskID:   taskID,
		Progress: progress,
		Message:  message,
	})
}

// SendResult uploads collected vuln records to the server.
func (c *Client) SendResult(taskID string, vulns []any) error {
	data, err := json.Marshal(vulns)
	if err != nil {
		return err
	}
	return c.sendMessage(MsgTypeResult, ResultPayload{
		TaskID: taskID,
		Status: "done",
		Data:   data,
	})
}

// SendError reports a fatal task error to the server.
func (c *Client) SendError(taskID, errMsg string) error {
	return c.sendMessage(MsgTypeResult, ResultPayload{
		TaskID: taskID,
		Status: "failed",
		Error:  errMsg,
	})
}

// SendHeartbeat pushes current node metrics to the server.
func (c *Client) SendHeartbeat(nodeID string, info *NodeInfo) error {
	return c.sendMessage(MsgTypeHeartbeat, HeartbeatPayload{
		NodeID: nodeID,
		Info:   info,
	})
}

// sendJSON is a legacy helper that sends raw JSON (used for backward compatibility).
// Deprecated: Use sendMessage instead.
func (c *Client) sendJSON(v any) error {
	c.mu.Lock()
	conn := c.conn
	c.mu.Unlock()
	if conn == nil {
		return fmt.Errorf("not connected")
	}
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return conn.WriteMessage(websocket.TextMessage, data)
}

// HTTPClient returns the underlying HTTP client for API calls.
// This is a placeholder for future HTTP API support.
func (c *Client) HTTPClient() *http.Client {
	return http.DefaultClient
}
