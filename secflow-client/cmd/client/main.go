// Package main is the entry point for the SecFlow client node.
//
// The client supports two operating modes:
//
//  1. Server Mode (default):
//     - Connects to the SecFlow server via WebSocket
//     - Receives task_assign frames from server
//     - Reports hardware metrics via heartbeats
//     - Uploads results back to the server
//
//  2. Standalone Mode:
//     - Runs independently without server connection
//     - Executes scheduled crawling tasks locally
//     - Stores all data in local SQLite database
//
// Usage:
//   - Set mode in config.yaml: mode: "standalone" or mode: "server"
//   - Or use environment variable: SECFLOW_MODE=standalone
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
	gnet "github.com/shirou/gopsutil/v3/net"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"

	"github.com/secflow/client/internal/config"
	"github.com/secflow/client/internal/db"
	"github.com/secflow/client/internal/engine"
	"github.com/secflow/client/internal/logger"
	"github.com/secflow/client/internal/task"
	iws "github.com/secflow/client/internal/ws"
	"github.com/secflow/client/pkg/articlegrabber"
	grabpkg "github.com/secflow/client/pkg/vulngrabber"
)

const version = "1.0.0"

func main() {
	app := &cli.App{
		Name:      "secflow-client",
		Usage:     "SecFlow client node — connects to the server and executes crawl tasks",
		Version:   version,
		UsageText: "secflow-client [global options] [arguments...]",
		Description: `SecFlow Client is a vulnerability intelligence collection node that connects
to SecFlow Server and executes crawling tasks.

EXAMPLES:
   # Run with default config file (client.yaml)
   secflow-client

   # Run with custom config file
   secflow-client -c /etc/secflow/client.yaml

   # Run with debug logging
   secflow-client --debug

   # Run with environment variables
   SECFLOW_TOKEN=xxx SECFLOW_API_URL=http://server:8080/api/v1 secflow-client

For more information, visit: https://github.com/secflow/secflow`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				Aliases: []string{"c"},
				Value:   "client.yaml",
				Usage:   "Path to client configuration file",
				EnvVars: []string{"SECFLOW_CONFIG"},
			},
			&cli.StringFlag{
				Name:    "mode",
				Usage:   "Operating mode: server or standalone",
				EnvVars: []string{"SECFLOW_MODE"},
			},
			&cli.StringFlag{
				Name:    "api-url",
				Usage:   "Server HTTP API URL (overrides config, server mode only)",
				EnvVars: []string{"SECFLOW_API_URL"},
			},
			&cli.StringFlag{
				Name:    "ws-url",
				Usage:   "Server WebSocket URL (overrides config, server mode only)",
				EnvVars: []string{"SECFLOW_WS_URL"},
			},
			&cli.StringFlag{
				Name:    "token",
				Usage:   "Authentication token (overrides config, server mode only)",
				EnvVars: []string{"SECFLOW_TOKEN"},
			},
			&cli.StringFlag{
				Name:    "node-id",
				Usage:   "Unique node identifier (overrides config)",
				EnvVars: []string{"SECFLOW_NODE_ID"},
			},
			&cli.StringFlag{
				Name:    "name",
				Usage:   "Node display name (overrides config)",
				EnvVars: []string{"SECFLOW_NODE_NAME"},
			},
			&cli.StringFlag{
				Name:    "proxy",
				Usage:   "HTTP/HTTPS proxy URL (overrides config)",
				EnvVars: []string{"SECFLOW_PROXY"},
			},
			&cli.StringFlag{
				Name:    "log-level",
				Usage:   "Log level: debug, info, warn, error",
				Value:   "info",
				EnvVars: []string{"SECFLOW_LOG_LEVEL"},
			},
			&cli.StringFlag{
				Name:    "log-path",
				Usage:   "Log file path (overrides config)",
				EnvVars: []string{"SECFLOW_LOG_PATH"},
			},
			&cli.StringFlag{
				Name:    "db-path",
				Usage:   "SQLite database path (overrides config)",
				EnvVars: []string{"SECFLOW_DB_PATH"},
			},
			&cli.BoolFlag{
				Name:    "debug",
				Aliases: []string{"d"},
				Usage:   "Enable debug logging (shorthand for --log-level=debug)",
				EnvVars: []string{"SECFLOW_DEBUG"},
			},
		},
		Action: run,
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(c *cli.Context) error {
	// ── Load configuration ────────────────────────────────────────────────
	cfgPath := c.String("config")
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// ── Apply command-line overrides ──────────────────────────────────────
	if v := c.String("mode"); v != "" {
		cfg.Mode = config.Mode(v)
	}
	if v := c.String("api-url"); v != "" {
		cfg.Server.APIURL = v
	}
	if v := c.String("ws-url"); v != "" {
		cfg.Server.WSURL = v
	}
	if v := c.String("token-key"); v != "" {
		cfg.Server.TokenKey = v
	}
	if v := c.String("node-id"); v != "" {
		cfg.NodeID = v
	}
	if v := c.String("name"); v != "" {
		cfg.Name = v
	}
	if v := c.String("proxy"); v != "" {
		cfg.Proxy = v
	}
	if v := c.String("log-path"); v != "" {
		cfg.LogPath = v
	}
	if v := c.String("db-path"); v != "" {
		cfg.DBPath = v
	}

	// Handle log level (debug flag takes precedence)
	if c.Bool("debug") {
		cfg.LogLevel = "debug"
	} else if v := c.String("log-level"); v != "" {
		cfg.LogLevel = v
	}

	// ── Generate node_id if empty ─────────────────────────────────────────
	if cfg.NodeID == "" {
		cfg.NodeID = uuid.New().String()
		// Save the generated node_id back to config file
		if err := saveNodeID(cfgPath, cfg.NodeID); err != nil {
			return fmt.Errorf("save node_id: %w", err)
		}
	}

	// ── Open local SQLite database ────────────────────────────────────────
	store, err := db.Open(cfg.DBPath)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer store.Close()

	// ── Init logger ───────────────────────────────────────────────────────
	debug := cfg.LogLevel == "debug"
	log, err := logger.New(cfg.LogPath, debug, store)
	if err != nil {
		return fmt.Errorf("init logger: %w", err)
	}
	defer log.Sync() //nolint:errcheck

	log.Info("starting secflow-client",
		zap.String("version", version),
		zap.String("mode", string(cfg.Mode)),
		zap.String("node_id", cfg.NodeID),
		zap.String("node_name", cfg.Name),
		zap.String("log_level", cfg.LogLevel),
	)

	// ── Create grabber engine ─────────────────────────────────────────────
	eng := engine.New(cfg.Proxy, log)

	// ── Initialize article grabbers ───────────────────────────────────────
	articlegrabber.Init()

	// ── Context with graceful shutdown ────────────────────────────────────
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// ── Run in appropriate mode ───────────────────────────────────────────
	if cfg.IsStandalone() {
		return runStandalone(ctx, cfg, eng, store, log)
	}

	return runServerMode(ctx, cfg, eng, store, log)
}

