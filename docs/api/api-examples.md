# SecFlow API 使用示例

本文档提供 SecFlow REST API 的详细使用示例。

## 目录

- [认证](#认证)
- [节点管理](#节点管理)
- [任务管理](#任务管理)
- [漏洞数据](#漏洞数据)
- [推送渠道](#推送渠道)
- [用户管理](#用户管理)

---

## 基础信息

- **Base URL**: `http://localhost:8080/api/v1`
- **认证方式**: JWT Bearer Token
- **Content-Type**: `application/json`

### 获取 Token

```bash
# 登录获取 JWT token
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username": "admin", "password": "your-password"}'
```

响应：
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "user": {
      "id": "65f1a2b3c4d5e6f7a8b9c0d1",
      "username": "admin",
      "role": "admin"
    }
  }
}
```

### 使用 Token

```bash
# 后续请求在 Header 中携带 token
curl http://localhost:8080/api/v1/vulns \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
```

---

## 认证

### 用户注册 (需要邀请码)

```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "newuser",
    "password": "SecurePass123!",
    "email": "user@example.com",
    "invite_code": "ABC123XYZ"
  }'
```

### 用户登录

```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "admin",
    "password": "your-password"
  }'
```

### 请求密码重置

```bash
curl -X POST http://localhost:8080/api/v1/auth/reset/request \
  -H "Content-Type: application/json" \
  -d '{"email": "user@example.com"}'
```

### 确认密码重置

```bash
curl -X POST http://localhost:8080/api/v1/auth/reset/confirm \
  -H "Content-Type: application/json" \
  -d '{
    "token": "64-char-hex-token-from-email",
    "new_password": "NewSecurePass123!"
  }'
```

### 获取当前用户信息

```bash
curl http://localhost:8080/api/v1/auth/me \
  -H "Authorization: Bearer YOUR_TOKEN"
```

---

## 节点管理

### 获取节点列表

```bash
# 获取所有节点
curl http://localhost:8080/api/v1/nodes \
  -H "Authorization: Bearer YOUR_TOKEN"

# 获取分页结果
curl "http://localhost:8080/api/v1/nodes?page=1&page_size=20" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

响应：
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "nodes": [
      {
        "id": "65f1a2b3c4d5e6f7a8b9c0d2",
        "node_id": "node-beijing-01",
        "name": "北京节点 01",
        "status": "online",
        "ip": "192.168.1.100",
        "os": "linux",
        "cpu_percent": 15.5,
        "mem_percent": 32.1,
        "last_seen": "2024-03-25T08:00:00Z"
      }
    ],
    "total": 5
  }
}
```

### 获取单个节点详情

```bash
curl http://localhost:8080/api/v1/nodes/65f1a2b3c4d5e6f7a8b9c0d2 \
  -H "Authorization: Bearer YOUR_TOKEN"
```

### 节点下线/暂停

```bash
curl -X PATCH http://localhost:8080/api/v1/nodes/NODE_ID/status \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"status": "paused"}'
```

### 删除节点

```bash
curl -X DELETE http://localhost:8080/api/v1/nodes/NODE_ID \
  -H "Authorization: Bearer YOUR_TOKEN"
```

### 获取节点性能统计

```bash
curl http://localhost:8080/api/v1/nodes/NODE_ID/stats \
  -H "Authorization: Bearer YOUR_TOKEN"
```

---

## 任务管理

### 获取任务列表

```bash
# 获取所有任务
curl http://localhost:8080/api/v1/tasks \
  -H "Authorization: Bearer YOUR_TOKEN"

# 按状态筛选
curl "http://localhost:8080/api/v1/tasks?status=running" \
  -H "Authorization: Bearer YOUR_TOKEN"

# 按类型筛选
curl "http://localhost:8080/api/v1/tasks?type=vuln_crawl" \
  -H "Authorization: Bearer YOUR_TOKEN"

# 组合筛选
curl "http://localhost:8080/api/v1/tasks?status=done&type=vuln_crawl&page=1&page_size=50" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

响应：
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "tasks": [
      {
        "id": "65f1a2b3c4d5e6f7a8b9c0d3",
        "task_id": "task-20240325-001",
        "type": "vuln_crawl",
        "status": "done",
        "progress": 100,
        "node_id": "node-beijing-01",
        "result_summary": {
          "vulns_found": 42,
          "sources_used": ["avd", "seebug", "nvd"]
        },
        "created_at": "2024-03-25T08:00:00Z",
        "completed_at": "2024-03-25T08:05:32Z"
      }
    ],
    "total": 128
  }
}
```

### 创建漏洞爬取任务

```bash
curl -X POST http://localhost:8080/api/v1/tasks \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "type": "vuln_crawl",
    "payload": {
      "sources": ["avd", "seebug", "nvd", "kev"],
      "page_limit": 2,
      "enable_github": false
    },
    "target_nodes": ["node-beijing-01", "node-shanghai-01"]
  }'
