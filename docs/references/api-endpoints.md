# 完整 API 端点参考

本文档基于 `internal/api/router.go` 提供完整的 REST API 端点参考。

## 基础信息

| 项目 | 值 |
|------|-----|
| 基础 URL | `http://localhost:8080/api/v1` |
| WebSocket URL | `ws://localhost:8080/api/v1/ws/node` |
| 认证方式 | Bearer Token (JWT) |
| 数据格式 | JSON |

## 统一响应格式

**成功响应：**
```json
{
  "code": 0,
  "message": "success",
  "data": { ... }
}
```

**分页响应：**
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "items": [...],
    "total": 100,
    "page": 1,
    "page_size": 20
  }
}
```

**错误响应：**
```json
{
  "code": 1001,
  "message": "参数错误",
  "detail": "page 参数必须为正整数",
  "request_id": "req_abc123"
}
```

---

## 认证相关 `/auth`

### 注册账号

```
POST /api/v1/auth/register
```

**请求体：**
```json
{
  "username": "string (3-32字符)",
  "password": "string (6-128字符)",
  "invite_code": "string (可选，用于邀请注册)"
}
```

**响应：**
```json
{
  "code": 0,
  "data": {
    "id": "ObjectID",
    "username": "string",
    "role": "viewer"
  }
}
```

---

### 登录

```
POST /api/v1/auth/login
```

**请求体：**
```json
{
  "username": "string",
  "password": "string"
}
```

**响应：**
```json
{
  "code": 0,
  "data": {
    "token": "JWT Token",
    "expires_in": 86400,
    "user": {
      "id": "ObjectID",
      "username": "string",
      "role": "admin|editor|viewer"
    }
  }
}
```

---

### 获取当前用户

```
GET /api/v1/auth/me
```

**需要认证：** 是

**响应：**
```json
{
  "code": 0,
  "data": {
    "id": "ObjectID",
    "username": "string",
    "email": "string",
    "role": "admin",
    "avatar": "string",
    "created_at": "2026-03-21T10:00:00Z"
  }
}
```

---

### 修改密码

```
PUT /api/v1/auth/password
```

**需要认证：** 是

**请求体：**
```json
{
  "old_password": "string",
  "new_password": "string (6-128字符)"
}
```

---

### 生成邀请码

```
POST /api/v1/auth/invite
```

**需要认证：** 是（需要 admin 角色）

**请求体：**
```json
{
  "is_admin": false,
  "max_uses": 1
}
```

**响应：**
```json
{
  "code": 0,
  "data": {
    "code": "INVITE-XXXX-YYYY",
    "is_admin": false,
    "created_at": "2026-03-21T10:00:00Z"
  }
}
```

---

### 列出邀请码

```
GET /api/v1/auth/invite
```

**需要认证：** 是

**响应：**
```json
{
  "code": 0,
  "data": [
    {
      "id": "ObjectID",
      "code": "INVITE-XXXX-YYYY",
      "is_admin": false,
      "used": false,
      "used_by_id": null,
      "created_at": "2026-03-21T10:00:00Z"
    }
  ]
}
```

---

## 用户管理 `/users` (admin only)

### 列出用户

```
GET /api/v1/users
```

**需要认证：** 是（需要 admin 角色）

**查询参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| `page` | int | 页码，默认 1 |
| `page_size` | int | 每页数量，默认 20 |

---

### 更新用户角色

```
PATCH /api/v1/users/:id
```

**需要认证：** 是（需要 admin 角色）

**请求体：**
```json
{
  "role": "admin|editor|viewer"
}
```

---

### 删除用户

```
DELETE /api/v1/users/:id
```

**需要认证：** 是（需要 admin 角色）

---

## 漏洞管理 `/vulns`

### 列出漏洞

```
GET /api/v1/vulns
```

**需要认证：** 是

**查询参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| `page` | int | 页码，默认 1 |
| `page_size` | int | 每页数量，默认 20 |
| `severity` | string | 严重程度：严重/高危/中危/低危 |
| `source` | string | 数据源筛选 |
| `cve` | string | CVE 编号精确匹配 |
| `keyword` | string | 关键词搜索（标题、描述） |
| `pushed` | bool | 是否已推送 |

**响应：**
```json
{
  "code": 0,
  "data": {
    "items": [
      {
        "id": "ObjectID",
        "key": "string",
        "title": "string",
        "description": "string",
        "severity": "高危",
        "cve": "CVE-2026-XXXXX",
        "disclosure": "2026-03-15",
        "solutions": "string",
        "references": ["url1", "url2"],
        "tags": ["tag1", "tag2"],
        "source": "avd",
        "url": "https://...",
        "pushed": false,
        "created_at": "2026-03-21T10:00:00Z"
      }
    ],
    "total": 100,
    "page": 1,
    "page_size": 20
  }
}
```

---

### 获取漏洞详情

```
GET /api/v1/vulns/:id
```

**需要认证：** 是

**说明：** `:id` 可以是 MongoDB ObjectID 或唯一 key

---

### 删除漏洞

```
DELETE /api/v1/vulns/:id
```

**需要认证：** 是（需要 editor 或 admin 角色）

---

### 获取漏洞统计

```
GET /api/v1/vulns/stats
```

**需要认证：** 是

**响应：**
```json
{
  "code": 0,
  "data": {
    "total": 1000,
    "by_severity": {
      "严重": 50,
      "高危": 200,
      "中危": 500,
      "低危": 250
    }
  }
}
```

---

### 导出漏洞

```
GET /api/v1/vulns/export
```

**需要认证：** 是

**查询参数：** 同 `/vulns` 列表接口

**说明：** 返回 CSV 文件流

---

## 文章管理 `/articles`

### 列出文章

```
GET /api/v1/articles
```

**需要认证：** 是

**查询参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| `page` | int | 页码 |
| `page_size` | int | 每页数量 |
| `source` | string | 数据源筛选 |
| `keyword` | string | 关键词搜索 |
| `pushed` | bool | 是否已推送 |

---

### 获取文章详情

```
GET /api/v1/articles/:id
```

**需要认证：** 是

---

### 删除文章

```
DELETE /api/v1/articles/:id
```

**需要认证：** 是（需要 editor 或 admin 角色）

---

### Upsert 文章

```
POST /api/v1/articles/upsert
```

**需要认证：** 是（需要 editor 或 admin 角色）

**请求体：**
```json
{
  "key": "unique-key",
  "title": "string",
  "summary": "string",
  "content": "string",
  "author": "string",
  "source": "string",
  "url": "string",
  "image": "string",
  "tags": ["tag1"],
  "published_at": "2026-03-21T10:00:00Z"
}
```

---

## 任务管理 `/tasks`

### 创建漏洞爬取任务

```
POST /api/v1/tasks/vuln-crawl
```

**需要认证：** 是（需要 editor 或 admin 角色）

**请求体：**
```json
{
  "sources": ["avd", "nvd"],
  "page_limit": 10,
  "enable_github": false,
  "proxy": "",
  "priority": 50
}
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `sources` | []string | 是 | 数据源列表 |
| `page_limit` | int | 否 | 爬取页数限制，默认 1 |
| `enable_github` | bool | 否 | 是否启用 GitHub 搜索 |
| `proxy` | string | 否 | 代理地址 |
| `priority` | int | 否 | 优先级 0-100，默认 50 |

