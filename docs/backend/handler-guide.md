# Handler 开发指南

本文档指导你如何在 SecFlow 后端项目中开发新的 Handler（API 处理器）。

## Handler 结构

Handler 是处理 HTTP 请求的核心组件。每个 Handler 负责一类资源的 CRUD 操作。

```
internal/api/handler/
├── article.go          # 文章 Handler
├── audit_log.go        # 审计日志 Handler
├── auth.go             # 认证 Handler
├── node.go             # 节点 Handler
├── push_channel.go     # 推送渠道 Handler
├── report.go           # 报告 Handler
├── response.go         # 统一响应工具
├── task.go             # 任务 Handler
├── user.go             # 用户 Handler
└── vuln.go             # 漏洞 Handler
```

---

## 创建新 Handler 的步骤

### Step 1: 定义 Handler 结构

```go
// internal/api/handler/example.go
package handler

import (
    "net/http"

    "github.com/gin-gonic/gin"
    "github.com/secflow/server/internal/repository"
)

// ExampleHandler 处理 example 资源
type ExampleHandler struct {
    exampleRepo *repository.ExampleRepo
}

// NewExampleHandler 构造函数
func NewExampleHandler(er *repository.ExampleRepo) *ExampleHandler {
    return &ExampleHandler{
        exampleRepo: er,
    }
}
```

---

### Step 2: 实现 List 方法（列表查询）

```go
// List 返回分页列表
//
// GET /api/v1/examples
func (h *ExampleHandler) List(c *gin.Context) {
    page, pageSize := pageParams(c)

    // 从查询参数构建筛选条件
    filter := repository.ExampleListFilter{
        Name:     c.Query("name"),
        Status:   c.Query("status"),
        Page:     page,
        PageSize: pageSize,
    }

    items, total, err := h.exampleRepo.List(c, filter)
    if err != nil {
        fail(c, http.StatusInternalServerError, err.Error())
        return
    }

    okPage(c, total, page, pageSize, items)
}
```

---

### Step 3: 实现 Get 方法（单个查询）

```go
// Get 返回单个资源
//
// GET /api/v1/examples/:id
func (h *ExampleHandler) Get(c *gin.Context) {
    id := c.Param("id")

    item, err := h.exampleRepo.GetByID(c, id)
    if err != nil {
        fail(c, http.StatusInternalServerError, err.Error())
        return
    }
    if item == nil {
        fail(c, http.StatusNotFound, "example not found")
        return
    }

    ok(c, item)
}
```

---

### Step 4: 实现 Create 方法（创建）

```go
// CreateExampleRequest 创建请求体
type CreateExampleRequest struct {
    Name    string `json:"name" binding:"required"`
    Type    string `json:"type"`
    Enabled bool   `json:"enabled"`
}

// Create 创建新资源
//
// POST /api/v1/examples
func (h *ExampleHandler) Create(c *gin.Context) {
    var req CreateExampleRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        fail(c, http.StatusBadRequest, err.Error())
        return
    }

    // 验证逻辑
    if len(req.Name) < 2 {
        fail(c, http.StatusBadRequest, "name must be at least 2 characters")
        return
    }

    item := &model.Example{
        Name:    req.Name,
        Type:    req.Type,
        Enabled: req.Enabled,
    }

    if err := h.exampleRepo.Create(c, item); err != nil {
        fail(c, http.StatusInternalServerError, "failed to create example: "+err.Error())
        return
    }

    ok(c, item)
}
```

---

### Step 5: 实现 Update 方法（更新）

```go
// UpdateExampleRequest 更新请求体
type UpdateExampleRequest struct {
    Name    string `json:"name"`
    Type    string `json:"type"`
    Enabled *bool  `json:"enabled"` // 使用指针区分"未设置"和"设置为 false"
}

// Update 更新资源
//
// PUT /api/v1/examples/:id
func (h *ExampleHandler) Update(c *gin.Context) {
    id := c.Param("id")

    // 先查询现有数据
    item, err := h.exampleRepo.GetByID(c, id)
    if err != nil {
        fail(c, http.StatusInternalServerError, err.Error())
        return
    }
    if item == nil {
        fail(c, http.StatusNotFound, "example not found")
        return
    }

    var req UpdateExampleRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        fail(c, http.StatusBadRequest, err.Error())
        return
    }

    // 只更新非空字段
    if req.Name != "" {
        item.Name = req.Name
    }
    if req.Type != "" {
        item.Type = req.Type
    }
    if req.Enabled != nil {
        item.Enabled = *req.Enabled
    }

    if err := h.exampleRepo.Update(c, id, item); err != nil {
        fail(c, http.StatusInternalServerError, "failed to update example: "+err.Error())
        return
    }

    ok(c, item)
}
```