```

### 创建文章爬取任务

```bash
curl -X POST http://localhost:8080/api/v1/tasks \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "type": "article_crawl",
    "payload": {
      "sources": ["qianxin", "venustech"],
      "limit": 20
    }
  }'
```

### 取消任务

```bash
curl -X DELETE http://localhost:8080/api/v1/tasks/TASK_ID \
  -H "Authorization: Bearer YOUR_TOKEN"
```

### 获取任务结果

```bash
curl http://localhost:8080/api/v1/tasks/TASK_ID/result \
  -H "Authorization: Bearer YOUR_TOKEN"
```

### 手动触发自动任务生成

```bash
curl -X POST http://localhost:8080/api/v1/tasks/generate \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "type": "vuln_crawl",
    "sources": ["avd", "seebug"]
  }'
```

---

## 漏洞数据

### 获取漏洞列表

```bash
# 获取所有漏洞
curl http://localhost:8080/api/v1/vulns \
  -H "Authorization: Bearer YOUR_TOKEN"

# 按严重等级筛选
curl "http://localhost:8080/api/v1/vulns?severity=HIGH,CRITICAL" \
  -H "Authorization: Bearer YOUR_TOKEN"

# 按来源筛选
curl "http://localhost:8080/api/v1/vulns?source=avd,nvd" \
  -H "Authorization: Bearer YOUR_TOKEN"

# 按 CVE 搜索
curl "http://localhost:8080/api/v1/vulns?cve=CVE-2024" \
  -H "Authorization: Bearer YOUR_TOKEN"

# 分页
curl "http://localhost:8080/api/v1/vulns?page=1&page_size=100&sort=created_at&order=desc" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

响应：
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "vulns": [
      {
        "id": "65f1a2b3c4d5e6f7a8b9c0d4",
        "key": "CVE-2024-1234",
        "title": "Apache Struts2 Remote Code Execution",
        "severity": "CRITICAL",
        "cve": "CVE-2024-1234",
        "source": "avd",
        "tags": ["rce", "apache", "struts"],
        "from": "https://avd.example.com/cve-2024-1234",
        "created_at": "2024-03-25T08:00:00Z"
      }
    ],
    "total": 1542,
    "page": 1,
    "page_size": 20
  }
}
```

### 获取单个漏洞详情

```bash
curl http://localhost:8080/api/v1/vulns/CVE-2024-1234 \
  -H "Authorization: Bearer YOUR_TOKEN"
```

### 搜索漏洞

```bash
curl -X POST http://localhost:8080/api/v1/vulns/search \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "query": "remote code execution",
    "severity": ["CRITICAL", "HIGH"],
    "date_from": "2024-01-01",
    "date_to": "2024-03-25",
    "tags": ["rce"]
  }'
```

### 获取漏洞统计

```bash
curl http://localhost:8080/api/v1/vulns/stats \
  -H "Authorization: Bearer YOUR_TOKEN"
```

响应：
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "total": 1542,
    "by_severity": {
      "CRITICAL": 42,
      "HIGH": 156,
      "MEDIUM": 389,
      "LOW": 955
    },
    "by_source": {
      "avd": 523,
      "seebug": 312,
      "nvd": 707
    },
    "last_24h": 12,
    "last_7d": 89
  }
}
```

### 获取文章列表

```bash
curl "http://localhost:8080/api/v1/articles?page=1&page_size=20" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

---

## 推送渠道

### 获取推送渠道列表

```bash
curl http://localhost:8080/api/v1/push-channels \
  -H "Authorization: Bearer YOUR_TOKEN"
```

响应：
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "channels": [
      {
        "id": "65f1a2b3c4d5e6f7a8b9c0d5",
        "name": "钉钉告警群",
        "type": "dingding",
        "enabled": true,
        "config": {
          "access_token": "***",
          "sign_secret": "***"
        },
        "filters": {
          "severity": ["CRITICAL", "HIGH"]
        }
      }
    ]
  }
}
```

### 创建推送渠道

**钉钉**

```bash
curl -X POST http://localhost:8080/api/v1/push-channels \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "钉钉安全告警",
    "type": "dingding",
    "enabled": true,
    "config": {
      "access_token": "your-dingtalk-access-token",
      "sign_secret": "your-sign-secret"
    },
    "filters": {
      "severity": ["CRITICAL", "HIGH"]
    }
  }'
```

**飞书**

```bash
curl -X POST http://localhost:8080/api/v1/push-channels \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "飞书告警",
    "type": "lark",
    "enabled": true,
    "config": {
      "access_token": "your-lark-access-token",
      "sign_secret": "your-sign-secret"
    }
  }'
```