**响应：**
```json
{
  "code": 0,
  "data": {
    "task_id": "uuid",
    "priority": 50
  }
}
```

---

### 创建文章爬取任务

```
POST /api/v1/tasks/article-crawl
```

**需要认证：** 是（需要 editor 或 admin 角色）

**请求体：**
```json
{
  "sources": ["qianxin-weekly"],
  "limit": 10
}
```

---

### 列出任务

```
GET /api/v1/tasks
```

**需要认证：** 是

**查询参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| `page` | int | 页码 |
| `page_size` | int | 每页数量 |
| `status` | string | pending/dispatched/running/done/failed |

---

### 获取任务详情

```
GET /api/v1/tasks/:id
```

**需要认证：** 是

**响应：**
```json
{
  "code": 0,
  "data": {
    "task": {
      "id": "ObjectID",
      "task_id": "uuid",
      "type": "vuln_crawl",
      "status": "running",
      "priority": 50,
      "progress": 65,
      "retry_count": 0,
      "error": "",
      "created_at": "2026-03-21T10:00:00Z",
      "started_at": "2026-03-21T10:01:00Z",
      "finished_at": null
    },
    "progress": {
      "percent": 65,
      "message": "正在爬取第 65/100 页"
    }
  }
}
```

---

### 删除任务

```
DELETE /api/v1/tasks/:id
```

**需要认证：** 是（需要 editor 或 admin 角色）

**说明：** 删除任务会同时取消队列中的任务

---

### 停止任务

```
POST /api/v1/tasks/:id/stop
```

**需要认证：** 是（需要 editor 或 admin 角色）

---

## 死信队列 `/dead-letters`

### 列出死信任务

```
GET /api/v1/dead-letters
```

