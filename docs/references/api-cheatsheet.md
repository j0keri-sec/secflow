# API 速查表

本文档提供 SecFlow REST API 的快速参考。

## 认证

### 登录获取 Token

```bash
POST /api/v1/auth/login
Content-Type: application/json

{
  "username": "admin",
  "password": "admin123"
}
```

**响应：**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIs...",
    "refresh_token": "eyJhbGciOiJIUzI1NiIs...",
    "expires_in": 86400
  }
}
```

### 刷新 Token

```bash
POST /api/v1/auth/refresh
Authorization: Bearer <refresh_token>
```

### 后续请求认证

```bash
Authorization: Bearer <token>
```

---

## 漏洞管理

### 创建漏洞爬取任务

```bash
POST /api/v1/tasks/vuln-crawl
Authorization: Bearer <token>
Content-Type: application/json

{
  "source": "nvd",           # 情报源: nvd/cve/avd/qianxin/venustech
  "page_limit": 10,          # 爬取页数限制
  "priority": 50,            # 优先级: 0/50/100
  "timeout": 300             # 超时时间(秒)
}
```

### 获取漏洞列表

```bash
GET /api/v1/vulns?page=1&page_size=20&source=nvd&severity=high
Authorization: Bearer <token>
```

| 参数 | 类型 | 说明 |
|------|------|------|
| `page` | int | 页码，默认 1 |
| `page_size` | int | 每页数量，默认 20 |
| `source` | string | 情报源筛选 |
| `severity` | string | 严重程度: low/medium/high/critical |
| `keyword` | string | 关键词搜索 |
| `start_date` | string | 开始日期 YYYY-MM-DD |
| `end_date` | string | 结束日期 YYYY-MM-DD |

### 获取漏洞详情

```bash
GET /api/v1/vulns/:id
Authorization: Bearer <token>
```

### 手动推送漏洞

```bash
POST /api/v1/push/vuln
Authorization: Bearer <token>
Content-Type: application/json

{
  "vuln_id": "vuln_abc123",
  "channel_ids": ["ch_xxx", "ch_yyy"],
  "message_type": "markdown"  # markdown/text
}
```

---

## 文章管理

### 创建文章爬取任务

```bash
POST /api/v1/tasks/article-crawl
Authorization: Bearer <token>
Content-Type: application/json

{
  "source": "qianxin-weekly",
  "article_type": "hot-week",  # hot-week/hot-day/security
  "priority": 50
}
```

### 获取文章列表

```bash
GET /api/v1/articles?page=1&page_size=20&source=qianxin-weekly
Authorization: Bearer <token>
```

### 获取文章详情

```bash
GET /api/v1/articles/:id
Authorization: Bearer <token>
```

---

## 任务管理

### 查询任务列表

```bash
GET /api/v1/tasks?status=running&type=vuln-crawl&page=1&page_size=20
Authorization: Bearer <token>
```

### 查询任务详情

```bash
GET /api/v1/tasks/:id
Authorization: Bearer <token>
```

### 查询任务进度

```bash
GET /api/v1/tasks/:id/progress
Authorization: Bearer <token>
```

**响应：**

```json
{
  "code": 0,
  "data": {
    "task_id": "task_xxx",
    "status": "running",
    "progress": 65,
    "total": 100,
    "completed": 65,
    "failed": 2,
    "message": "正在爬取第 65/100 页",
    "started_at": "2026-03-21T10:00:00Z",
    "updated_at": "2026-03-21T10:05:30Z"
  }
}
```

### 停止任务

```bash
POST /api/v1/tasks/:id/stop
Authorization: Bearer <token>
```

### 删除任务

```bash
DELETE /api/v1/tasks/:id
Authorization: Bearer <token>
```

---

## 节点管理

### 获取节点列表

```bash
GET /api/v1/nodes?status=online
Authorization: Bearer <token>
```

### 获取节点详情

```bash
GET /api/v1/nodes/:id
Authorization: Bearer <token>
```

**响应：**

```json
{
  "code": 0,
  "data": {
    "id": "node_xxx",
    "name": "内网爬取节点-1",
    "status": "online",
    "ip": "192.168.1.100",
    "tags": ["production", "vuln"],
    "capabilities": ["vuln-crawl", "article-crawl"],
    "stats": {
      "tasks_completed": 1250,
      "tasks_failed": 12,
      "success_rate": 99.04,
      "avg_duration": "45s"
    },
    "performance": {
      "cpu_usage": 35.2,
      "memory_usage": 62.8,
      "concurrent_tasks": 3
    },
    "last_heartbeat": "2026-03-21T10:10:00Z",
    "registered_at": "2026-03-01T08:00:00Z"
  }
}
```

### 暂停节点

```bash
POST /api/v1/nodes/:id/pause
Authorization: Bearer <token>
```

### 恢复节点

```bash
POST /api/v1/nodes/:id/resume
Authorization: Bearer <token>
```

### 获取节点日志

```bash
GET /api/v1/nodes/:id/logs?limit=100&level=error
Authorization: Bearer <token>
```

---

## 推送渠道管理

### 获取渠道列表

```bash
GET /api/v1/channels
Authorization: Bearer <token>
```

### 创建推送渠道

```bash
POST /api/v1/channels
Authorization: Bearer <token>
Content-Type: application/json

