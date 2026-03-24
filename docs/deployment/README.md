# SecFlow 部署运维文档

## 目录

- [环境要求](#环境要求)
- [开发环境部署](#开发环境部署)
- [生产环境部署](#生产环境部署)
- [Docker 部署](#docker-部署)
- [配置说明](#配置说明)
- [监控运维](#监控运维)
- [故障排查](#故障排查)

---

## 环境要求

### 硬件要求

| 环境 | CPU | 内存 | 磁盘 | 说明 |
|------|-----|------|------|------|
| 开发 | 2 核 | 4 GB | 20 GB | - |
| 小规模生产 | 4 核 | 8 GB | 100 GB | 支持 10 个节点 |
| 中等规模生产 | 8 核 | 16 GB | 200 GB | 支持 50 个节点 |
| 大规模生产 | 16 核 | 32 GB | 500 GB | 支持 100+ 节点 |

### 软件要求

| 软件 | 版本 | 说明 |
|------|------|------|
| Go | 1.21+ | 后端开发语言 |
| Node.js | 18+ | 前端构建 |
| MongoDB | 7.0+ | 主数据库 |
| Redis | 7.0+ | 任务队列、缓存 |
| Chrome/Chromium | 最新版 | go-rod 依赖 |
| Docker | 24+ | 容器化部署 |
| docker-compose | 2.0+ | 容器编排 |

### 网络要求

- **服务端**: 公网可访问，开放 8080 端口
- **客户端**: 能访问服务端公网地址即可（无需开放端口）

---

## 开发环境部署

### 1. 安装依赖

```bash
# macOS
brew install go node mongodb-community redis

# Ubuntu/Debian
curl -fsSL https://deb.nodesource.com/setup_18.x | sudo -E bash -
sudo apt-get install -y nodejs golang-go mongodb redis-server
```

### 2. 启动依赖服务

```bash
# macOS
brew services start mongodb-community
brew services start redis

# Ubuntu/Debian
sudo systemctl start mongodb
sudo systemctl start redis-server
```

### 3. 配置服务端

```bash
cd secflow-server

# 复制配置示例
cp config/config.yaml.example config/config.yaml

# 编辑配置
vim config/config.yaml
```

**配置内容**:

```yaml
server:
  host: "0.0.0.0"
  port: 8080
  mode: "debug"

mongodb:
  uri: "mongodb://secflow:secflow_pass@127.0.0.1:27017/secflow?authSource=admin"
  database: "secflow"

redis:
  addr: "127.0.0.1:6379"
  password: "secflow_redis_pass"
  db: 0

jwt:
  secret: "your-jwt-secret-change-in-production"
  expire: "72h"

node:
  token_key: "your-node-token-key"

scheduler:
  max_retries: 3
  retry_interval: "5m"
  batch_size: 3
  task_timeout: "30m"
  timeout_check: "1m"
```

### 4. 启动服务端

```bash
cd secflow-server
go run cmd/server/main.go
```

### 5. 配置前端

```bash
cd secflow-web

# 安装依赖
npm install

# 开发模式启动
npm run dev
```

### 6. 配置客户端

```bash
cd secflow-client

# 复制配置示例
cp client.example.yaml client.yaml

# 编辑配置
vim client.yaml
```

**配置内容**:

```yaml
server:
  api_url: "http://localhost:8080/api/v1"
  ws_url: "ws://localhost:8080/api/v1/ws/node"
  token_key: "your-node-token-key"

db_path: "./secflow.db"
heartbeat_interval: 30s
log_level: info
name: "dev-node-01"
```

### 7. 启动客户端

```bash
cd secflow-client
go run cmd/client/main.go
```

---

## 生产环境部署

### 1. 服务器准备

```bash
# 创建部署目录
sudo mkdir -p /opt/secflow
sudo chown -R $USER:$USER /opt/secflow

# 创建数据目录
sudo mkdir -p /opt/secflow/{data,logs,uploads}
sudo chown -R $USER:$USER /opt/secflow/{data,logs,uploads}
```

### 2. 编译二进制文件

```bash
# 编译服务端
cd secflow-server
GOOS=linux GOARCH=amd64 go build -o bin/secflow-server ./cmd/server/main.go

# 编译客户端
cd secflow-client
GOOS=linux GOARCH=amd64 go build -o bin/secflow-client ./cmd/client/main.go

# 打包前端
cd secflow-web
npm install
npm run build
```

### 3. 配置 Nginx

```nginx
# /etc/nginx/sites-available/secflow

server {
    listen 80;
    server_name secflow.example.com;

    # 重定向到 HTTPS
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name secflow.example.com;

    # SSL 配置
    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256;
    ssl_prefer_server_ciphers on;

    # 前端静态文件
    location / {
        root /opt/secflow/secflow-web/dist;
        try_files $uri $uri/ /index.html;
        expires 7d;
        add_header Cache-Control "public, immutable";
    }

    # API 代理
    location /api/ {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_cache_bypass $http_upgrade;
    }

    # WebSocket 代理
    location /ws/ {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_read_timeout 86400;
    }

    # 上传文件
    location /uploads/ {
        alias /opt/secflow/uploads/;
        expires 30d;
        add_header Cache-Control "public";
    }

    # 日志
    access_log /var/log/nginx/secflow-access.log;
    error_log /var/log/nginx/secflow-error.log;
}
```

### 4. 配置 Systemd 服务

**服务端服务文件** (`/etc/systemd/system/secflow-server.service`):

```ini
[Unit]
Description=SecFlow Server
After=network.target mongodb.service redis.service
Wants=mongodb.service redis.service

[Service]
Type=simple
User=secflow
WorkingDirectory=/opt/secflow
Environment="SECFLOW_CONFIG=/opt/secflow/config/config.yaml"
ExecStart=/opt/secflow/bin/secflow-server
Restart=always
RestartSec=5
StandardOutput=append:/opt/secflow/logs/server.log
StandardError=append:/opt/secflow/logs/server-error.log

[Install]
WantedBy=multi-user.target
```

**启用服务**:

```bash
# 重载 systemd
sudo systemctl daemon-reload

# 启用开机自启
sudo systemctl enable secflow-server

# 启动服务
sudo systemctl start secflow-server

# 查看状态
sudo systemctl status secflow-server
```

### 5. 配置防火墙

```bash
# Ubuntu/Debian (ufw)
sudo ufw allow 22/tcp    # SSH
sudo ufw allow 80/tcp    # HTTP
sudo ufw allow 443/tcp   # HTTPS
sudo ufw enable

# CentOS/RHEL (firewalld)
sudo firewall-cmd --permanent --add-port=80/tcp
sudo firewall-cmd --permanent --add-port=443/tcp
sudo firewall-cmd --reload
```

---

## Docker 部署

### docker-compose.yml

```yaml
version: '3.8'

services:
  # MongoDB
  mongodb:
    image: mongo:7.0
    container_name: secflow-mongodb
    restart: unless-stopped
    environment:
      MONGO_INITDB_ROOT_USERNAME: secflow
      MONGO_INITDB_ROOT_PASSWORD: ${MONGO_PASSWORD:-secflow_pass}
      MONGO_INITDB_DATABASE: secflow
    volumes:
      - mongodb_data:/data/db
    ports:
      - "27017:27017"
    networks:
      - secflow-network

  # Redis
  redis:
    image: redis:7-alpine
    container_name: secflow-redis
    restart: unless-stopped
    command: redis-server --requirepass ${REDIS_PASSWORD:-secflow_redis_pass}
    volumes:
      - redis_data:/data
    ports:
      - "6379:6379"
    networks:
      - secflow-network

  # 服务端
  server:
    build:
      context: ./secflow-server
      dockerfile: Dockerfile
    container_name: secflow-server
    restart: unless-stopped
    environment:
      - CONFIG_PATH=/app/config/config.yaml
    volumes:
      - ./secflow-server/config:/app/config:ro
      - ./secflow-data/uploads:/app/uploads
      - ./secflow-data/logs:/app/logs
    depends_on:
      - mongodb
      - redis
    networks:
      - secflow-network

  # 前端 (可选，生产环境建议 Nginx)
  web:
    build:
      context: ./secflow-web
      dockerfile: Dockerfile
    container_name: secflow-web
    restart: unless-stopped
    volumes:
      - ./secflow-web/dist:/usr/share/nginx/html:ro
    ports:
      - "3000:80"
    networks:
      - secflow-network

volumes:
  mongodb_data:
  redis_data:

networks:
  secflow-network:
    driver: bridge
```

### 环境变量文件 (.env)

```bash
# 数据库密码
MONGO_PASSWORD=secflow_pass
REDIS_PASSWORD=secflow_redis_pass

# JWT 配置
JWT_SECRET=your-super-secret-jwt-key-change-in-production
NODE_TOKEN_KEY=your-node-token-key
```

### 启动集群

```bash
# 构建并启动所有服务
docker-compose up -d

# 查看服务状态
docker-compose ps

# 查看日志
docker-compose logs -f server

# 停止服务
docker-compose down

# 完全清理
docker-compose down -v
```

### 客户端 Docker 部署

```dockerfile
# secflow-client/Dockerfile
FROM golang:1.21-alpine AS builder

WORKDIR /build
COPY . .
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o secflow-client ./cmd/client/main.go

FROM alpine:latest
RUN apk add --no-cache chromium

COPY --from=builder /build/secflow-client /usr/local/bin/
COPY --from=builder /build/client.yaml /etc/secflow/client.yaml

ENV SECFLOW_CONFIG=/etc/secflow/client.yaml
ENTRYPOINT ["secflow-client"]
```

```bash
# 在各节点上运行客户端
docker run -d \
  --name secflow-client \
  --restart unless-stopped \
  -v /path/to/client.yaml:/etc/secflow/client.yaml \
  secflow-client:latest
```

---

## 配置说明

### 服务端配置 (config.yaml)

```yaml
server:
  host: "0.0.0.0"
  port: 8080
  mode: "release"  # debug | release

mongodb:
  uri: "mongodb://user:pass@host:27017/secflow?authSource=admin"
  database: "secflow"

redis:
  addr: "host:6379"
  password: "redis-password"
  db: 0

jwt:
  secret: "${JWT_SECRET}"  # 使用环境变量
  expire: "72h"

node:
  token_key: "${NODE_TOKEN_KEY}"

scheduler:
  max_retries: 3           # 最大重试次数
  retry_interval: "5m"     # 重试间隔
  batch_size: 5            # 批量分发大小
  task_timeout: "30m"      # 任务超时
  timeout_check: "1m"      # 超时检查间隔
```

### 安全加固

```yaml
# 生产环境建议的安全配置

jwt:
  secret: ""  # 必须设置强密码
  expire: "24h"  # 生产环境建议 24 小时

server:
  mode: "release"

scheduler:
  max_retries: 3
  task_timeout: "15m"
```

---

## 监控运维

### Prometheus 监控

配置 Prometheus (`monitoring/prometheus.yml`):

```yaml
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: 'secflow-server'
    static_configs:
      - targets: ['server:8080']
    metrics_path: '/metrics'

  - job_name: 'secflow-nodes'
    static_configs:
      - targets: ['node1:8080', 'node2:8080']
```

### Grafana Dashboard

关键监控指标:

| 指标 | 说明 | 告警阈值 |
|------|------|----------|
| `secflow_queue_length` | 任务队列长度 | > 100 |
| `secflow_tasks_total{status="failed"}` | 失败任务数 | > 10/min |
| `secflow_nodes_total` | 在线节点数 | = 0 |
| `secflow_http_duration_seconds` | API 延迟 | > 1s |

### 日志管理

```bash
# 服务端日志
tail -f /opt/secflow/logs/server.log | jq

# 客户端日志
docker logs -f secflow-client

# 查看错误日志
grep -i error /opt/secflow/logs/server.log | tail -100
```

### Redis 监控

```bash
# 查看队列长度
redis-cli -a secflow_redis_pass LLEN secflow:tasks:pending
redis-cli -a secflow_redis_pass ZCARD secflow:tasks:priority
redis-cli -a secflow_redis_pass ZCARD secflow:tasks:retry

# 查看在线节点
redis-cli -a secflow_redis_pass ZRANGE secflow:nodes:heartbeat 0 -1 WITHSCORES

# 清理过期数据
redis-cli -a secflow_redis_pass FLUSHDB  # ⚠️ 慎用
```

### MongoDB 监控

```bash
# 连接数据库
mongosh "mongodb://secflow:secflow_pass@localhost:27017/secflow"

# 查看集合统计
db.vuln_records.stats()
db.tasks.stats()
db.nodes.stats()

# 查看慢查询
db.setProfilingLevel(1, { slowms: 100 })
db.system.profile.find().pretty()

# 清理过期数据
db.vuln_records.deleteMany({
  created_at: { $lt: new Date(Date.now() - 90 * 24 * 60 * 60 * 1000) }
})
```

---

## 故障排查

### 常见问题

#### 1. 服务启动失败

```bash
# 查看错误日志
journalctl -u secflow-server -n 100 --no-pager

# 检查端口占用
ss -tlnp | grep 8080

# 检查配置文件语法
go run cmd/server/main.go --validate-config
```

#### 2. MongoDB 连接失败

```bash
# 测试 MongoDB 连接
mongosh "mongodb://secflow:secflow_pass@localhost:27017/secflow" --eval "db.runCommand({ ping: 1 })"

# 查看 MongoDB 日志
docker logs secflow-mongodb
```

#### 3. Redis 连接失败

```bash
# 测试 Redis 连接
redis-cli -a secflow_redis_pass ping

# 查看 Redis 日志
docker logs secflow-redis
```

#### 4. 节点不在线

```bash
# 1. 检查节点服务状态
systemctl status secflow-client
docker logs secflow-client

# 2. 检查网络连通性
curl -v http://server:8080/health

# 3. 检查 Token 配置
grep token_key /etc/secflow/client.yaml

# 4. 重启节点服务
systemctl restart secflow-client
```

#### 5. 任务卡住

```bash
# 1. 查看任务状态
curl http://localhost:8080/api/v1/tasks?status=running

# 2. 检查 Redis 队列
redis-cli LLEN secflow:tasks:pending

# 3. 查看节点日志
curl http://localhost:8080/api/v1/nodes

# 4. 手动停止任务
curl -X POST http://localhost:8080/api/v1/tasks/{task_id}/stop

# 5. 清理卡住的任务
redis-cli -a pass ZREMRANGEBYSCORE secflow:tasks:retry -inf $(date +%s)
```

#### 6. 前端无法访问

```bash
# 检查 Nginx 状态
systemctl status nginx

# 检查 Nginx 配置
nginx -t

# 查看 Nginx 日志
tail -f /var/log/nginx/secflow-error.log
```

### 性能优化

```bash
# MongoDB 索引优化
mongosh "mongodb://user:pass@host/secflow"
db.vuln_records.createIndex({ "key": 1 }, { unique: true })
db.vuln_records.createIndex({ "created_at": -1 })
db.vuln_records.createIndex({ "source": 1, "created_at": -1 })

# Redis 内存优化
redis-cli -a pass CONFIG SET maxmemory 2gb
redis-cli -a pass CONFIG SET maxmemory-policy allkeys-lru

# 客户端并发优化
# 在 client.yaml 中调整浏览器池大小
browser_pool_size: 4  # 根据 CPU 核心数调整
```

### 数据备份

```bash
# MongoDB 备份
mongodump "mongodb://user:pass@host:27017/secflow" \
  --out /opt/secflow/backups/mongo-$(date +%Y%m%d)

# Redis 备份
redis-cli -a pass SAVE
cp /var/lib/redis/dump.rdb /opt/secflow/backups/redis-$(date +%Y%m%d).rdb

# 自动化备份脚本
0 2 * * * /opt/secflow/scripts/backup.sh
```
