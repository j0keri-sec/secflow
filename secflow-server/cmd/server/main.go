// Command server is the secflow platform server entry point.
package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"

	"github.com/secflow/server/config"
	"github.com/secflow/server/internal/api"
	"github.com/secflow/server/internal/api/handler"
	"github.com/secflow/server/internal/queue"
	"github.com/secflow/server/internal/report"
	"github.com/secflow/server/internal/repository"
	"github.com/secflow/server/internal/scheduler"
	"github.com/secflow/server/internal/ws"
	"github.com/secflow/server/pkg/auth"
	"github.com/secflow/server/pkg/logger"
	"github.com/secflow/server/pkg/notify"
)

func main() {
	cfgPath := flag.String("config", "config/config.yaml", "path to config file")
	flag.Parse()

	// ── Load configuration ─────────────────────────────────────────────────
	cfg, err := config.Load(*cfgPath)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load config")
	}
	if err = cfg.Validate(); err != nil {
		log.Fatal().Err(err).Msg("invalid config")
	}

	// ── Init logger ────────────────────────────────────────────────────────
	logger.Init(cfg.Log.Level, cfg.Log.Format)

	// ── Connect MongoDB ────────────────────────────────────────────────────
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	db, err := repository.New(ctx, cfg.MongoDB.URI, cfg.MongoDB.Database)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect mongodb")
	}
	defer db.Close(context.Background())
	log.Info().Str("db", cfg.MongoDB.Database).Msg("mongodb connected")

	// ── Connect Redis ──────────────────────────────────────────────────────
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	if err = rdb.Ping(context.Background()).Err(); err != nil {
		log.Fatal().Err(err).Msg("failed to connect redis")
	}
	log.Info().Str("addr", cfg.Redis.Addr).Msg("redis connected")

	// ── Wire dependencies ──────────────────────────────────────────────────
	q         := queue.New(rdb)
	authSvc   := auth.New(cfg.JWT.Secret, cfg.JWT.Expire)

	vulnRepo       := repository.NewVulnRepo(db)
	userRepo       := repository.NewUserRepo(db)
	invRepo        := repository.NewInviteCodeRepo(db)
	nodeRepo       := repository.NewNodeRepo(db)
	taskRepo       := repository.NewTaskRepo(db)
	articleRepo    := repository.NewArticleRepository(db)
	pushRepo       := repository.NewPushChannelRepository(db)
	auditRepo      := repository.NewAuditLogRepository(db)
	reportRepo     := repository.NewReportRepository(db)
	resetTokenRepo := repository.NewPasswordResetTokenRepo(db)
	// Initialize report generator with Minimax AI (if configured)
	reportGen := report.NewGenerator(vulnRepo, articleRepo)
	// To enable AI: report.NewGeneratorWithAI(vulnRepo, articleRepo, "your-api-key", "your-group-id", report.AIModelGPT4)

	// Email sender for password reset and notifications
	var emailSender *notify.EmailSender
	if cfg.Email.Enabled && cfg.Email.Host != "" {
		emailSender = notify.NewEmailSender(notify.EmailConfig{
			Host:     cfg.Email.Host,
			Port:     cfg.Email.Port,
			Username: cfg.Email.Username,
			Password: cfg.Email.Password,
			From:     cfg.Email.From,
			FromName: cfg.Email.FromName,
			UseTLS:   cfg.Email.UseTLS,
		})
		log.Info().
			Str("host", cfg.Email.Host).
			Int("port", cfg.Email.Port).
			Str("from", cfg.Email.From).
			Msg("email sender configured")
	} else {
		log.Warn().Msg("email sender not configured, password reset tokens will be logged only")
	}

	// Base URL for password reset links
	resetBaseURL := os.Getenv("RESET_BASE_URL")
	if resetBaseURL == "" {
		resetBaseURL = fmt.Sprintf("http://localhost:%d", cfg.Server.Port)
	}

	// ── Task Scheduler (dispatches tasks to nodes) ──────────────────────────
	taskScheduler := scheduler.NewWithConfig(q, taskRepo, nodeRepo, nil, cfg.Scheduler)
	taskScheduler.Start(ctx)

	// ── Task Schedule Repository ────────────────────────────────────────────
	scheduleRepo := repository.NewTaskScheduleRepo(db)
	systemH := handler.NewSystemHandler(scheduleRepo)

	// ── WebSocket Hub ──────────────────────────────────────────────────────
	// Configure WebSocket origin validation
	ws.SetAllowedOrigins(cfg.Server.CORSOrigins)
	// Create hub with node handler callbacks (hub needs handler methods, so create handler first with temp nil hub)
	tempNodeH := handler.NewNodeHandlerWithScheduler(nodeRepo, taskRepo, vulnRepo, articleRepo, q, nil, cfg.Node.TokenKey, taskScheduler)
	hub := ws.NewHub(tempNodeH.OnMessage, tempNodeH.OnConnect, tempNodeH.OnDisconnect)
	// Inject hub into scheduler and create final node handler with hub
	taskScheduler.SetHub(hub)
	nodeH := handler.NewNodeHandlerWithScheduler(nodeRepo, taskRepo, vulnRepo, articleRepo, q, hub, cfg.Node.TokenKey, taskScheduler)

	// ── Task Generator (automatically creates crawl tasks) ──────────────────
	// Default sources: all rod-based grabbers
	defaultVulnSources := []string{
		"avd-rod", "seebug-rod", "ti-rod", "nox-rod",
		"kev-rod", "struts2-rod", "chaitin-rod",
		"oscs-rod", "threatbook-rod", "venustech-rod",
	}
	taskGenerator := scheduler.NewTaskGenerator(taskRepo, q, nodeRepo, defaultVulnSources, nil)
	taskGenerator.Start(ctx)

	// ── Build router ───────────────────────────────────────────────────────
	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}
	taskH := handler.NewTaskHandler(taskRepo, q)
	passwordResetH := handler.NewPasswordResetHandler(userRepo, resetTokenRepo, emailSender, resetBaseURL)
	r := api.Router(
		cfg,
		authSvc,
		handler.NewAuthHandler(userRepo, invRepo, auditRepo, authSvc),
		handler.NewVulnHandler(vulnRepo),
		nodeH,
		taskH,
		handler.NewUserHandler(userRepo),
		handler.NewArticleHandler(articleRepo),
		handler.NewPushChannelHandler(pushRepo),
		handler.NewAuditLogHandler(auditRepo),
		handler.NewReportHandler(reportGen, vulnRepo, reportRepo),
		systemH,
		passwordResetH,
	)
	// Inject scheduler into task handler for stop functionality
	taskH.SetScheduler(taskScheduler)

	// ── HTTP Server ────────────────────────────────────────────────────────
	srv := &http.Server{
		Addr:         cfg.Server.Addr(),
		Handler:      r,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	go func() {
		log.Info().Str("addr", srv.Addr).Msg("server starting")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("server error")
		}
	}()

	// ── Graceful shutdown ──────────────────────────────────────────────────
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info().Msg("shutting down...")

	// Stop schedulers
	taskScheduler.Stop()
	taskGenerator.Stop()

	shutCtx, shutCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutCancel()
	if err = srv.Shutdown(shutCtx); err != nil {
		log.Error().Err(err).Msg("server forced shutdown")
	}
	log.Info().Msg("server stopped")
}
