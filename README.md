# SecFlow 安全信息流平台

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat-square&logo=go" alt="Go">
  <img src="https://img.shields.io/badge/Vue-3.4+-4FC08D?style=flat-square&logo=vue.js" alt="Vue">
  <img src="https://img.shields.io/badge/MongoDB-7.0+-47A248?style=flat-square&logo=mongodb" alt="MongoDB">
  <img src="https://img.shields.io/badge/Redis-7.0+-DC382D?style=flat-square&logo=redis" alt="Redis">
  <img src="https://img.shields.io/badge/License-MIT-yellow?style=flat-square" alt="License">
</p>

> 基于 watchvuln 改造的**分布式安全信息流平台**，支持多节点分布式爬取漏洞情报和安全资讯。

## 核心特性

- **分布式架构**：服务端-客户端分离，支持多节点并行爬取
- **实时任务调度**：基于 Redis 的任务队列，支持优先级和重试机制
- **智能负载均衡**：多维度评分（任务数、CPU、内存、成功率）
- **WebSocket 通信**：长连接实时任务分发和进度上报
- **多渠道推送**：钉钉、飞书、企业微信、Slack、Telegram、Webhook
- **22 个情报源**：覆盖国内外主流漏洞库和安全资讯平台

## 系统架构

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              【公网服务器】                                    │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                        secflow-server                               │   │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌────────────┐ │   │
│  │  │   Gin API   │  │  WebSocket  │  │   Redis     │  │  MongoDB   │ │   │
│  │  │   Router    │  │    Hub      │  │   Queue     │  │   Store    │ │   │
│  │  └─────────────┘  └─────────────┘  └─────────────┘  └────────────┘ │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────────┘
                                      │
                                      │ WebSocket / HTTP API
                                      ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                              【内网节点 × N】                                  │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                        secflow-client                                │   │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────────┐ │   │
│  │  │   WS Client │  │  Dispatcher  │  │   Rod-based Grabbers (22)   │ │   │
│  │  └─────────────┘  └─────────────┘  └─────────────────────────────┘ │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────────┘
```

## 技术栈

| 模块 | 技术 | 说明 |
|------|------|------|
| **secflow-server** | Go + Gin + MongoDB + Redis | 服务端 API、任务调度、WebSocket Hub |
| **secflow-client** | Go + go-rod + SQLite | 分布式爬虫节点 |
| **secflow-web** | Vue 3 + TypeScript + TailwindCSS | 前端管理界面 |
| **部署** | Docker + docker-compose | 容器化部署 |

## 快速开始

### 环境要求

- Go 1.21+
- Node.js 18+
- MongoDB 7.0+
- Redis 7.0+
- Chrome/Chromium（go-rod 依赖）

### 安装部署

```bash
# 1. 克隆项目
git clone https://github.com/your-org/secflow.git
cd secflow

# 2. 启动依赖服务
brew services start mongodb-community redis

# 3. 配置服务端
cd secflow-server
cp config/config.yaml.example config/config.yaml
# 编辑 config.yaml 配置数据库连接

# 4. 启动服务端
go run cmd/server/main.go

# 5. 配置客户端
cd ../secflow-client
cp client.yaml.example client.yaml
# 编辑 client.yaml 配置服务端地址

# 6. 启动客户端
go run cmd/client/main.go

# 7. 启动前端
cd ../secflow-web
npm install
npm run dev
```

详细部署文档请参考 [部署指南](docs/deployment/README.md)。

## 项目结构

```
secflow/
├── docs/                      # 技术文档
│   ├── architecture/          # 架构设计文档
│   ├── backend/               # 后端开发文档
│   ├── frontend/              # 前端开发文档
│   ├── api/                   # API 接口文档
│   ├── client/                # 客户端开发文档
│   ├── deployment/            # 部署运维文档
│   └── user-guide/            # 使用手册
│
├── secflow-server/            # 服务端
│   ├── cmd/server/            # 入口文件
│   ├── config/                # 配置
│   ├── internal/
│   │   ├── api/               # HTTP API (handler, middleware, router)
│   │   ├── model/             # 数据模型
│   │   ├── queue/             # Redis 任务队列
│   │   ├── repository/        # MongoDB 操作
│   │   ├── scheduler/         # 任务调度器
│   │   └── ws/                # WebSocket Hub
│   └── pkg/
│       ├── auth/              # JWT 认证
│       ├── logger/            # 日志
│       └── pusher/            # 推送服务
│
├── secflow-client/            # 客户端
│   ├── cmd/client/            # 入口文件
│   ├── internal/
│   │   ├── engine/            # 执行引擎
│   │   ├── task/              # 任务分发
│   │   ├── ws/                # WebSocket 客户端
│   │   └── db/                # 本地 SQLite
│   └── pkg/
│       ├── vulngrabber/       # 漏洞爬虫 (15个源)
│       ├── articlegrabber/     # 文章爬虫 (7个源)
│       ├── rodutil/            # Rod 工具库
│       └── httputil/          # HTTP 工具
│
└── secflow-web/               # 前端
    ├── src/
    │   ├── api/               # API 调用封装
    │   ├── components/         # 公共组件
    │   ├── pages/              # 页面组件
    │   ├── stores/             # Pinia 状态管理
    │   ├── types/              # TypeScript 类型定义
    │   └── utils/              # 工具函数
    └── public/                 # 静态资源
