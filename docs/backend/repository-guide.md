# Repository 开发指南

本文档指导你如何开发 Repository（数据访问层）。

## Repository 模式

Repository 模式封装了所有数据访问逻辑，Handler 只负责请求处理和响应组装。

```
Handler (API 层)
    ↓
Repository (数据访问层)
    ↓
MongoDB / Redis
```

---

## Repository 结构

```
internal/repository/
├── db.go              # 数据库连接
├── repository.go      # 基础 Repository 接口
├── extra.go           # 扩展查询方法
├── user.go            # User Repository
├── node.go            # Node Repository
├── task.go            # Task Repository
├── vuln.go            # VulnRecord Repository
├── article.go         # Article Repository
├── push_channel.go    # PushChannel Repository
├── audit_log.go       # AuditLog Repository
└── report.go          # Report Repository
```

---

## 创建新 Repository

### Step 1: 定义模型

```go
// internal/model/example.go
package model

type Example struct {
    ID        bson.ObjectID `bson:"_id,omitempty" json:"id"`
    Name      string        `bson:"name"           json:"name"`
    Type      string        `bson:"type"           json:"type"`
    Status    string        `bson:"status"         json:"status"`
    CreatedAt time.Time     `bson:"created_at"     json:"created_at"`
    UpdatedAt time.Time     `bson:"updated_at"     json:"updated_at"`
}
```

---

### Step 2: 定义列表筛选器

```go
// internal/repository/example.go
package repository

// ExampleListFilter 定义列表查询筛选条件
type ExampleListFilter struct {
    Name   string
    Type   string
    Status string
    Page   int
    PageSize int
}
```

---

### Step 3: 实现 Repository

```go
// ExampleRepo 数据访问层
type ExampleRepo struct {
    collection *mongo.Collection
}

// NewExampleRepo 构造函数
func NewExampleRepo(db *mongo.Database) *ExampleRepo {
    return &ExampleRepo{
        collection: db.Collection("examples"),
    }
}

// List 返回分页列表
func (r *ExampleRepo) List(ctx context.Context, filter ExampleListFilter) ([]*model.Example, int64, error) {
    // 构建查询条件
    query := bson.M{}
    if filter.Name != "" {
        query["name"] = bson.M{"$regex": filter.Name, "$options": "i"}
    }
    if filter.Type != "" {
        query["type"] = filter.Type
    }
    if filter.Status != "" {
        query["status"] = filter.Status
    }

    // 计数
    total, err := r.collection.CountDocuments(ctx, query)
    if err != nil {
        return nil, 0, err
    }

    // 分页查询
    skip := int64((filter.Page - 1) * filter.PageSize)
    limit := int64(filter.PageSize)

    cursor, err := r.collection.Find(ctx, query,
        options.Find().
            SetSkip(skip).
            SetLimit(limit).
            SetSort(bson.D{{Key: "created_at", Value: -1}}),
    )
    if err != nil {
        return nil, 0, err
    }
    defer cursor.Close(ctx)

    var items []*model.Example
    if err := cursor.All(ctx, &items); err != nil {
        return nil, 0, err
    }

    return items, total, nil
}

// GetByID 根据 ID 查询
func (r *ExampleRepo) GetByID(ctx context.Context, id string) (*model.Example, error) {
    objectID, err := primitive.ObjectIDFromHex(id)
    if err != nil {
        return nil, err
    }

    var item model.Example
    err = r.collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&item)
    if err != nil {
        if err == mongo.ErrNoDocuments {
            return nil, nil
        }
        return nil, err
    }

    return &item, nil
}

// GetByName 根据名称查询
func (r *ExampleRepo) GetByName(ctx context.Context, name string) (*model.Example, error) {
    var item model.Example
    err := r.collection.FindOne(ctx, bson.M{"name": name}).Decode(&item)
    if err != nil {
        if err == mongo.ErrNoDocuments {
            return nil, nil
        }
        return nil, err
    }

    return &item, nil
}

// Create 创建新记录
func (r *ExampleRepo) Create(ctx context.Context, item *model.Example) error {
    item.CreatedAt = time.Now()
    item.UpdatedAt = time.Now()

    result, err := r.collection.InsertOne(ctx, item)
    if err != nil {
        return err
    }

    item.ID = result.InsertedID.(primitive.ObjectID)
    return nil
}

// Update 更新记录
func (r *ExampleRepo) Update(ctx context.Context, id string, item *model.Example) error {
    objectID, err := primitive.ObjectIDFromHex(id)
    if err != nil {
        return err
    }

    item.UpdatedAt = time.Now()

    _, err = r.collection.UpdateOne(ctx,
        bson.M{"_id": objectID},
        bson.M{"$set": item},
    )
    return err
}

// Delete 删除记录
func (r *ExampleRepo) Delete(ctx context.Context, id string) error {
    objectID, err := primitive.ObjectIDFromHex(id)
    if err != nil {
        return err
    }

    _, err = r.collection.DeleteOne(ctx, bson.M{"_id": objectID})
    return err
}

// Count 统计总数
func (r *ExampleRepo) Count(ctx context.Context) (int64, error) {
    return r.collection.CountDocuments(ctx, bson.M{})
}
```

