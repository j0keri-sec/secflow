# SecFlow 客户端开发文档

## 目录

- [项目结构](#项目结构)
- [配置说明](#配置说明)
- [核心模块](#核心模块)
- [爬虫开发](#爬虫开发)
- [任务处理](#任务处理)
- [结果上报](#结果上报)
- [调试技巧](#调试技巧)

---

## 项目结构

```
secflow-client/
├── cmd/client/
│   └── main.go              # 程序入口
│
├── internal/                 # 私有包
│   ├── config/
│   │   └── config.go        # 配置加载
│   │
│   ├── db/
│   │   └── db.go            # SQLite 本地存储
│   │
│   ├── engine/
│   │   └── engine.go        # 执行引擎
│   │
│   ├── task/
│   │   └── dispatcher.go    # 任务分发
│   │
│   └── ws/
│       └── client.go        # WebSocket 客户端
│
├── pkg/
│   ├── vulngrabber/          # 漏洞爬虫
│   │   ├── grabber.go       # 接口定义
│   │   ├── registry.go      # 注册表
│   │   ├── sources.go       # 源配置
│   │   ├── rod_base.go      # Rod 基类
│   │   │   ├── avd_rod.go
│   │   │   ├── seebug_rod.go
│   │   │   ├── ti_rod.go
│   │   │   ├── kev_rod.go
│   │   │   └── ...
│   │   └── http_base.go     # HTTP 基类
│   │       ├── kev.go
│   │       └── chaitin.go
│   │
│   ├── articlegrabber/       # 文章爬虫
│   │   ├── grabber.go
│   │   ├── rod_base.go
│   │   ├── qianxin_weekly.go
│   │   └── venustech.go
│   │
│   ├── rodutil/
│   │   ├── browser.go       # 浏览器池
│   │   ├── bypass.go        # WAF 绕过
│   │   └── helpers.go       # 辅助函数
│   │
│   └── httputil/
│       └── client.go        # HTTP 客户端
│
├── client.yaml               # 配置文件
├── client.example.yaml       # 配置示例
└── go.mod
```

---

## 配置说明

### 配置文件 (client.yaml)

```yaml
# 服务端配置
server:
  api_url: "http://localhost:8080/api/v1"    # 服务端 API 地址
  ws_url: "ws://localhost:8080/api/v1/ws/node"  # WebSocket 地址
  token_key: "secflow-node-token-2026"       # 节点认证密钥

# 本地数据库
db_path: "./secflow.db"

# 心跳间隔
heartbeat_interval: 30s

# 日志配置
log_level: "info"    # debug | info | warn | error
log_path: ""         # 空则输出到 stdout

# 节点名称
name: "my-node"

# 节点 ID（自动生成，通常不需要修改）
node_id: "550e8400-e29b-41d4-a716-446655440000"

# 数据源（为空表示使用所有源）
sources: []
```

### 环境变量

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `SECFLOW_API_URL` | 服务端 API 地址 | - |
| `SECFLOW_WS_URL` | WebSocket 地址 | - |
| `SECFLOW_TOKEN_KEY` | 认证密钥 | - |
| `SECFLOW_NODE_NAME` | 节点名称 | - |
| `SECFLOW_LOG_LEVEL` | 日志级别 | info |
| `SECFLOW_DB_PATH` | 数据库路径 | ./secflow.db |

---

## 核心模块

### 1. WebSocket 客户端

位置：`internal/ws/client.go`

```go
type Client struct {
    ctx        context.Context
    cancel     context.CancelFunc
    config     *Config
    hub        *ws.Hub        // 服务端的 Hub（连接时创建）
    wsConn     *websocket.Conn // WebSocket 连接
    onMessage  func(msg *ws.Message)
    onConnect  func()
    onDisconnect func(err error)
}

func NewClient(config *Config) *Client {
    ctx, cancel := context.WithCancel(context.Background())
    return &Client{
        ctx:    ctx,
        cancel: cancel,
        config: config,
    }
}

func (c *Client) Connect() error {
    // 1. 建立 WebSocket 连接
    url := fmt.Sprintf("%s?token=%s", c.config.WSURL, c.config.Token)
    conn, _, err := websocket.DefaultDialer.DialContext(c.ctx, url, nil)
    if err != nil {
        return fmt.Errorf("dialing websocket: %w", err)
    }
    c.wsConn = conn

    // 2. 发送注册消息
    regMsg := &ws.Message{
        Type: ws.MsgTypeRegister,
        Payload: mustMarshal(ws.RegisterPayload{
            NodeID:  c.config.NodeID,
            Token:   c.config.Token,
            Name:    c.config.Name,
            Sources: c.config.Sources,
        }),
    }
    if err := c.wsConn.WriteJSON(regMsg); err != nil {
        return fmt.Errorf("sending register: %w", err)
    }

    // 3. 启动读写循环
    go c.readLoop()
    go c.writeLoop()
    go c.heartbeatLoop()

    return nil
}
```

### 2. 任务分发器

位置：`internal/task/dispatcher.go`

```go
type Dispatcher struct {
    engine  *engine.Engine
    db      *db.DB
}

func NewDispatcher(cfg *Config, eng *engine.Engine, database *db.DB) *Dispatcher {
    return &Dispatcher{
        engine: eng,
        db:     database,
    }
}

func (d *Dispatcher) Dispatch(task *model.Task) error {
    switch task.Type {
    case model.TaskTypeVulnCrawl:
        return d.engine.RunVulnCrawl(d.ctx, task)
    case model.TaskTypeArticleCrawl:
        return d.engine.RunArticleCrawl(d.ctx, task)
    default:
        return fmt.Errorf("unknown task type: %s", task.Type)
    }
}
```

### 3. 执行引擎

位置：`internal/engine/engine.go`

```go
type Engine struct {
    vulnGrabbers    map[string]vulngrabber.Grabber
    articleGrabbers map[string]articlegrabber.Grabber
    browserPool     *rodutil.BrowserPool
    httpClient      *httputil.Client
    reporter        *reporter.HTTPReporter
}

func NewEngine(config *Config) (*Engine, error) {
    // 初始化浏览器池
    pool, err := rodutil.NewBrowserPool(4)
    if err != nil {
        return nil, fmt.Errorf("creating browser pool: %w", err)
    }

    // 初始化漏洞爬虫
    vulnGrabbers := vulngrabber.NewRegistry()
    for name, grabber := range vulnGrabbers {
        log.Info().Str("source", name).Msg("registered vuln grabber")
    }

    // 初始化 HTTP 客户端
    httpClient := httputil.NewClient()

    // 初始化结果上报器
    reporter := reporter.NewHTTPReporter(config.Server.APIURL, config.Token)

    return &Engine{
        vulnGrabbers:    vulnGrabbers,
        browserPool:     pool,
        httpClient:      httpClient,
        reporter:        reporter,
    }, nil
}

func (e *Engine) RunVulnCrawl(ctx context.Context, task *model.Task) error {
    var payload model.VulnCrawlPayload
    if err := json.Unmarshal(task.Payload, &payload); err != nil {
        return fmt.Errorf("unmarshaling payload: %w", err)
    }

    results := make([]*model.VulnRecord, 0)

    for i, source := range payload.Sources {
        grabber, ok := e.vulnGrabbers[source]
        if !ok {
            log.Warn().Str("source", source).Msg("unknown grabber")
            continue
        }

        // 进度回调
        progress := (i + 1) * 100 / len(payload.Sources)
        task.UpdateProgress(progress, fmt.Sprintf("Processing %s...", source))

        // 执行爬取
        vulns, err := grabber.Crawl(ctx, payload.PageLimit)
        if err != nil {
            log.Error().Str("source", source).Err(err).Msg("grabber failed")
            continue
        }

        results = append(results, vulns...)
    }

    // 上报结果
    return e.reporter.ReportVulns(ctx, task.TaskID, results)
}
```

---

## 爬虫开发

### 接口定义

```go
// pkg/vulngrabber/grabber.go

// Grabber 漏洞爬虫接口
type Grabber interface {
    // Name 返回爬虫名称
    Name() string

    // Crawl 执行爬取
    // pageLimit: 最多爬取页面数
    // 返回: 漏洞记录列表
    Crawl(ctx context.Context, pageLimit int) ([]*VulnRecord, error)
}

// VulnRecord 漏洞记录
type VulnRecord struct {
    Key          string   `json:"key"`
    Title        string   `json:"title"`
    Description  string   `json:"description"`
    Severity     string   `json:"severity"`
    CVE          string   `json:"cve"`
    Disclosure   string   `json:"disclosure"`
    Solutions    string   `json:"solutions"`
    References   []string `json:"references"`
    Tags         []string `json:"tags"`
    GithubSearch []string `json:"github_search"`
    From         string   `json:"from"`
    URL          string   `json:"url"`
}
```

### Rod 爬虫模板

```go
// pkg/vulngrabber/avd_rod.go

type AVDGrabber struct {
    *RodBase
}

func NewAVDGrabber(pool *rodutil.BrowserPool) *AVDGrabber {
    return &AVDGrabber{
        RodBase: NewRodBase(pool, "avd.aliyun.com"),
    }
}

func (g *AVDGrabber) Name() string {
    return "avd-rod"
}

func (g *AVDGrabber) Crawl(ctx context.Context, pageLimit int) ([]*VulnRecord, error) {
    // 1. 获取浏览器页面
    page, err := g.pool.GetPage(ctx)
    if err != nil {
        return nil, fmt.Errorf("getting page: %w", err)
    }
    defer g.pool.PutPage(page)

    // 2. 访问目标页面
    if err := page.Navigate("https://avd.aliyun.com/"); err != nil {
        return nil, fmt.Errorf("navigating: %w", err)
    }

    // 3. 等待内容加载
    if err := page.WaitLoad().Timeout(30 * time.Second).Do(); err != nil {
        log.Warn().Err(err).Msg("wait load timeout")
    }

    // 4. 提取数据
    var results []*VulnRecord

    for i := 0; i < pageLimit; i++ {
        // 使用 JavaScript 提取数据
        data, err := page.Evaluate(`() => {
            const items = document.querySelectorAll('.vuln-item');
            return Array.from(items).map(item => ({
                title: item.querySelector('.title')?.textContent?.trim(),
                cve: item.querySelector('.cve')?.textContent?.trim(),
                severity: item.querySelector('.severity')?.textContent?.trim(),
                url: item.querySelector('a')?.href
            }));
        }`)

        if err != nil {
            log.Warn().Err(err).Msg("evaluate failed")
            break
        }

        // 转换为结果
        items := data.Value.Explode()
        for _, item := range items {
            results = append(results, &VulnRecord{
                Key:      fmt.Sprintf("avd:%s", item.Get("cve").Str()),
                Title:    item.Get("title").Str(),
                CVE:      item.Get("cve").Str(),
                Severity: item.Get("severity").Str(),
                URL:      item.Get("url").Str(),
                From:     "阿里云漏洞库",
                Source:   "avd-rod",
            })
        }

        // 翻页
        if i < pageLimit-1 {
            nextBtn := page.MustElement(".next-page")
            if nextBtn == nil {
                break
            }
            if err := nextBtn.Click(); err != nil {
                break
            }
            page.WaitLoad()
        }
    }

    return results, nil
}
```

### HTTP 爬虫模板

```go
// pkg/vulngrabber/kev.go

type KEVGrabber struct {
    httpClient *httputil.Client
}

func NewKEVGrabber(client *httputil.Client) *KEVGrabber {
    return &KEVGrabber{
        httpClient: client,
    }
}

func (g *KEVGrabber) Name() string {
    return "kev-rod"  // 保持命名一致
}

func (g *KEVGrabber) Crawl(ctx context.Context, pageLimit int) ([]*VulnRecord, error) {
    // 调用 CISA KEV API
    url := "https://www.cisa.gov/sites/default/files/feeds/known_exploited_vulnerabilities.json"

    resp, err := g.httpClient.Get(ctx, url)
    if err != nil {
        return nil, fmt.Errorf("fetching KEV: %w", err)
    }
    defer resp.Body.Close()

    var kevResp KEVResponse
    if err := json.NewDecoder(resp.Body).Decode(&kevResp); err != nil {
        return nil, fmt.Errorf("decoding response: %w", err)
    }

    var results []*VulnRecord
    for _, vuln := range kevResp.Vulnerabilities {
        results = append(results, &VulnRecord{
            Key:        fmt.Sprintf("kev:%s", vuln.CVEID),
            Title:      vuln.VulnerabilityName,
            CVE:        vuln.CVEID,
            Severity:   "高危",  // KEV 都是已遭利用的漏洞
            Disclosure: vuln.DateAdded,
            URL:        vuln.VendorAdvisoryLink,
            From:       "CISA KEV",
            Source:     "kev-rod",
        })
    }

    return results, nil
}

type KEVResponse struct {
    Title        string   `json:"title"`
    CatalogVersion string `json:"catalogVersion"`
    DateReleased string   `json:"dateReleased"`
    Vulnerabilities []struct {
        CVEID              string `json:"cveID"`
        VulnerabilityName string `json:"vulnerabilityName"`
        DateAdded         string `json:"dateAdded"`
        ShortDescription  string `json:"shortDescription"`
        VendorAdvisoryLink string `json:"vendorAdvisoryLink"`
    } `json:"vulnerabilities"`
}
```

### 注册爬虫

```go
// pkg/vulngrabber/registry.go

type Registry map[string]Grabber

func NewRegistry() Registry {
    pool := rodutil.NewBrowserPool(4)
    httpClient := httputil.NewClient()

    return Registry{
        "avd-rod":        NewAVDGrabber(pool),
        "seebug-rod":     NewSeebugGrabber(pool),
        "ti-rod":         NewTIGrabber(pool),
        "kev-rod":        NewKEVGrabber(httpClient),
        "struts2-rod":    NewStruts2Grabber(pool),
        "chaitin-rod":    NewChaitinGrabber(pool),
        "oscs-rod":       NewOSCSGrabber(pool),
        "threatbook-rod": NewThreatBookGrabber(pool),
        "venustech-rod":  NewVenustechGrabber(pool),
        // ...
    }
}
```

---

## 任务处理

### 接收任务

```go
func (c *Client) handleMessage(msg *ws.Message) error {
    switch msg.Type {
    case ws.MsgTypeTask:
        return c.handleTask(msg)

    case ws.MsgTypeTaskCancel:
        return c.handleTaskCancel(msg)

    case ws.MsgTypePing:
        return c.sendPong()
    }
    return nil
}

func (c *Client) handleTask(msg *ws.Message) error {
    var payload struct {
        TaskID  string          `json:"task_id"`
        Type    string          `json:"type"`
        Sources []string        `json:"sources"`
        Payload json.RawMessage `json:"payload"`
    }

    if err := json.Unmarshal(msg.Payload, &payload); err != nil {
        return fmt.Errorf("unmarshaling task: %w", err)
    }

    // 记录任务
    task := &model.Task{
        TaskID: payload.TaskID,
        Type:   payload.Type,
        Status: model.TaskRunning,
        Payload: payload.Payload,
    }

    // 分配到分发器
    go func() {
        if err := c.dispatcher.Dispatch(task); err != nil {
            log.Error().Str("task_id", task.TaskID).Err(err).Msg("dispatch failed")
            c.reportResult(task.TaskID, "failed", nil, err.Error())
        } else {
            c.reportResult(task.TaskID, "done", task.Result, "")
        }
    }()

    return nil
}
```

### 进度上报

```go
func (c *Client) reportProgress(taskID string, progress int, message string) error {
    msg := &ws.Message{
        Type: ws.MsgTypeProgress,
        Payload: mustMarshal(ws.ProgressPayload{
            TaskID:   taskID,
            Progress: progress,
            Message:  message,
        }),
    }

    return c.wsConn.WriteJSON(msg)
}

func (c *Client) reportResult(taskID, status string, data interface{}, errMsg string) error {
    payload := ws.ResultPayload{
        TaskID: taskID,
        Status: status,
    }

    if data != nil {
        payload.Data = mustMarshal(data)
    }
    if errMsg != "" {
        payload.Error = errMsg
    }

    msg := &ws.Message{
        Type:    ws.MsgTypeResult,
        Payload: mustMarshal(payload),
    }

    return c.wsConn.WriteJSON(msg)
}
```

---

## 结果上报

### HTTP 上报（推荐）

```go
// pkg/reporter/http_reporter.go

type HTTPReporter struct {
    apiURL    string
    token     string
    httpClient *http.Client
}

func NewHTTPReporter(apiURL, token string) *HTTPReporter {
    return &HTTPReporter{
        apiURL: apiURL,
        token:  token,
        httpClient: &http.Client{
            Timeout: 30 * time.Second,
        },
    }
}

func (r *HTTPReporter) ReportVulns(ctx context.Context, taskID string, vulns []*VulnRecord) error {
    url := fmt.Sprintf("%s/report", r.apiURL)

    body := ReportRequest{
        TaskID: taskID,
        Type:   "vuln_crawl",
        Data:   vulns,
    }

    data, err := json.Marshal(body)
    if err != nil {
        return fmt.Errorf("marshaling report: %w", err)
    }

    req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
    if err != nil {
        return fmt.Errorf("creating request: %w", err)
    }

    req.Header.Set("Authorization", "Bearer "+r.token)
    req.Header.Set("Content-Type", "application/json")

    resp, err := r.httpClient.Do(req)
    if err != nil {
        return fmt.Errorf("sending report: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("server returned %d", resp.StatusCode)
    }

    return nil
}

type ReportRequest struct {
    TaskID string        `json:"task_id"`
    Type   string        `json:"type"`
    Data   interface{}   `json:"data"`
}
```

---

## 调试技巧

### 本地测试爬虫

```go
// 测试单个爬虫
func TestAVDGrabber(t *testing.T) {
    pool := rodutil.NewBrowserPool(1)
    defer pool.Close()

    grabber := NewAVDGrabber(pool)

    ctx := context.Background()
    vulns, err := grabber.Crawl(ctx, 1)
    if err != nil {
        t.Fatalf("crawl failed: %v", err)
    }

    for _, vuln := range vulns {
        t.Logf("CVE: %s, Title: %s", vuln.CVE, vuln.Title)
    }
}
```

### 调试 WAF 绕过

```go
// 查看当前指纹
func DebugFingerprint(page *rod.Page) {
    ua, _ := page.Evaluate(`() => navigator.userAgent`)
    t.Logf("User-Agent: %s", ua)

    platform, _ := page.Evaluate(`() => navigator.platform`)
    t.Logf("Platform: %s", platform)
}
```

### 日志配置

```bash
# 启动客户端并输出详细日志
SECFLOW_LOG_LEVEL=debug go run cmd/client/main.go

# 输出到文件
SECFLOW_LOG_LEVEL=debug SECFLOW_LOG_PATH=./client.log go run cmd/client/main.go
```
