# SecFlow Client

SecFlow Client 是一个漏洞情报采集节点，连接到 SecFlow Server 并执行漏洞爬取任务。

## 功能特性

- **WebSocket 连接**: 与服务器保持实时连接，接收任务分配
- **多数据源采集**: 支持 10+ 个漏洞数据源
  - 阿里云漏洞库 (aliyun-avd)
  - 长亭漏洞库 (chaitin)
  - 奇安信威胁情报中心 (qianxin-ti)
  - OSCS开源安全情报预警 (oscs)
  - Seebug 漏洞平台 (seebug)
  - 微步在线研究响应中心 (threatbook)
  - Apache Struts2 Security Bulletins (struts2)
  - CISA KEV (Known Exploited Vulnerabilities)
  - 启明星辰漏洞通告 (venustech)
- **自动重连**: WebSocket 断线后自动重连
- **本地缓存**: SQLite 本地存储任务状态和漏洞数据
- **日志存储**: 所有日志同时写入 SQLite 数据库，便于查询和审计
- **心跳上报**: 定期上报节点状态（CPU、内存、磁盘等）

## 快速开始

### 1. 配置

复制示例配置文件：

```bash
cp client.example.yaml client.yaml
```

编辑 `client.yaml`，配置服务器地址和认证令牌：

```yaml
server:
  api_url: "http://localhost:8080/api/v1"
  ws_url: "ws://localhost:8080/api/v1/ws/node"
  token: "your-node-token-here"

name: "my-secflow-node"
```

### 配置文件详解

完整的配置选项说明：

```yaml
# =============================================================================
# Server Connection Settings
# =============================================================================
server:
  api_url: "http://localhost:8080/api/v1"      # HTTP API 地址
  ws_url: "ws://localhost:8080/api/v1/ws/node" # WebSocket 地址
  token: "your-node-token"                      # 认证令牌

# =============================================================================
# Node Identity
# =============================================================================
node_id: "node-001"        # 节点唯一标识（留空则自动生成）
name: "采集节点1"           # 节点显示名称

# =============================================================================
# Logging Settings
# =============================================================================
log_level: "info"          # 日志级别: debug, info, warn, error
log_path: ""               # 日志文件路径（留空则只输出到控制台）

# =============================================================================
# Database Settings
# =============================================================================
db_path: "secflow_client.db"  # SQLite 数据库路径

# =============================================================================
# Connection Settings
# =============================================================================
connection:
  reconnect_interval: 5s        # 断线重连间隔
  timeout: 10s                  # 连接超时时间
  auto_reconnect: true          # 启用自动重连
  max_reconnect_attempts: 0     # 最大重连次数（0 表示无限）

# =============================================================================
# Heartbeat Settings
# =============================================================================
heartbeat_interval: 30s       # 心跳上报间隔

# =============================================================================
# Task Execution Settings
# =============================================================================
task:
  default_page_limit: 1       # 默认分页限制
  max_concurrent: 1           # 最大并发任务数
  timeout: 30m                # 任务超时时间
  enable_github_search: false # 启用 GitHub CVE 搜索

# =============================================================================
# Grabber Settings
# =============================================================================
grabber:
  request_timeout: 30s        # HTTP 请求超时
  retry_attempts: 3           # 失败重试次数
  retry_interval: 5s          # 重试间隔
  user_agent: "SecFlow-Client/1.0"  # User-Agent
  tls_verify: true            # TLS 证书验证
  sources: []                 # 启用的数据源（空表示全部）
  disabled_sources: []        # 禁用的数据源

# =============================================================================
# Proxy Settings
# =============================================================================
proxy: ""                     # HTTP/HTTPS 代理，如 "http://127.0.0.1:7890"
```

### 2. 获取 Node Token

在服务器端创建节点并获取 token：