```

## 文档导航

### 入门指南

| 文档 | 说明 |
|------|------|
| [📖 快速入门](quick-start.md) | 5分钟快速上手，Docker一键启动 |
| [🚀 部署运维](deployment/README.md) | 开发/生产环境部署指南 |

### 开发指南

| 文档 | 说明 |
|------|------|
| [🏗️ 架构设计](architecture/README.md) | 系统架构、模块设计、通信协议 |
| [⚙️ 后端开发](backend/README.md) | 服务端开发指南、代码规范 |
| [🔧 Handler 开发](backend/handler-guide.md) | API Handler 开发教程 |
| [💾 Repository 开发](backend/repository-guide.md) | 数据访问层开发教程 |
| [📝 开发工作流程](backend/workflow-guide.md) | Git 工作流、代码规范、测试 |
| [🎨 前端开发](frontend/README.md) | Vue3 前端开发指南 |
| [📐 前端开发规范](frontend/development-guide.md) | TypeScript 规范、组件规范 |
| [🕷️ 客户端开发](client/README.md) | 爬虫节点开发指南 |
| [📡 API 接口](api/README.md) | RESTful API 完整参考 |

### 参考文档

| 文档 | 说明 |
|------|------|
| [⚙️ 配置参考](references/config.md) | 所有配置项完整说明 |
| [📋 API 端点](references/api-endpoints.md) | 完整 API 端点参考 |
| [📄 API 速查表](references/api-cheatsheet.md) | 常用 API 快速参考 |
| [🗄️ 数据模型](references/data-models.md) | MongoDB 数据模型完整参考 |
| [🚨 错误码参考](references/error-codes.md) | 错误码及处理方法 |
| [🕷️ 爬虫源配置](references/crawler-sources.md) | 22 个情报源详细配置 |

### 使用手册

| 文档 | 说明 |
|------|------|
| [📘 使用手册](user-guide/README.md) | 功能使用指南、操作说明 |

## 支持的情报源

### 漏洞情报源 (15个)

| 数据源 | 类型 | 技术方案 |
|--------|------|----------|
| AVD (阿里云漏洞库) | HTML | go-rod |
| Seebug (知道创宇) | HTML | go-rod |
| TI (腾讯安全) | API/HTML | go-rod |
| NOX (奇安信) | HTML | go-rod |
| KEV (CISA) | API | HTTP |
| Struts2 | HTML | go-rod |
| Chaitin (长亭) | API/HTML | HTTP/go-rod |
| OSCS (开源安全) | API/HTML | HTTP/go-rod |
| ThreatBook (微步) | API | HTTP |
| Venustech (启明) | HTML | go-rod |
| NVD | API | HTTP |
| PacketStorm | HTML | go-rod |
| ExploitDB | HTML | go-rod |
| 360-CERT | HTML | go-rod |
| Vulhub | HTML | go-rod |

### 安全资讯源 (7个)

| 数据源 | 类型 | 技术方案 |
|--------|------|----------|
| 奇安信周报 | API + HTML | go-rod |
| 启明星辰文章 | HTML | go-rod |
| ... | ... | ... |

## 配置示例

### 服务端配置 (config.yaml)

```yaml
server:
  host: "0.0.0.0"
  port: 8080
  mode: "release"

mongodb:
  uri: "mongodb://user:pass@127.0.0.1:27017/secflow?authSource=admin"
  database: "secflow"

redis:
  addr: "127.0.0.1:6379"
  password: "your-redis-password"
  db: 0

jwt:
  secret: "your-jwt-secret-key"
  expire: "72h"

node:
  token_key: "your-node-token-key"
```

### 客户端配置 (client.yaml)

```yaml
server:
  api_url: "https://your-server.com/api/v1"
  ws_url: "wss://your-server.com/api/v1/ws/node"
  token_key: "your-node-token-key"

db_path: "./secflow.db"
heartbeat_interval: 30s
log_level: info
name: "beijing-node-01"
```

## 许可证

MIT License - 详见 [LICENSE](LICENSE) 文件。

## 贡献指南

欢迎提交 Issue 和 Pull Request！

1. Fork 本仓库
2. 创建特性分支 (`git checkout -b feature/amazing-feature`)
3. 提交更改 (`git commit -m 'Add amazing feature'`)
4. 推送到分支 (`git push origin feature/amazing-feature`)
5. 创建 Pull Request

---

**SecFlow** - 让安全情报触手可及
