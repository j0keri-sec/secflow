# 配置参考

本文档提供所有配置项的完整说明。

## 服务端配置 (secflow-server)

### config.yaml

```yaml
# 服务端配置示例
server:
  host: "0.0.0.0"
  port: 8080
  mode: "release"              # debug / release / test
  read_timeout: 30
  write_timeout: 30

# MongoDB 配置
mongodb:
  host: "localhost"
  port: 27017
  database: "secflow"
  username: ""
  password: ""
  auth_source: "admin"
  replica_set: ""              # 集群模式，如 "rs0"
  max_pool_size: 100
  min_pool_size: 10

# Redis 配置
redis:
  host: "localhost"
  port: 6379
  password: ""
  db: 0
  pool_size: 100
  min_idle_conns: 20

# JWT 配置
jwt:
  secret: "your-secret-key-change-in-production"
  expiry: 24h                 # token 过期时间
  refresh_expiry: 168h        # refresh token 过期时间

# WebSocket 配置
websocket:
  ping_interval: 30s          # 心跳间隔
  pong_timeout: 60s            # 心跳超时
  max_message_size: 65536      # 最大消息大小

# 任务队列配置
queue:
  max_retries: 3               # 最大重试次数
  retry_delay: 10s             # 基础重试延迟
  max_retry_delay: 1h          # 最大重试延迟
  task_ttl: 24h                # 任务结果保留时间
  batch_size: 10               # 调度批次大小

# 爬虫配置
crawler:
  timeout: 60s                 # 爬取超时
  user_agent: "SecFlowBot/1.0"
  max_depth: 3                 # 最大爬取深度
  delay: 500ms                 # 请求间隔

# 推送配置
pusher:
  timeout: 10s                 # 推送超时
  retry: 2                     # 推送重试次数
  workers: 5                   # 并发推送数

# 日志配置
log:
  level: "info"                # debug / info / warn / error
  format: "json"               # json / text
  output: "stdout"             # stdout / file
  file_path: "logs/server.log"
  max_size: 100                # MB
  max_backups: 7
  max_age: 30                  # 天

# CORS 配置
cors:
  enabled: true
  allowed_origins:
    - "http://localhost:3000"
    - "https://your-domain.com"
  allowed_methods:
    - "GET"
    - "POST"
    - "PUT"
    - "DELETE"
  allowed_headers:
    - "Authorization"
    - "Content-Type"
  allow_credentials: true
  max_age: 86400
```

### 配置项详解

#### server

| 配置项 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `host` | string | `"0.0.0.0"` | 服务监听地址 |
| `port` | int | `8080` | 服务监听端口 |
| `mode` | string | `"debug"` | 运行模式，影响日志输出和错误处理 |
| `read_timeout` | duration | `30s` | HTTP 读取超时 |
| `write_timeout` | duration | `30s` | HTTP 写入超时 |

#### mongodb

| 配置项 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `host` | string | `"localhost"` | MongoDB 主机地址 |
| `port` | int | `27017` | MongoDB 端口 |
| `database` | string | `"secflow"` | 数据库名称 |
| `username` | string | `""` | 用户名（可选） |
| `password` | string | `""` | 密码（可选） |
| `auth_source` | string | `"admin"` | 认证数据库 |
| `replica_set` | string | `""` | 副本集名称（集群模式） |
| `max_pool_size` | int | `100` | 最大连接池大小 |
| `min_pool_size` | int | `10` | 最小连接池大小 |

#### redis

| 配置项 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `host` | string | `"localhost"` | Redis 主机地址 |
| `port` | int | `6379` | Redis 端口 |
| `password` | string | `""` | 密码（可选） |
| `db` | int | `0` | 数据库编号 |
| `pool_size` | int | `100` | 连接池大小 |
| `min_idle_conns` | int | `20` | 最小空闲连接数 |

#### queue

| 配置项 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `max_retries` | int | `3` | 任务最大重试次数 |
| `retry_delay` | duration | `10s` | 基础重试延迟 |
| `max_retry_delay` | duration | `1h` | 最大重试延迟（用于指数退避） |
| `task_ttl` | duration | `24h` | 任务结果在 Redis 中的保留时间 |
| `batch_size` | int | `10` | 每次调度分发的任务数量 |

## 客户端配置 (secflow-client)

### client.yaml