```bash
# 使用服务器 API 创建节点
curl -X POST http://localhost:8080/api/v1/nodes \
  -H "Authorization: Bearer <your-jwt-token>" \
  -H "Content-Type: application/json" \
  -d '{
    "node_id": "node-001",
    "name": "采集节点1",
    "sources": ["avd", "chaitin", "oscs"]
  }'
```

响应中会返回 `token` 字段，将其填入 client.yaml。

### 3. 运行

```bash
go run ./cmd/client
```

或使用编译后的二进制：

```bash
go build -o secflow-client ./cmd/client
./secflow-client
```

### 4. 命令行选项

```bash
./secflow-client --help

NAME:
   secflow-client - SecFlow client node — connects to the server and executes crawl tasks

USAGE:
   secflow-client [global options] [arguments...]

VERSION:
   1.0.0

DESCRIPTION:
   SecFlow Client is a vulnerability intelligence collection node that connects
   to SecFlow Server and executes crawling tasks.

   EXAMPLES:
      # Run with default config file (client.yaml)
      secflow-client

      # Run with custom config file
      secflow-client -c /etc/secflow/client.yaml

      # Run with debug logging
      secflow-client --debug

      # Run with environment variables
      SECFLOW_TOKEN=xxx SECFLOW_API_URL=http://server:8080/api/v1 secflow-client

GLOBAL OPTIONS:
   --config value, -c value  Path to client configuration file (default: "client.yaml") [$SECFLOW_CONFIG]
   --api-url value           Server HTTP API URL (overrides config) [$SECFLOW_API_URL]
   --ws-url value            Server WebSocket URL (overrides config) [$SECFLOW_WS_URL]
   --token value             Authentication token (overrides config) [$SECFLOW_TOKEN]
   --node-id value           Unique node identifier (overrides config) [$SECFLOW_NODE_ID]
   --name value              Node display name (overrides config) [$SECFLOW_NODE_NAME]
   --proxy value             HTTP/HTTPS proxy URL (overrides config) [$SECFLOW_PROXY]
   --log-level value         Log level: debug, info, warn, error (default: "info") [$SECFLOW_LOG_LEVEL]
   --log-path value          Log file path (overrides config) [$SECFLOW_LOG_PATH]
   --db-path value           SQLite database path (overrides config) [$SECFLOW_DB_PATH]
   --debug, -d               Enable debug logging (shorthand for --log-level=debug) [$SECFLOW_DEBUG]
   --help, -h                show help
   --version, -v             print the version
```

## 环境变量

可以通过环境变量覆盖配置文件中的设置：

| 变量名 | 说明 | 示例 |
|--------|------|------|
| `SECFLOW_CONFIG` | 配置文件路径 | `/etc/secflow/client.yaml` |
| `SECFLOW_API_URL` | HTTP API 地址 | `http://localhost:8080/api/v1` |
| `SECFLOW_WS_URL` | WebSocket 地址 | `ws://localhost:8080/api/v1/ws/node` |
| `SECFLOW_TOKEN` | 节点认证令牌 | `node-token-xxx` |
| `SECFLOW_PROXY` | HTTP 代理 | `http://proxy:8080` |
| `SECFLOW_LOG_LEVEL` | 日志级别 | `debug`, `info`, `warn`, `error` |
| `SECFLOW_LOG_PATH` | 日志文件路径 | `/var/log/secflow/client.log` |
| `SECFLOW_NODE_ID` | 节点 ID | `node-001` |
| `SECFLOW_NODE_NAME` | 节点名称 | `采集节点1` |
| `SECFLOW_DB_PATH` | 数据库路径 | `/var/lib/secflow/client.db` |
| `SECFLOW_DEBUG` | 启用调试日志 | `true` |

## 工作原理

