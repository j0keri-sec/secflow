package tests

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/secflow/client/internal/config"
)

func TestConfigLoad(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "client.yaml")

	validConfig := `
mode: server
server:
  api_url: "http://127.0.0.1:8080/api/v1"
  ws_url: "ws://127.0.0.1:8080/ws/node"
  token_key: "test-token-key"
db_path: "./test.db"
log_level: info
node_id: "test-node-001"
name: "test-node"
heartbeat_interval: 30s
`

	err := os.WriteFile(configPath, []byte(validConfig), 0644)
	require.NoError(t, err)

	cfg, err := config.Load(configPath)
	require.NoError(t, err)
	assert.Equal(t, config.ModeServer, cfg.Mode)
	assert.Equal(t, "http://127.0.0.1:8080/api/v1", cfg.Server.APIURL)
	assert.Equal(t, "ws://127.0.0.1:8080/ws/node", cfg.Server.WSURL)
	assert.Equal(t, "test-token-key", cfg.Server.TokenKey)
	assert.Equal(t, "test-node-001", cfg.NodeID)
	assert.Equal(t, "test-node", cfg.Name)
	assert.Equal(t, "info", cfg.LogLevel)
}

func TestConfigLoadStandaloneMode(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "client.yaml")

	// Standalone mode doesn't require server config
	standaloneConfig := `
mode: standalone
db_path: "./test.db"
log_level: debug
scheduler:
  enabled: true
  interval: 1h
`

	err := os.WriteFile(configPath, []byte(standaloneConfig), 0644)
	require.NoError(t, err)

	cfg, err := config.Load(configPath)
	require.NoError(t, err)
	assert.Equal(t, config.ModeStandalone, cfg.Mode)
	assert.True(t, cfg.IsStandalone())
	assert.False(t, cfg.IsServerMode())
	assert.Equal(t, "debug", cfg.LogLevel)
}

func TestConfigValidation(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name    string
		config  string
		wantErr string
	}{
		{
			name: "server mode missing api_url",
			config: `
mode: server
ws_url: "ws://127.0.0.1:8080/ws/node"
token_key: "test-key"
`,
			wantErr: "server.api_url is required",
		},
		{
			name: "server mode missing ws_url",
			config: `
mode: server
api_url: "http://127.0.0.1:8080/api/v1"
token_key: "test-key"
`,
			wantErr: "server.ws_url is required",
		},
		{
			name: "server mode missing token_key",
			config: `
mode: server
api_url: "http://127.0.0.1:8080/api/v1"
ws_url: "ws://127.0.0.1:8080/ws/node"
`,
			wantErr: "server.token_key is required",
		},
		{
			name: "invalid log level",
			config: `
mode: standalone
log_level: invalid
`,
			wantErr: "invalid log_level",
		},
		{
			name: "invalid mode",
			config: `
mode: unknown
api_url: "http://127.0.0.1:8080/api/v1"
ws_url: "ws://127.0.0.1:8080/ws/node"
token_key: "test-key"
`,
			wantErr: "invalid mode",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configPath := filepath.Join(tmpDir, tt.name+".yaml")
			err := os.WriteFile(configPath, []byte(tt.config), 0644)
			require.NoError(t, err)

			_, err = config.Load(configPath)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestConfigEnvOverrides(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "client.yaml")

	baseConfig := `
mode: server
server:
  api_url: "http://127.0.0.1:8080/api/v1"
  ws_url: "ws://127.0.0.1:8080/ws/node"
  token_key: "original-key"
db_path: "./test.db"
log_level: info
`

	err := os.WriteFile(configPath, []byte(baseConfig), 0644)
	require.NoError(t, err)

	// Set environment variables
	os.Setenv("SECFLOW_API_URL", "http://env:8080/api/v1")
	os.Setenv("SECFLOW_LOG_LEVEL", "debug")
	os.Setenv("SECFLOW_NODE_NAME", "env-node")
	defer func() {
		os.Unsetenv("SECFLOW_API_URL")
		os.Unsetenv("SECFLOW_LOG_LEVEL")
		os.Unsetenv("SECFLOW_NODE_NAME")
	}()

	cfg, err := config.Load(configPath)
	require.NoError(t, err)

	// Environment should override file values
	assert.Equal(t, "http://env:8080/api/v1", cfg.Server.APIURL)
	assert.Equal(t, "debug", cfg.LogLevel)
	assert.Equal(t, "env-node", cfg.Name)
	// Original token_key should remain since env not set
	assert.Equal(t, "original-key", cfg.Server.TokenKey)
}

func TestConfigDefaults(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "minimal.yaml")

	// Minimal config - should apply defaults
	minimalConfig := `
mode: standalone
`

	err := os.WriteFile(configPath, []byte(minimalConfig), 0644)
	require.NoError(t, err)

	cfg, err := config.Load(configPath)
	require.NoError(t, err)

	// Check defaults
	assert.Equal(t, "info", cfg.LogLevel)
	assert.Equal(t, int64(30), int64(cfg.HeartbeatInterval.Seconds()))
	assert.Equal(t, 5*60*1e9, int64(cfg.Connection.ReconnectInterval.Nanoseconds()))
	assert.Equal(t, 10*60*1e9, int64(cfg.Connection.Timeout.Nanoseconds()))
	assert.Equal(t, 1, cfg.Task.DefaultPageLimit)
	assert.Equal(t, 1, cfg.Task.MaxConcurrent)
	assert.Equal(t, 30*60, int64(cfg.Task.Timeout.Seconds()))
	assert.Equal(t, 3, cfg.Grabber.RetryAttempts)
	assert.Equal(t, "SecFlow-Client/1.0", cfg.Grabber.UserAgent)
	assert.True(t, cfg.Grabber.TLSVerify)
}

func TestGetEnabledSources(t *testing.T) {
	cfg := &config.Config{
		Grabber: config.GrabberConfig{
			Sources:         []string{"avd", "seebug"},
			DisabledSources: []string{"nvd"},
		},
	}

	allSources := []string{"avd", "seebug", "nvd", "kev", "threatbook"}

	// When specific sources are configured, return them
	enabled := cfg.GetEnabledSources(allSources)
	assert.Equal(t, []string{"avd", "seebug"}, enabled)

	// When no specific sources, filter disabled ones
	cfg.Grabber.Sources = nil
	enabled = cfg.GetEnabledSources(allSources)
	assert.Equal(t, []string{"avd", "seebug", "kev", "threatbook"}, enabled)
}

func TestConfigFileNotFound(t *testing.T) {
	_, err := config.Load("/nonexistent/path/client.yaml")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "read config")
}

func TestConfigInvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid.yaml")

	err := os.WriteFile(configPath, []byte("invalid: yaml: content: ["), 0644)
	require.NoError(t, err)

	_, err = config.Load(configPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse config")
}