{
  "name": "钉钉告警群",
  "type": "dingtalk",          # dingtalk/feishu/wecom/slack/telegram/webhook/bark
  "config": {
    "webhook_url": "https://oapi.dingtalk.com/robot/send?access_token=xxx",
    "secret": "SECxxx"         # 加签密钥
  },
  "enabled": true
}
```

### 测试推送渠道

```bash
POST /api/v1/channels/:id/test
Authorization: Bearer <token>
```

### 更新推送渠道

```bash
PUT /api/v1/channels/:id
Authorization: Bearer <token>
Content-Type: application/json

{
  "name": "钉钉告警群-新",
  "enabled": true
}
```

### 删除推送渠道

```bash
DELETE /api/v1/channels/:id
Authorization: Bearer <token>
```

---

## 统计与监控

### 获取统计数据

```bash
GET /api/v1/stats?period=week
Authorization: Bearer <token>
```

**响应：**

```json
{
  "code": 0,
  "data": {
    "period": "week",
    "vulns": {
      "total": 1250,
      "critical": 45,
      "high": 230,
      "medium": 680,
      "low": 295
    },
    "articles": {
      "total": 356
    },
    "tasks": {
      "total": 89,
      "success_rate": 94.4,
      "avg_duration": "52s"
    },
    "nodes": {
      "online": 5,
      "offline": 1
    }
  }
}
```

### 获取趋势数据

```bash
GET /api/v1/stats/trend?metric=vulns&period=30d
Authorization: Bearer <token>
```

---

## WebSocket API

### 客户端连接

```
ws://localhost:8080/api/v1/ws/node?token=<token>&node_id=<node_id>&name=<name>
```

### 接收服务端消息

```json
{
  "type": "task",
  "payload": {
    "task_id": "task_xxx",
    "task_type": "vuln-crawl",
    "params": {
      "source": "nvd",
      "page_limit": 10
    }
  }
}
```

```json
{
  "type": "task_cancel",
  "payload": {
    "task_id": "task_xxx",
    "reason": "用户取消"
  }
}
```

```json
{
  "type": "ping"
}
```

### 发送客户端消息

**任务进度上报：**

```json
{
  "type": "progress",
  "payload": {
    "task_id": "task_xxx",
    "progress": 50,
    "message": "已爬取 50/100 页"
  }
}
```

**任务结果上报：**

```json
{
  "type": "result",
  "payload": {
    "task_id": "task_xxx",
    "status": "completed",
    "data": {
      "total": 100,
      "success": 98,
      "failed": 2
    }
  }
}
```

**心跳：**

```json
{
  "type": "heartbeat",
  "payload": {
    "status": "idle",
    "cpu_usage": 25.5,
    "memory_usage": 45.2,
    "concurrent_tasks": 2
  }
}
```

---

## 错误码

| 错误码 | 说明 |
|--------|------|
| `0` | 成功 |
| `1001` | 参数错误 |
| `1002` | 认证失败 |
| `1003` | 权限不足 |
| `2001` | 资源不存在 |
| `2002` | 资源已存在 |
| `3001` | 任务不存在 |
| `3002` | 任务执行失败 |
| `3003` | 任务已取消 |
| `4001` | 节点不在线 |
| `4002` | 节点繁忙 |
| `5001` | 推送失败 |
| `5002` | 爬取失败 |
| `5003` | 目标站点不可达 |
| `5004` | WAF 拦截 |

---

## curl 示例集合

```bash
# 1. 登录
TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}' | jq -r '.data.token')

# 2. 创建漏洞爬取任务
curl -X POST http://localhost:8080/api/v1/tasks/vuln-crawl \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"source":"nvd","page_limit":10,"priority":50}'

# 3. 查看任务列表
curl http://localhost:8080/api/v1/tasks \
  -H "Authorization: Bearer $TOKEN"

# 4. 查看节点列表
curl http://localhost:8080/api/v1/nodes \
  -H "Authorization: Bearer $TOKEN"

# 5. 获取漏洞列表
curl "http://localhost:8080/api/v1/vulns?page=1&page_size=10&severity=high" \
  -H "Authorization: Bearer $TOKEN"

# 6. 创建推送渠道
curl -X POST http://localhost:8080/api/v1/channels \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"钉钉","type":"dingtalk","config":{"webhook_url":"https://..."}}'

# 7. 获取统计数据
curl http://localhost:8080/api/v1/stats \
  -H "Authorization: Bearer $TOKEN"
```
