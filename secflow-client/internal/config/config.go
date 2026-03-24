// Package config handles client configuration loading from YAML / env variables.
package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Mode represents the client operating mode.
type Mode string

const (
	// ModeServer connects to SecFlow server for distributed task execution.
	ModeServer Mode = "server"
	// ModeStandalone runs independently without server connection.
	ModeStandalone Mode = "standalone"
)

// Config is the root client configuration structure.
type Config struct {
	// Operating mode: "server" or "standalone" (default: "server")
	// - server: connects to SecFlow server, receives tasks via WebSocket
	// - standalone: runs independently, executes scheduled tasks locally
	Mode Mode `yaml:"mode"`

	// Server-side connection parameters (read from local file at startup).
	// Only required when mode is "server".
	Server ServerConfig `yaml:"server"`

	// Local SQLite database path.
	DBPath string `yaml:"db_path"`

	// Log file path (leave empty to log to stdout only).
	LogPath string `yaml:"log_path"`

	// Log level: debug, info, warn, error (default: info)
	LogLevel string `yaml:"log_level"`

	// Node identity — auto-generated on first run if empty.
	NodeID string `yaml:"node_id"`
	Name   string `yaml:"name"`

	// Heartbeat interval sent to server.
	HeartbeatInterval time.Duration `yaml:"heartbeat_interval"`

	// Proxy used by all HTTP grabbers.
	Proxy string `yaml:"proxy"`

	// Connection settings
	Connection ConnectionConfig `yaml:"connection"`

	// Task execution settings
	Task TaskConfig `yaml:"task"`

	// Grabber settings
	Grabber GrabberConfig `yaml:"grabber"`

	// Scheduler settings for standalone mode
	Scheduler SchedulerConfig `yaml:"scheduler"`
}

// SchedulerConfig contains settings for standalone mode task scheduling.
type SchedulerConfig struct {
	// Enable scheduled task execution (default: true in standalone mode)
	Enabled bool `yaml:"enabled"`

	// Interval between scheduled crawls (default: 1h)
	Interval time.Duration `yaml:"interval"`

	// Sources to crawl in standalone mode (if empty, uses all enabled sources)
	Sources []string `yaml:"sources"`
}

// ServerConfig contains server endpoint and authentication.
type ServerConfig struct {
	// Base HTTP API endpoint, e.g. http://127.0.0.1:8080/api/v1
	APIURL string `yaml:"api_url"`

	// WebSocket endpoint, e.g. ws://127.0.0.1:8080/ws/node
	WSURL string `yaml:"ws_url"`

	// TokenKey is the shared secret key for node authentication.
	// Must match the server's NODE_TOKEN_KEY configuration.
	TokenKey string `yaml:"token_key"`
}

// ConnectionConfig contains WebSocket connection settings.
type ConnectionConfig struct {
	// Reconnect interval when connection is lost (default: 5s)
	ReconnectInterval time.Duration `yaml:"reconnect_interval"`

	// Connection timeout (default: 10s)
	Timeout time.Duration `yaml:"timeout"`

	// Enable auto reconnect (default: true)
	AutoReconnect bool `yaml:"auto_reconnect"`

	// Max reconnect attempts, 0 means unlimited (default: 0)
	MaxReconnectAttempts int `yaml:"max_reconnect_attempts"`
}

// TaskConfig contains task execution settings.
type TaskConfig struct {
	// Default page limit for grabbers (default: 1)
	DefaultPageLimit int `yaml:"default_page_limit"`

	// Max concurrent tasks (default: 1)
	MaxConcurrent int `yaml:"max_concurrent"`

	// Task timeout (default: 30m)
	Timeout time.Duration `yaml:"timeout"`

	// Enable GitHub search for CVE (default: false)
	EnableGithubSearch bool `yaml:"enable_github_search"`
}

// GrabberConfig contains grabber-specific settings.
type GrabberConfig struct {
	// Request timeout for HTTP requests (default: 30s)
	RequestTimeout time.Duration `yaml:"request_timeout"`

	// Retry attempts for failed requests (default: 3)
	RetryAttempts int `yaml:"retry_attempts"`

	// Retry interval (default: 5s)
	RetryInterval time.Duration `yaml:"retry_interval"`

	// User agent string
	UserAgent string `yaml:"user_agent"`

	// Enable TLS verification (default: true)
	TLSVerify bool `yaml:"tls_verify"`

	// Sources to enable (empty means all)
	Sources []string `yaml:"sources"`

	// Sources to disable
	DisabledSources []string `yaml:"disabled_sources"`

	// ThreatBook API key (optional)
	ThreatbookAPIKey string `yaml:"threatbook_api_key"`
}

const defaultConfigPath = "client.yaml"

// Load reads the YAML config file, falling back to environment variables.
func Load(path string) (*Config, error) {
	if path == "" {
		path = defaultConfigPath
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}

	cfg := defaultConfig()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	// Apply environment variable overrides
	cfg.applyEnvOverrides()

	// Apply defaults for empty values
	cfg.applyDefaults()

	return cfg, cfg.Validate()
}

