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
}

// NodeConfig holds node authentication settings.
type NodeConfig struct {
	// TokenKey is the shared secret key for node authentication.
	// Clients use this key to connect and auto-register.
	TokenKey string `yaml:"token_key"`
}

// ServerConfig holds HTTP server parameters.
type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
	Mode string `yaml:"mode"` // debug | release
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
func (c *Config) applyEnv() {
	if v := os.Getenv("MONGO_URI"); v != "" {
		c.MongoDB.URI = v
	}
	if v := os.Getenv("REDIS_ADDR"); v != "" {
		c.Redis.Addr = v
	}
	if v := os.Getenv("JWT_SECRET"); v != "" {
		c.JWT.Secret = v
	}
	if v := os.Getenv("SERVER_MODE"); v != "" {
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
}

// Validate checks that required fields are present.
func (c *Config) Validate() error {
	if c.MongoDB.URI == "" {
		return fmt.Errorf("mongodb.uri is required")
	}
	if c.JWT.Secret == "secflow-change-me" {
		// warn only — allow dev startup
		_, _ = fmt.Fprintln(os.Stderr, "[WARN] jwt.secret is using the default value, please change it")
	}
	if !strings.Contains(c.Grabber.Interval, "m") && !strings.Contains(c.Grabber.Interval, "h") {
		return fmt.Errorf("grabber.interval must be a Go duration string (e.g. 30m, 1h)")
	}
	if c.Scheduler.BatchSize < 1 || c.Scheduler.BatchSize > 100 {
		return fmt.Errorf("scheduler.batch_size must be between 1 and 100")
	}
	return nil
}