```yaml
# 客户端配置示例
client:
  node_id: "node-001"          # 节点唯一标识
  name: "内网爬取节点-1"        # 节点显示名称
  server_url: "ws://localhost:8080/api/v1/ws/node"
  token: ""                    # 节点认证 token
  max_concurrent: 5            # 最大并发任务数

# 浏览器配置
browser:
  headless: false              # 是否无头模式
  user_agent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"
  window_size: "1920,1080"
  proxy: ""                    # 代理地址，如 "http://proxy:8080"
  stealth: true                # 启用隐身模式
  timeout: 30s                # 页面加载超时
  wait_for_selector: 5s        # 等待元素超时

# 浏览器池配置
browser_pool:
  min_size: 2                  # 最小浏览器实例数
  max_size: 10                 # 最大浏览器实例数
  idle_timeout: 5m             # 空闲浏览器回收时间
  max_age: 30m                 # 浏览器最大生命周期

# 爬虫配置
crawler:
  timeout: 60s                 # 爬取超时
  delay_min: 1000             # 请求最小延迟（毫秒）
  delay_max: 3000              # 请求最大延迟（毫秒）
  max_retries: 3               # 最大重试次数
  retry_delay: 5s              # 重试延迟

# HTTP 爬虫配置
http_crawler:
  timeout: 30s
  follow_redirects: true
  max_redirects: 5
  headers:
    Accept: "text/html,application/xhtml+xml,application/xml;q=0.9"
    Accept-Language: "zh-CN,zh;q=0.9,en;q=0.8"

# 日志配置
log:
  level: "info"
  format: "json"
  output: "stdout"
  file_path: "logs/client.log"

# 性能配置
performance:
  enable_metrics: true        # 启用性能指标上报
  metrics_interval: 30s       # 指标上报间隔
  cpu_threshold: 80           # CPU 使用率告警阈值
  memory_threshold: 85        # 内存使用率告警阈值
```

### 配置项详解

#### client

| 配置项 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `node_id` | string | 自动生成 | 节点唯一标识 |
| `name` | string | `"Anonymous Node"` | 节点显示名称 |
| `server_url` | string | 必填 | WebSocket 服务端地址 |
| `token` | string | `""` | 认证 token |
| `max_concurrent` | int | `5` | 单节点最大并发任务数 |

#### browser

| 配置项 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `headless` | bool | `true` | 是否无头模式运行 |
| `user_agent` | string | Chrome 默认 | User-Agent 字符串 |
| `window_size` | string | `"1920,1080"` | 窗口大小 |
| `proxy` | string | `""` | HTTP/SOCKS5 代理 |
| `stealth` | bool | `true` | 启用反检测措施 |
| `timeout` | duration | `30s` | 页面加载超时 |
| `wait_for_selector` | duration | `5s` | 等待元素出现超时 |

#### browser_pool

| 配置项 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `min_size` | int | `2` | 最小浏览器实例 |
| `max_size` | int | `5` | 最大浏览器实例 |
| `idle_timeout` | duration | `5m` | 空闲回收时间 |
| `max_age` | duration | `30m` | 浏览器生命周期 |

## 前端配置

### .env 文件

```bash
# API 配置
VITE_API_BASE_URL=http://localhost:8080
VITE_WS_URL=ws://localhost:8080/api/v1/ws

# OAuth 配置（可选）
VITE_OAUTH_ENABLED=false
VITE_OAUTH_PROVIDER=github

# 功能开关
VITE_ENABLE_DEBUG=false
VITE_MOCK_API=false

# 主题配置
VITE_DEFAULT_THEME=dark
```

## 环境变量覆盖

配置可以通过环境变量覆盖，格式为 `SECFLOW_*`：

```bash
# 服务端环境变量
SECFLOW_SERVER_PORT=8080
SECFLOW_MONGODB_HOST=localhost
SECFLOW_MONGODB_PORT=27017
SECFLOW_REDIS_HOST=localhost
SECFLOW_REDIS_PORT=6379
SECFLOW_JWT_SECRET=your-secret

# 客户端环境变量
SECFLOW_CLIENT_SERVER_URL=ws://localhost:8080/api/v1/ws/node
SECFLOW_CLIENT_NODE_ID=node-001
SECFLOW_BROWSER_HEADLESS=true
SECFLOW_CRAWLER_DELAY_MIN=1000
```

## 生产环境配置建议

### 高可用配置

```yaml
# MongoDB 副本集
mongodb:
  host: "mongo1,mongo2,mongo3"
  replica_set: "rs0"

# Redis 哨兵/集群
redis:
  host: "redis1,redis2,redis3"
  sentinel_master: "mymaster"
  sentinel_password: "sentinel-pass"
```

### 安全配置

```yaml
# 生产环境必须修改
jwt:
  secret: "${JWT_SECRET}"      # 使用强随机密钥

cors:
  allowed_origins:
    - "https://your-secure-domain.com"

# 启用 TLS
server:
  cert_file: "/path/to/cert.pem"
  key_file: "/path/to/key.pem"
```

### 性能优化配置

```yaml
# 提升连接池
mongodb:
  max_pool_size: 200
  min_pool_size: 50

redis:
  pool_size: 200
  min_idle_conns: 50

# 调整任务队列
queue:
  batch_size: 50               # 增大调度批次
  max_retries: 5              # 增加重试次数
```