// defaultConfig returns a config with default values.
func defaultConfig() *Config {
	return &Config{
		Mode:              ModeServer, // default to server mode
		DBPath:            "secflow_client.db",
		LogLevel:          "info",
		HeartbeatInterval: 30 * time.Second,
		Connection: ConnectionConfig{
			ReconnectInterval:    5 * time.Second,
			Timeout:              10 * time.Second,
			AutoReconnect:        true,
			MaxReconnectAttempts: 0,
		},
		Task: TaskConfig{
			DefaultPageLimit:   1,
			MaxConcurrent:      1,
			Timeout:            30 * time.Minute,
			EnableGithubSearch: false,
		},
		Grabber: GrabberConfig{
			RequestTimeout: 30 * time.Second,
			RetryAttempts:  3,
			RetryInterval:  5 * time.Second,
			UserAgent:      "SecFlow-Client/1.0",
			TLSVerify:      true,
		},
		Scheduler: SchedulerConfig{
			Enabled:  true,
			Interval: 1 * time.Hour,
		},
	}
}

// applyEnvOverrides applies environment variable overrides.
func (c *Config) applyEnvOverrides() {
	if v := os.Getenv("SECFLOW_MODE"); v != "" {
		c.Mode = Mode(v)
	}
	if v := os.Getenv("SECFLOW_API_URL"); v != "" {
		c.Server.APIURL = v
	}
	if v := os.Getenv("SECFLOW_WS_URL"); v != "" {
		c.Server.WSURL = v
	}
	if v := os.Getenv("SECFLOW_TOKEN_KEY"); v != "" {
		c.Server.TokenKey = v
	}
	if v := os.Getenv("SECFLOW_PROXY"); v != "" {
		c.Proxy = v
	}
	if v := os.Getenv("SECFLOW_LOG_LEVEL"); v != "" {
		c.LogLevel = v
	}
	if v := os.Getenv("SECFLOW_NODE_ID"); v != "" {
		c.NodeID = v
	}
	if v := os.Getenv("SECFLOW_NODE_NAME"); v != "" {
		c.Name = v
	}
	if v := os.Getenv("SECFLOW_DB_PATH"); v != "" {
		c.DBPath = v
	}
}

// applyDefaults applies default values for empty fields.
func (c *Config) applyDefaults() {
	if c.LogLevel == "" {
		c.LogLevel = "info"
	}
	if c.HeartbeatInterval == 0 {
		c.HeartbeatInterval = 30 * time.Second
	}
	if c.Connection.ReconnectInterval == 0 {
		c.Connection.ReconnectInterval = 5 * time.Second
	}
	if c.Connection.Timeout == 0 {
		c.Connection.Timeout = 10 * time.Second
	}
	if c.Task.DefaultPageLimit == 0 {
		c.Task.DefaultPageLimit = 1
	}
	if c.Task.MaxConcurrent == 0 {
		c.Task.MaxConcurrent = 1
	}
	if c.Task.Timeout == 0 {
		c.Task.Timeout = 30 * time.Minute
	}
	if c.Grabber.RequestTimeout == 0 {
		c.Grabber.RequestTimeout = 30 * time.Second
	}
	if c.Grabber.RetryAttempts == 0 {
		c.Grabber.RetryAttempts = 3
	}
	if c.Grabber.RetryInterval == 0 {
		c.Grabber.RetryInterval = 5 * time.Second
	}
	if c.Grabber.UserAgent == "" {
		c.Grabber.UserAgent = "SecFlow-Client/1.0"
	}
}

// Validate ensures required fields are present.
func (c *Config) Validate() error {
	// Validate mode
	switch c.Mode {
	case ModeServer, ModeStandalone:
		// valid
	case "":
		c.Mode = ModeServer // default to server mode if not specified
	default:
		return fmt.Errorf("invalid mode: %s (must be 'server' or 'standalone')", c.Mode)
	}

	// Server config only required in server mode
	if c.Mode == ModeServer {
		if c.Server.APIURL == "" {
			return fmt.Errorf("server.api_url is required in server mode")
		}
		if c.Server.WSURL == "" {
			return fmt.Errorf("server.ws_url is required in server mode")
		}
		if c.Server.TokenKey == "" {
			return fmt.Errorf("server.token_key is required in server mode (must match server's NODE_TOKEN_KEY)")
		}
	}

	// Validate log level
	validLogLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !validLogLevels[c.LogLevel] {
		return fmt.Errorf("invalid log_level: %s (must be debug, info, warn, or error)", c.LogLevel)
	}

	return nil
}

// IsStandalone returns true if the client is running in standalone mode.
func (c *Config) IsStandalone() bool {
	return c.Mode == ModeStandalone
}

// IsServerMode returns true if the client is running in server mode.
func (c *Config) IsServerMode() bool {
	return c.Mode == ModeServer
}

// GetEnabledSources returns the list of enabled grabber sources.
func (c *Config) GetEnabledSources(allSources []string) []string {
	// If specific sources are configured, use them
	if len(c.Grabber.Sources) > 0 {
		return c.Grabber.Sources
	}

	// Otherwise, filter out disabled sources
	enabled := make([]string, 0, len(allSources))
	disabled := make(map[string]bool)
	for _, s := range c.Grabber.DisabledSources {
		disabled[s] = true
	}

	for _, s := range allSources {
		if !disabled[s] {
			enabled = append(enabled, s)
		}
	}
	return enabled
}
