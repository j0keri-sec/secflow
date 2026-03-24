# 快速入门指南

本文档帮助你快速上手 SecFlow 平台，从零开始了解系统的核心功能和使用方法。

## 环境要求

| 组件 | 最低版本 | 推荐版本 |
|------|----------|----------|
| Go | 1.21 | 1.21+ |
| Node.js | 18.0 | 20.0+ |
| MongoDB | 6.0 | 7.0 |
| Redis | 7.0 | 7.2 |
| Chrome/Chromium | 120 | 最新稳定版 |

## 快速启动（Docker Compose）

### 1. 克隆项目

```bash
git clone https://github.com/your-org/secflow.git
cd secflow
```

### 2. 一键启动所有服务

```bash
# 开发环境
docker-compose up -d

# 生产环境
docker-compose -f docker-compose.prod.yml up -d
```

### 3. 访问服务

| 服务 | 地址 | 默认凭证 |
|------|------|----------|
| 前端界面 | http://localhost:3000 | admin / admin123 |
| API 接口 | http://localhost:8080/api/v1 | - |
| Prometheus | http://localhost:9090 | - |
| Grafana | http://localhost:3001 | admin / admin |

## 本地开发环境搭建

### 服务端

```bash
cd secflow-server

# 安装依赖
go mod tidy

# 配置环境变量
cp config/config.yaml.example config/config.yaml
# 编辑 config.yaml 填入你的配置

# 启动开发服务器（热重载）
air

# 或直接运行
go run cmd/server/main.go
```

### 客户端

```bash
cd secflow-client

# 安装依赖
go mod tidy

# 配置
cp client.yaml.example client.yaml
# 编辑 client.yaml

# 启动客户端节点
go run cmd/client/main.go
```

### 前端

```bash
cd secflow-web

# 安装依赖
pnpm install

# 启动开发服务器
pnpm dev

# 构建生产版本
pnpm build
```

## 核心概念

### 1. 节点（Node）

节点是部署在内网的爬取客户端，通过 WebSocket 与服务端保持长连接。

```
节点角色：
├── 调度器 (Scheduler) - 负责任务分发
├── 执行器 (Executor) - 执行具体爬取任务
└── 浏览器池 (Browser Pool) - 管理 Chrome 实例
```

### 2. 任务（Task）

任务是爬取工作的基本单元。

```yaml
任务类型：
├── vuln-crawl     # 漏洞情报爬取
└── article-crawl # 安全资讯爬取

任务状态：
├── pending       # 等待调度
├── running       # 执行中
├── completed     # 已完成
├── failed        # 失败
└── cancelled     # 已取消
```

### 3. 优先级队列

系统采用多级优先级机制：

| 优先级 | 分数 | 使用场景 |
|--------|------|----------|
| 紧急 | 100 | 安全事件应急响应 |
| 高 | 50 | 常规漏洞情报 |
| 普通 | 0 | 后台批量任务 |

### 4. 情报源

系统支持 22 个情报源：

**国际源（15个）**
- NVD (National Vulnerability Database)
- CVE.org
- Exploit-DB
- GitHub Advisory Database
- OSV
- NIST NVD
- Vulners
- CISA Known Exploited Vulnerabilities
- Rapid7
- Tenable Nessus
- Qualys
- Snyk
- HackerOne
- Bugcrowd
- ZeroDayInitiative

**国内源（7个）**
- 奇安信威胁情报中心
- 绿盟科技
- 启明星辰
- 深信服
- 安恒信息
- 天融信
- 知道创宇

## 首次使用流程

### Step 1: 配置推送渠道

1. 进入「系统设置」→「推送配置」
2. 添加推送渠道（钉钉/飞书/企业微信等）
3. 配置 Webhook URL 和密钥
4. 点击「测试」验证连接

### Step 2: 添加客户端节点

1. 在内网服务器部署 secflow-client
2. 配置 server_url 指向你的服务端地址
3. 启动客户端，自动注册到服务端
4. 在前端「节点管理」页面确认节点在线

### Step 3: 创建爬取任务

**漏洞情报爬取：**

```bash
curl -X POST http://localhost:8080/api/v1/tasks/vuln-crawl \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "source": "nvd",
    "page_limit": 10,
    "priority": 50
  }'
```

**安全资讯爬取：**

```bash
curl -X POST http://localhost:8080/api/v1/tasks/article-crawl \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "source": "qianxin-weekly",
    "article_type": "hot-week",
    "priority": 50
  }'
```

### Step 4: 查看爬取结果

1. 在前端「漏洞列表」/「文章列表」查看数据
2. 点击任意条目查看详情
3. 配置自动推送，数据会自动推送到已配置的渠道

## 常见操作示例

### 查看任务状态

```bash
# 查看所有任务
curl http://localhost:8080/api/v1/tasks

# 查看指定任务
curl http://localhost:8080/api/v1/tasks/<task_id>

# 查看任务进度
curl http://localhost:8080/api/v1/tasks/<task_id>/progress
```

### 管理节点

```bash
# 查看所有节点
curl http://localhost:8080/api/v1/nodes

# 暂停节点
curl -X POST http://localhost:8080/api/v1/nodes/<node_id>/pause

# 恢复节点
curl -X POST http://localhost:8080/api/v1/nodes/<node_id>/resume
```

### 手动触发推送

```bash
# 推送指定漏洞
curl -X POST http://localhost:8080/api/v1/push/vuln \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "vuln_id": "<vuln_id>",
    "channel_ids": ["<channel_id1>", "<channel_id2>"]
  }'
```

## 监控与日志

### 查看日志

```bash
# 服务端日志
tail -f secflow-server/server.log

# 客户端日志
tail -f secflow-client/client.log

# Docker 日志
docker-compose logs -f secflow-server
docker-compose logs -f secflow-client
```

### Prometheus 指标

访问 http://localhost:9090 查看监控面板。

常用查询：

```promql
# 任务队列长度
secflow_tasks_queue_length

# 节点处理能力
secflow_node_tasks_processed_total

# API 请求延迟
secflow_api_request_duration_seconds_bucket

# 爬取成功率
rate(secflow_crawl_success_total[5m]) / rate(secflow_crawl_total[5m])
```

## 下一步

- 📖 阅读 [架构文档](./architecture/README.md) 深入了解系统设计
- 🔧 查看 [开发文档](./backend/README.md) 开始开发
- 🚀 参考 [部署文档](./deployment/README.md) 进行生产部署
- 💬 如有问题，查看 [使用手册](./user-guide/README.md) 的 FAQ 部分
