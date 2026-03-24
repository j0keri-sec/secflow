// Cleaner service entry point.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/secflow/server/pkg/cleaner"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("shutting down...")
		cancel()
	}()

	// Get configuration from environment
	mongoURI := getEnv("SECFLOW_MONGO_URI", "mongodb://localhost:27017/secflow")
	retentionDays := getEnvInt("SECFLOW_RETENTION_DAYS", 90)
	archiveEnabled := getEnvBool("SECFLOW_ARCHIVE_ENABLED", true)
	archivePath := getEnv("SECFLOW_ARCHIVE_PATH", "/archive")

	// Connect to MongoDB
	fmt.Printf("connecting to MongoDB: %s\n", mongoURI)
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to connect to MongoDB: %v\n", err)
		os.Exit(1)
	}
	defer client.Disconnect(ctx)

	// Ping database
	if err := client.Ping(ctx, nil); err != nil {
		fmt.Fprintf(os.Stderr, "failed to ping MongoDB: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("connected to MongoDB")

	db := client.Database("secflow")

	// Create cleaner
	config := cleaner.Config{
		RetentionDays:  retentionDays,
		ArchiveEnabled: archiveEnabled,
		ArchivePath:    archivePath,
	}

	c := cleaner.New(config, db)

	// Start cleaner
	fmt.Printf("starting cleaner with retention=%d days, archive=%v\n", retentionDays, archiveEnabled)
	if err := c.Start(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "cleaner error: %v\n", err)
		os.Exit(1)
	}
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return defaultVal
}

func getEnvBool(key string, defaultVal bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return defaultVal
}
