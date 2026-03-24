# 数据模型参考

本文档基于 `internal/model/model.go` 提供完整的数据模型参考。

## MongoDB 集合一览

| 集合名 | 模型 | 说明 |
|--------|------|------|
| `users` | User | 用户账号 |
| `invite_codes` | InviteCode | 邀请码 |
| `nodes` | Node | 客户端节点 |
| `tasks` | Task | 爬取任务 |
| `vuln_records` | VulnRecord | 漏洞记录 |
| `articles` | Article | 安全文章 |
| `push_channels` | PushChannel | 推送渠道 |
| `audit_logs` | AuditLog | 审计日志 |
| `reports` | Report | 报告 |

---

## User 用户

```go
type User struct {
    ID           bson.ObjectID `bson:"_id,omitempty"  json:"id"`
    Username     string        `bson:"username"        json:"username"`
    Email        string        `bson:"email"           json:"email"`
    PasswordHash string        `bson:"password_hash"   json:"-"`           // 不暴露
    Role         RoleType      `bson:"role"            json:"role"`
    InviteCode   string        `bson:"invite_code"     json:"invite_code,omitempty"`
    InvitedBy    bson.ObjectID `bson:"invited_by,omitempty" json:"invited_by,omitempty"`
    Avatar       string        `bson:"avatar"          json:"avatar"`
    Active       bool          `bson:"active"          json:"active"`
    CreatedAt    time.Time     `bson:"created_at"      json:"created_at"`
    UpdatedAt    time.Time     `bson:"updated_at"      json:"updated_at"`
}
```

### RoleType 角色类型

```go
const (
    RoleAdmin  RoleType = "admin"   // 管理员
    RoleEditor RoleType = "editor" // 编辑
    RoleViewer RoleType = "viewer" // 访客
)
```

### 权限说明

| 角色 | 权限 |
|------|------|
| admin | 所有操作 |
| editor | 创建任务、管理漏洞/文章、配置推送 |
| viewer | 仅查看数据 |

---

## InviteCode 邀请码

```go
type InviteCode struct {
    ID        bson.ObjectID `bson:"_id,omitempty" json:"id"`
    Code      string        `bson:"code"           json:"code"`
    OwnerID   bson.ObjectID `bson:"owner_id"       json:"owner_id"`
    UsedByID  bson.ObjectID `bson:"used_by_id,omitempty" json:"used_by_id,omitempty"`
    Used      bool          `bson:"used"           json:"used"`
    IsAdmin   bool          `bson:"is_admin"       json:"is_admin"`
    CreatedAt time.Time     `bson:"created_at"     json:"created_at"`
    UsedAt    time.Time     `bson:"used_at,omitempty" json:"used_at,omitempty"`
}
```

---

## Node 节点

```go
type Node struct {
    ID           bson.ObjectID `bson:"_id,omitempty"  json:"id"`
    NodeID       string        `bson:"node_id"         json:"node_id"`      // 客户端生成的 UUID
    Name         string        `bson:"name"            json:"name"`
    Token        string        `bson:"token"           json:"token"`        // 预共享认证 Token
    Status       NodeStatus    `bson:"status"          json:"status"`
    Info         NodeInfo      `bson:"info"            json:"info"`
    Sources      []string      `bson:"sources"         json:"sources"`      // 负责的数据源
    LastSeenAt   time.Time     `bson:"last_seen_at"    json:"last_seen_at"`
    RegisteredAt  time.Time     `bson:"registered_at"   json:"registered_at"`
    TaskStats    NodeTaskStats `bson:"task_stats"      json:"task_stats"`   // 性能指标
}
```

### NodeStatus 节点状态

```go
const (
    NodeOnline  NodeStatus = "online"   // 在线
    NodeOffline NodeStatus = "offline"  // 离线
    NodeBusy    NodeStatus = "busy"     // 繁忙
    NodePaused  NodeStatus = "paused"   // 已暂停
)
```

### NodeInfo 节点信息

```go
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
```

### NodeTaskStats 任务统计