---

## 高级查询

### 聚合查询

```go
// GetStatsByStatus 按状态分组统计
func (r *ExampleRepo) GetStatsByStatus(ctx context.Context) (map[string]int64, error) {
    pipeline := mongo.Pipeline{
        // 按 status 分组
        {{Key: "$group", Value: bson.D{
            {Key: "_id", Value: "$status"},
            {Key: "count", Value: bson.D{{Key: "$sum", Value: 1}}},
        }}},
    }

    cursor, err := r.collection.Aggregate(ctx, pipeline)
    if err != nil {
        return nil, err
    }
    defer cursor.Close(ctx)

    stats := make(map[string]int64)
    for cursor.Next(ctx) {
        var result struct {
            Status string `bson:"_id"`
            Count  int64  `bson:"count"`
        }
        if err := cursor.Decode(&result); err != nil {
            continue
        }
        stats[result.Status] = result.Count
    }

    return stats, nil
}
```

### 批量操作

```go
// BulkCreate 批量创建
func (r *ExampleRepo) BulkCreate(ctx context.Context, items []*model.Example) error {
    if len(items) == 0 {
        return nil
    }

    docs := make([]interface{}, len(items))
    now := time.Now()
    for i, item := range items {
        item.CreatedAt = now
        item.UpdatedAt = now
        docs[i] = item
    }

    _, err := r.collection.InsertMany(ctx, docs)
    return err
}

// BulkUpdate 批量更新
func (r *ExampleRepo) BulkUpdate(ctx context.Context, ids []string, update bson.M) error {
    if len(ids) == 0 {
        return nil
    }

    objectIDs := make([]primitive.ObjectID, 0, len(ids))
    for _, id := range ids {
        if oid, err := primitive.ObjectIDFromHex(id); err == nil {
            objectIDs = append(objectIDs, oid)
        }
    }

    _, err := r.collection.UpdateMany(ctx,
        bson.M{"_id": bson.M{"$in": objectIDs}},
        bson.M{"$set": update},
    )
    return err
}
```

### 模糊搜索

```go
// Search 搜索
func (r *ExampleRepo) Search(ctx context.Context, keyword string, page, pageSize int) ([]*model.Example, int64, error) {
    query := bson.M{
        "$or": []bson.M{
            {"name": bson.M{"$regex": keyword, "$options": "i"}},
            {"description": bson.M{"$regex": keyword, "$options": "i"}},
        },
    }

    total, err := r.collection.CountDocuments(ctx, query)
    if err != nil {
        return nil, 0, err
    }

    skip := int64((page - 1) * pageSize)
    cursor, err := r.collection.Find(ctx, query,
        options.Find().
            SetSkip(skip).
            SetLimit(int64(pageSize)).
            SetSort(bson.D{{Key: "created_at", Value: -1}}),
    )
    if err != nil {
        return nil, 0, err
    }
    defer cursor.Close(ctx)

    var items []*model.Example
    if err := cursor.All(ctx, &items); err != nil {
        return nil, 0, err
    }

    return items, total, nil
}
```

### 存在性检查

```go
// Exists 检查记录是否存在
func (r *ExampleRepo) Exists(ctx context.Context, name string) (bool, error) {
    count, err := r.collection.CountDocuments(ctx, bson.M{"name": name})
    if err != nil {
        return false, err
    }
    return count > 0, nil
}

// ExistsByField 根据字段检查是否存在
func (r *ExampleRepo) ExistsByField(ctx context.Context, field, value string) (bool, error) {
    count, err := r.collection.CountDocuments(ctx, bson.M{field: value})
    if err != nil {
        return false, err
    }
    return count > 0, nil
}
```

---

## 索引管理

### 自动创建索引

```go
// NewExampleRepo 构造函数中创建索引
func NewExampleRepo(db *mongo.Database) *ExampleRepo {
    repo := &ExampleRepo{
        collection: db.Collection("examples"),
    }

    // 创建索引
    ctx := context.Background()
    repo.ensureIndexes(ctx)

    return repo
}

func (r *ExampleRepo) ensureIndexes(ctx context.Context) error {
    indexes := []mongo.IndexModel{
        {
            // 名称唯一索引
            Keys:    bson.D{{Key: "name", Value: 1}},
            Options: options.Index().SetUnique(true),
        },
        {
            // 状态索引（用于筛选）
            Keys: bson.D{{Key: "status", Value: 1}},
        },
        {
            // 创建时间倒序索引（用于列表查询）
            Keys: bson.D{{Key: "created_at", Value: -1}},
        },
        {
            // 复合索引
            Keys: bson.D{
                {Key: "type", Value: 1},
                {Key: "status", Value: 1},
            },
        },
    }

    _, err := r.collection.Indexes().CreateMany(ctx, indexes)
    return err
}
```

### 常用索引模式

