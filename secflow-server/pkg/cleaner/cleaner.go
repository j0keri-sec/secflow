// Package cleaner provides data cleanup and archiving functionality.
package cleaner

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Config holds configuration for the cleaner.
type Config struct {
	RetentionDays   int    // 数据保留天数
	ArchiveEnabled  bool   // 是否启用归档
	ArchivePath     string // 归档文件路径
	CleanCron       string // 清理定时表达式 (cron格式)
}

// Cleaner handles data cleanup and archiving.
type Cleaner struct {
	config Config
	mongo  *mongo.Database
}

// New creates a new Cleaner.
func New(config Config, mongoDB *mongo.Database) *Cleaner {
	return &Cleaner{
		config: config,
		mongo:  mongoDB,
	}
}

// Start starts the cleaner background job.
func (c *Cleaner) Start(ctx context.Context) error {
	// 立即执行一次清理
	if err := c.Cleanup(ctx); err != nil {
		return fmt.Errorf("initial cleanup failed: %w", err)
	}

	// 设置定时清理
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := c.Cleanup(ctx); err != nil {
				fmt.Printf("cleanup failed: %v\n", err)
			}
		}
	}
}

// Cleanup performs cleanup of old data.
func (c *Cleaner) Cleanup(ctx context.Context) error {
	cutoffDate := time.Now().AddDate(0, 0, -c.config.RetentionDays)

	fmt.Printf("starting cleanup for data older than %s\n", cutoffDate.Format("2006-01-02"))

	// 清理漏洞数据
	if err := c.cleanupVulns(ctx, cutoffDate); err != nil {
		return fmt.Errorf("cleanup vulns: %w", err)
	}

	// 清理文章数据
	if err := c.cleanupArticles(ctx, cutoffDate); err != nil {
		return fmt.Errorf("cleanup articles: %w", err)
	}

	// 清理任务历史
	if err := c.cleanupTasks(ctx, cutoffDate); err != nil {
		return fmt.Errorf("cleanup tasks: %w", err)
	}

	// 清理审计日志
	if err := c.cleanupAuditLogs(ctx, cutoffDate); err != nil {
		return fmt.Errorf("cleanup audit logs: %w", err)
	}

	fmt.Println("cleanup completed successfully")
	return nil
}

func (c *Cleaner) cleanupVulns(ctx context.Context, cutoffDate time.Time) error {
	collection := c.mongo.Collection("vulns")

	filter := bson.M{
		"created_at": bson.M{"$lt": cutoffDate},
	}

	if c.config.ArchiveEnabled {
		// 先归档再删除
		if err := c.archiveCollection(ctx, collection, filter, "vulns"); err != nil {
			return fmt.Errorf("archive vulns: %w", err)
		}
	}

	result, err := collection.DeleteMany(ctx, filter)
	if err != nil {
		return fmt.Errorf("delete vulns: %w", err)
	}

	fmt.Printf("deleted %d old vulnerabilities\n", result.DeletedCount)
	return nil
}

func (c *Cleaner) cleanupArticles(ctx context.Context, cutoffDate time.Time) error {
	collection := c.mongo.Collection("articles")

	filter := bson.M{
		"created_at": bson.M{"$lt": cutoffDate},
	}

	if c.config.ArchiveEnabled {
		if err := c.archiveCollection(ctx, collection, filter, "articles"); err != nil {
			return fmt.Errorf("archive articles: %w", err)
		}
	}

	result, err := collection.DeleteMany(ctx, filter)
	if err != nil {
		return fmt.Errorf("delete articles: %w", err)
	}

	fmt.Printf("deleted %d old articles\n", result.DeletedCount)
	return nil
}

func (c *Cleaner) cleanupTasks(ctx context.Context, cutoffDate time.Time) error {
	collection := c.mongo.Collection("tasks")

	filter := bson.M{
		"created_at": bson.M{"$lt": cutoffDate},
		"status":     bson.M{"$in": []string{"completed", "failed"}},
	}

	result, err := collection.DeleteMany(ctx, filter)
	if err != nil {
		return fmt.Errorf("delete tasks: %w", err)
	}

	fmt.Printf("deleted %d old tasks\n", result.DeletedCount)
	return nil
}