```go
type NodeTaskStats struct {
    TotalTasks      int       `bson:"total_tasks"           json:"total_tasks"`
    SuccessTasks    int       `bson:"success_tasks"         json:"success_tasks"`
    FailedTasks     int       `bson:"failed_tasks"          json:"failed_tasks"`
    AvgResponseTime int64     `bson:"avg_response_time_ms"  json:"avg_response_time_ms"`
    CurrentTasks    int       `bson:"-"                     json:"current_tasks,omitempty"` // 瞬时值，不持久化
    CPUPercent      float64   `bson:"cpu_percent,omitempty"  json:"cpu_percent,omitempty"`
    MemPercent      float64   `bson:"mem_percent,omitempty"  json:"mem_percent,omitempty"`
    UpdatedAt       time.Time `bson:"updated_at"            json:"updated_at"`
}

// 成功率计算
func (s NodeTaskStats) SuccessRate() float64 {
    if s.TotalTasks == 0 {
        return 1.0 // 新节点默认 100%
    }
    return float64(s.SuccessTasks) / float64(s.TotalTasks)
}

// 负载评分计算（越低越好）
func (s NodeTaskStats) LoadScore() float64 {
    score := float64(s.CurrentTasks) * 10.0

    // CPU 超过 70% 增加惩罚
    if s.CPUPercent > 70.0 {
        score += (s.CPUPercent - 70.0) * 2.0
    }

    // 内存超过 80% 增加惩罚
    if s.MemPercent > 80.0 {
        score += (s.MemPercent - 80.0) * 2.0
    }

    return score
}
```

---

## Task 任务

```go
type Task struct {
    ID             bson.ObjectID `bson:"_id,omitempty"          json:"id"`
    TaskID         string        `bson:"task_id"                json:"task_id"`         // UUID
    Type           TaskType      `bson:"type"                   json:"type"`
    Status         TaskStatus    `bson:"status"                 json:"status"`
    AssignedTo     string        `bson:"assigned_to,omitempty"   json:"assigned_to,omitempty"`
    Payload        []byte        `bson:"payload"                json:"payload"`         // JSON 编码的任务配置
    Result         []byte        `bson:"result,omitempty"        json:"result,omitempty"`
    Error          string        `bson:"error,omitempty"        json:"error,omitempty"`
    Progress       int           `bson:"progress"               json:"progress"`        // 0-100
    Priority       int           `bson:"priority"               json:"priority"`        // 0-100
    CreatedAt      time.Time     `bson:"created_at"             json:"created_at"`
    UpdatedAt      time.Time     `bson:"updated_at"             json:"updated_at"`
    FinishedAt     time.Time     `bson:"finished_at,omitempty"   json:"finished_at,omitempty"`

    // 重试跟踪
    RetryCount    int       `bson:"retry_count,omitempty"    json:"retry_count,omitempty"`
    MaxRetries    int       `bson:"max_retries,omitempty"    json:"max_retries,omitempty"`
    LastRetryAt   time.Time `bson:"last_retry_at,omitempty"  json:"last_retry_at,omitempty"`
    NextRetryAt   time.Time `bson:"next_retry_at,omitempty"  json:"next_retry_at,omitempty"`
    RetryErrors   []string  `bson:"retry_errors,omitempty"   json:"retry_errors,omitempty"`

    // 超时设置
    TimeoutSeconds int       `bson:"timeout_seconds,omitempty" json:"timeout_seconds,omitempty"`
    StartedAt      time.Time `bson:"started_at,omitempty"     json:"started_at,omitempty"`
}
```

### TaskType 任务类型

```go
const (
    TaskTypeVulnCrawl    TaskType = "vuln_crawl"     // 漏洞爬取
    TaskTypeArticleCrawl TaskType = "article_crawl"  // 文章爬取
)
```

### TaskStatus 任务状态

```go
const (
    TaskPending    TaskStatus = "pending"     // 等待调度
    TaskDispatched TaskStatus = "dispatched"  // 已分发
    TaskRunning    TaskStatus = "running"     // 执行中
    TaskDone       TaskStatus = "done"        // 已完成
    TaskFailed     TaskStatus = "failed"      // 失败
)
```

### 任务状态流转

```
pending → dispatched → running → done
                ↓         ↓
              failed    failed
```

### VulnCrawlPayload 漏洞爬取负载

```go
type VulnCrawlPayload struct {
    Sources      []string `json:"sources"`
    PageLimit   int      `json:"page_limit"`
    EnableGithub bool     `json:"enable_github"`
    Proxy        string   `json:"proxy,omitempty"`
}
```

