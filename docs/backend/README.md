# SecFlow 后端开发文档

## 目录

- [项目结构](#项目结构)
- [代码规范](#代码规范)
- [核心模块](#核心模块)
- [API 开发](#api-开发)
- [数据库操作](#数据库操作)
- [WebSocket 开发](#websocket-开发)
- [测试](#测试)
- [常见问题](#常见问题)

---

## 项目结构

```
secflow-server/
├── cmd/server/
│   └── main.go              # 程序入口
│
├── config/
│   ├── config.go            # 配置加载
│   └── config.yaml          # 配置文件
│
├── internal/                 # 私有包（不导出）
│   ├── api/
│   │   ├── router.go        # 路由定义
│   │   ├── handler/         # HTTP 处理器
│   │   │   ├── auth.go      # 认证
│   │   │   ├── vuln.go      # 漏洞
│   │   │   ├── article.go   # 文章
│   │   │   ├── task.go      # 任务
│   │   │   ├── node.go      # 节点
│   │   │   ├── push_channel.go  # 推送渠道
│   │   │   └── response.go  # 统一响应
│   │   └── middleware/
│   │       ├── middleware.go    # 中间件
│   │       └── metrics.go       # 监控指标
│   │
│   ├── model/
│   │   └── model.go         # MongoDB 数据模型
│   │
│   ├── repository/
│   │   ├── db.go            # 数据库连接
│   │   ├── repository.go    # CRUD 操作
│   │   └── extra.go        # 扩展查询
│   │
│   ├── queue/
│   │   └── queue.go        # Redis 任务队列
│   │
│   ├── scheduler/
│   │   ├── scheduler.go    # 任务调度器
│   │   └── task_generator.go # 任务生成器
│   │
│   └── ws/
│       └── hub.go          # WebSocket Hub
│
└── pkg/                     # 公共包（可导出）
    ├── auth/
    │   └── auth.go         # JWT 认证
    │
    ├── logger/
    │   └── logger.go      # 日志配置
    │
    └── pusher/              # 推送服务
        ├── interface.go    # 推送接口
        ├── factory.go      # 推送器工厂
        ├── service.go     # 推送服务
        ├── template.go    # 消息模板
        ├── dingding.go    # 钉钉
        ├── lark.go        # 飞书
        ├── wechat_work.go # 企业微信
        ├── slack.go       # Slack
        ├── telegram.go    # Telegram
        └── webhook.go     # Webhook
```

---

## 代码规范

### Go 编码规范

遵循 [projectdiscovery/nuclei](https://github.com/projectdiscovery/nuclei) 的代码风格：

1. **清晰优先**：代码意图对读者清楚，无歧义命名
2. **简约风格**：以最简单方式完成，不过度抽象
3. **高信噪比**：无冗余代码，无 magic number
4. **错误处理**：使用 `fmt.Errorf` 包装错误，格式为 `"doing something: %w"`

### 命名规范

```go
// ✅ 好的命名
func (f *Fetcher) FetchVulns(ctx context.Context, source string) ([]*Vuln, error)
func (q *Queue) EnqueueTask(ctx context.Context, task *Task) error

// ❌ 避免的命名
func (fetcher *VulnFetcher) GetVulnsFromSource(sourceUrl string) ([]*Model, error)
```

### 错误处理

```go
// ✅ 正确：使用 fmt.Errorf 包装
if err != nil {
    return nil, fmt.Errorf("fetching %s: %w", source, err)
}

// ✅ 正确：使用 errors.Is/As 检查
if errors.Is(err, redis.Nil) {
    return nil, nil // Not found
}

// ❌ 错误：忽略错误
data, _ := json.Marshal(v)
```

### Context 传递

```go
// ✅ 正确：使用 context.Context
func (s *Service) FetchData(ctx context.Context, url string) error {
    req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
    if err != nil {
        return fmt.Errorf("creating request: %w", err)
    }
    // ...
}

// ❌ 错误：不传递 context
func (s *Service) FetchData(url string) error {
    req, err := http.NewRequest(http.MethodGet, url, nil) // 缺少 context
    // ...
}
```

---

## 核心模块

### 1. 配置管理

位置：`config/config.go`

```go
type Config struct {
    Server   ServerConfig   `yaml:"server"`
    MongoDB  MongoDBConfig  `yaml:"mongodb"`
    Redis    RedisConfig    `yaml:"redis"`
    JWT      JWTConfig      `yaml:"jwt"`
    Node     NodeConfig     `yaml:"node"`
    Log      LogConfig      `yaml:"log"`
    Scheduler SchedulerConfig `yaml:"scheduler"`
}

type ServerConfig struct {
    Host string `yaml:"host"`
    Port int    `yaml:"port"`
    Mode string `yaml:"mode"` // debug | release
}

type MongoDBConfig struct {
    URI      string `yaml:"uri"`
    Database string `yaml:"database"`
}
```

**配置加载**：

```go
func LoadConfig(path string) (*Config, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, fmt.Errorf("reading config: %w", err)
    }

    var cfg Config
    if err := yaml.Unmarshal(data, &cfg); err != nil {
        return nil, fmt.Errorf("parsing config: %w", err)
    }

    return &cfg, nil
}
```

### 2. 数据库连接

位置：`internal/repository/db.go`

```go
func NewDB(ctx context.Context, cfg *MongoDBConfig) (*mongo.Database, error) {
    client, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.URI))
    if err != nil {
        return nil, fmt.Errorf("connecting to MongoDB: %w", err)
    }

    // Ping 检查连接
    if err := client.Database(cfg.Database).Client().Ping(ctx, nil); err != nil {
        return nil, fmt.Errorf("pinging MongoDB: %w", err)
    }

    return client.Database(cfg.Database), nil
}
```

### 3. 路由定义

位置：`internal/api/router.go`

```go
func Router(
    authSvc *auth.Service,
    vulnH *handler.VulnHandler,
    nodeH *handler.NodeHandler,
    taskH *handler.TaskHandler,
    // ... 其他 handler
) *gin.Engine {
    r := gin.New()
    r.Use(middleware.Recovery())
    r.Use(middleware.CORS())

    v1 := r.Group("/api/v1")

    // 认证路由
    authGroup := v1.Group("/auth")
    {
        authGroup.POST("/register", authH.Register)
        authGroup.POST("/login", authH.Login)
    }

    // 需要 JWT 认证的路由
    api := v1.Group("", middleware.JWTAuth(authSvc))
    {
        vulns := api.Group("/vulns")
        vulns.GET("", vulnH.List)
        vulns.GET("/:id", vulnH.Get)
        // ...
    }

    return r
}
```

---

## API 开发

### Handler 模板

```go
// internal/api/handler/example.go
package handler

type ExampleHandler struct {
    repo *repository.Repository
    auth *auth.Service
}

type ListRequest struct {
    Page     int    `form:"page" binding:"min=1"`
    PageSize int    `form:"page_size" binding:"min=1,max=100"`
    Keyword  string `form:"keyword"`
}

type ListResponse struct {
    Items []*Example `json:"items"`
    Total int64      `json:"total"`
    Page  int        `json:"page"`
}

// List 获取列表
// @Summary 获取列表
// @Tags example
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Success 200 {object} Response{data=ListResponse}
// @Router /examples [get]
func (h *ExampleHandler) List(c *gin.Context) {
    var req ListRequest
    if err := c.ShouldBindQuery(&req); err != nil {
        Error(c, ErrInvalidParams.WithDetail(err.Error()))
        return
    }

    // 设置默认值
    if req.Page <= 0 {
        req.Page = 1
    }
    if req.PageSize <= 0 {
        req.PageSize = 20
    }

    // 业务逻辑
    items, total, err := h.repo.ListExamples(c.Request.Context(), req.Page, req.PageSize, req.Keyword)
    if err != nil {
        Error(c, ErrInternal.WithDetail(err.Error()))
        return
    }

    Success(c, ListResponse{
        Items: items,
        Total: total,
        Page:  req.Page,
    })
}
```

### 统一响应

```go
// internal/api/handler/response.go
type Response struct {
    Code int         `json:"code"`
    Msg  string      `json:"msg"`
    Data interface{} `json:"data,omitempty"`
}

func Success(c *gin.Context, data interface{}) {
    c.JSON(200, Response{
        Code: 0,
        Msg:  "success",
        Data: data,
    })
}

func Error(c *gin.Context, err *AppError) {
    c.JSON(err.Status, Response{
        Code: err.Code,
        Msg:  err.Msg,
    })
}
```

### 中间件

```go
// internal/api/middleware/middleware.go

// JWTAuth JWT 认证中间件
func JWTAuth(authSvc *auth.Service) gin.HandlerFunc {
    return func(c *gin.Context) {
        token := c.GetHeader("Authorization")
        if token == "" {
            c.AbortWithStatusJSON(401, gin.H{"code": 401, "msg": "missing token"})
            return
        }

        // 去掉 "Bearer " 前缀
        token = strings.TrimPrefix(token, "Bearer ")

        claims, err := authSvc.ValidateToken(token)
        if err != nil {
            c.AbortWithStatusJSON(401, gin.H{"code": 401, "msg": "invalid token"})
            return
        }

        c.Set("user_id", claims.UserID)
        c.Set("username", claims.Username)
        c.Next()
    }
}

// RequireRole 角色权限中间件
func RequireRole(roles ...string) gin.HandlerFunc {
    return func(c *gin.Context) {
        userRole, _ := c.Get("role")
        for _, role := range roles {
            if userRole == role {
                c.Next()
                return
            }
        }
        c.AbortWithStatusJSON(403, gin.H{"code": 403, "msg": "forbidden"})
    }
}
```

---

## 数据库操作

### Repository 模式

```go
// internal/repository/repository.go
type Repository struct {
    db *mongo.Database
}

func New(db *mongo.Database) *Repository {
    return &Repository{db: db}
}

// ListVulns 查询漏洞列表
func (r *Repository) ListVulns(ctx context.Context, page, size int, filter bson.M) ([]*model.VulnRecord, int64, error) {
    coll := r.db.Collection(model.CollVulnRecords)

    // 计算总数
    total, err := coll.CountDocuments(ctx, filter)
    if err != nil {
        return nil, 0, fmt.Errorf("counting vulns: %w", err)
    }

    // 查询列表
    skip := (page - 1) * size
    cursor, err := coll.Find(ctx, filter,
        options.Find().
            SetSkip(int64(skip)).
            SetLimit(int64(size)).
            SetSort(bson.D{{Key: "created_at", Value: -1}}),
    )
    if err != nil {
        return nil, 0, fmt.Errorf("finding vulns: %w", err)
    }
    defer cursor.Close(ctx)

    var vulns []*model.VulnRecord
    if err := cursor.All(ctx, &vulns); err != nil {
        return nil, 0, fmt.Errorf("decoding vulns: %w", err)
    }

    return vulns, total, nil
}

// CreateVuln 创建漏洞记录
func (r *Repository) CreateVuln(ctx context.Context, vuln *model.VulnRecord) error {
    coll := r.db.Collection(model.CollVulnRecords)

    vuln.CreatedAt = time.Now()
    vuln.UpdatedAt = time.Now()

    _, err := coll.InsertOne(ctx, vuln)
    if err != nil {
        return fmt.Errorf("inserting vuln: %w", err)
    }

    return nil
}

// UpsertVuln 批量 upsert 漏洞
func (r *Repository) UpsertVuln(ctx context.Context, vuln *model.VulnRecord) error {
    coll := r.db.Collection(model.CollVulnRecords)

    vuln.UpdatedAt = time.Now()

    filter := bson.M{"key": vuln.Key}
    update := bson.M{
        "$set": vuln,
        "$setOnInsert": bson.M{"created_at": time.Now()},
    }

    _, err := coll.UpdateOne(ctx, filter, update, options.Update().SetUpsert(true))
    if err != nil {
        return fmt.Errorf("upserting vuln: %w", err)
    }

    return nil
}
```

### MongoDB 索引

```go
// 在程序启动时创建索引
func ensureIndexes(ctx context.Context, db *mongo.Database) error {
    // vuln_records 索引
    vulns := db.Collection(model.CollVulnRecords)
    vulns.Indexes().CreateMany(ctx, []mongo.IndexModel{
        {Keys: bson.D{{Key: "key", Value: 1}}, Options: options.Index().SetUnique(true)},
        {Keys: bson.D{{Key: "source", Value: 1}}},
        {Keys: bson.D{{Key: "severity", Value: 1}}},
        {Keys: bson.D{{Key: "created_at", Value: -1}}},
        {Keys: bson.D{{Key: "cve", Value: 1}}},
    })

    // tasks 索引
    tasks := db.Collection(model.CollTasks)
    tasks.Indexes().CreateMany(ctx, []mongo.IndexModel{
        {Keys: bson.D{{Key: "task_id", Value: 1}}, Options: options.Index().SetUnique(true)},
        {Keys: bson.D{{Key: "status", Value: 1}}},
        {Keys: bson.D{{Key: "created_at", Value: -1}}},
    })

    // nodes 索引
    nodes := db.Collection(model.CollNodes)
    nodes.Indexes().CreateMany(ctx, []mongo.IndexModel{
        {Keys: bson.D{{Key: "node_id", Value: 1}}, Options: options.Index().SetUnique(true)},
        {Keys: bson.D{{Key: "status", Value: 1}}},
    })

    return nil
}
```

---

## WebSocket 开发

### Hub 使用

```go
// cmd/server/main.go
hub := ws.NewHub(
    onMessage:    handleWSMessage,
    onConnect:    handleNodeConnect,
    onDisconnect: handleNodeDisconnect,
)

// 发送任务到指定节点
taskMsg := &ws.Message{
    Type:    ws.MsgTypeTask,
    Payload: payloadBytes,
}
hub.Send(nodeID, taskMsg)

// 广播消息到所有节点
hub.Broadcast(&ws.Message{
    Type:    ws.MsgTypePing,
    Payload: nil,
})

// 获取所有在线节点
onlineNodes := hub.ConnectedNodes()
```

### 消息处理

```go
func handleWSMessage(nodeID string, msg *ws.Message) {
    switch msg.Type {
    case ws.MsgTypeRegister:
        var payload ws.RegisterPayload
        if err := json.Unmarshal(msg.Payload, &payload); err != nil {
            return
        }
        handleNodeRegister(nodeID, &payload)

    case ws.MsgTypeHeartbeat:
        var payload ws.HeartbeatPayload
        if err := json.Unmarshal(msg.Payload, &payload); err != nil {
            return
        }
        handleHeartbeat(nodeID, &payload)

    case ws.MsgTypeProgress:
        var payload ws.ProgressPayload
        if err := json.Unmarshal(msg.Payload, &payload); err != nil {
            return
        }
        handleProgress(nodeID, &payload)

    case ws.MsgTypeResult:
        var payload ws.ResultPayload
        if err := json.Unmarshal(msg.Payload, &payload); err != nil {
            return
        }
        handleResult(nodeID, &payload)

    case ws.MsgTypePong:
        // 心跳响应，无需处理
    }
}
```

---

## 测试

### 单元测试

```go
// internal/repository/repository_test.go
package repository

import (
    "context"
    "testing"
    "time"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestRepository_ListVulns(t *testing.T) {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    repo := setupTestRepo(t)

    // 创建测试数据
    vuln := &model.VulnRecord{
        Key:      "test:vuln-001",
        Title:    "Test Vulnerability",
        Severity: model.SeverityHigh,
        Source:   "test",
    }
    err := repo.CreateVuln(ctx, vuln)
    require.NoError(t, err)

    // 测试查询
    vulns, total, err := repo.ListVulns(ctx, 1, 10, nil)
    require.NoError(t, err)
    assert.Equal(t, int64(1), total)
    assert.Len(t, vulns, 1)
}
```

### 集成测试

```go
// integration_test.go
func TestFullWorkflow(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }

    // 启动测试服务
    srv := startTestServer(t)
    defer srv.Close()

    // 创建客户端
    client := NewClient(srv.URL)

    // 测试登录
    token, err := client.Login("admin", "password")
    require.NoError(t, err)

    // 测试创建任务
    task, err := client.CreateTask(&TaskConfig{
        Type: "vuln_crawl",
        Sources: []string{"avd-rod"},
    })
    require.NoError(t, err)
    assert.NotEmpty(t, task.ID)
}
```

---

## 常见问题

### Q: MongoDB 连接失败

```
error: connecting to MongoDB: server selection error: context canceled
```

**解决方案**：
1. 检查 MongoDB 服务是否启动：`brew services list | grep mongodb`
2. 检查 URI 是否正确：`mongodb://user:pass@host:port/db?authSource=admin`
3. 检查网络连接：`mongosh "mongodb://host:port"`

### Q: Redis 连接失败

```
error: connecting to Redis: connection refused
```

**解决方案**：
1. 检查 Redis 服务：`brew services list | grep redis`
2. 测试连接：`redis-cli ping`（应返回 PONG）
3. 检查密码配置

### Q: JWT Token 无效

```
error: invalid token
```

**解决方案**：
1. 检查 JWT secret 是否一致
2. 检查 token 是否过期
3. 重新登录获取新 token

### Q: 任务一直处于 pending 状态

**可能原因**：
1. 没有在线的客户端节点
2. 调度器未启动
3. Redis 队列为空

**排查步骤**：
```bash
# 1. 检查节点状态
curl http://localhost:8080/api/v1/nodes

# 2. 检查队列长度
redis-cli LLEN secflow:tasks:pending

# 3. 查看服务端日志
tail -f server.log
```