```go
// 唯一索引
options.Index().SetUnique(true)

// 稀疏索引（只索引存在该字段的文档）
options.Index().SetSparse(true)

// TTL 索引（自动过期）
options.Index().SetExpireAfterSeconds(3600)

// 部分索引
options.Index().SetPartialFilterExpression(bson.M{
    "status": "active",
})
```

---

## 错误处理

### 常见错误处理

```go
// 检查 ErrDuplicateKeyError
import "go.mongodb.org/mongo-driver/v2/mongo"

func (r *ExampleRepo) Create(ctx context.Context, item *model.Example) error {
    _, err := r.collection.InsertOne(ctx, item)
    if err != nil {
        if mongo.IsDuplicateKeyError(err) {
            return fmt.Errorf("example with name %s already exists", item.Name)
        }
        return err
    }
    return nil
}

// 检查 ErrNoDocuments
func (r *ExampleRepo) GetByID(ctx context.Context, id string) (*model.Example, error) {
    var item model.Example
    err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&item)
    if err != nil {
        if err == mongo.ErrNoDocuments {
            return nil, nil // 不返回错误，只返回 nil
        }
        return nil, err
    }
    return &item, nil
}
```

---

## 事务处理

```go
// Transaction 执行事务
func (r *ExampleRepo) Transaction(ctx context.Context, fn func(sessCtx mongo.SessionContext) error) error {
    session, err := r.collection.Database().Client().StartSession()
    if err != nil {
        return err
    }
    defer session.EndSession(ctx)

    _, err = session.WithTransaction(ctx, func(sessCtx mongo.SessionContext) (interface{}, error) {
        return nil, fn(sessCtx)
    })
    return err
}

// 使用示例
func (r *ExampleRepo) TransferExample(ctx context.Context, fromID, toID, exampleID string) error {
    return r.Transaction(ctx, func(sessCtx mongo.SessionContext) error {
        // 1. 从源删除
        _, err := r.collection.DeleteOne(sessCtx, bson.M{"_id": fromID, "example_id": exampleID})
        if err != nil {
            return err
        }

        // 2. 添加到目标
        _, err = r.collection.InsertOne(sessCtx, bson.M{"_id": toID, "example_id": exampleID})
        if err != nil {
            return err
        }

        return nil
    })
}
```

---

## 测试 Repository

```go
// internal/repository/example_test.go
package repository

import (
    "context"
    "testing"
    "time"

    "github.com/testcontainers/testcontainers-go"
    "github.com/testcontainers/testcontainers-go/modules/mongodb"
)

func setupTestDB(t *testing.T) (*mongo.Database, func()) {
    ctx := context.Background()

    // 启动 MongoDB 容器
    container, err := mongodb.RunContainer(ctx,
        testcontainers.WithImage("mongo:7"),
    )
    if err != nil {
        t.Skipf("Failed to start MongoDB container: %v", err)
    }

    // 获取连接字符串
    connStr, err := container.ConnectionString(ctx)
    if err != nil {
        t.Fatal(err)
    }

    // 连接 MongoDB
    client, err := mongo.Connect(ctx, options.Client().ApplyURI(connStr))
    if err != nil {
        t.Fatal(err)
    }

    db := client.Database("test_secflow")

    // 清理函数
    cleanup := func() {
        db.Drop(ctx)
        container.Terminate(ctx)
    }

    return db, cleanup
}

func TestExampleRepo_Create(t *testing.T) {
    db, cleanup := setupTestDB(t)
    defer cleanup()

    repo := NewExampleRepo(db)

    item := &model.Example{
        Name:   "test",
        Type:   "test-type",
        Status: "active",
    }

    err := repo.Create(context.Background(), item)
    if err != nil {
        t.Fatalf("Failed to create: %v", err)
    }

    if item.ID.IsZero() {
        t.Error("Expected ID to be set")
    }
}

func TestExampleRepo_List(t *testing.T) {
    db, cleanup := setupTestDB(t)
    defer cleanup()

    repo := NewExampleRepo(db)

    // 创建测试数据
    for i := 0; i < 25; i++ {
        repo.Create(context.Background(), &model.Example{
            Name:   "test",
            Type:   "test-type",
            Status: "active",
        })
    }

    // 测试分页
    items, total, err := repo.List(context.Background(), ExampleListFilter{
        Page:     1,
        PageSize: 10,
    })
    if err != nil {
        t.Fatalf("Failed to list: %v", err)
    }

    if len(items) != 10 {
        t.Errorf("Expected 10 items, got %d", len(items))
    }

    if total != 25 {
        t.Errorf("Expected total 25, got %d", total)
    }
}
```

---

## 最佳实践

1. **单一职责** - 每个 Repository 只负责一个集合
2. **筛选器模式** - 使用结构体封装查询条件
3. **错误处理** - 区分"未找到"和"错误"
4. **索引管理** - 在构造函数中创建索引
5. **分页查询** - 始终支持分页
6. **事务支持** - 提供事务封装方法
7. **可测试性** - 使用依赖注入，便于测试
8. **性能优化** - 批量操作、使用索引
