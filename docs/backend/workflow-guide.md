# 开发工作流程指南

本文档描述 SecFlow 项目的开发流程和规范。

## 环境准备

### 1. 安装依赖

```bash
# 安装 Go 1.21+
go version

# 安装 Node.js 18+
node -v

# 安装 pnpm
npm install -g pnpm

# 安装 Docker Desktop
docker --version
```

### 2. 克隆项目

```bash
git clone https://github.com/your-org/secflow.git
cd secflow
```

### 3. 启动基础设施

```bash
# 启动 MongoDB 和 Redis
brew services start mongodb-community
brew services start redis

# 或使用 Docker
docker run -d --name mongodb -p 27017:27017 mongo:7
docker run -d --name redis -p 6379:6379 redis:7
```

### 4. 配置环境

```bash
# 服务端配置
cd secflow-server
cp config/config.yaml.example config/config.yaml
# 编辑 config.yaml

# 客户端配置
cd ../secflow-client
cp client.yaml.example client.yaml
# 编辑 client.yaml

# 前端依赖
cd ../secflow-web
pnpm install
```

---

## 开发流程

### 日常开发循环

```
1. 创建/切换分支
   ↓
2. 编写代码
   ↓
3. 运行测试
   ↓
4. 本地验证
   ↓
5. 提交代码
   ↓
6. 推送并创建 PR
```

---

## Git 工作流程

### 1. 创建功能分支

```bash
# 确保在最新主分支
git checkout main
git pull origin main

# 创建功能分支
git checkout -b feature/your-feature-name

# 或创建修复分支
git checkout -b fix/bug-description
```

### 2. 提交规范

