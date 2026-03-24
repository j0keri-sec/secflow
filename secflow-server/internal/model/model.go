// Package model defines the MongoDB document schemas for the secflow platform.
// Each model corresponds to one MongoDB collection.
package model

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// --------------------------------------------------------------------------
// User & Auth
// --------------------------------------------------------------------------

// RoleType identifies a user's permission level.
type RoleType string

const (
	RoleAdmin  RoleType = "admin"
	RoleEditor RoleType = "editor"
	RoleViewer RoleType = "viewer"
)

// User represents a platform user account.
// Collection: users
type User struct {
	ID           bson.ObjectID `bson:"_id,omitempty"  json:"id"`
	Username     string        `bson:"username"        json:"username"`
	Email        string        `bson:"email"           json:"email"`
	PasswordHash string        `bson:"password_hash"   json:"-"`
	Role         RoleType      `bson:"role"            json:"role"`
	// InviteCode is the invite code used when this account was created.
	InviteCode   string        `bson:"invite_code"     json:"invite_code,omitempty"`
	// InvitedBy is the ObjectID of the user who owns the invite code used.
	InvitedBy    bson.ObjectID `bson:"invited_by,omitempty" json:"invited_by,omitempty"`
	Avatar       string        `bson:"avatar"          json:"avatar"`
	Active       bool          `bson:"active"          json:"active"`
	CreatedAt    time.Time     `bson:"created_at"      json:"created_at"`
	UpdatedAt    time.Time     `bson:"updated_at"      json:"updated_at"`
}

// InviteCode is an invitation token that allows account registration.
// Collection: invite_codes
type InviteCode struct {
	ID        bson.ObjectID `bson:"_id,omitempty" json:"id"`
	Code      string        `bson:"code"           json:"code"`
	// OwnerID is the user who generated this code.
	OwnerID   bson.ObjectID `bson:"owner_id"       json:"owner_id"`
	// UsedByID is populated when the code has been redeemed.
	UsedByID  bson.ObjectID `bson:"used_by_id,omitempty" json:"used_by_id,omitempty"`
	Used      bool          `bson:"used"           json:"used"`
	// MaxAdmin controls whether admins bypass the per-user limit.
	IsAdmin   bool          `bson:"is_admin"       json:"is_admin"`
	CreatedAt time.Time     `bson:"created_at"     json:"created_at"`
	UsedAt    time.Time     `bson:"used_at,omitempty" json:"used_at,omitempty"`
}

// --------------------------------------------------------------------------
// Node (Client)
// --------------------------------------------------------------------------

// NodeStatus describes the operational state of a connected client node.
type NodeStatus string

const (
	NodeOnline  NodeStatus = "online"
	NodeOffline NodeStatus = "offline"
	NodeBusy    NodeStatus = "busy"
	NodePaused  NodeStatus = "paused"
)

// NodeTaskStats tracks task execution performance for load balancing decisions.
type NodeTaskStats struct {
	// Total tasks executed
	TotalTasks int `bson:"total_tasks" json:"total_tasks"`
	// Successful tasks count
	SuccessTasks int `bson:"success_tasks" json:"success_tasks"`
	// Failed tasks count
	FailedTasks int `bson:"failed_tasks" json:"failed_tasks"`
	// Average response time in milliseconds
	AvgResponseTime int64 `bson:"avg_response_time_ms" json:"avg_response_time_ms"`
	// Current task count (transient, not persisted)
	CurrentTasks int `bson:"-" json:"current_tasks,omitempty"`
	// CPU usage percentage from last heartbeat
	CPUPercent float64 `bson:"cpu_percent,omitempty" json:"cpu_percent,omitempty"`
	// Memory usage percentage from last heartbeat
	MemPercent float64 `bson:"mem_percent,omitempty" json:"mem_percent,omitempty"`
	// Last updated timestamp
	UpdatedAt time.Time `bson:"updated_at" json:"updated_at"`
}

// SuccessRate calculates the task success rate (0.0 to 1.0)
func (s NodeTaskStats) SuccessRate() float64 {
	if s.TotalTasks == 0 {
		return 1.0 // Default to 100% for new nodes
	}
	return float64(s.SuccessTasks) / float64(s.TotalTasks)
}

