# SecFlow 测试指南

本文档介绍 SecFlow 项目的测试策略和运行方法。

## 测试架构

```
┌─────────────────────────────────────────────────────────────┐
│                     测试金字塔                                 │
└─────────────────────────────────────────────────────────────┘

        ▲
       ╱ ╲
      ╱   ╲        集成测试 (tests/integration)
     ╱─────╲       - MongoDB 连接测试
    ╱       ╲      - Redis 连接测试
   ╱─────────╲     - WebSocket 通信测试
  ╱           ╲
 ╱─────────────╲   单元测试 (各包 *_test.go)
╱               ╲  - pkg/auth: 72.7%
                    - middleware: 78.7%
                    - handler: 1.3%
```

## 运行测试

### 快速运行

```bash
# 运行所有测试（含覆盖率）
cd secflow-server && go test ./... -cover

# 运行所有测试（详细输出）
go test ./... -v

# 运行测试并显示覆盖率
go test ./... -coverprofile=coverage.out && go tool cover -html=coverage.out
```

### 按包运行

```bash
# 认证包测试
go test ./pkg/auth/... -v

# 中间件测试
go test ./internal/api/middleware/... -v

# Handler 测试
go test ./internal/api/handler/... -v

# 模型测试
go test ./internal/model/... -v

# 集成测试（需要 MongoDB/Redis）
go test ./tests/integration/... -v
```

### 运行特定测试

```bash
# 按测试名称过滤
go test ./pkg/auth/... -v -run TestGenerateToken

# 运行包含关键词的测试
go test ./... -v -run "TestJWT|TestPassword"
```

## 测试覆盖

### 当前覆盖率

| 包 | 覆盖率 | 说明 |
|---|--------|------|
| `pkg/auth` | **72.7%** | JWT、bcrypt 密码测试 |
| `internal/api/middleware` | **78.7%** | 认证、CORS、角色、超时 |
| `internal/api/handler` | 1.3% | 密码重置测试 |
| `internal/model` | 0.0% | 模型结构测试 |
| `tests/integration` | 3.5% | 端到端测试 |

### 查看详细覆盖率

```bash
# 生成覆盖率报告
go test ./... -coverprofile=coverage.out -covermode=atomic

# 查看覆盖率前20的函数
go tool cover -func=coverage.out | head -20

# 生成 HTML 覆盖率报告
go tool cover -html=coverage.out -o coverage.html
# 然后在浏览器打开 coverage.html
```

## 测试文件结构

```
secflow-server/
├── pkg/
│   └── auth/
│       ├── auth.go           # 认证实现
│       └── auth_test.go     # 认证测试
│
├── internal/
│   ├── api/
│   │   ├── middleware/
│   │   │   ├── middleware.go
│   │   │   └── middleware_test.go
│   │   └── handler/
│   │       ├── password_reset.go
│   │       └── password_reset_test.go
│   │
│   └── model/
│       ├── model.go
│       └── model_test.go
│
└── tests/
    ├── auth_test.go         # Handler 集成测试
    ├── handler_test.go
    └── integration/         # 端到端测试
```

## 测试工具

### Mock 对象

项目使用手工 Mock 而非代码生成工具：

```go
// 示例：mockUserRepo
type mockUserRepo struct {
    users map[string]*model.User
}

func (m *mockUserRepo) GetByUsername(ctx context.Context, username string) (*model.User, error) {
    u, ok := m.users[username]
    if !ok {
        return nil, nil
    }
    return u, nil
}
```

### 测试辅助函数

```go
// 初始化 Gin 测试模式
gin.SetMode(gin.TestMode)

// 创建测试 HTTP 请求
w := httptest.NewRecorder()
c, _ := gin.CreateTestContext(w)
c.Request, _ = http.NewRequest("GET", "/test", nil)

// 验证响应
assert.Equal(t, http.StatusOK, w.Code)
assert.Contains(t, w.Body.String(), "expected")
```

## 编写新测试

### 单元测试示例

```go
package auth

import (
    "testing"
    "time"
    "github.com/stretchr/testify/assert"
)

func TestGenerateToken(t *testing.T) {
    svc := New("test-secret", 24*time.Hour)
    
    token, err := svc.GenerateToken("user123", "testuser", "admin")
    assert.NoError(t, err)
    assert.NotEmpty(t, token)
}

func TestParseToken(t *testing.T) {
    svc := New("test-secret", 24*time.Hour)
    
    // 生成 token
    token, _ := svc.GenerateToken("user123", "testuser", "admin")
    
    // 解析 token
    claims, err := svc.ParseToken(token)
    assert.NoError(t, err)
    assert.Equal(t, "user123", claims.UserID)
}
```

### Handler 测试示例

```go
package handler

import (
    "bytes"
    "net/http"
    "net/http/httptest"
    "testing"
    "github.com/gin-gonic/gin"
    "github.com/stretchr/testify/assert"
)

func TestPasswordResetRequest(t *testing.T) {
    gin.SetMode(gin.TestMode)
    
    h := &PasswordResetHandler{}
    
    w := httptest.NewRecorder()
    c, _ := gin.CreateTestContext(w)
    c.Request, _ = http.NewRequest("POST", "/", bytes.NewBufferString(`{"email":"test@example.com"}`))
    c.Request.Header.Set("Content-Type", "application/json")
    
    h.RequestReset(c)
    
    // 验证响应
    assert.Equal(t, http.StatusOK, w.Code)
}
```

## CI/CD 集成

### GitHub Actions 示例

```yaml
# .github/workflows/test.yml
name: Test

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    
    services:
      mongodb:
        image: mongo:7.0
        ports:
          - 27017:27017
      
      redis:
        image: redis:7.2-alpine
        ports:
          - 6379:6379
    
    steps:
      - uses: actions/checkout@v4
      
      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.23'
      
      - name: Run tests
        run: |
          cd secflow-server
          go test ./... -coverprofile=coverage.out
      
      - name: Upload coverage
        uses: codecov/codecov-action@v4
        with:
          file: secflow-server/coverage.out
```

## 常见问题

### Q: 集成测试需要 MongoDB/Redis 吗？

A: 是的。`tests/integration` 需要 MongoDB 和 Redis 运行：

```bash
# 使用 Docker 启动
docker run -d --name mongodb -p 27017:27017 mongo:7.0
docker run -d --name redis -p 6379:6379 redis:7.2-alpine

# 然后运行集成测试
go test ./tests/integration/... -v
```

### Q: 如何跳过慢速测试？

A: 使用 `-short` 标志：

```bash
go test ./... -short
```

### Q: 测试失败如何调试？

A: 使用 `-trace` 查看详细输出：

```bash
go test ./pkg/auth/... -v -trace
```