// runServerMode runs the client in server-connected mode.
func runServerMode(ctx context.Context, cfg *config.Config, eng *engine.Engine, store *db.DB, log *zap.Logger) error {
	log.Info("running in server mode", zap.String("server", cfg.Server.APIURL))

	// ── Build the WS client (handler attached below via dispatcher) ───────
	var dispatcher *task.Dispatcher

	// Get available grabber sources for registration
	allSources := grabpkg.Available()
	sources := cfg.GetEnabledSources(allSources)

	wsClient := iws.New(
		cfg.Server.WSURL,
		cfg.Server.TokenKey,
		cfg.NodeID,
		cfg.Name,
		sources,
		&handlerProxy{getDispatcher: func() *task.Dispatcher { return dispatcher }},
		log,
	)

	dispatcher = task.New(eng, wsClient, store, cfg.NodeID, log)

	// ── Heartbeat goroutine ───────────────────────────────────────────────
	go runHeartbeat(ctx, wsClient, cfg, log)

	// ── WebSocket main loop (blocks until ctx is done) ───────────────────
	wsClient.Run(ctx)

	log.Info("client stopped gracefully")
	return nil
}

// runStandalone runs the client in standalone mode without server connection.
func runStandalone(ctx context.Context, cfg *config.Config, eng *engine.Engine, store *db.DB, log *zap.Logger) error {
	log.Info("running in standalone mode")

	// Create standalone task runner
	runner := NewStandaloneRunner(cfg, eng, store, log)

	// Start the scheduler
	if err := runner.Start(ctx); err != nil {
		return fmt.Errorf("start standalone runner: %w", err)
	}

	// Wait for context cancellation
	<-ctx.Done()

	log.Info("standalone client stopped gracefully")
	return nil
}

// handlerProxy delays the binding between ws.Client and task.Dispatcher,
// avoiding an import cycle.
type handlerProxy struct {
	getDispatcher func() *task.Dispatcher
}

func (h *handlerProxy) OnTaskAssign(msg *iws.TaskAssignMsg) {
	if d := h.getDispatcher(); d != nil {
		d.OnTaskAssign(msg)
	}
}

func (h *handlerProxy) OnTaskCancel(taskID string) {
	if d := h.getDispatcher(); d != nil {
		d.OnTaskCancel(taskID)
	}
}