// LoadScore calculates a load score (lower is better)
func (s NodeTaskStats) LoadScore() float64 {
	// Base score from current task count
	score := float64(s.CurrentTasks) * 10.0
	
	// Add CPU penalty if over 70%
	if s.CPUPercent > 70.0 {
		score += (s.CPUPercent - 70.0) * 2.0
	}
	
	// Add memory penalty if over 80%
	if s.MemPercent > 80.0 {
		score += (s.MemPercent - 80.0) * 2.0
	}
	
	return score
}

// NodeInfo contains the hardware/OS snapshot reported by a client.
type NodeInfo struct {
	IP         string   `bson:"ip"         json:"ip"`
	PublicIP   string   `bson:"public_ip"  json:"public_ip"`
	MAC        string   `bson:"mac"        json:"mac"`
	OS         string   `bson:"os"         json:"os"`
	Arch       string   `bson:"arch"       json:"arch"`
	CPUModel   string   `bson:"cpu_model"  json:"cpu_model"`
	CPUCores   int      `bson:"cpu_cores"  json:"cpu_cores"`
	MemTotal   uint64   `bson:"mem_total"  json:"mem_total"`
	MemUsed    uint64   `bson:"mem_used"   json:"mem_used"`
	DiskTotal  uint64   `bson:"disk_total" json:"disk_total"`
	DiskUsed   uint64   `bson:"disk_used"  json:"disk_used"`
	CPUPercent float64  `bson:"cpu_pct"    json:"cpu_pct"`
	NetCards   []string `bson:"net_cards"  json:"net_cards"`
}

// Node represents a registered client node.
// Collection: nodes
type Node struct {
	ID          bson.ObjectID `bson:"_id,omitempty"  json:"id"`
	NodeID      string        `bson:"node_id"         json:"node_id"` // stable client-generated UUID
	Name        string        `bson:"name"            json:"name"`
	Token       string        `bson:"token"           json:"token"` // pre-shared auth token
	Status      NodeStatus    `bson:"status"          json:"status"`
	Info        NodeInfo      `bson:"info"            json:"info"`
	// Sources is the list of grabber source names this node is responsible for.
	Sources     []string      `bson:"sources"         json:"sources"`
	LastSeenAt  time.Time     `bson:"last_seen_at"    json:"last_seen_at"`
	RegisteredAt time.Time    `bson:"registered_at"   json:"registered_at"`
	
	// Performance metrics for intelligent load balancing
	TaskStats   NodeTaskStats `bson:"task_stats"      json:"task_stats"`
}

// --------------------------------------------------------------------------
// Task
// --------------------------------------------------------------------------

// TaskType describes the kind of work a task represents.
type TaskType string

const (
	TaskTypeVulnCrawl    TaskType = "vuln_crawl"
	TaskTypeArticleCrawl TaskType = "article_crawl"
)

// TaskStatus is the execution state of a task.
type TaskStatus string

const (
	TaskPending   TaskStatus = "pending"
	TaskDispatched TaskStatus = "dispatched"
	TaskRunning   TaskStatus = "running"
	TaskDone      TaskStatus = "done"
	TaskFailed    TaskStatus = "failed"
)