---

### Step 6: 实现 Delete 方法（删除）

```go
// Delete 删除资源
//
// DELETE /api/v1/examples/:id
func (h *ExampleHandler) Delete(c *gin.Context) {
    id := c.Param("id")

    // 先检查是否存在
    item, err := h.exampleRepo.GetByID(c, id)
    if err != nil {
        fail(c, http.StatusInternalServerError, err.Error())
        return
    }
    if item == nil {
        fail(c, http.StatusNotFound, "example not found")
        return
    }

    if err := h.exampleRepo.Delete(c, id); err != nil {
        fail(c, http.StatusInternalServerError, "failed to delete example: "+err.Error())
        return
    }

    ok(c, nil)
}
```

---

## 响应工具函数

Handler 使用统一的响应格式：

```go
// 成功响应
ok(c, data)                    // 返回 data
okPage(c, total, page, size, items) // 返回分页数据

// 失败响应
fail(c, http.StatusBadRequest, "错误信息")
```

### 完整响应示例

```go
// ok - 成功
func ok(c *gin.Context, data interface{}) {
    c.JSON(http.StatusOK, gin.H{
        "code": 0,
        "message": "success",
        "data": data,
    })
}

// okPage - 成功（分页）
func okPage(c *gin.Context, total int64, page, pageSize int, items interface{}) {
    c.JSON(http.StatusOK, gin.H{
        "code": 0,
        "message": "success",
        "data": gin.H{
            "items": items,
            "total": total,
            "page": page,
            "page_size": pageSize,
        },
    })
}

// fail - 失败
func fail(c *gin.Context, status int, message string) {
    c.JSON(status, gin.H{
        "code": status,
        "message": message,
    })
}
```

---

## 分页参数解析

```go
// pageParams 从请求中解析分页参数
func pageParams(c *gin.Context) (page, pageSize int) {
    page = 1
    pageSize = 20

    if p := c.Query("page"); p != "" {
        if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
            page = parsed
        }
    }

    if ps := c.Query("page_size"); ps != "" {
        if parsed, err := strconv.Atoi(ps); err == nil && parsed > 0 {
            pageSize = parsed
            // 限制最大值
            if pageSize > 100 {
                pageSize = 100
            }
        }
    }

    return
}
```

---

## 路由注册

在 `internal/api/router.go` 中注册新 Handler：

```go
// 1. 引入 Handler
exampleH := handler.NewExampleHandler(exampleRepo)

// 2. 在路由组中注册
examples := api.Group("/examples")
{
    examples.GET("",      exampleH.List)
    examples.GET("/:id",  exampleH.Get)
    examples.POST("",      exampleH.Create)
    examples.PUT("/:id",  exampleH.Update)
    examples.DELETE("/:id", exampleH.Delete)
}

// 3. 添加权限控制（如需要）
examples.POST("", middleware.RequireRole("editor"), exampleH.Create)
```

---

## 权限控制

使用中间件控制访问权限：

```go
// RequireRole 返回需要特定角色的中间件
func RequireRole(roles ...string) gin.HandlerFunc {
    // ...
}

// 示例
api.POST("/examples", middleware.RequireRole("admin", "editor"), exampleH.Create)
api.DELETE("/examples/:id", middleware.RequireRole("admin"), exampleH.Delete)
```

---

## 请求绑定与验证

### 使用 Gin 绑定

```go
// JSON 绑定
var req CreateRequest
if err := c.ShouldBindJSON(&req); err != nil {
    fail(c, http.StatusBadRequest, err.Error())
    return
}

// 表单绑定
var req FormRequest
if err := c.ShouldBind(&req); err != nil {
    fail(c, http.StatusBadRequest, err.Error())
    return
}

// URI 参数绑定
var req URIRequest
if err := c.ShouldBindUri(&req); err != nil {
    fail(c, http.StatusBadRequest, err.Error())
    return
}
```