**需要认证：** 是

---

### 获取死信详情

```
GET /api/v1/dead-letters/:id
```

**需要认证：** 是

---

### 重试死信任务

```
POST /api/v1/dead-letters/:id/retry
```

**需要认证：** 是（需要 editor 或 admin 角色）

**说明：** 将死信任务重新放入主队列

---

### 删除死信任务

```
DELETE /api/v1/dead-letters/:id
```

**需要认证：** 是（需要 editor 或 admin 角色）

---

### 死信统计

```
GET /api/v1/dead-letters/stats
```

**需要认证：** 是

---

## 节点管理 `/nodes`

### 列出节点

```
GET /api/v1/nodes
```

**需要认证：** 是

---

### 创建节点

```
POST /api/v1/nodes
```

**需要认证：** 是（需要 admin 角色）

**请求体：**
```json
{
  "name": "string",
  "sources": ["avd", "nvd"]
}
```

---

### 获取节点日志

```
GET /api/v1/nodes/:id/logs
```

**需要认证：** 是

**查询参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| `limit` | int | 日志条数，默认 100 |
| `level` | string | 日志级别：debug/info/warn/error |

---

### 删除节点

```
DELETE /api/v1/nodes/:id
```

**需要认证：** 是（需要 admin 角色）

---

### 重新生成节点 Token

```
POST /api/v1/nodes/:id/regenerate-token
```

**需要认证：** 是（需要 admin 角色）

---

### 暂停节点

```
POST /api/v1/nodes/:id/pause
```

**需要认证：** 是（需要 admin 角色）

**说明：** 暂停后节点不再接收新任务

---

### 恢复节点

```
POST /api/v1/nodes/:id/resume
```

**需要认证：** 是（需要 admin 角色）

---

### 断开节点连接

```
POST /api/v1/nodes/:id/disconnect
```

**需要认证：** 是（需要 admin 角色）

---

## 推送渠道管理 `/push-channels`

### 列出渠道

```
GET /api/v1/push-channels
```

**需要认证：** 是

---

### 创建渠道

```
POST /api/v1/push-channels
```

**需要认证：** 是（需要 editor 或 admin 角色）

**请求体：**
```json
{
  "name": "钉钉告警群",
  "type": "dingding",
  "config": {
    "webhook_url": "https://oapi.dingtalk.com/robot/send?access_token=xxx",
    "secret": "SECxxx"
  },
  "enabled": true
}
```

---

### 更新渠道

```
PATCH /api/v1/push-channels/:id
```

**需要认证：** 是（需要 editor 或 admin 角色）

---

### 删除渠道

```
DELETE /api/v1/push-channels/:id
```

**需要认证：** 是（需要 editor 或 admin 角色）

---

## 审计日志 `/audit-logs`

### 列出审计日志

```
GET /api/v1/audit-logs
```

**需要认证：** 是

**查询参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| `page` | int | 页码 |
| `page_size` | int | 每页数量 |
| `user_id` | string | 按用户筛选 |
| `action` | string | 按操作类型筛选 |
| `start_date` | string | 开始日期 |
| `end_date` | string | 结束日期 |

---

## 报告管理 `/reports`

### 列出报告

```
GET /api/v1/reports
```

**需要认证：** 是

---

### 创建报告

```
POST /api/v1/reports
```

**需要认证：** 是

**请求体：**
```json
{
  "title": "2026年3月漏洞月报",
  "description": "描述",
  "period": "2026-03-01 ~ 2026-03-31"
}
```

---

### 删除报告

```
DELETE /api/v1/reports/:id
```

**需要认证：** 是（需要 editor 或 admin 角色）

---

## WebSocket 连接 `/ws/node`

### 客户端连接

```
ws://localhost:8080/api/v1/ws/node?token=<node_token>&node_id=<node_id>&name=<node_name>
```

**连接参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `token` | string | 是 | 节点认证 Token |
| `node_id` | string | 是 | 节点唯一 ID |
| `name` | string | 是 | 节点显示名称 |

---

## 权限角色

| 角色 | 说明 |
|------|------|
| `admin` | 管理员，拥有所有权限 |
| `editor` | 编辑，可以创建任务、管理漏洞和文章、配置推送 |
| `viewer` | 访客，仅可查看数据 |

---

## HTTP 状态码

| 状态码 | 说明 |
|--------|------|
| 200 | 成功 |
| 400 | 请求参数错误 |
| 401 | 未认证或 Token 过期 |
| 403 | 权限不足 |
| 404 | 资源不存在 |
| 500 | 服务器内部错误 |