### 优先级说明

| 优先级 | 值 | 使用场景 |
|--------|-----|----------|
| 紧急 | 100 | 安全事件应急响应 |
| 高 | 75 | 重要漏洞情报 |
| 中 | 50 | 常规爬取任务 |
| 低 | 25 | 后台批量任务 |
| 普通 | 0 | 默认值 |

---

## VulnRecord 漏洞记录

```go
type VulnRecord struct {
    ID           bson.ObjectID `bson:"_id,omitempty"  json:"id"`
    Key          string        `bson:"key"             json:"key"`             // 数据源唯一键
    Title        string        `bson:"title"           json:"title"`
    Description  string        `bson:"description"     json:"description"`
    Severity     SeverityLevel `bson:"severity"        json:"severity"`
    CVE          string        `bson:"cve"             json:"cve"`
    Disclosure   string        `bson:"disclosure"      json:"disclosure"`      // 披露日期
    Solutions    string        `bson:"solutions"       json:"solutions"`
    References   []string      `bson:"references"      json:"references"`
    Tags         []string      `bson:"tags"            json:"tags"`
    GithubSearch []string      `bson:"github_search"   json:"github_search"`
    From         string        `bson:"from"            json:"from"`
    Source       string        `bson:"source"          json:"source"`          // 爬虫名称
    URL          string        `bson:"url"             json:"url"`             // 原始链接
    Pushed       bool          `bson:"pushed"          json:"pushed"`          // 是否已推送
    ReportedBy   string        `bson:"reported_by"     json:"reported_by"`     // 节点 ID
    CreatedAt    time.Time     `bson:"created_at"      json:"created_at"`
    UpdatedAt    time.Time     `bson:"updated_at"      json:"updated_at"`
}
```

### SeverityLevel 严重程度

```go
const (
    SeverityLow      SeverityLevel = "低危"
    SeverityMedium   SeverityLevel = "中危"
    SeverityHigh     SeverityLevel = "高危"
    SeverityCritical SeverityLevel = "严重"
)
```

### Key 生成规则

`Key` 是数据源内的唯一标识，格式为 `{source}:{id}`，例如：
- `avd:CVE-2026-28472`
- `nvd:CVE-2026-12345`

---

## Article 文章

```go
type Article struct {
    ID          bson.ObjectID `bson:"_id,omitempty" json:"id"`
    Title       string        `bson:"title"          json:"title"`
    Summary     string        `bson:"summary"        json:"summary"`
    Content     string        `bson:"content"        json:"content"`
    Author      string        `bson:"author"         json:"author"`
    Source      string        `bson:"source"         json:"source"`         // 来源名称
    URL         string        `bson:"url"            json:"url"`
    Image       string        `bson:"image"          json:"image"`         // 封面图
    Tags        []string      `bson:"tags"           json:"tags"`
    Pushed      bool          `bson:"pushed"         json:"pushed"`
    ReportedBy   string        `bson:"reported_by"    json:"reported_by"`
    PublishedAt time.Time     `bson:"published_at"   json:"published_at"`
    CreatedAt   time.Time     `bson:"created_at"     json:"created_at"`
}
```

---

## PushChannel 推送渠道

```go
type PushChannel struct {
    ID        bson.ObjectID     `bson:"_id,omitempty" json:"id"`
    Name      string            `bson:"name"           json:"name"`
    Type      string            `bson:"type"           json:"type"`           // dingding/lark/wechat/webhook
    Config    map[string]string `bson:"config"         json:"config"`
    Enabled   bool              `bson:"enabled"        json:"enabled"`
    CreatedAt time.Time         `bson:"created_at"     json:"created_at"`
    UpdatedAt time.Time         `bson:"updated_at"     json:"updated_at"`
}
```

### 渠道类型

| 类型 | 说明 | 必需配置 |
|------|------|----------|
| `dingding` | 钉钉机器人 | webhook_url, secret |
| `lark` | 飞书机器人 | webhook_url, secret |
| `wechat` | 企业微信机器人 | webhook_url |
| `slack` | Slack Incoming Webhook | webhook_url |
| `telegram` | Telegram Bot | bot_token, chat_id |
| `webhook` | 通用 Webhook | webhook_url |
| `bark` | Bark 推送 | bark_url |

### 渠道配置示例

