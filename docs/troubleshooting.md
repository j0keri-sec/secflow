# SecFlow 故障排查指南

本文档帮助您解决 SecFlow 部署和运行中的常见问题。

## 目录

- [快速检查清单](#快速检查清单)
- [服务端问题](#服务端问题)
- [客户端问题](#客户端问题)
- [数据库问题](#数据库问题)
- [网络和连接问题](#网络和连接问题)
- [性能问题](#性能问题)

---

## 快速检查清单

遇到问题时，先执行以下检查：

```bash
# 1. 检查服务状态
docker ps

# 2. 查看服务日志
docker logs secflow-server
docker logs secflow-client

# 3. 检查端口占用
netstat -tlnp | grep -E '8080|27017|6379'

# 4. 运行健康检查
cd /path/to/secflow
./scripts/healthcheck.sh
```

---

## 服务端问题

### 服务启动失败

**症状**: `secflow-server` 容器无法启动

**排查步骤**:

1. 检查配置文件
```bash
# 查看日志
docker logs secflow-server

# 常见错误：
# - "configuration file not found" → 检查 config.yaml 路径
# - "MongoDB connection refused" → 检查 MongoDB 是否运行
# - "JWT secret is using default value" → 生产环境必须修改密钥
```

2. 验证配置
```bash
# 检查必填配置项
grep -E "secret|token_key|api_key" config.yaml
```

3. 常见解决方案
| 错误 | 解决方案 |
|------|----------|
| `port already in use` | 修改 `server.port` 或停止占用端口的进程 |
| `mongodb: connection refused` | 检查 MongoDB 容器是否运行 `docker ps mongodb` |
| `redis: connection refused` | 检查 Redis 容器是否运行 `docker ps redis` |

### API 请求返回 500 错误

**排查步骤**:

1. 检查服务端日志
```bash
docker logs secflow-server 2>&1 | grep -A5 "500"
```

2. 常见原因
- MongoDB 查询失败
- Redis 连接断开
- JWT Token 过期

3. 修复
```bash
# 重启服务
docker-compose restart secflow-server
```

### WebSocket 连接失败

**症状**: 客户端无法连接到服务端

**排查步骤**:

1. 检查服务端 WebSocket 端点
```bash
curl -i -N \
  -H "Connection: Upgrade" \
  -H "Upgrade: websocket" \
  -H "Sec-WebSocket-Version: 13" \
  -H "Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==" \
  http://localhost:8080/api/v1/ws/node
```

2. 检查反向代理配置 (Nginx)
```nginx
# 确保 Nginx 支持 WebSocket
location /api/v1/ws/ {
    proxy_pass http://127.0.0.1:8080;
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection "upgrade";
    proxy_read_timeout 86400;
}
```

---

## 客户端问题

### 客户端无法连接服务器

**排查步骤**:

1. 检查客户端配置
```bash
cat client.yaml
```

2. 验证网络连通性
```bash
# 测试 API 端点
curl -I https://your-server.com/api/v1/health

# 测试 WebSocket 端点
wscat -c wss://your-server.com/api/v1/ws/node
```

3. 常见问题

| 问题 | 原因 | 解决方案 |
|------|------|----------|
| `401 Unauthorized` | Token 不匹配 | 确保客户端和服务端 `token_key` 一致 |
| `connection refused` | URL 错误 | 检查 `api_url` 和 `ws_url` 配置 |
| `certificate signed by unknown authority` | TLS 证书问题 | 生产环境配置正确证书，或在开发环境跳过验证 |

### 客户端节点不执行任务

**排查步骤**:

1. 检查节点状态
```bash
# 登录 Web UI → 节点管理
# 查看节点状态是否为 "online"
```

2. 检查任务队列
```bash
# 查看 Redis 队列
docker exec secflow-redis redis-cli LLEN secflow:tasks

# 查看待处理任务
docker exec secflow-redis redis-cli LRANGE secflow:tasks 0 -1
```

3. 查看客户端日志
```bash
docker logs secflow-client 2>&1 | grep -i task
```

### 爬虫无数据返回

**排查步骤**:

1. 检查单个爬虫
```go
// 在客户端测试
grabber, _ := vulngrabber.ByName("avd")
vulns, err := grabber.GetUpdate(context.Background(), 1)
fmt.Printf("Found %d vulns\n", len(vulns))
```

2. 常见原因
- **目标网站访问限制**: 使用代理 `proxy: "http://proxy:8080"`
- **WAF 拦截**: Rod 浏览器被识别
- **网络问题**: 客户端网络无法访问目标

3. 解决方案
```yaml
# client.yaml
proxy: "http://your-proxy:8080"  # 使用代理
grabber:
  user_agent: "Mozilla/5.0 (compatible; Bot/1.0)"  # 伪装 UA
```

---

## 数据库问题

### MongoDB 连接失败

**排查步骤**:

1. 检查 MongoDB 容器
```bash
docker ps | grep mongodb
docker logs secflow-mongodb
```

2. 测试连接
```bash
mongosh "mongodb://localhost:27017/secflow" --eval "db.adminCommand('ping')"
```

3. 常见问题

| 问题 | 解决方案 |
|------|----------|
| `Authentication failed` | 检查用户名密码配置 |
| `Database exists in recovery state` | 重启 MongoDB `docker-compose restart mongodb` |
| `Disk full` | 清理磁盘空间或扩容 |

### Redis 连接失败

**排查步骤**:

1. 检查 Redis 容器
```bash
docker ps | grep redis
docker logs secflow-redis
```

2. 测试连接
```bash
docker exec secflow-redis redis-cli ping
# 应返回 PONG
```

3. 常见问题
```bash
# Redis 内存不足
docker exec secflow-redis redis-cli INFO memory

# 清除所有键 (慎用!)
docker exec secflow-redis redis-cli FLUSHALL
```

### 数据迁移问题

```bash
# 使用迁移脚本
./scripts/migrate.sh \
  -s "mongodb://old:27017/secflow" \
  -t "mongodb://new:27017/secflow" \
  -c "users,vulns,tasks"

# 验证迁移
mongosh "mongodb://new:27017/secflow" --eval "db.users.countDocuments()"
```

---

## 网络和连接问题

### 节点频繁断开重连

**原因**: 网络不稳定或服务端负载过高

**解决方案**:

```yaml
# client.yaml
connection:
  reconnect_interval: 10s  # 增加重连间隔
  timeout: 30s             # 增加超时时间
  auto_reconnect: true
```

### Nginx 反向代理问题

**排查步骤**:

```bash
# 检查 Nginx 配置
nginx -t

# 查看错误日志
tail -f /var/log/nginx/error.log
```

**常见配置问题**:

```nginx
# CORS 问题
add_header 'Access-Control-Allow-Origin' '*';

# WebSocket 支持
proxy_set_header Upgrade $http_upgrade;
proxy_set_header Connection "upgrade";

# 长连接超时
proxy_read_timeout 86400;
```

---

## 性能问题

### 服务端响应缓慢

**排查步骤**:

1. 检查资源使用
```bash
docker stats --no-stream
```

2. 查看慢查询
```javascript
// 在 mongosh 中
db.getProfilingLevel()
db.setProfilingLevel(1, { slowms: 100 })
```

3. 常见原因
- MongoDB 索引缺失
- Redis 队列积压
- 内存不足

**解决方案**:

```bash
# 添加索引
db.vulns.createIndex({ "cve": 1 }, { unique: true })
db.vulns.createIndex({ "created_at": -1 })
db.tasks.createIndex({ "status": 1, "created_at": -1 })

# 重启服务清理缓存
docker-compose restart
```

### 客户端资源占用高

**原因**: Rod 浏览器占用大量内存

**解决方案**:

```yaml
# client.yaml
grabber:
  browser_limit: 2  # 限制同时运行的浏览器实例
task:
  max_concurrent: 1  # 限制并发任务数
```

### 告警频繁触发

**排查步骤**:

```bash
# 查看告警状态
docker exec secflow-alertmanager amtool --alertmanager.url=http://localhost:9093 alert query

# 检查告警规则
cat monitoring/alerts.yml
```

**调整阈值**:

```yaml
# monitoring/alerts.yml
- alert: HighMemoryUsage
  expr: container_memory_usage_bytes / container_spec_memory_limit_bytes > 0.8
  for: 5m  # 增加等待时间，减少瞬时告警
```

---

## 日志分析

### 常用日志命令

```bash
# 收集所有日志
./scripts/log-collector.sh

# 实时查看错误日志
docker logs -f secflow-server 2>&1 | grep -i error

# 统计错误类型
docker logs secflow-server 2>&1 | \
  grep -oE '"level":"error"' | \
  sort | uniq -c | sort -rn

# 查看特定时间范围
docker logs --since "2024-01-15T10:00:00" secflow-server
```

### Loki 日志查询

```logql
# 查询所有错误
{job="secflow-server"} |= "level=error"

# 查询特定任务日志
{job="secflow-server"} | json | task_id="abc123"

# 查询最近 5 分钟的警告
{job="secflow-server"} |~ "WARN|ERROR" | __error__!="JSON"
```

---

## 获取帮助

如果以上方案无法解决问题：

1. 收集诊断信息
```bash
# 创建诊断包
mkdir diag && cd diag
./scripts/healthcheck.sh > healthcheck.log
docker logs secflow-server > server.log
docker logs secflow-client > client.log
docker logs secflow-mongodb > mongodb.log
tar -czf diag.tar.gz *
```

2. 提交 Issue
   - GitHub: https://github.com/secflow/secflow/issues
   - 请附上诊断包和复现步骤