**Slack**

```bash
curl -X POST http://localhost:8080/api/v1/push-channels \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Slack 安全频道",
    "type": "slack",
    "enabled": true,
    "config": {
      "webhook_url": "https://hooks.slack.com/services/xxx/yyy/zzz"
    }
  }'
```

**Telegram**

```bash
curl -X POST http://localhost:8080/api/v1/push-channels \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Telegram 告警",
    "type": "telegram",
    "enabled": true,
    "config": {
      "bot_token": "your-bot-token",
      "chat_id": "your-chat-id"
    }
  }'
```

**Discord**

```bash
curl -X POST http://localhost:8080/api/v1/push-channels \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Discord 安全告警",
    "type": "discord",
    "enabled": true,
    "config": {
      "webhook_url": "https://discord.com/api/webhooks/YOUR_WEBHOOK_ID/TOKEN"
    },
    "filters": {
      "severity": ["CRITICAL", "HIGH"]
    }
  }'
```

**Webhook**

```bash
curl -X POST http://localhost:8080/api/v1/push-channels \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "自定义 Webhook",
    "type": "webhook",
    "enabled": true,
    "config": {
      "url": "https://your-server.com/webhook",
      "method": "POST"
    }
  }'
```

### 更新推送渠道

```bash
curl -X PATCH http://localhost:8080/api/v1/push-channels/CHANNEL_ID \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "enabled": false,
    "filters": {
      "severity": ["CRITICAL"]
    }
  }'
```

### 测试推送渠道

```bash
curl -X POST http://localhost:8080/api/v1/push-channels/CHANNEL_ID/test \
  -H "Authorization: Bearer YOUR_TOKEN"
```

### 删除推送渠道

```bash
curl -X DELETE http://localhost:8080/api/v1/push-channels/CHANNEL_ID \
  -H "Authorization: Bearer YOUR_TOKEN"
```

---

## 用户管理

### 获取用户列表 (admin)

```bash
curl http://localhost:8080/api/v1/users \
  -H "Authorization: Bearer YOUR_TOKEN"
```

### 获取当前用户

```bash
curl http://localhost:8080/api/v1/users/me \
  -H "Authorization: Bearer YOUR_TOKEN"
```

### 更新用户信息

```bash
curl -X PATCH http://localhost:8080/api/v1/users/USER_ID \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "new-email@example.com"
  }'
```

### 更新用户角色 (admin)

```bash
curl -X PATCH http://localhost:8080/api/v1/users/USER_ID/role \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"role": "editor"}'
```

### 删除用户 (admin)

```bash
curl -X DELETE http://localhost:8080/api/v1/users/USER_ID \
  -H "Authorization: Bearer YOUR_TOKEN"
```

### 生成邀请码 (admin/editor)

```bash
curl -X POST http://localhost:8080/api/v1/users/invite-codes \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "is_admin": false,
    "max_uses": 5
  }'
```

响应：
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "code": "ABC123XYZ789",
    "is_admin": false,
    "max_uses": 5,
    "uses": 0,
    "created_at": "2024-03-25T08:00:00Z"
  }
}
```

---

## 审计日志

### 获取审计日志 (admin)

```bash
curl "http://localhost:8080/api/v1/audit-logs?page=1&page_size=50" \
  -H "Authorization: Bearer YOUR_TOKEN"

# 按操作类型筛选
curl "http://localhost:8080/api/v1/audit-logs?action=delete" \
  -H "Authorization: Bearer YOUR_TOKEN"

# 按用户筛选
curl "http://localhost:8080/api/v1/audit-logs?user_id=USER_ID" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

---

## 系统

### 健康检查

```bash
curl http://localhost:8080/api/v1/health
```

### 获取系统统计

```bash
curl http://localhost:8080/api/v1/system/stats \
  -H "Authorization: Bearer YOUR_TOKEN"
```

### 获取可用爬虫源

```bash
curl http://localhost:8080/api/v1/system/sources \
  -H "Authorization: Bearer YOUR_TOKEN"
```

响应：
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "vuln_sources": [
      {"name": "avd", "display_name": "AVD", "enabled": true},
      {"name": "seebug", "display_name": "Seebug", "enabled": true}
    ],
    "article_sources": [
      {"name": "qianxin", "display_name": "奇安信", "enabled": true}
    ]
  }
}
```

---

## 错误响应

所有 API 错误返回统一格式：

```json
{
  "code": 40001,
  "message": "invalid token",
  "data": null
}
```

### 常见错误码

| code | 说明 |
|------|------|
| 0 | 成功 |
| 40001 | 无效或过期 token |
| 40002 | 权限不足 |
| 40003 | 资源不存在 |
| 40004 | 请求参数错误 |
| 40005 | 邀请码无效 |
| 50001 | 服务器内部错误 |
