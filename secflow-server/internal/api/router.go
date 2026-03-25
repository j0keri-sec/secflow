// Package api wires all routes together.
package api

import (
	"compress/gzip"
	"io"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/secflow/server/config"
	"github.com/secflow/server/internal/api/handler"
	"github.com/secflow/server/internal/api/middleware"
	"github.com/secflow/server/pkg/auth"
)

// gzipMiddleware compresses HTTP responses using gzip for eligible clients.
func gzipMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !strings.Contains(c.GetHeader("Accept-Encoding"), "gzip") {
			c.Next()
			return
		}
		c.Header("Content-Encoding", "gzip")
		c.Header("Vary", "Accept-Encoding")

		gz, err := gzip.NewWriterLevel(c.Writer, gzip.BestSpeed)
		if err != nil {
			c.Next()
			return
		}
		defer gz.Close()

		c.Writer = &gzipWriter{Writer: gz, ResponseWriter: c.Writer}
		c.Next()
	}
}

type gzipWriter struct {
	gin.ResponseWriter
	io.Writer
}

func (g *gzipWriter) Write(b []byte) (int, error) { return g.Writer.Write(b) }

func (g *gzipWriter) ReadFrom(r io.Reader) (int64, error) {
	return io.Copy(g.Writer, r)
}

// Router returns a configured Gin engine with all routes registered.
func Router(
	cfg        *config.Config,
	authSvc    *auth.Service,
	authH      *handler.AuthHandler,
	vulnH      *handler.VulnHandler,
	nodeH      *handler.NodeHandler,
	taskH      *handler.TaskHandler,
	userH      *handler.UserHandler,
	articleH   *handler.ArticleHandler,
	pushH      *handler.PushChannelHandler,
	auditH     *handler.AuditLogHandler,
	reportH    *handler.ReportHandler,
	systemH    *handler.SystemHandler,
) *gin.Engine {
	r := gin.New()
	r.Use(middleware.Recovery())
	r.Use(middleware.CORS(cfg.Server.CORSOrigins))
	r.Use(middleware.RequestID())
	r.Use(gzipMiddleware())

	// Health probe — no auth required.
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Prometheus metrics endpoint — no auth required.
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// ── Static file serving for uploaded images (no auth required for nodes)
	r.Static("/uploads", "./uploads")

	v1 := r.Group("/api/v1")

	// ── Auth ────────────────────────────────────────────────────────────────
	authGroup := v1.Group("/auth")
	{
		authGroup.POST("/register", authH.Register)
		authGroup.POST("/login",    authH.Login)

		authed := authGroup.Group("", middleware.JWTAuth(authSvc))
		{
			authed.GET("/me",        authH.Me)
			authed.PUT("/password",  authH.ChangePassword)
			authed.POST("/invite",   authH.GenerateInviteCode)
			authed.GET("/invite",    authH.ListInviteCodes)
		}
	}

	// ── WebSocket (node connection) ─────────────────────────────────────────
	v1.GET("/ws/node", nodeH.ServeWS)

	// ── All routes below require JWT ────────────────────────────────────────
	api := v1.Group("", middleware.JWTAuth(authSvc))

	// ── Vulnerabilities ─────────────────────────────────────────────────────
	vulns := api.Group("/vulns")
	{
		vulns.GET("",         vulnH.List)
		vulns.GET("/stats",   vulnH.Stats)
		vulns.GET("/export",  vulnH.Export)
		vulns.GET("/:id",     vulnH.Get)
		vulns.DELETE("/:id",  middleware.RequireRole("editor"), vulnH.Delete)
	}

	// ── Articles ─────────────────────────────────────────────────────────────
	articles := api.Group("/articles")
	{
		articles.GET("",              articleH.List)
		articles.GET("/:id",          articleH.Get)
		articles.DELETE("/:id",       middleware.RequireRole("editor"), articleH.Delete)
		articles.POST("/upsert",      middleware.RequireRole("editor"), articleH.Upsert)
	}

	// ── Tasks ────────────────────────────────────────────────────────────────
	tasks := api.Group("/tasks")
	{
		tasks.GET("",              taskH.List)
		tasks.GET("/:id",          taskH.Get)
		tasks.DELETE("/:id",       middleware.RequireRole("editor"), taskH.Delete)
		tasks.POST("/:id/stop",    middleware.RequireRole("editor"), taskH.Stop)
		tasks.POST("/vuln-crawl",  middleware.RequireRole("editor"), taskH.CreateVulnCrawl)
		tasks.POST("/article-crawl", middleware.RequireRole("editor"), taskH.CreateArticleCrawl)
	}

	// ── Dead Letter Queue ──────────────────────────────────────────────────
	deadLetters := api.Group("/dead-letters")
	{
		deadLetters.GET("",              taskH.ListDeadLetters)
		deadLetters.GET("/stats",       taskH.GetDeadLetterStats)
		deadLetters.GET("/:id",         taskH.GetDeadLetter)
		deadLetters.POST("/:id/retry",  middleware.RequireRole("editor"), taskH.RetryDeadLetter)
		deadLetters.DELETE("/:id",       middleware.RequireRole("editor"), taskH.DeleteDeadLetter)
	}

	// ── Task Schedules ──────────────────────────────────────────────────
	taskSchedules := api.Group("/task-schedules")
	{
		taskSchedules.GET("",     systemH.GetTaskSchedules)
		taskSchedules.PUT("",     middleware.RequireRole("editor"), systemH.UpdateTaskSchedule)
	}

	// ── Nodes ────────────────────────────────────────────────────────────────
	nodes := api.Group("/nodes")
	{
		nodes.GET("", nodeH.List)
		nodes.POST("", middleware.RequireRole("admin"), nodeH.Create)
		nodes.GET("/:id/logs", nodeH.GetLogs)
		nodes.DELETE("/:id", middleware.RequireRole("admin"), nodeH.Delete)
		nodes.POST("/:id/regenerate-token", middleware.RequireRole("admin"), nodeH.RegenerateToken)
		nodes.POST("/:id/pause", middleware.RequireRole("admin"), nodeH.Pause)
		nodes.POST("/:id/resume", middleware.RequireRole("admin"), nodeH.Resume)
		nodes.POST("/:id/disconnect", middleware.RequireRole("admin"), nodeH.Disconnect)
	}

	// ── Push Channels ─────────────────────────────────────────────────────────
	push := api.Group("/push-channels")
	{
		push.GET("",      pushH.List)
		push.POST("",     middleware.RequireRole("editor"), pushH.Create)
		push.PATCH("/:id", middleware.RequireRole("editor"), pushH.Update)
		push.DELETE("/:id", middleware.RequireRole("editor"), pushH.Delete)
	}

	// ── Audit Logs ────────────────────────────────────────────────────────────
	api.GET("/audit-logs", auditH.List)

	// ── Reports ──────────────────────────────────────────────────────────────
	reports := api.Group("/reports")
	{
		reports.GET("",      reportH.List)
		reports.POST("",     reportH.Create)
		reports.DELETE("/:id", middleware.RequireRole("editor"), reportH.Delete)
	}

	// ── Users (admin only) ───────────────────────────────────────────────────
	users := api.Group("/users", middleware.RequireRole("admin"))
	{
		users.GET("",        userH.List)
		users.PATCH("/:id",  userH.UpdateRole)
		users.DELETE("/:id", userH.Delete)
	}

	return r
}