func (c *Cleaner) cleanupAuditLogs(ctx context.Context, cutoffDate time.Time) error {
	collection := c.mongo.Collection("audit_logs")

	filter := bson.M{
		"created_at": bson.M{"$lt": cutoffDate},
	}

	result, err := collection.DeleteMany(ctx, filter)
	if err != nil {
		return fmt.Errorf("delete audit logs: %w", err)
	}

	fmt.Printf("deleted %d old audit logs\n", result.DeletedCount)
	return nil
}

func (c *Cleaner) archiveCollection(ctx context.Context, collection *mongo.Collection, filter bson.M, name string) error {
	// 创建归档目录
	archiveDir := filepath.Join(c.config.ArchivePath, time.Now().Format("2006-01"))
	if err := os.MkdirAll(archiveDir, 0755); err != nil {
		return fmt.Errorf("create archive directory: %w", err)
	}

	// 创建归档文件
	archiveFile := filepath.Join(archiveDir, fmt.Sprintf("%s_%s.tar.gz", name, time.Now().Format("20060102_150405")))
	file, err := os.Create(archiveFile)
	if err != nil {
		return fmt.Errorf("create archive file: %w", err)
	}
	defer file.Close()

	// 创建 gzip writer
	gzWriter := gzip.NewWriter(file)
	defer gzWriter.Close()

	// 创建 tar writer
	tarWriter := tar.NewWriter(gzWriter)
	defer tarWriter.Close()

	// 查询数据
	opts := options.Find().SetBatchSize(1000)
	cursor, err := collection.Find(ctx, filter, opts)
	if err != nil {
		return fmt.Errorf("find documents: %w", err)
	}
	defer cursor.Close(ctx)

	// 写入数据到 tar
	docCount := 0
	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			return fmt.Errorf("decode document: %w", err)
		}

		data, err := json.Marshal(doc)
		if err != nil {
			return fmt.Errorf("marshal document: %w", err)
		}

		header := &tar.Header{
			Name: fmt.Sprintf("%s_%d.json", name, docCount),
			Size: int64(len(data)),
			Mode: 0644,
		}

		if err := tarWriter.WriteHeader(header); err != nil {
			return fmt.Errorf("write tar header: %w", err)
		}

		if _, err := tarWriter.Write(data); err != nil {
			return fmt.Errorf("write tar data: %w", err)
		}

		docCount++
	}

	if err := cursor.Err(); err != nil {
		return fmt.Errorf("cursor error: %w", err)
	}

	fmt.Printf("archived %d %s documents to %s\n", docCount, name, archiveFile)
	return nil
}

// Restore restores archived data back to the database.
func (c *Cleaner) Restore(ctx context.Context, archiveFile string, collectionName string) error {
	file, err := os.Open(archiveFile)
	if err != nil {
		return fmt.Errorf("open archive file: %w", err)
	}
	defer file.Close()

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("create gzip reader: %w", err)
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)
	collection := c.mongo.Collection(collectionName)

	docCount := 0
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("read tar header: %w", err)
		}

		data := make([]byte, header.Size)
		if _, err := io.ReadFull(tarReader, data); err != nil {
			return fmt.Errorf("read tar data: %w", err)
		}

		var doc bson.M
		if err := json.Unmarshal(data, &doc); err != nil {
			return fmt.Errorf("unmarshal document: %w", err)
		}

		// 删除 _id 避免冲突
		delete(doc, "_id")

		if _, err := collection.InsertOne(ctx, doc); err != nil {
			return fmt.Errorf("insert document: %w", err)
		}

		docCount++
	}

	fmt.Printf("restored %d documents to %s\n", docCount, collectionName)
	return nil
}

// ListArchives lists all available archive files.
func (c *Cleaner) ListArchives() ([]string, error) {
	var archives []string

	err := filepath.Walk(c.config.ArchivePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && filepath.Ext(path) == ".gz" {
			archives = append(archives, path)
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("walk archive path: %w", err)
	}

	return archives, nil
}
