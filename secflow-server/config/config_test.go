package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name      string
		config    Config
		wantErr   bool
		errMsg    string
		callSetDefaults bool
	}{
		{
			name: "valid config in dev mode",
			config: Config{
				Server:  ServerConfig{Mode: "debug"},
				MongoDB: MongoConfig{URI: "mongodb://localhost"},
				JWT:     JWTConfig{Secret: "dev-secret"},
				Node:    NodeConfig{TokenKey: "dev-token"},
				Grabber: GrabberConfig{Interval: "1h"},
			},
			wantErr:        false,
			callSetDefaults: true,
		},
		{
			name: "valid config in release mode with strong secrets",
			config: Config{
				Server:  ServerConfig{Mode: "release"},
				MongoDB: MongoConfig{URI: "mongodb://localhost"},
				JWT:     JWTConfig{Secret: "strong-production-secret-at-least-32-chars"},
				Node:    NodeConfig{TokenKey: "strong-node-token-key-at-least-32-chars"},
				Grabber: GrabberConfig{Interval: "1h"},
			},
			wantErr:        false,
			callSetDefaults: true,
		},
		{
			name: "missing mongodb URI",
			config: Config{
				Server: ServerConfig{Mode: "debug"},
				Grabber: GrabberConfig{Interval: "1h"},
			},
			wantErr: true,
			errMsg:  "mongodb.uri is required",
		},
		{
			name: "release mode with default JWT secret",
			config: Config{
				Server:  ServerConfig{Mode: "release"},
				MongoDB: MongoConfig{URI: "mongodb://localhost"},
				JWT:     JWTConfig{Secret: "secflow-change-me"},
				Node:    NodeConfig{TokenKey: "strong-node-token"},
				Grabber: GrabberConfig{Interval: "1h"},
			},
			wantErr: true,
			errMsg:  "jwt.secret is required in production mode",
		},
		{
			name: "release mode with default node token",
			config: Config{
				Server:  ServerConfig{Mode: "release"},
				MongoDB: MongoConfig{URI: "mongodb://localhost"},
				JWT:     JWTConfig{Secret: "strong-production-secret"},
				Node:    NodeConfig{TokenKey: "secflow-node-token-key-change-me"},
				Grabber: GrabberConfig{Interval: "1h"},
			},
			wantErr: true,
			errMsg:  "node.token_key is required in production mode",
		},
		{
			name: "invalid scheduler batch size (0)",
			config: Config{
				Server:  ServerConfig{Mode: "debug"},
				MongoDB: MongoConfig{URI: "mongodb://localhost"},
				Grabber: GrabberConfig{Interval: "1h"},
				Scheduler: SchedulerConfig{BatchSize: 0},
			},
			wantErr: true,
			errMsg:  "scheduler.batch_size must be between 1 and 100",
		},
		{
			name: "scheduler batch size too large",
			config: Config{
				Server:  ServerConfig{Mode: "debug"},
				MongoDB: MongoConfig{URI: "mongodb://localhost"},
				Grabber: GrabberConfig{Interval: "1h"},
				Scheduler: SchedulerConfig{BatchSize: 101},
			},
			wantErr: true,
			errMsg:  "scheduler.batch_size must be between 1 and 100",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.callSetDefaults {
				tt.config.setDefaults()
			}
			err := tt.config.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfig_ApplyEnv(t *testing.T) {
	// Save original env
	origHost := os.Getenv("SMTP_HOST")
	origPort := os.Getenv("SMTP_PORT")
	origEnabled := os.Getenv("EMAIL_ENABLED")
	defer func() {
		os.Setenv("SMTP_HOST", origHost)
		os.Setenv("SMTP_PORT", origPort)
		os.Setenv("EMAIL_ENABLED", origEnabled)
	}()

	cfg := &Config{}
	cfg.applyEnv()

	// Test SMTP_HOST
	os.Setenv("SMTP_HOST", "smtp.test.com")
	cfg = &Config{}
	cfg.applyEnv()
	assert.Equal(t, "smtp.test.com", cfg.Email.Host)

	// Test SMTP_PORT
	os.Setenv("SMTP_PORT", "465")
	cfg = &Config{}
	cfg.applyEnv()
	assert.Equal(t, 465, cfg.Email.Port)

	// Test EMAIL_ENABLED
	os.Setenv("EMAIL_ENABLED", "true")
	cfg = &Config{}
	cfg.applyEnv()
	assert.True(t, cfg.Email.Enabled)
}

func TestConfig_SetDefaults(t *testing.T) {
	cfg := &Config{}
	cfg.setDefaults()

	assert.Equal(t, "0.0.0.0", cfg.Server.Host)
	assert.Equal(t, 8080, cfg.Server.Port)
	assert.Equal(t, "release", cfg.Server.Mode)
	assert.Equal(t, "secflow", cfg.MongoDB.Database)
	assert.Equal(t, "127.0.0.1:6379", cfg.Redis.Addr)
	assert.Equal(t, 24*60*60*1000000000, int(cfg.JWT.Expire)) // 24 hours in nanoseconds
	assert.Equal(t, "secflow-change-me", cfg.JWT.Secret)
	assert.Equal(t, "info", cfg.Log.Level)
	assert.Equal(t, "json", cfg.Log.Format)
	assert.Equal(t, 3, cfg.Grabber.InitPageLimit)
	assert.Equal(t, 1, cfg.Grabber.UpdatePageLimit)
	assert.Equal(t, 587, cfg.Email.Port)
	assert.Equal(t, "SecFlow", cfg.Email.FromName)
}
