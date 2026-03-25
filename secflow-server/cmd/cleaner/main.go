// Cleaner service entry point.
package main

import (
	"context"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/secflow/server/pkg/cleaner"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	// Setup zerolog
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Info().Msg("shutting down...")
		cancel()
	}()

	// Get configuration from environment
	mongoURI := getEnv("SECFLOW_MONGO_URI", "mongodb://localhost:27017/secflow")
	retentionDays := getEnvInt("SECFLOW_RETENTION_DAYS", 90)
	archiveEnabled := getEnvBool("SECFLOW_ARCHIVE_ENABLED", true)
	archivePath := getEnv("SECFLOW_ARCHIVE_PATH", "/archive")

	// Connect to MongoDB
	log.Info().Str("mongo_uri", mongoURI).Msg("connecting to MongoDB")
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to MongoDB")
	}
	defer client.Disconnect(ctx)

	// Ping database
	if err := client.Ping(ctx, nil); err != nil {
		log.Fatal().Err(err).Msg("failed to ping MongoDB")
	}
	log.Info().Msg("connected to MongoDB")

	db := client.Database("secflow")

	// Create cleaner
	config := cleaner.Config{
		RetentionDays:  retentionDays,
		ArchiveEnabled: archiveEnabled,
		ArchivePath:    archivePath,
	}

	c := cleaner.New(config, db)

	// Start cleaner
	log.Info().Int("retention_days", retentionDays).Bool("archive", archiveEnabled).Msg("starting cleaner")
	if err := c.Start(ctx); err != nil {
		log.Fatal().Err(err).Msg("cleaner error")
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