// Task is a unit of work dispatched to a client node via Redis queue.
// Collection: tasks
type Task struct {
	ID         bson.ObjectID `bson:"_id,omitempty" json:"id"`
	TaskID     string        `bson:"task_id"        json:"task_id"` // UUID
	Type       TaskType      `bson:"type"           json:"type"`
	Status     TaskStatus    `bson:"status"         json:"status"`
	// AssignedTo is the NodeID of the client that picked up this task.
	AssignedTo string        `bson:"assigned_to,omitempty" json:"assigned_to,omitempty"`
	// Payload is the JSON-encoded task configuration sent to the client.
	Payload    []byte        `bson:"payload"        json:"payload"`
	// Result is the JSON-encoded result uploaded by the client.
	Result     []byte        `bson:"result,omitempty" json:"result,omitempty"`
	Error      string        `bson:"error,omitempty" json:"error,omitempty"`
	Progress   int           `bson:"progress"       json:"progress"` // 0-100
	Priority   int           `bson:"priority"       json:"priority"` // Task priority (0-100)
	CreatedAt  time.Time     `bson:"created_at"     json:"created_at"`
	UpdatedAt  time.Time     `bson:"updated_at"     json:"updated_at"`
	FinishedAt time.Time     `bson:"finished_at,omitempty" json:"finished_at,omitempty"`
	
	// Retry tracking fields
	RetryCount    int       `bson:"retry_count,omitempty"    json:"retry_count,omitempty"`    // Current retry count
	MaxRetries    int       `bson:"max_retries,omitempty"    json:"max_retries,omitempty"`    // Maximum allowed retries
	LastRetryAt   time.Time `bson:"last_retry_at,omitempty"  json:"last_retry_at,omitempty"`  // Last retry timestamp
	NextRetryAt   time.Time `bson:"next_retry_at,omitempty"  json:"next_retry_at,omitempty"`  // Next retry scheduled time
	RetryErrors   []string  `bson:"retry_errors,omitempty"   json:"retry_errors,omitempty"`   // Error history from retries
	
	// Timeout fields
	TimeoutSeconds int       `bson:"timeout_seconds,omitempty" json:"timeout_seconds,omitempty"` // Task timeout in seconds
	StartedAt      time.Time `bson:"started_at,omitempty"     json:"started_at,omitempty"`     // Task start time
}

// VulnCrawlPayload is the payload for a TaskTypeVulnCrawl task.
type VulnCrawlPayload struct {
	Sources         []string `json:"sources"`
	PageLimit       int      `json:"page_limit"`
	EnableGithub    bool     `json:"enable_github"`
	Proxy           string   `json:"proxy,omitempty"`
}

// --------------------------------------------------------------------------
// Vulnerability
// --------------------------------------------------------------------------

// SeverityLevel is the threat severity.
type SeverityLevel string

const (
	SeverityLow      SeverityLevel = "低危"
	SeverityMedium   SeverityLevel = "中危"
	SeverityHigh     SeverityLevel = "高危"
	SeverityCritical SeverityLevel = "严重"
)

// VulnRecord is the persisted vulnerability document.
// Collection: vuln_records
type VulnRecord struct {
	ID           bson.ObjectID `bson:"_id,omitempty"  json:"id"`
	Key          string        `bson:"key"             json:"key"`   // unique per source
	Title        string        `bson:"title"           json:"title"`
	Description  string        `bson:"description"     json:"description"`
	Severity     SeverityLevel `bson:"severity"        json:"severity"`
	CVE          string        `bson:"cve"             json:"cve"`
	Disclosure   string        `bson:"disclosure"      json:"disclosure"`
	Solutions    string        `bson:"solutions"       json:"solutions"`
	References   []string      `bson:"references"      json:"references"`
	Tags         []string      `bson:"tags"            json:"tags"`
	GithubSearch []string      `bson:"github_search"   json:"github_search"`
	From         string        `bson:"from"            json:"from"`
	Source       string        `bson:"source"          json:"source"` // grabber name
	URL          string        `bson:"url"             json:"url"`   // source URL
	Pushed       bool          `bson:"pushed"          json:"pushed"`
	// ReportedBy is the NodeID of the client that submitted this record.
	ReportedBy   string        `bson:"reported_by"     json:"reported_by"`
	CreatedAt    time.Time     `bson:"created_at"      json:"created_at"`
	UpdatedAt    time.Time     `bson:"updated_at"      json:"updated_at"`
}

// --------------------------------------------------------------------------
// Article (technical blog/forum)
// --------------------------------------------------------------------------

// Article is a technical article or forum post.
// Collection: articles
type Article struct {
	ID          bson.ObjectID `bson:"_id,omitempty" json:"id"`
	Title       string        `bson:"title"          json:"title"`
	Summary     string        `bson:"summary"        json:"summary"`
	Content     string        `bson:"content"        json:"content"`
	Author      string        `bson:"author"         json:"author"`
	Source      string        `bson:"source"         json:"source"` // forum/blog name
	URL         string        `bson:"url"            json:"url"`
	Image       string        `bson:"image"          json:"image"`   // cover image URL
	Tags        []string      `bson:"tags"           json:"tags"`
	Pushed      bool          `bson:"pushed"         json:"pushed"`
	ReportedBy  string        `bson:"reported_by"    json:"reported_by"`
	PublishedAt time.Time     `bson:"published_at"   json:"published_at"`
	CreatedAt   time.Time     `bson:"created_at"     json:"created_at"`
}

