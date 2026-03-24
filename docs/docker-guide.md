# SecFlow Docker 部署指南

## 快速开始

### 1. 环境要求

```bash
# 检查 Docker 版本
docker --version        # >= 24.0
docker compose version   # >= 2.0
```

### 2. 启动服务 (开发环境)

```bash
# 方式一: 使用快速脚本
./scripts/docker-quickstart.sh start

# 方式二: 直接使用 docker-compose
docker-compose up -d
```

### 3. 启动服务 (生产环境)

```bash
# 复制并编辑环境变量
cp .env.example .env
vim .env  # 修改密码和其他配置

# 启动生产环境
docker-compose -f docker-compose.prod.yml up -d
```

### 4. 访问服务

| 服务 | 地址 | 说明 |
|------|------|------|
| 前端界面 | http://localhost:3000 | Web UI |
| API 接口 | http://localhost:8080 | 后端 API |
| MongoDB | localhost:27017 | 数据库 |
| Redis | localhost:6379 | 缓存/队列 |

**默认账号**: `admin` / `admin123`

### 5. 查看日志

```bash
# 查看所有服务日志
docker-compose logs -f

# 查看指定服务日志
docker-compose logs -f server
docker-compose logs -f web
docker-compose logs -f mongo
```

### 6. 停止服务

```bash
docker-compose down

# 清理数据卷
docker-compose down -v
```

## 配置说明

### 环境变量 (.env)

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `MONGO_ROOT_USER` | secflow | MongoDB 用户名 |
| `MONGO_ROOT_PASS` | secflow_pass | MongoDB 密码 |
| `REDIS_PASSWORD` | secflow_redis_pass | Redis 密码 |
| `JWT_SECRET` | - | JWT 密钥 (**必须修改**) |
| `NODE_TOKEN_KEY` | - | 节点认证密钥 |

### 端口映射

开发环境 (`docker-compose.yml`):

| 主机端口 | 容器端口 | 服务 |
|----------|----------|------|
| 3000 | 80 | 前端 Web |
| 8080 | 8080 | 服务端 API |
| 27017 | 27017 | MongoDB |
| 6379 | 6379 | Redis |

### 客户端节点

在 `docker-compose.prod.yml` 中配置了三个地理分布的客户端节点:

- `client-beijing` - 北京节点
- `client-shanghai` - 上海节点
- `client-guangzhou` - 广州节点

如需启用客户端，取消对应服务的注释即可。

## 运维命令

```bash
# 重启服务
docker-compose restart

# 重新构建镜像
docker-compose build --no-cache

# 进入容器调试
docker exec -it secflow_server sh

# 查看资源使用
docker stats

# 查看服务健康状态
docker-compose ps
```

## 数据持久化

所有数据存储在 Docker volumes 中:

- `mongo_data` - MongoDB 数据
- `redis_data` - Redis 持久化数据
- `server_logs` - 服务端日志
- `grafana_data` - Grafana 配置和数据

## SSL/HTTPS 配置 (生产环境)

1. 获取 SSL 证书 (Let's Encrypt):

```bash
# 使用 certbot 获取证书
certbot certonly -d your-domain.com
```

2. 将证书复制到 `nginx/ssl/` 目录:

```bash
cp /etc/letsencrypt/live/your-domain.com/fullchain.pem nginx/ssl/
cp /etc/letsencrypt/live/your-domain.com/privkey.pem nginx/ssl/
```

3. 重启 Nginx 服务:

```bash
docker-compose -f docker-compose.prod.yml restart web
```

## 故障排查

### 服务启动失败

```bash
# 检查容器日志
docker-compose logs

# 检查端口占用
lsof -i :8080
```

### 数据库连接失败

```bash
# 检查 MongoDB 健康状态
docker-compose ps mongo
docker-compose logs mongo

# 手动测试连接
docker exec -it secflow_mongo mongosh -u secflow -p
```

### 客户端无法连接

```bash
# 检查服务端是否正常运行
curl http://localhost:8080/api/v1/health

# 检查客户端日志
docker-compose logs client-beijing
```

## 性能优化建议

1. **MongoDB**: 生产环境建议分配至少 4GB 内存
2. **Redis**: 限制内存使用 (`--maxmemory 1gb`)
3. **客户端**: 根据 CPU 核心数调整并发数量 (`SECFLOW_MAX_CONCURRENT`)
4. **日志**: 生产环境设置 `SECFLOW_LOG_LEVEL=warn`

## 备份与恢复

```bash
# 备份 MongoDB
docker exec secflow_mongo mongodump --archive=/tmp/backup.gz --gzip

# 恢复 MongoDB
docker exec -i secflow_mongo mongorestore --gzip --archive=/tmp/backup.gz
```