// runHeartbeat periodically collects system metrics and sends them to the server.
func runHeartbeat(ctx context.Context, ws *iws.Client, cfg *config.Config, log *zap.Logger) {
	interval := cfg.HeartbeatInterval
	if interval <= 0 {
		interval = 30 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Send first heartbeat immediately
	info := collectNodeInfo()
	info.PublicIP = getPublicIP()
	if err := ws.SendHeartbeat(cfg.NodeID, info); err != nil {
		log.Warn("initial heartbeat failed", zap.Error(err))
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			info := collectNodeInfo()
			info.PublicIP = getPublicIP()
			if err := ws.SendHeartbeat(cfg.NodeID, info); err != nil {
				log.Warn("heartbeat failed", zap.Error(err))
			}
		}
	}
}

// collectNodeInfo gathers hardware / OS metrics using gopsutil.
func collectNodeInfo() *iws.NodeInfo {
	info := &iws.NodeInfo{
		OS:   runtime.GOOS,
		Arch: runtime.GOARCH,
	}

	// CPU - 获取使用率和核心数
	if pcts, err := cpu.Percent(0, false); err == nil && len(pcts) > 0 {
		info.CPUPct = pcts[0]
	}
	if counts, err := cpu.Counts(true); err == nil {
		info.CPUCores = counts
	}

	// CPU 型号信息
	if cpuInfo, err := cpu.Info(); err == nil && len(cpuInfo) > 0 {
		info.CPUModel = cpuInfo[0].ModelName
	}

	// Memory
	if vm, err := mem.VirtualMemory(); err == nil {
		info.MemTotal = vm.Total
		info.MemUsed = vm.Used
	}

	// Disk
	if usage, err := disk.Usage("/"); err == nil {
		info.DiskTotal = usage.Total
		info.DiskUsed = usage.Used
	}

	// Network interfaces - 获取所有网卡信息
	if ifaces, err := gnet.Interfaces(); err == nil {
		for _, iface := range ifaces {
			// 检查接口是否启用 (up)
			isUp := false
			isLoopback := false
			for _, flag := range iface.Flags {
				if flag == "up" {
					isUp = true
				}
				if flag == "loopback" {
					isLoopback = true
				}
			}
			if isUp {
				info.NetCards = append(info.NetCards, iface.Name)
			}
			// 获取第一个非 loopback 的 MAC 地址
			if iface.HardwareAddr != "" && info.MAC == "" && !isLoopback {
				// 排除常见的虚拟网卡
				if !strings.Contains(iface.Name, "lo") &&
					!strings.Contains(iface.Name, "docker") &&
					!strings.Contains(iface.Name, "veth") &&
					!strings.Contains(iface.Name, "br-") {
					info.MAC = iface.HardwareAddr
				}
			}
		}
	}

	// 获取主机信息
	if hostInfo, err := host.Info(); err == nil {
		if info.OS == "" {
			info.OS = hostInfo.OS
		}
	}

	return info
}

// publicIPCache caches the public IP to avoid repeated API calls
var (
	publicIPCache     string
	publicIPCacheTime time.Time
	publicIPMutex     sync.RWMutex
)

// getPublicIP retrieves the public IP address from external API
func getPublicIP() string {
	publicIPMutex.RLock()
	if time.Since(publicIPCacheTime) < 5*time.Minute && publicIPCache != "" {
		publicIPMutex.RUnlock()
		return publicIPCache
	}
	publicIPMutex.RUnlock()

	publicIPMutex.Lock()
	defer publicIPMutex.Unlock()

	// Double check after acquiring write lock
	if time.Since(publicIPCacheTime) < 5*time.Minute && publicIPCache != "" {
		return publicIPCache
	}

	// Try multiple IP detection services
	services := []string{
		"https://api.ipify.org?format=json",
		"https://httpbin.org/ip",
		"https://api.ip.sb/json",
	}

	client := &http.Client{Timeout: 5 * time.Second}

	for _, url := range services {
		resp, err := client.Get(url)
		if err != nil {
			continue
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			continue
		}

		// Try to parse JSON response
		var result map[string]interface{}
		if err := json.Unmarshal(body, &result); err == nil {
			if ip, ok := result["origin"].(string); ok && ip != "" {
				publicIPCache = ip
				publicIPCacheTime = time.Now()
				return ip
			}
			if ip, ok := result["ip"].(string); ok && ip != "" {
				publicIPCache = ip
				publicIPCacheTime = time.Now()
				return ip
			}
		}

		// Try plain text response
		ip := strings.TrimSpace(string(body))
		if ip != "" && !strings.Contains(ip, "{") {
			publicIPCache = ip
			publicIPCacheTime = time.Now()
			return ip
		}
	}

	return ""
}

// saveNodeID saves the generated node_id back to the config file.
func saveNodeID(configPath, nodeID string) error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("read config: %w", err)
	}

	var cfg map[string]interface{}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("parse config: %w", err)
	}

	// Update node_id
	cfg["node_id"] = nodeID

	// Marshal back to YAML
	out, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, out, 0600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	return nil
}