```
┌─────────────┐      WebSocket      ┌─────────────┐
│   Server    │ ◄──────────────────► │   Client    │
│  (SecFlow)  │    任务分配/结果上报  │  (本程序)    │
└─────────────┘                      └──────┬──────┘
                                            │
                                            ▼
                                    ┌─────────────┐
                                    │   Grabber   │
                                    │   Engine    │
                                    └──────┬──────┘
                                           │
                    ┌──────────────────────┼──────────────────────┐
                    ▼                      ▼                      ▼
              ┌─────────┐           ┌─────────┐           ┌─────────┐
              │  AVD    │           │ Chaitin │           │  OSCS   │
              │ 阿里云   │           │ 长亭    │           │ 墨菲安全 │
              └─────────┘           └─────────┘           └─────────┘
```

### 通信协议

Client 与 Server 使用 WebSocket 进行通信，消息格式为 JSON envelope：

**Client → Server:**
- `register`: 节点注册
- `heartbeat`: 心跳上报
- `progress`: 任务进度更新
- `result`: 任务结果上报

**Server → Client:**
- `ping`: 心跳检测
- `task`: 任务分配
- `task_cancel`: 任务取消

## 开发

### 项目结构

```
secflow-client/
├── cmd/client/          # 主程序入口
│   └── main.go
├── internal/
│   ├── config/          # 配置管理
│   ├── db/              # SQLite 数据库
│   ├── engine/          # Grabber 引擎
│   ├── logger/          # 日志
│   ├── task/            # 任务调度
│   └── ws/              # WebSocket 客户端
├── pkg/
│   ├── grabber/         # 漏洞采集器实现
│   │   ├── avd.go
│   │   ├── chaitin.go
│   │   ├── ...
│   │   └── sources.go   # 注册所有采集器
│   └── httputil/        # HTTP 工具
├── client.yaml          # 配置文件
└── go.mod
```

### 添加新的数据源

1. 在 `pkg/grabber/` 下创建新的采集器文件
2. 实现 `Grabber` 接口：

```go
type MyCrawler struct {
    client *req.Client
    log    *golog.Logger
}

func NewMyCrawler() Grabber {
    return &MyCrawler{
        client: httputil.NewClient(),
        log:    golog.Child("[my-source]"),
    }
}

func (c *MyCrawler) ProviderInfo() *Provider {
    return &Provider{
        Name:        "my-source",
        DisplayName: "My Source Name",
        Link:        "https://example.com/vulns",
    }
}

func (c *MyCrawler) GetUpdate(ctx context.Context, pageLimit int) ([]*VulnInfo, error) {
    // 实现爬取逻辑
}

func (c *MyCrawler) IsValuable(info *VulnInfo) bool {
    // 判断漏洞是否值得关注
    return info.Severity == SeverityHigh || info.Severity == SeverityCritical
}
```

3. 在 `sources.go` 中注册：

```go
func init() {
    Register("my-source", func() Grabber { return NewMyCrawler() })
}
```

## 故障排查

### 无法连接到服务器

1. 检查服务器地址配置是否正确
2. 检查网络连通性：`curl http://server:port/api/v1/health`
3. 检查防火墙设置

### 认证失败

1. 确认 token 是否正确
2. 确认 node_id 是否与服务器注册的一致
3. 检查 token 是否过期（在服务器上重新生成）

### 采集器报错

1. 启用调试日志：`--debug` 或 `SECFLOW_DEBUG=true`
2. 检查代理设置
3. 检查目标网站是否可访问

### 查看本地日志

Client 将所有日志存储在 SQLite 数据库中，可以使用以下 SQL 查询：

```bash
# 查看最近的 100 条日志
sqlite3 secflow_client.db "SELECT * FROM logs ORDER BY created_at DESC LIMIT 100;"

# 查看错误级别的日志
sqlite3 secflow_client.db "SELECT * FROM logs WHERE level = 'error' ORDER BY created_at DESC;"

# 查看任务执行记录
sqlite3 secflow_client.db "SELECT * FROM tasks ORDER BY received_at DESC;"
```

## 许可证

MIT License
