# SecFlow 部署指南

## 目录

- [快速开始](#快速开始)
- [生产环境部署](#生产环境部署)
- [macvlan 网络配置](#macvlan-网络配置)
- [配置参考](#配置参考)
- [故障排查](#故障排查)

---

## 快速开始

### 1. 下载部署包
```bash
tar -xzvf secflow-deploy.tar.gz
cd secflow
```

### 2. 配置环境变量
```bash
cp .env.example .env
# 编辑 .env 设置密码和密钥
```

### 3. 启动服务
```bash
docker compose up -d
```

### 4. 检查状态
```bash
docker ps
curl http://localhost:8989/health
```

### 5. 注册管理员
访问 http://localhost:3000，注册第一个用户（自动成为管理员）

---

## 生产环境部署

### 1. 服务器要求

| 配置 | 最低要求 |
|------|----------|
| CPU | 2 核 |
| 内存 | 4 GB |
| 磁盘 | 50 GB |
| 系统 | Ubuntu 20.04+ / Debian 11+ |

### 2. 安装 Docker
```bash
# 安装 Docker
curl -fsSL https://get.docker.com | sh

# 添加用户到 docker 组
sudo usermod -aG docker $USER
```

### 3. 配置防火墙
```bash
# 开放端口
sudo ufw allow 8989/tcp  # 后端 API
sudo ufw allow 3000/tcp  # 前端 Web
```

### 4. 使用 systemd 管理服务
```bash
sudo tee /etc/systemd/system/secflow.service << 'EOF'
[Unit]
Description=SecFlow Service
Requires=docker.service
After=docker.service

[Service]
Type=oneshot
RemainAfterExit=yes
WorkingDirectory=/opt/secflow
ExecStart=/usr/local/bin/docker compose up -d
ExecStop=/usr/local/bin/docker compose down
TimeoutStartSec=0

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
sudo systemctl enable secflow
```

---

## macvlan 网络配置

### 什么是 macvlan？

macvlan 允许容器直接使用物理网络 IP，实现容器与宿主机在同一网络中通信。

### 1. 创建 macvlan 网络

```bash
# 查看宿主机网卡名称
ip addr

# 创建 macvlan 网络
docker network create -d macvlan \
  --subnet=172.16.100.0/24 \
  --gateway=172.16.100.254 \
  -o parent=ens33 \
  pub_net
```

**参数说明：**
- `--subnet`: 物理网络网段 (172.16.100.0/24)
- `--gateway`: 物理网络网关 (172.16.100.254)
- `-o parent`: 宿主机网卡名称

### 2. 配置文件

创建 `.env` 文件：
```bash
# 网络配置
SECFLOW_NETWORK=pub_net
SECFLOW_NETWORK_EXTERNAL=true

# MongoDB
MONGO_ROOT_USER=secflow
MONGO_ROOT_PASS=your_secure_password
MONGO_DATABASE=secflow

# Redis
REDIS_PASSWORD=your_redis_password

# JWT
JWT_SECRET=your_jwt_secret_min_32_chars

# Node Token (客户端认证)
NODE_TOKEN_KEY=your_node_token_key
```

### 3. 分配静态 IP

docker-compose.yml 中已预配置静态 IP：

```yaml
services:
  mongo:   # 172.16.100.30
  redis:   # 172.16.100.31
  server:  # 172.16.100.32
  web:     # 172.16.100.33
  client:  # 172.16.100.34
```

**IP 分配表：**

| 服务 | IP 地址 | 端口 | 说明 |
|------|---------|------|------|
| mongo | 172.16.100.30 | 27017 | MongoDB 数据库 |
| redis | 172.16.100.31 | 6379 | Redis 缓存/队列 |
| server | 172.16.100.32 | 8989 | 后端 API |
| web | 172.16.100.33 | 80/3000 | 前端界面 |
| client | 172.16.100.34 | - | 漏洞爬取节点 |

### 4. 启动服务
```bash
docker compose up -d
```

### 5. 验证网络
```bash
# 检查容器 IP
docker exec secflow_server ip addr

# 测试连通性
docker exec secflow_server ping -c 3 192.168.20.1
```

---

## 客户端节点部署

### 客户端连接说明

客户端（secflow-client）以 **server 模式** 运行时，需要通过 WebSocket 连接到服务器。

### 1. 服务端配置

确保服务器有固定 IP，例如 `192.168.20.12`：
```yaml
server:
  networks:
    secflow-net:
      ipv4_address: 192.168.20.12
```

### 2. 客户端配置

客户端配置 `client.yaml`：
```yaml
server:
  api_url: "http://192.168.20.12:8989/api/v1"
  ws_url: "ws://192.168.20.12:8989/api/v1/ws/node"
  token_key: "your-node-token-key"
```

### 3. 多节点部署

在不同机器上部署客户端时，只需要修改 `client.yaml` 中的服务器地址：

```yaml
# 北京节点
server:
  api_url: "http://服务器公网IP:8989/api/v1"
  ws_url: "ws://服务器公网IP:8989/api/v1/ws/node"
  token_key: "your-node-token-key"
```

### 4. 客户端容器部署

客户端使用 Docker 部署时，确保网络可以访问服务器：
```bash
docker run -d \
  --name secflow-client \
  -e SECFLOW_API_URL="http://192.168.20.12:8989/api/v1" \
  -e SECFLOW_WS_URL="ws://192.168.20.12:8989/api/v1/ws/node" \
  -e SECFLOW_TOKEN_KEY="your-node-token-key" \
  secflow/client:latest
```

---

## 配置参考

### 环境变量

| 变量名 | 默认值 | 说明 |
|--------|--------|------|
| `SECFLOW_PORT` | 8989 | 服务端口 |
| `SECFLOW_NETWORK` | secflow_dev_network | 网络名称 |
| `SECFLOW_NETWORK_EXTERNAL` | false | 是否使用外部网络 |
| `MONGO_ROOT_USER` | secflow | MongoDB 用户名 |
| `MONGO_ROOT_PASS` | secflow_pass | MongoDB 密码 |
| `MONGO_DATABASE` | secflow | MongoDB 数据库名 |
| `REDIS_PASSWORD` | secflow_redis_pass | Redis 密码 |
| `JWT_SECRET` | dev-secret... | JWT 密钥 |
| `NODE_TOKEN_KEY` | secflow-node-token | 客户端认证密钥 |

### 端口映射

| 服务 | 容器端口 | 宿主机端口 | 说明 |
|------|----------|------------|------|
| server | 8989 | 8989 | 后端 API |
| web | 80 | 3000 | 前端 Web |
| mongo | 27017 | 27017 | MongoDB |
| redis | 6379 | 6379 | Redis |

---

## 故障排查

### 1. 服务无法启动
```bash
# 查看日志
docker compose logs server
docker compose logs mongo
docker compose logs redis

# 检查端口占用
netstat -tlnp | grep 8989
```

### 2. 数据库连接失败
```bash
# 检查 MongoDB
docker exec secflow_mongo mongosh -u secflow -p password

# 检查 Redis
docker exec secflow_redis redis-cli -a password
```

### 3. 客户端无法连接
```bash
# 检查 token 是否匹配
grep TOKEN_KEY .env
docker exec secflow_client env | grep TOKEN

# 查看客户端日志
docker logs secflow_client
```

### 4. 网络问题
```bash
# 查看网络
docker network ls
docker network inspect secflow_pub_net

# 测试网络连通性
docker exec secflow_server ping -c 3 mongo
```

### 5. 重置数据库
```bash
# 删除所有数据
docker compose down -v

# 重新启动
docker compose up -d
```

---

## 安全建议

1. **修改所有默认密码**
   - MongoDB 密码
   - Redis 密码
   - JWT Secret
   - Node Token Key

2. **使用 HTTPS**
   - 配置 Nginx 反向代理
   - 启用 SSL/TLS 证书

3. **限制网络访问**
   - 使用防火墙规则
   - 禁止外部访问数据库端口

4. **定期更新**
   ```bash
   docker compose pull
   docker compose up -d
   ```