// --------------------------------------------------------------------------
// Pusher Config
// --------------------------------------------------------------------------

// PushChannel stores a configured notification channel.
// Collection: push_channels
type PushChannel struct {
	ID        bson.ObjectID     `bson:"_id,omitempty" json:"id"`
	Name      string            `bson:"name"           json:"name"`
	Type      string            `bson:"type"           json:"type"` // dingding | lark | webhook | ...
	Config    map[string]string `bson:"config"         json:"config"`
	Enabled   bool              `bson:"enabled"        json:"enabled"`
	CreatedAt time.Time         `bson:"created_at"     json:"created_at"`
	UpdatedAt time.Time         `bson:"updated_at"     json:"updated_at"`
}

// --------------------------------------------------------------------------
// Audit Log
// --------------------------------------------------------------------------

// AuditLog records user operations for audit trail.
// Collection: audit_logs
type AuditLog struct {
	ID        bson.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID    bson.ObjectID `bson:"user_id"        json:"user_id"`
	Username  string        `bson:"username"       json:"username"`
	Action    string        `bson:"action"         json:"action"`
	Resource  string        `bson:"resource"       json:"resource"`
	Detail    string        `bson:"detail"         json:"detail"`
	IP        string        `bson:"ip"             json:"ip"`
	CreatedAt time.Time     `bson:"created_at"     json:"created_at"`
}

// --------------------------------------------------------------------------
// Report
// --------------------------------------------------------------------------

// ReportStatus is the generation state of a report.
type ReportStatus string

const (
	ReportPending    ReportStatus = "pending"
	ReportGenerating ReportStatus = "generating"
	ReportDone       ReportStatus = "done"
	ReportFailed     ReportStatus = "failed"
)

// Report represents a generated security report.
// Collection: reports
type Report struct {
	ID          bson.ObjectID `bson:"_id,omitempty" json:"id"`
	Title       string        `bson:"title"          json:"title"`
	Description string        `bson:"description"    json:"description"`
	Status      ReportStatus  `bson:"status"         json:"status"`
	// Period is a human-readable range string e.g. "2024-01-01 ~ 2024-01-31"
	Period      string        `bson:"period"         json:"period"`
	// Content is the rendered report markdown/HTML.
	Content     string        `bson:"content,omitempty" json:"content,omitempty"`
	FilePath    string        `bson:"file_path,omitempty" json:"file_path,omitempty"`
	CreatedBy   bson.ObjectID `bson:"created_by"     json:"created_by"`
	CreatedAt   time.Time     `bson:"created_at"     json:"created_at"`
	UpdatedAt   time.Time     `bson:"updated_at"     json:"updated_at"`
}

// CollectionNames maps model types to their MongoDB collection names.
const (
	CollUsers         = "users"
	CollInviteCodes   = "invite_codes"
	CollNodes         = "nodes"
	CollTasks         = "tasks"
	CollVulnRecords   = "vuln_records"
	CollArticles      = "articles"
	CollPushChannels  = "push_channels"
	CollAuditLogs     = "audit_logs"
	CollReports       = "reports"
	CollTaskSchedules = "task_schedules"
)

// TaskSchedule stores the periodic task generation configuration.
// Collection: task_schedules
type TaskSchedule struct {
	ID        bson.ObjectID `bson:"_id,omitempty" json:"id"`
	Type      TaskType      `bson:"type"           json:"type"` // vuln_crawl or article_crawl
	Enabled   bool          `bson:"enabled"        json:"enabled"`
	Interval  int           `bson:"interval"       json:"interval"` // interval in minutes
	Sources   []string      `bson:"sources"        json:"sources"`   // data sources to crawl
	CreatedAt time.Time     `bson:"created_at"     json:"created_at"`
	UpdatedAt time.Time     `bson:"updated_at"     json:"updated_at"`
}
