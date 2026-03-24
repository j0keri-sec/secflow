# SecFlow REST API 接口文档

## 目录

- [概述](#概述)
- [认证](#认证)
- [用户管理](#用户管理)
- [漏洞管理](#漏洞管理)
- [文章管理](#文章管理)
- [任务管理](#任务管理)
- [节点管理](#节点管理)
- [推送渠道](#推送渠道)
- [WebSocket](#websocket)
- [错误码](#错误码)

---

## 概述

### 基础信息

- **Base URL**: `http://localhost:8080/api/v1`
- **认证方式**: JWT Bearer Token
- **Content-Type**: `application/json`

### 请求头

```
Authorization: Bearer <your_jwt_token>
Content-Type: application/json
```

### 通用响应格式

```json
{
  "code": 0,
  "msg": "success",
  "data": { ... }
}
```

### 分页响应格式

```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "items": [...],
    "total": 100,
    "page": 1,
    "page_size": 20
  }
}
```

---

## 认证

### 登录

```
POST /api/v1/auth/login
```

**请求体**:

```json
{
  "username": "admin",
  "password": "password123"
}
```

**响应**:

```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIs...",
    "user": {
      "id": "507f1f77bcf86cd799439011",
      "username": "admin",
      "email": "admin@example.com",
      "role": "admin"
    }
  }
}
```

### 注册

```
POST /api/v1/auth/register
```

**请求体**:

```json
{
  "username": "newuser",
  "password": "password123",
  "email": "user@example.com",
  "invite_code": "ABC123"
}
```

### 获取当前用户

```
GET /api/v1/auth/me
```

**响应**:

```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "id": "507f1f77bcf86cd799439011",
    "username": "admin",
    "email": "admin@example.com",
    "role": "admin",
    "avatar": "",
    "created_at": "2024-01-01T00:00:00Z"
  }
}
```

### 修改密码

```
PUT /api/v1/auth/password
```

**请求体**:

```json
{
  "old_password": "oldpass123",
  "new_password": "newpass123"
}
```

---

## 漏洞管理

### 漏洞列表

```
GET /api/v1/vulns
```

**查询参数**:

| 参数 | 类型 | 说明 |
|------|------|------|
| page | int | 页码 (默认 1) |
| page_size | int | 每页数量 (默认 20, 最大 100) |
| severity | string | 风险等级: 严重, 高危, 中危, 低危 |
| source | string | 数据来源 |
| cve | string | CVE 编号 |
| keyword | string | 关键词搜索 |
| pushed | string | 是否已推送: true, false |

**响应**:

```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "items": [
      {
        "id": "507f1f77bcf86cd799439011",
        "key": "avd:CVE-2024-1234",
        "title": "Apache Log4j 远程代码执行漏洞",
        "description": "漏洞详细描述...",
        "severity": "严重",
        "cve": "CVE-2024-1234",
        "disclosure": "2024-01-15",
        "solutions": "升级到最新版本",
        "references": ["https://nvd.nist.gov/..."],
        "tags": ["RCE", "Log4j"],
        "from": "阿里云漏洞库",
        "source": "avd-rod",
        "url": "https://avd.aliyun.com/...",
        "pushed": false,
        "created_at": "2024-01-20T10:30:00Z",
        "updated_at": "2024-01-20T10:30:00Z"
      }
    ],
    "total": 500,
    "page": 1,
    "page_size": 20
  }
}
```

### 漏洞详情

```
GET /api/v1/vulns/:id
```

### 漏洞统计

```
GET /api/v1/vulns/stats
```

**响应**:

```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "total": 500,
    "today": 15,
    "by_severity": {
      "严重": 50,
      "高危": 150,
      "中危": 200,
      "低危": 100
    },
    "by_source": {
      "avd-rod": 200,
      "seebug-rod": 150
    }
  }
}
```

### 导出漏洞

```
GET /api/v1/vulns/export
```

**响应**: CSV 文件下载

### 删除漏洞

```
DELETE /api/v1/vulns/:id
```

---

## 文章管理

### 文章列表

```
GET /api/v1/articles
```

**查询参数**:

| 参数 | 类型 | 说明 |
|------|------|------|
| page | int | 页码 |
| page_size | int | 每页数量 |
| source | string | 数据来源 |
| keyword | string | 关键词搜索 |

### 文章详情

```
GET /api/v1/articles/:id
```

### 创建/更新文章

```
POST /api/v1/articles/upsert
```

### 删除文章

```
DELETE /api/v1/articles/:id
```

---

## 任务管理

### 任务列表

```
GET /api/v1/tasks
```

**查询参数**:

| 参数 | 类型 | 说明 |
|------|------|------|
| page | int | 页码 |
| page_size | int | 每页数量 |
| status | string | pending, dispatched, running, done, failed |
| type | string | vuln_crawl, article_crawl |

### 创建漏洞爬取任务

```
POST /api/v1/tasks/vuln-crawl
```

**请求体**:

```json
{
  "sources": ["avd-rod", "seebug-rod"],
  "page_limit": 3,
  "priority": 50,
  "timeout_seconds": 1800
}
```

### 创建文章爬取任务

```
POST /api/v1/tasks/article-crawl
```

### 停止任务

```
POST /api/v1/tasks/:id/stop
```

### 删除任务

```
DELETE /api/v1/tasks/:id
```

### 死信队列

```
GET  /api/v1/dead-letters
GET  /api/v1/dead-letters/:id
POST /api/v1/dead-letters/:id/retry
DELETE /api/v1/dead-letters/:id
```

---

## 节点管理

### 节点列表

```
GET /api/v1/nodes
```

**响应**:

```json
{
  "code": 0,
  "msg": "success",
  "data": [
    {
      "id": "507f1f77bcf86cd799439014",
      "node_id": "550e8400-e29b-41d4-a716-446655440000",
      "name": "北京节点-01",
      "status": "online",
      "info": {
        "ip": "192.168.1.100",
        "os": "linux/amd64",
        "cpu_cores": 4,
        "cpu_percent": 45.5
      },
      "sources": ["avd-rod", "seebug-rod"],
      "task_stats": {
        "total_tasks": 100,
        "success_tasks": 95,
        "failed_tasks": 5,
        "current_tasks": 2
      },
      "last_seen_at": "2024-01-20T10:30:00Z"
    }
  ]
}
```

### 创建节点

```
POST /api/v1/nodes
```

### 节点操作

```
POST /api/v1/nodes/:id/pause           # 暂停
POST /api/v1/nodes/:id/resume          # 恢复
POST /api/v1/nodes/:id/disconnect      # 断开
POST /api/v1/nodes/:id/regenerate-token
```

### 删除节点

```
DELETE /api/v1/nodes/:id
```

---

## 推送渠道

### 渠道列表

```
GET /api/v1/push-channels
```

### 创建渠道

```
POST /api/v1/push-channels
```

**钉钉**:

```json
{
  "name": "钉钉告警群",
  "type": "dingding",
  "config": {
    "webhook": "https://oapi.dingtalk.com/robot/send?access_token=xxx",
    "secret": "xxx"
  },
  "enabled": true
}
```

**飞书**:

```json
{
  "name": "飞书告警群",
  "type": "lark",
  "config": {
    "webhook": "https://open.feishu.cn/open-apis/bot/v2/hook/xxx"
  },
  "enabled": true
}
```

**企业微信**:

```json
{
  "name": "企业微信",
  "type": "wechat_work",
  "config": {
    "webhook": "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=xxx"
  },
  "enabled": true
}
```

**Telegram**:

```json
{
  "name": "Telegram",
  "type": "telegram",
  "config": {
    "bot_token": "xxx",
    "chat_id": "xxx"
  },
  "enabled": true
}
```

**Slack**:

```json
{
  "name": "Slack",
  "type": "slack",
  "config": {
    "webhook": "https://hooks.slack.com/services/xxx"
  },
  "enabled": true
}
```

### 更新渠道

```
PATCH /api/v1/push-channels/:id
```

### 删除渠道

```
DELETE /api/v1/push-channels/:id
```

---

## WebSocket

### 连接

```
ws://localhost:8080/api/v1/ws/node?token=<node_token>
```

### 消息类型

#### Server → Client

| 类型 | 说明 | Payload |
|------|------|---------|
| `task` | 下发任务 | `TaskPayload` |
| `task_cancel` | 取消任务 | `{ task_id }` |
| `ping` | 心跳检测 | - |

#### Client → Server

| 类型 | 说明 | Payload |
|------|------|---------|
| `register` | 注册节点 | `{ node_id, token, name, info, sources }` |
| `heartbeat` | 心跳上报 | `{ node_id, info }` |
| `progress` | 进度上报 | `{ task_id, progress, message }` |
| `result` | 结果上报 | `{ task_id, status, data?, error? }` |
| `pong` | 心跳响应 | - |

### 任务 Payload

```json
{
  "type": "task",
  "payload": {
    "task_id": "550e8400-e29b-41d4-a716-446655440000",
    "type": "vuln_crawl",
    "sources": ["avd-rod"],
    "page_limit": 3
  }
}
```

### 进度上报

```json
{
  "type": "progress",
  "payload": {
    "task_id": "550e8400-e29b-41d4-a716-446655440000",
    "progress": 50,
    "message": "Processing avd-rod..."
  }
}
```

### 结果上报

```json
{
  "type": "result",
  "payload": {
    "task_id": "550e8400-e29b-41d4-a716-446655440000",
    "status": "done",
    "data": [...]
  }
}
```

---

## 错误码

| HTTP | 业务码 | 说明 |
|------|--------|------|
| 200 | 0 | 成功 |
| 400 | 1001 | 参数错误 |
| 401 | 1002 | 认证失败 |
| 401 | 1003 | Token 过期 |
| 403 | 1004 | 权限不足 |
| 404 | 2001 | 资源不存在 |
| 500 | 5000 | 服务器错误 |

**错误响应格式**:

```json
{
  "code": 1001,
  "msg": "参数错误: page 必须大于 0"
}
```
