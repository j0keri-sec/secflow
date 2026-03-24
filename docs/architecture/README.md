# SecFlow 技术架构文档

## 目录

- [系统概述](#系统概述)
- [架构设计](#架构设计)
- [核心组件](#核心组件)
- [通信协议](#通信协议)
- [数据模型](#数据模型)
- [任务调度](#任务调度)
- [负载均衡](#负载均衡)
- [监控指标](#监控指标)

---

## 系统概述

### 项目背景

SecFlow 是一个分布式安全信息流平台，核心功能：
- 实时/定时爬取各厂商漏洞情报、安全资讯
- 数据存储在 MongoDB 中
- 推送至钉钉、飞书、企业微信等消息平台
- 多节点分布式执行爬取任务

### 设计目标

1. **高可用性**：客户端断线自动重连，任务失败自动重试
2. **可扩展性**：支持水平扩展多个客户端节点
3. **高性能**：Redis 任务队列，支持批量分发
4. **可观测性**：Prometheus 监控指标，结构化日志

---

## 架构设计

### 整体架构

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              【公网服务器】                                    │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                        secflow-server                               │   │
│  │                                                                      │   │
│  │  ┌─────────────────────────────────────────────────────────────┐   │   │
│  │  │                      Gin HTTP Server                         │   │   │
│  │  │  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐        │   │   │
│  │  │  │ Auth API │ │ Vuln API │ │Task API  │ │Push API  │        │   │   │
│  │  │  └──────────┘ └──────────┘ └──────────┘ └──────────┘        │   │   │
│  │  └─────────────────────────────────────────────────────────────┘   │   │
│  │                              │                                     │   │
│  │  ┌────────────────────────────▼────────────────────────────────┐   │   │
│  │  │                    WebSocket Hub                            │   │   │
│  │  │  - 任务下发 (task)                                           │   │   │
│  │  │  - 进度上报 (progress)                                      │   │   │
│  │  │  - 结果上报 (result)                                        │   │   │
│  │  │  - 心跳检测 (heartbeat)                                     │   │   │
│  │  └─────────────────────────────────────────────────────────────┘   │   │
│  │                              │                                     │   │
│  │  ┌────────────────────────────▼────────────────────────────────┐   │   │
│  │  │                   Task Scheduler                              │   │   │
│  │  │  ┌────────────┐  ┌────────────┐  ┌─────────────────────┐     │   │   │
│  │  │  │ Priority   │  │  Retry     │  │    Dead Letter     │     │   │   │
│  │  │  │ Queue      │  │  Queue     │  │    Queue           │     │   │   │
│  │  │  └────────────┘  └────────────┘  └─────────────────────┘     │   │   │
│  │  └─────────────────────────────────────────────────────────────┘   │   │
│  │                              │                                     │   │
│  │  ┌────────────────────────────▼────────────────────────────────┐   │   │
│  │  │                    Task Generator                             │   │   │
│  │  │  - 每30分钟生成漏洞爬取任务                                   │   │   │
│  │  │  - 每小时生成文章爬取任务                                     │   │   │
│  │  └─────────────────────────────────────────────────────────────┘   │   │
│  │                                                                      │   │
│  │  ┌─────────────────────────────────────────────────────────────┐   │   │
│  │  │                        Redis                                 │   │   │
│  │  │  - secflow:tasks:pending (List)     待处理任务队列           │   │   │
│  │  │  - secflow:tasks:priority (ZSet)    优先级队列              │   │   │
│  │  │  - secflow:tasks:retry (ZSet)       重试队列               │   │   │
│  │  │  - secflow:tasks:deadletter (List)  死信队列               │   │   │
│  │  │  - secflow:nodes:heartbeat (ZSet)   节点心跳               │   │   │
│  │  │  - secflow:tasks:progress (Hash)    任务进度               │   │   │
│  │  └─────────────────────────────────────────────────────────────┘   │   │
│  │                                                                      │   │
│  │  ┌─────────────────────────────────────────────────────────────┐   │   │
│  │  │                       MongoDB                                │   │   │
│  │  │  - users          用户表                                     │   │   │
│  │  │  - nodes          节点表                                     │   │   │
│  │  │  - tasks          任务表                                     │   │   │
│  │  │  - vuln_records   漏洞记录表                                 │   │   │
│  │  │  - articles       文章表                                     │   │   │
│  │  │  - push_channels  推送渠道表                                 │   │   │
│  │  └─────────────────────────────────────────────────────────────┘   │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────────┘
                                      │
                    WebSocket 长连接 + HTTP API (结果上报)
                                      │
┌─────────────────────────────────────────────────────────────────────────────┐
│                              【内网节点 × N】                                  │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                        secflow-client                                │   │
│  │                                                                      │   │
│  │  ┌─────────────────────────────────────────────────────────────┐   │   │
│  │  │                   WebSocket Client                           │   │   │
│  │  │  - 接收任务 (task)                                           │   │   │
│  │  │  - 心跳上报 (heartbeat)                                      │   │   │
│  │  │  - 进度上报 (progress)                                       │   │   │
│  │  │  - 结果上报 (result via HTTP)                                │   │   │
│  │  └─────────────────────────────────────────────────────────────┘   │   │
│  │                              │                                     │   │
│  │  ┌────────────────────────────▼────────────────────────────────┐   │   │
│  │  │                    Task Dispatcher                            │   │   │
│  │  │  - vuln_crawl → VulnEngine                                   │   │   │
│  │  │  - article_crawl → ArticleEngine                             │   │   │
│  │  └─────────────────────────────────────────────────────────────┘   │   │
│  │                              │                                     │   │
│  │  ┌─────────────────────────────────────────────────────────────┐   │   │
│  │  │                   Rod Browser Pool                           │   │   │
│  │  │  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐          │   │   │
│  │  │  │ Browser │ │ Browser │ │ Browser │ │ Browser │          │   │   │
│  │  │  │  Pool   │ │  Pool   │ │  Pool   │ │  Pool   │          │   │   │
│  │  │  └─────────┘ └─────────┘ └─────────┘ └─────────┘          │   │   │
│  │  └─────────────────────────────────────────────────────────────┘   │   │
│  │                              │                                     │   │
│  │  ┌────────────────────────────▼────────────────────────────────┐   │   │
│  │  │                   Grabbers (22个源)                           │   │   │
│  │  │  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐      │   │   │
│  │  │  │ avd-rod  │ │ seebug   │ │ ti-rod   │ │ kev-rod  │      │   │   │
│  │  │  └──────────┘ └──────────┘ └──────────┘ └──────────┘      │   │   │
│  │  │  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐      │   │   │
│  │  │  │ struts2  │ │ chaitin  │ │ oscs     │ │threatbook│      │   │   │
│  │  │  └──────────┘ └──────────┘ └──────────┘ └──────────┘      │   │   │
│  │  └─────────────────────────────────────────────────────────────┘   │   │
│  │                                                                      │   │
│  │  ┌─────────────────────────────────────────────────────────────┐   │   │
│  │  │                    SQLite (本地状态)                          │   │   │
│  │  │  - 节点配置                                                  │   │   │
│  │  │  - 任务状态缓存                                              │   │   │
│  │  └─────────────────────────────────────────────────────────────┘   │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 模块职责

| 模块 | 部署位置 | 职责 |
|------|----------|------|
| `secflow-server` | 公网服务器 | Web API、任务调度、数据存储、WebSocket Hub |
| `secflow-web` | 与 Server 同机 | 前端管理界面 |
| `secflow-client` | 内网节点 | 接收任务、执行爬取、上报结果 |

> **重要**：secflow-client 和 secflow-agent 已合并为同一个客户端程序。

---

## 核心组件

### 1. WebSocket Hub

位置：`secflow-server/internal/ws/hub.go`

**功能**：
- 管理所有客户端节点的长连接
- 消息类型：`task`, `task_cancel`, `ping`, `register`, `heartbeat`, `progress`, `result`, `pong`
- 自动处理节点连接/断开
- 批量任务分发

**核心结构**：

```go
type Hub struct {
    mu      sync.RWMutex
    clients map[string]*Client  // nodeID → Client
    taskCounts map[string]int   // 节点任务计数（负载均衡）
    registerCh   chan *Client
    unregisterCh chan *Client
    onMessage func(nodeID string, msg *Message)
    onConnect func(nodeID string)
    onDisconnect func(nodeID string)
}
```

### 2. 任务队列 (Redis)

位置：`secflow-server/internal/queue/queue.go`

**队列类型**：

| 队列 | 类型 | 用途 |
|------|------|------|
| `secflow:tasks:pending` | List | 普通任务队列 (FIFO) |
| `secflow:tasks:priority` | Sorted Set | 优先级队列 (score=priority) |
| `secflow:tasks:retry` | Sorted Set | 重试队列 (score=retry_at) |
| `secflow:tasks:deadletter` | List | 死信队列（永久失败） |
| `secflow:nodes:heartbeat` | Sorted Set | 节点心跳 (score=timestamp) |

**优先级定义**：
- `PriorityHigh = 100`：紧急安全事件
- `PriorityMedium = 50`：常规爬取任务（默认）
- `PriorityLow = 0`：后台批量任务

### 3. 任务调度器

位置：`secflow-server/internal/scheduler/scheduler.go`

**功能**：
- 从 Redis 队列中取出任务
- 分发给在线的客户端节点（智能负载均衡）
- 更新任务状态：`pending` → `dispatched` → `running` → `done/failed`
- 支持超时检测和自动重试

**调度策略**：
1. 优先从优先级队列取任务
2. 使用 `GetBestNodeIntelligent()` 选择最优节点
3. 支持批量分发（可配置批次大小）

### 4. 任务生成器

位置：`secflow-server/internal/scheduler/task_generator.go`

**功能**：
- 自动定时任务生成
- 支持的漏洞源（15个）：
  - avd-rod, seebug-rod, ti-rod, nox-rod
  - kev-rod, struts2-rod, chaitin-rod
  - oscs-rod, threatbook-rod, venustech-rod
  - nvd-rod, packetstorm-rod, exploitdb-rod
  - 360cert-rod, vulhub-rod

### 5. Rod Browser Pool

位置：`secflow-client/pkg/rodutil/browser.go`

**功能**：
- 管理 Chrome 浏览器实例池
- 复用 browser/page，避免频繁创建销毁
- 内置 WAF bypass（随机 UA、viewport、行为模拟）

---

## 通信协议

### WebSocket 消息格式

```json
// Server → Client: 任务下发
{
  "type": "task",
  "payload": {
    "task_id": "550e8400-e29b-41d4-a716-446655440000",
    "type": "vuln_crawl",
    "sources": ["avd-rod", "seebug-rod"],
    "page_limit": 3
  }
}

// Client → Server: 心跳上报
{
  "type": "heartbeat",
  "payload": {
    "node_id": "node-001",
    "info": {
      "cpu_percent": 45.2,
      "mem_percent": 62.8,
      "current_tasks": 2
    }
  }
}

// Client → Server: 进度上报
{
  "type": "progress",
  "payload": {
    "task_id": "550e8400-e29b-41d4-a716-446655440000",
    "progress": 50,
    "message": "Processing avd-rod..."
  }
}

// Client → Server: 结果上报
{
  "type": "result",
  "payload": {
    "task_id": "550e8400-e29b-41d4-a716-446655440000",
    "status": "done",
    "data": [
      {
        "key": "avd:CVE-2024-1234",
        "title": "XXX 漏洞",
        "cve": "CVE-2024-1234",
        "severity": "高危",
        "source": "avd-rod"
      }
    ]
  }
}
```

### HTTP API (结果上报)

```
POST /api/v1/report
Authorization: Bearer <node_token>
Content-Type: application/json

{
  "task_id": "550e8400-e29b-41d4-a716-446655440000",
  "node_id": "node-001",
  "status": "done",
  "data": [...]
}
```

---

## 数据模型

### MongoDB Collections

#### users (用户表)

```go
type User struct {
    ID           bson.ObjectID `bson:"_id,omitempty"`
    Username     string        `bson:"username"`
    Email        string        `bson:"email"`
    PasswordHash string        `bson:"password_hash"`
    Role         RoleType      `bson:"role"`         // admin | editor | viewer
    InviteCode   string        `bson:"invite_code"`
    Avatar       string        `bson:"avatar"`
    Active       bool          `bson:"active"`
    CreatedAt    time.Time     `bson:"created_at"`
    UpdatedAt    time.Time     `bson:"updated_at"`
}
```

#### nodes (节点表)

```go
type Node struct {
    ID          bson.ObjectID `bson:"_id,omitempty"`
    NodeID      string        `bson:"node_id"`       // 客户端生成的 UUID
    Name        string        `bson:"name"`
    Token       string        `bson:"token"`         // 预共享认证 token
    Status      NodeStatus    `bson:"status"`       // online | offline | busy | paused
    Info        NodeInfo      `bson:"info"`          // 硬件/OS 信息
    Sources     []string      `bson:"sources"`       // 该节点负责的数据源
    TaskStats   NodeTaskStats `bson:"task_stats"`   // 性能指标
    LastSeenAt  time.Time     `bson:"last_seen_at"`
    RegisteredAt time.Time    `bson:"registered_at"`
}
```

#### tasks (任务表)

```go
type Task struct {
    ID         bson.ObjectID `bson:"_id,omitempty"`
    TaskID     string        `bson:"task_id"`        // UUID
    Type       TaskType      `bson:"type"`          // vuln_crawl | article_crawl
    Status     TaskStatus    `bson:"status"`        // pending | dispatched | running | done | failed
    AssignedTo string        `bson:"assigned_to"`    // 分配的节点 ID
    Payload    []byte        `bson:"payload"`        // JSON 编码的任务配置
    Result     []byte        `bson:"result"`         // JSON 编码的执行结果
    Progress   int           `bson:"progress"`       // 0-100
    Priority   int           `bson:"priority"`      // 优先级 0-100
    RetryCount int           `bson:"retry_count"`   // 重试次数
    MaxRetries int           `bson:"max_retries"`   // 最大重试次数
    TimeoutSeconds int       `bson:"timeout_seconds"`
    CreatedAt  time.Time     `bson:"created_at"`
    UpdatedAt  time.Time     `bson:"updated_at"`
    FinishedAt time.Time     `bson:"finished_at"`
}
```

#### vuln_records (漏洞记录表)

```go
type VulnRecord struct {
    ID           bson.ObjectID `bson:"_id,omitempty"`
    Key          string        `bson:"key"`           // source:unique_id
    Title        string        `bson:"title"`
    Description  string        `bson:"description"`
    Severity     SeverityLevel `bson:"severity"`      // 低危 | 中危 | 高危 | 严重
    CVE          string        `bson:"cve"`
    Disclosure   string        `bson:"disclosure"`
    Solutions    string        `bson:"solutions"`
    References   []string      `bson:"references"`
    Tags         []string      `bson:"tags"`
    GithubSearch []string      `bson:"github_search"`
    From         string        `bson:"from"`
    Source       string        `bson:"source"`        // grabber name
    URL          string        `bson:"url"`
    Pushed       bool          `bson:"pushed"`
    ReportedBy   string        `bson:"reported_by"`    // 上报的节点 ID
    CreatedAt    time.Time     `bson:"created_at"`
    UpdatedAt    time.Time     `bson:"updated_at"`
}
```

#### articles (文章表)

```go
type Article struct {
    ID          bson.ObjectID `bson:"_id,omitempty"`
    Title       string        `bson:"title"`
    Summary     string        `bson:"summary"`
    Content     string        `bson:"content"`
    Author      string        `bson:"author"`
    Source      string        `bson:"source"`
    URL         string        `bson:"url"`
    Image       string        `bson:"image"`
    Tags        []string      `bson:"tags"`
    Pushed      bool          `bson:"pushed"`
    ReportedBy  string        `bson:"reported_by"`
    PublishedAt time.Time     `bson:"published_at"`
    CreatedAt   time.Time     `bson:"created_at"`
}
```

---

## 任务调度

### 调度流程

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                            任务生命周期                                       │
└─────────────────────────────────────────────────────────────────────────────┘

[创建任务]
    │
    ▼
[pending] ─────────────────────────────────────────────────┐
    │                                                     │
    │ 调度器取出任务                                        │
    ▼                                                     │
[dispatched]  ────────────────────────────────────────────┤
    │                                                     │
    │ WebSocket 发送到客户端                                │
    ▼                                                     │
[running] ────────────────────────────────────────────────┤
    │                    │                                 │
    │ 客户端执行中         │ 超时/失败                        │
    │ 进度上报            ▼                                 │
    │                [retry] ───→ [deadletter]             │
    │                    │                                 │
    │                    │ 重试成功                         │
    │                    ▼                                 │
    │                 [running]                            │
    │                                                     │
    ▼                                                     │
[done/failed] ◀───────────────────────────────────────────┘
```

### 重试策略

```go
// 指数退避算法
baseDelay := time.Duration(1<<uint(retryCount)) * time.Second
if baseDelay > time.Hour {
    baseDelay = time.Hour
}
// 添加 0-20% 随机抖动
jitter := time.Duration(float64(baseDelay) * 0.2 * rand.Float64())
```

### 死信队列

当任务超过最大重试次数后，移入死信队列：
- 保留 30 天
- 支持手动重试
- 支持删除

---

## 负载均衡

### 智能负载均衡算法

```go
func calculateNodeScore(stats NodeTaskStats) float64 {
    score := 100.0 // 基础分数

    // 1. 任务负载因子 (30分)
    score -= float64(stats.CurrentTasks) * 10.0

    // 2. CPU 使用率惩罚 (20分)
    if stats.CPUPercent > 70.0 {
        score -= (stats.CPUPercent - 70.0) * 0.5
    }

    // 3. 内存使用率惩罚 (20分)
    if stats.MemPercent > 80.0 {
        score -= (stats.MemPercent - 80.0) * 0.5
    }

    // 4. 历史成功率奖励 (30分)
    score += stats.SuccessRate() * 30.0

    // 5. 平均响应时间惩罚
    avgSeconds := float64(stats.AvgResponseTime) / 1000.0
    if avgSeconds > 30.0 {
        score -= (avgSeconds - 30.0) * 0.1
    }

    return score
}
```

### 评分权重

| 指标 | 权重 | 说明 |
|------|------|------|
| 当前任务数 | 30% | 任务越少越好 |
| CPU 使用率 | 20% | 超过 70% 开始惩罚 |
| 内存使用率 | 20% | 超过 80% 开始惩罚 |
| 历史成功率 | 30% | 成功率越高越好 |

---

## 监控指标

### Prometheus 指标

#### 任务指标

| 指标 | 类型 | 说明 |
|------|------|------|
| `secflow_tasks_total` | Counter | 任务总数（按类型、状态分组） |
| `secflow_tasks_duration_seconds` | Histogram | 任务执行时长 |
| `secflow_tasks_retry_total` | Counter | 重试次数 |
| `secflow_queue_length` | Gauge | 队列长度（pending/priority/retry） |

#### 节点指标

| 指标 | 类型 | 说明 |
|------|------|------|
| `secflow_nodes_total` | Gauge | 在线节点数 |
| `secflow_node_tasks` | Gauge | 节点当前任务数 |
| `secflow_node_cpu_percent` | Gauge | 节点 CPU 使用率 |
| `secflow_node_memory_percent` | Gauge | 节点内存使用率 |

#### API 指标

| 指标 | 类型 | 说明 |
|------|------|------|
| `secflow_http_requests_total` | Counter | HTTP 请求数（按路径、方法分组） |
| `secflow_http_duration_seconds` | Histogram | HTTP 请求延迟 |

### Redis 监控命令

```bash
# 查看队列长度
LLEN secflow:tasks:pending
ZCARD secflow:tasks:priority
ZCARD secflow:tasks:retry

# 查看节点心跳
ZCARD secflow:nodes:heartbeat
ZRANGEBYSCORE secflow:nodes:heartbeat -inf {cutoff}

# 查看任务进度
HGET secflow:tasks:progress {task_id}

# 查看 Redis 内存
INFO memory
```

---

## 扩展开发

### 添加新的 Grabber 源

1. 在 `secflow-client/pkg/vulngrabber/` 创建 `<source>_rod.go`
2. 实现 `Grabber` 接口
3. 在 `secflow-client/pkg/vulngrabber/registry.go` 中注册
4. 更新服务端的 `defaultVulnSources`

### 添加新的任务类型

1. 在 `secflow-server/internal/model/model.go` 添加新的 `TaskType`
2. 在 `secflow-client/internal/task/dispatcher.go` 添加处理逻辑
3. 在 `secflow-client/internal/engine/engine.go` 添加执行器
4. 在 `secflow-server/internal/api/handler/task.go` 添加 API 端点

### 添加新的推送渠道

1. 在 `secflow-server/pkg/pusher/` 创建 `<channel>.go`
2. 实现 `Pusher` 接口
3. 在 `secflow-server/pkg/pusher/factory.go` 注册
4. 前端添加渠道配置界面
