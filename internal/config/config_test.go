package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig(t *testing.T) {
	// Create temporary config file
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.yaml")
	
	configContent := `
server:
  host: "0.0.0.0"
  port: 8080
  mode: "debug"
  graceful_shutdown_timeout: 30s

logging:
  level: "info"
  format: "json"
  output: "stdout"

database:
  driver: "postgres"
  host: "localhost"
  port: 5432
  name: "aos_test"
  username: "test_user"
  password: "test_password"
  ssl_mode: "disable"
  max_open_conns: 10
  max_idle_conns: 5
  conn_max_lifetime: 3600s

mode: "test"
debug: true
`

	err := os.WriteFile(configFile, []byte(configContent), 0644)
	require.NoError(t, err)

	// Test loading config
	cfg, err := LoadConfig(configFile)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Verify config values
	assert.Equal(t, "test", cfg.Mode)
	assert.True(t, cfg.Debug)
	assert.Equal(t, "0.0.0.0", cfg.Server.Host)
	assert.Equal(t, 8080, cfg.Server.Port)
	assert.Equal(t, "debug", cfg.Server.Mode)
	assert.Equal(t, 30*time.Second, cfg.Server.GracefulShutdownTimeout)
	assert.Equal(t, "info", cfg.Logging.Level)
	assert.Equal(t, "json", cfg.Logging.Format)
	assert.Equal(t, "stdout", cfg.Logging.Output)
	assert.Equal(t, "postgres", cfg.Database.Driver)
	assert.Equal(t, "localhost", cfg.Database.Host)
	assert.Equal(t, 5432, cfg.Database.Port)
	assert.Equal(t, "aos_test", cfg.Database.Name)
}

func TestLoadConfig_FileNotFound(t *testing.T) {
	_, err := LoadConfig("nonexistent.yaml")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read config file")
}

func TestLoadConfig_InvalidYAML(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "invalid.yaml")
	
	invalidContent := `invalid: yaml: content:`
	err := os.WriteFile(configFile, []byte(invalidContent), 0644)
	require.NoError(t, err)

	_, err = LoadConfig(configFile)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to unmarshal config")
}

func TestDefaultConfig(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "empty.yaml")

	emptyContent := `
mode: "development"
debug: true
`
	err := os.WriteFile(configFile, []byte(emptyContent), 0644)
	require.NoError(t, err)

	cfg, err := LoadConfig(configFile)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, "development", cfg.Mode)
	assert.True(t, cfg.Debug)
	assert.Equal(t, 8080, cfg.Server.Port)
}

func TestConfigValidation(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "invalid_config.yaml")

	invalidConfig := `
server:
  port: -1
mode: "invalid"
`
	err := os.WriteFile(configFile, []byte(invalidConfig), 0644)
	require.NoError(t, err)

	_, err = LoadConfig(configFile)
	assert.Error(t, err, "negative port should trigger validation error")
}

func TestGetDefaultConfigPath(t *testing.T) {
	// Test with explicit path
	explicitPath := "/path/to/config.yaml"
	result := getDefaultConfigPath(explicitPath)
	assert.Equal(t, explicitPath, result)

	// Test with empty string - should return default
	result = getDefaultConfigPath("")
	assert.Equal(t, defaultConfigPath, result)

	// Test with relative path
	relPath := "configs/local.yaml"
	result = getDefaultConfigPath(relPath)
	assert.Equal(t, relPath, result)
}

func TestNewConfig(t *testing.T) {
	cfg := NewConfig()
	require.NotNil(t, cfg)

	assert.Equal(t, "development", cfg.Mode)
	assert.False(t, cfg.Debug)
	assert.Equal(t, 8080, cfg.Server.Port)
	assert.Equal(t, "0.0.0.0", cfg.Server.Host)
}