```go
// 钉钉
map[string]string{
    "webhook_url": "https://oapi.dingtalk.com/robot/send?access_token=xxx",
    "secret": "SECxxx",
}

// 飞书
map[string]string{
    "webhook_url": "https://open.feishu.cn/open-apis/bot/v2/hook/xxx",
    "secret": "xxx",
}

// 企业微信
map[string]string{
    "webhook_url": "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=xxx",
}

// Telegram
map[string]string{
    "bot_token": "123456:ABC-DEF",
    "chat_id": "-100123456789",
}

// Bark
map[string]string{
    "bark_url": "https://api.day.app/your-bark-key",
}
```

---

## AuditLog 审计日志

```go
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
```

### 操作类型

| Action | 说明 |
|--------|------|
| `login` | 用户登录 |
| `logout` | 用户登出 |
| `create_task` | 创建任务 |
| `stop_task` | 停止任务 |
| `delete_task` | 删除任务 |
| `create_channel` | 创建推送渠道 |
| `update_channel` | 更新推送渠道 |
| `delete_channel` | 删除推送渠道 |
| `create_node` | 创建节点 |
| `delete_node` | 删除节点 |
| `pause_node` | 暂停节点 |
| `resume_node` | 恢复节点 |

---

## Report 报告

```go
type Report struct {
    ID          bson.ObjectID `bson:"_id,omitempty" json:"id"`
    Title       string        `bson:"title"          json:"title"`
    Description string        `bson:"description"    json:"description"`
    Status      ReportStatus  `bson:"status"         json:"status"`
    Period      string        `bson:"period"         json:"period"`         // 如 "2026-03-01 ~ 2026-03-31"
    Content     string        `bson:"content,omitempty" json:"content,omitempty"`
    FilePath    string        `bson:"file_path,omitempty" json:"file_path,omitempty"`
    CreatedBy   bson.ObjectID `bson:"created_by"     json:"created_by"`
    CreatedAt   time.Time     `bson:"created_at"     json:"created_at"`
    UpdatedAt   time.Time     `bson:"updated_at"     json:"updated_at"`
}
```

### ReportStatus 报告状态

```go
const (
    ReportPending    ReportStatus = "pending"     // 生成中
    ReportGenerating ReportStatus = "generating"  // 生成中
    ReportDone       ReportStatus = "done"       // 已完成
    ReportFailed     ReportStatus = "failed"      // 生成失败
)
```

---

## 索引设计

### users 集合

```javascript
// username 唯一索引
{ "username": 1 }, { unique: true }

// email 唯一索引
{ "email": 1 }, { unique: true, sparse: true }
```

### nodes 集合

```javascript
// node_id 唯一索引
{ "node_id": 1 }, { unique: true }

// status + last_seen_at 复合索引（查询在线节点）
{ "status": 1, "last_seen_at": -1 }
```

### tasks 集合

```javascript
// task_id 唯一索引
{ "task_id": 1 }, { unique: true }

// status + created_at 复合索引（查询待处理任务）
{ "status": 1, "created_at": -1 }

// assigned_to + status 复合索引（查询节点任务）
{ "assigned_to": 1, "status": 1 }
```

### vuln_records 集合

```javascript
// key 唯一索引
{ "key": 1 }, { unique: true }

// source + cve 复合索引
{ "source": 1, "cve": 1 }

// severity + created_at 复合索引
{ "severity": 1, "created_at": -1 }

// cve 索引
{ "cve": 1 }

// pushed + created_at 复合索引
{ "pushed": 1, "created_at": -1 }

// 文本索引
{ "title": "text", "description": "text" }
```

### articles 集合

```javascript
// source + url 复合索引
{ "source": 1, "url": 1 }, { unique: true }

// published_at 索引
{ "published_at": -1 }

// pushed + created_at 复合索引
{ "pushed": 1, "created_at": -1 }

// 文本索引
{ "title": "text", "summary": "text" }
```

---

## 数据保留策略

| 数据类型 | 保留时间 | 说明 |
|----------|----------|------|
| tasks | 7 天 | 完成后保留一周 |
| dead-letters | 30 天 | 死信队列保留一个月 |
| audit_logs | 90 天 | 审计日志保留三个月 |
| vuln_records | 永久 | 漏洞记录永久保留 |
| articles | 永久 | 文章永久保留 |