遵循 [Conventional Commits](https://www.conventionalcommits.org/)：

```
<type>(<scope>): <subject>

[optional body]

[optional footer]
```

**Type 类型：**

| Type | 说明 |
|------|------|
| `feat` | 新功能 |
| `fix` | Bug 修复 |
| `docs` | 文档更新 |
| `style` | 代码格式（不影响功能） |
| `refactor` | 重构（不是新功能或修复） |
| `perf` | 性能优化 |
| `test` | 测试 |
| `chore` | 构建/工具 |

**示例：**

```bash
# 好的提交
git commit -m "feat(queue): add priority queue support"
git commit -m "fix(crawler): handle WAF timeout correctly"
git commit -m "docs(api): update endpoint documentation"
git commit -m "refactor(vuln): extract common parser logic"

# 不好的提交
git commit -m "fix stuff"
git commit -m "update"
git commit -m "WIP"
```

### 3. 提交消息模板

创建 `.gitmessage` 文件：

```bash
git config commit.template .gitmessage
```

```txt
# <type>(<scope>): <subject>
#
# Body lines should wrap at 72 characters.
#
# Footer: Closes #<issue>

# Types: feat | fix | docs | style | refactor | perf | test | chore
# Scope: api | queue | crawler | pusher | frontend | etc.
```

---

## 代码规范

### Go 代码规范

1. **格式化**
   ```bash
   go fmt ./...
   ```

2. **代码检查**
   ```bash
   go vet ./...
   golint ./...
   ```

3. **运行所有检查**
   ```bash
   make lint
   # 或
   go run honnef.co/go/tools/cmd/staticcheck ./...
   ```

4. **导入排序**
   ```go
   import (
       // 标准库
       "context"
       "encoding/json"
       "fmt"

       // 第三方库
       "github.com/gin-gonic/gin"
       "github.com/rs/zerolog"

       // 项目内部
       "github.com/secflow/server/internal/model"
   )
   ```

### TypeScript 代码规范

1. **ESLint + Prettier**
   ```bash
   cd secflow-web
   pnpm lint
   pnpm lint:fix
   ```

2. **类型优先**
   ```typescript
   // ✅ 好的
   interface User {
     id: string;
     name: string;
   }

   // ❌ 避免
   const user = { id: '1', name: 'test' };
   ```

---

## 测试规范

### Go 测试

```bash
# 运行所有测试
go test ./...

# 运行测试并显示覆盖率
go test -cover ./...

# 运行测试并显示详细输出
go test -v ./...

# 运行特定包的测试
go test -v ./internal/api/handler/

# 运行特定测试
go test -v -run TestExampleHandler_List

# 生成覆盖率报告
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

### 前端测试

```bash
cd secflow-web

# 运行所有测试
pnpm test

# 监听模式
pnpm test:watch

# 生成覆盖率
pnpm test:coverage
```

### E2E 测试

```bash
# 启动服务
docker-compose up -d

# 运行 E2E 测试
pnpm test:e2e

# 运行特定 E2E 测试
pnpm test:e2e --spec tests/example.spec.ts
```

---

## 本地验证

### 服务端

```bash
cd secflow-server

# 开发模式（热重载）
air

# 或普通运行
go run cmd/server/main.go
```

### 客户端

```bash
cd secflow-client

# 运行客户端
go run cmd/client/main.go
```

### 前端

```bash
cd secflow-web

# 开发服务器
pnpm dev

# 生产构建
pnpm build

# 预览生产构建
pnpm preview
```

### 完整堆栈

```bash
# 使用 Docker Compose 启动所有服务
docker-compose up -d

# 查看日志
docker-compose logs -f

# 查看特定服务日志
docker-compose logs -f secflow-server
```

---

## 代码审查

### 提交前检查清单

```markdown
- [ ] 代码符合格式规范
- [ ] 通过所有测试
- [ ] 添加了必要的单元测试
- [ ] 更新了相关文档
- [ ] 提交消息符合规范
- [ ] 分支已同步最新主分支
```

### Pull Request 规范

**PR 标题格式：**
```
feat(scope): add new feature
fix(scope): fix bug description
docs(scope): update documentation
refactor(scope): improve code structure
```

**PR 描述模板：**

```markdown
## 描述
简要说明这个 PR 做了什么。

## 变更内容
- 列出主要的变更点
- ...

## 相关 Issue
Closes #123

## 测试
- [ ] 单元测试通过
- [ ] E2E 测试通过
- [ ] 手动测试通过

## 截图（如有 UI 变更）
...
```

### 审查检查清单

- [ ] 代码逻辑正确
- [ ] 没有明显的 Bug
- [ ] 符合代码规范
- [ ] 有适当的测试覆盖
- [ ] 没有性能问题
- [ ] 文档已更新
- [ ] 没有安全漏洞

---

## 部署流程

### 开发环境

```bash
# 使用 docker-compose
docker-compose up -d

# 或手动启动
# 1. 启动 MongoDB 和 Redis
# 2. 启动服务端
# 3. 启动客户端
# 4. 启动前端
```

### 生产环境

```bash
# 1. 确保在 main 分支
git checkout main
git pull origin main

# 2. 构建镜像
docker build -t secflow-server:latest ./secflow-server
docker build -t secflow-client:latest ./secflow-client
docker build -t secflow-web:latest ./secflow-web

# 3. 使用生产配置
docker-compose -f docker-compose.prod.yml up -d

# 4. 查看日志
docker-compose -f docker-compose.prod.yml logs -f
```

---

## 故障排查

### 服务无法启动

```bash
# 检查端口占用
lsof -i :8080
lsof -i :3000

# 检查配置
cat config/config.yaml

# 检查日志
tail -f server.log
```

### 数据库连接问题

```bash
# 测试 MongoDB 连接
mongosh "mongodb://localhost:27017/secflow"

# 测试 Redis 连接
redis-cli ping
```

### 客户端连接问题

```bash
# 检查服务端是否运行
curl http://localhost:8080/health

# 检查 WebSocket 端点
wscat -c ws://localhost:8080/api/v1/ws/node

# 查看客户端日志
tail -f client.log
```

---

## 常用脚本

### Makefile

```makefile
# 开发
make dev          # 启动开发环境
make test        # 运行测试
make lint        # 代码检查

# 构建
make build       # 构建所有组件
make build-server
make build-client
make build-web

# 部署
make deploy      # 部署到生产环境
make logs        # 查看日志

# 清理
make clean       # 清理构建产物
make prune       # 清理 Docker 资源
```

---

## 版本管理

### 语义化版本

```
主版本.次版本.修订号
major.minor.patch

例：v1.2.3
- v1: 主版本（不兼容的 API 变更）
- v2: 次版本（向后兼容的功能添加）
- 3: 修订号（向后兼容的修复）
```

### 发布流程

```bash
# 1. 更新版本号
git tag -a v1.2.0 -m "Release v1.2.0"

# 2. 推送标签
git push origin v1.2.0

# 3. GitHub Actions 自动构建和发布
```

---

## 持续集成

### GitHub Actions 工作流

```yaml
# .github/workflows/ci.yml
name: CI

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.21'
      - name: Run tests
        run: go test -v ./...

  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Run linter
        run: golint ./...
```

---

## 协作规范

### 命名规范

| 类型 | 规范 | 示例 |
|------|------|------|
| 分支 | `feature/xxx` / `fix/xxx` | `feature/add-waf-bypass` |
| 提交 | `<type>(<scope>): <subject>` | `feat(queue): add priority` |
| PR | `<type>: <description>` | `feat: add new crawler` |

### 沟通

- 使用 GitHub Issues 跟踪任务
- 使用 PR 进行代码审查
- 在 PR 中描述变更内容
- 及时响应 Review 意见

---

## 资源链接

- [Go 官方文档](https://go.dev/doc/)
- [Gin 框架文档](https://gin-gonic.com/)
- [Vue 3 文档](https://vuejs.org/)
- [TypeScript 文档](https://www.typescriptlang.org/)
- [MongoDB 文档](https://docs.mongodb.com/)
- [Redis 文档](https://redis.io/documentation)
