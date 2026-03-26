// Package config handles server-side application configuration.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Config is the top-level server configuration.
type Config struct {
	Server    ServerConfig    `yaml:"server"`
	MongoDB   MongoConfig     `yaml:"mongodb"`
	Redis     RedisConfig     `yaml:"redis"`
	JWT       JWTConfig       `yaml:"jwt"`
	Log       LogConfig       `yaml:"log"`
	Grabber   GrabberConfig   `yaml:"grabber"`
	Node      NodeConfig      `yaml:"node"`
	Scheduler SchedulerConfig `yaml:"scheduler"`
	Email     EmailConfig     `yaml:"email"`
}

// NodeConfig holds node authentication settings.
type NodeConfig struct {
	// TokenKey is the shared secret key for node authentication.
	// Clients use this key to connect and auto-register.
	TokenKey string `yaml:"token_key"`
}

// ServerConfig holds HTTP server parameters.
type ServerConfig struct {
	Host        string   `yaml:"host"`
	Port        int      `yaml:"port"`
	Mode        string   `yaml:"mode"` // debug | release
	CORSOrigins []string `yaml:"cors_origins"` // Allowed CORS origins; empty means all
}

func (s *ServerConfig) Addr() string {
	return fmt.Sprintf("%s:%d", s.Host, s.Port)
}

// MongoConfig holds MongoDB connection parameters.
type MongoConfig struct {
	URI      string `yaml:"uri"`
	Database string `yaml:"database"`
}