### 自定义验证

```go
type CreateRequest struct {
    Name string `json:"name" binding:"required,min=2,max=100"`
    Age  int    `json:"age" binding:"required,min=0,max=150"`
    Email string `json:"email" binding:"required,email"`
}

// 或使用自定义验证器
binding.AddRule("valid_status", func(v interface{}) error {
    s := v.(string)
    if s != "active" && s != "inactive" {
        return errors.New("status must be active or inactive")
    }
    return nil
})
```

---

## 错误处理

### 统一错误格式

```go
// 业务错误
fail(c, http.StatusBadRequest, "参数错误")

// 资源不存在
fail(c, http.StatusNotFound, "example not found")

// 权限不足
fail(c, http.StatusForbidden, "insufficient permissions")

// 服务器错误
fail(c, http.StatusInternalServerError, err.Error())
```

### 包装错误

```go
// 使用 fmt.Errorf 包装错误
err := doSomething()
if err != nil {
    return fmt.Errorf("doing something: %w", err)
}

// 在 Handler 中返回
if err != nil {
    fail(c, http.StatusInternalServerError, "failed to create: "+err.Error())
    return
}
```

---

## 审计日志

记录关键操作的审计日志：

```go
// 在 handler 中记录审计日志
func (h *ExampleHandler) Create(c *gin.Context) {
    // ... 创建逻辑 ...

    // 获取当前用户
    user, _ := c.Get("user")

    // 记录审计日志
    auditLog := &model.AuditLog{
        UserID:   user.ID,
        Username: user.Username,
        Action:   "create_example",
        Resource: "example",
        Detail:   fmt.Sprintf("Created example: %s", item.Name),
        IP:       c.ClientIP(),
    }
    h.auditRepo.Create(c, auditLog)

    ok(c, item)
}
```

---

## 完整示例：Stats 接口

有些 Handler 需要返回聚合统计数据：

```go
// Stats 返回统计数据
//
// GET /api/v1/examples/stats
func (h *ExampleHandler) Stats(c *gin.Context) {
    total, _ := h.exampleRepo.Count(c)

    // 按状态分组统计
    statuses := []string{"active", "inactive"}
    byStatus := make(map[string]int64, len(statuses))
    for _, status := range statuses {
        _, cnt, err := h.exampleRepo.List(c, repository.ExampleListFilter{
            Status:   status,
            PageSize: 1,
        })
        if err == nil {
            byStatus[status] = cnt
        }
    }

    ok(c, gin.H{
        "total":     total,
        "by_status": byStatus,
    })
}
```

---

## 测试 Handler

```go
// internal/api/handler/example_test.go
package handler

import (
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/gin-gonic/gin"
)

func TestExampleHandler_List(t *testing.T) {
    gin.SetMode(gin.TestMode)

    // 设置路由
    r := gin.New()
    handler := &ExampleHandler{}
    r.GET("/examples", handler.List)

    // 创建请求
    req, _ := http.NewRequest("GET", "/examples?page=1&page_size=10", nil)
    w := httptest.NewRecorder()

    r.ServeHTTP(w, req)

    // 验证响应
    if w.Code != http.StatusOK {
        t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
    }

    var resp map[string]interface{}
    json.Unmarshal(w.Body.Bytes(), &resp)

    if resp["code"].(float64) != 0 {
        t.Errorf("Expected code 0, got %v", resp["code"])
    }
}
```

---

## 最佳实践

1. **保持 Handler 简洁** - Handler 只负责请求解析和响应组装，业务逻辑放在 Repository
2. **统一错误处理** - 使用 `fail()` 函数返回错误
3. **参数验证** - 在 Handler 层验证输入参数
4. **分页限制** - 始终限制最大分页大小
5. **审计日志** - 记录关键操作的审计日志
6. **权限控制** - 使用中间件控制访问权限
7. **日志记录** - 在关键位置添加日志
8. **单元测试** - 为 Handler 编写单元测试