// RedisConfig holds Redis connection parameters.
type RedisConfig struct {
	Addr     string `yaml:"addr"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

// JWTConfig holds JWT signing parameters.
type JWTConfig struct {
	Secret string        `yaml:"secret"`
	Expire time.Duration `yaml:"expire"`
}

// LogConfig holds logging parameters.
type LogConfig struct {
	Level  string `yaml:"level"`  // debug | info | warn | error
	Format string `yaml:"format"` // json | text
}

// GrabberConfig holds crawl scheduling parameters.
type GrabberConfig struct {
	// Interval is the default check interval for grabber tasks.
	Interval string `yaml:"interval"`
	// InitPageLimit is max pages fetched on initial crawl.
	InitPageLimit int `yaml:"init_page_limit"`
	// UpdatePageLimit is max pages fetched per tick.
	UpdatePageLimit int `yaml:"update_page_limit"`
}

// SchedulerConfig holds task scheduler parameters.
type SchedulerConfig struct {
	// MaxRetries is the maximum number of retry attempts for failed tasks.
	MaxRetries int `yaml:"max_retries"`
	// RetryInterval is the base retry interval.
	RetryInterval time.Duration `yaml:"retry_interval"`
	// BatchSize is the number of tasks to dispatch in one batch (1-100).
	BatchSize int `yaml:"batch_size"`
	// TaskTimeout is the default task timeout duration.
	TaskTimeout time.Duration `yaml:"task_timeout"`
	// TimeoutCheck is the interval for checking task timeouts.
	TimeoutCheck time.Duration `yaml:"timeout_check"`
}

// EmailConfig holds SMTP email configuration.
type EmailConfig struct {
	Enabled  bool   `yaml:"enabled"`
	Host     string `yaml:"host"`     // SMTP server host (e.g., smtp.gmail.com)
	Port     int    `yaml:"port"`     // SMTP server port (e.g., 587)
	Username string `yaml:"username"` // SMTP username (usually email address)
	Password string `yaml:"password"` // SMTP password or app password
	From     string `yaml:"from"`     // Sender email address
	FromName string `yaml:"from_name"` // Sender display name
	UseTLS   bool   `yaml:"use_tls"`  // Use STARTTLS (port 587)
}

// Load reads the YAML config file at path, then applies env overrides.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %q: %w", path, err)
	}
	var cfg Config
	if err = yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	cfg.applyEnv()
	cfg.setDefaults()
	return &cfg, nil
}

// applyEnv overrides fields with environment variables when set.
// Supports both plain env vars (MONGO_URI, REDIS_ADDR) and SECFLOW_ prefixed vars.
func (c *Config) applyEnv() {
	// MongoDB - support both SECFLOW_MONGO_URI and MONGO_URI
	if v := os.Getenv("SECFLOW_MONGO_URI"); v != "" {
		c.MongoDB.URI = v
	} else if v := os.Getenv("MONGO_URI"); v != "" {
		c.MongoDB.URI = v
	}
	// Redis - support both SECFLOW_REDIS_URL and REDIS_ADDR
	// SECFLOW_REDIS_URL can be redis:// URL format or host:port format
	if v := os.Getenv("SECFLOW_REDIS_URL"); v != "" {
		c.Redis.Addr = parseRedisURL(v)
		c.Redis.Password = parseRedisPassword(v)
	} else if v := os.Getenv("REDIS_ADDR"); v != "" {
		c.Redis.Addr = v
	}
	// Redis password can also be set directly
	if v := os.Getenv("SECFLOW_REDIS_PASSWORD"); v != "" {
		c.Redis.Password = v
	} else if v := os.Getenv("REDIS_PASSWORD"); v != "" {
		c.Redis.Password = v
	}
	// JWT Secret - support both SECFLOW_JWT_SECRET and JWT_SECRET
	if v := os.Getenv("SECFLOW_JWT_SECRET"); v != "" {
		c.JWT.Secret = v
	} else if v := os.Getenv("JWT_SECRET"); v != "" {
		c.JWT.Secret = v
	}
	// Server Mode - support both SECFLOW_ENV and SERVER_MODE
	if v := os.Getenv("SECFLOW_ENV"); v != "" {
		c.Server.Mode = v
	} else if v := os.Getenv("SERVER_MODE"); v != "" {
		c.Server.Mode = v
	}
	if v := os.Getenv("SECFLOW_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			c.Server.Port = port
		}
	}
	if v := os.Getenv("NODE_TOKEN_KEY"); v != "" {
		c.Node.TokenKey = v
	}
	if v := os.Getenv("CORS_ORIGINS"); v != "" {
		c.Server.CORSOrigins = strings.Split(v, ",")
	}
	if v := os.Getenv("SMTP_HOST"); v != "" {
		c.Email.Host = v
	}
	if v := os.Getenv("SMTP_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			c.Email.Port = port
		}
	}
	if v := os.Getenv("SMTP_USERNAME"); v != "" {
		c.Email.Username = v
	}
	if v := os.Getenv("SMTP_PASSWORD"); v != "" {
		c.Email.Password = v
	}
	if v := os.Getenv("SMTP_FROM"); v != "" {
		c.Email.From = v
	}
	if v := os.Getenv("EMAIL_ENABLED"); v == "true" || v == "1" {
		c.Email.Enabled = true
	}
}

// setDefaults fills in sensible defaults for unset fields.
func (c *Config) setDefaults() {
	if c.Server.Host == "" {
		c.Server.Host = "0.0.0.0"
	}
	if c.Server.Port == 0 {
		c.Server.Port = 8080
	}
	if c.Server.Mode == "" {
		c.Server.Mode = "release"
	}
	if c.MongoDB.Database == "" {
		c.MongoDB.Database = "secflow"
	}
	if c.Redis.Addr == "" {
		c.Redis.Addr = "127.0.0.1:6379"
	}
	if c.JWT.Expire == 0 {
		c.JWT.Expire = 24 * time.Hour
	}
	if c.JWT.Secret == "" {
		c.JWT.Secret = "secflow-change-me"
	}
	if c.Log.Level == "" {
		c.Log.Level = "info"
	}
	if c.Log.Format == "" {
		c.Log.Format = "json"
	}
	if c.Grabber.Interval == "" {
		c.Grabber.Interval = "1h"
	}
	if c.Grabber.InitPageLimit == 0 {
		c.Grabber.InitPageLimit = 3
	}
	if c.Grabber.UpdatePageLimit == 0 {
		c.Grabber.UpdatePageLimit = 1
	}
	if c.Node.TokenKey == "" {
		c.Node.TokenKey = "secflow-node-token-key-change-me"
	}
	
	// Scheduler defaults
	if c.Scheduler.MaxRetries == 0 {
		c.Scheduler.MaxRetries = 3
	}
	if c.Scheduler.RetryInterval == 0 {
		c.Scheduler.RetryInterval = 5 * time.Minute
	}
	if c.Scheduler.BatchSize == 0 {
		c.Scheduler.BatchSize = 3
	}
	if c.Scheduler.BatchSize < 1 {
		c.Scheduler.BatchSize = 1
	}
	if c.Scheduler.BatchSize > 100 {
		c.Scheduler.BatchSize = 100
	}
	if c.Scheduler.TaskTimeout == 0 {
		c.Scheduler.TaskTimeout = 30 * time.Minute
	}
	if c.Scheduler.TimeoutCheck == 0 {
		c.Scheduler.TimeoutCheck = 1 * time.Minute
	}

	// Email defaults
	if c.Email.Port == 0 {
		c.Email.Port = 587 // Default TLS port
	}
	if c.Email.FromName == "" {
		c.Email.FromName = "SecFlow"
	}
}

// Validate checks that required fields are present.
func (c *Config) Validate() error {
	if c.MongoDB.URI == "" {
		return fmt.Errorf("mongodb.uri is required")
	}

	// In production mode, enforce strong security secrets
	if c.Server.Mode == "release" {
		if c.JWT.Secret == "" || c.JWT.Secret == "secflow-change-me" {
			return fmt.Errorf("jwt.secret is required in production mode: must be a strong, unique secret")
		}
		if c.Node.TokenKey == "" || c.Node.TokenKey == "secflow-node-token-key-change-me" {
			return fmt.Errorf("node.token_key is required in production mode: must be a strong, unique key")
		}
	} else {
		// Development mode warnings
		if c.JWT.Secret == "secflow-change-me" {
			_, _ = fmt.Fprintln(os.Stderr, "[WARN] jwt.secret is using the default value in dev mode, please change it for production")
		}
		if c.Node.TokenKey == "secflow-node-token-key-change-me" {
			_, _ = fmt.Fprintln(os.Stderr, "[WARN] node.token_key is using the default value in dev mode, please change it for production")
		}
	}

	if !strings.Contains(c.Grabber.Interval, "m") && !strings.Contains(c.Grabber.Interval, "h") {
		return fmt.Errorf("grabber.interval must be a Go duration string (e.g. 30m, 1h)")
	}
	if c.Scheduler.BatchSize < 1 || c.Scheduler.BatchSize > 100 {
		return fmt.Errorf("scheduler.batch_size must be between 1 and 100")
	}
	return nil
}

// parseRedisURL parses a redis:// URL and returns host:port format.
// Supports formats like:
//   - redis://localhost:6379/0
//   - redis://:password@localhost:6379/0
//   - localhost:6379 (passthrough)
func parseRedisURL(redisURL string) string {
	// If not a URL, assume it's already host:port
	if !strings.HasPrefix(redisURL, "redis://") {
		return redisURL
	}

	// Manually parse redis:// URL
	// Format: redis://[user:password@]host[:port][/db]
	rest := redisURL[8:] // Remove "redis://"

	// Find @ to skip auth info
	if idx := strings.Index(rest, "@"); idx != -1 {
		rest = rest[idx+1:]
	}

	// Remove trailing db path if present
	if dbIdx := strings.Index(rest, "/"); dbIdx != -1 {
		rest = rest[:dbIdx]
	}

	// Now rest is host:port or just host
	if colonIdx := strings.Index(rest, ":"); colonIdx != -1 {
		host := rest[:colonIdx]
		port := rest[colonIdx+1:]
		if port == "" {
			port = "6379"
		}
		return host + ":" + port
	}

	// No port specified
	return rest + ":6379"
}

// parseRedisPassword extracts password from redis:// URL format.
// Returns empty string if no password is present.
func parseRedisPassword(redisURL string) string {
	if !strings.HasPrefix(redisURL, "redis://") {
		return ""
	}

	rest := redisURL[8:] // Remove "redis://"

	// Find @ to get password
	if idx := strings.Index(rest, "@"); idx != -1 {
		password := rest[:idx]
		// Skip username if present (user:password format)
		if colonIdx := strings.Index(password, ":"); colonIdx != -1 {
			return password[colonIdx+1:]
		}
		return password
	}

	return ""
}
