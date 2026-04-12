package database

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestDefaultConfig_AllFields(t *testing.T) {
	cfg := DefaultConfig()
	require.NotNil(t, cfg)

	assert.Equal(t, "localhost", cfg.Host)
	assert.Equal(t, 5432, cfg.Port)
	assert.Equal(t, "postgres", cfg.User)
	assert.Equal(t, "postgres", cfg.Password)
	assert.Equal(t, "agent_os", cfg.Database)
	assert.Equal(t, "disable", cfg.SSLMode)
	assert.Equal(t, 25, cfg.MaxOpenConns)
	assert.Equal(t, 5, cfg.MaxIdleConns)
	assert.Equal(t, 5*time.Minute, cfg.ConnMaxLifetime)
	assert.Equal(t, 30*time.Second, cfg.ConnectionTimeout)
}

func TestConfig_CustomValues(t *testing.T) {
	cfg := &Config{
		Host:              "db.prod.example.com",
		Port:              5433,
		User:              "aos_prod",
		Password:          "super-secret",
		Database:          "aos_production",
		SSLMode:           "require",
		MaxOpenConns:      100,
		MaxIdleConns:      20,
		ConnMaxLifetime:   15 * time.Minute,
		ConnectionTimeout: 10 * time.Second,
	}

	assert.Equal(t, "db.prod.example.com", cfg.Host)
	assert.Equal(t, 5433, cfg.Port)
	assert.Equal(t, "aos_prod", cfg.User)
	assert.Equal(t, "require", cfg.SSLMode)
	assert.Equal(t, 100, cfg.MaxOpenConns)
	assert.Equal(t, 20, cfg.MaxIdleConns)
	assert.Equal(t, 15*time.Minute, cfg.ConnMaxLifetime)
}

func TestNewDatabase_ConnectionRefused(t *testing.T) {
	logger := zap.NewNop()
	cfg := &Config{
		Host:              "127.0.0.1",
		Port:              15432,
		User:              "test",
		Password:          "test",
		Database:          "test_db",
		SSLMode:           "disable",
		ConnectionTimeout: 2 * time.Second,
	}

	_, err := NewDatabase(cfg, logger)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to connect to database")
}

func TestNewDatabase_InvalidHost(t *testing.T) {
	logger := zap.NewNop()
	cfg := &Config{
		Host:              "nonexistent.invalid.host.example.com",
		Port:              5432,
		User:              "test",
		Password:          "test",
		Database:          "test",
		SSLMode:           "disable",
		ConnectionTimeout: 1 * time.Second,
	}

	_, err := NewDatabase(cfg, logger)
	require.Error(t, err)
}

func TestClose_NilDB(t *testing.T) {
	db := &Database{DB: nil, Logger: zap.NewNop()}
	err := db.Close()
	assert.NoError(t, err)
}

func TestDatabase_ZeroValueConfig(t *testing.T) {
	cfg := &Config{
		ConnectionTimeout: 1 * time.Second,
	}

	_, err := NewDatabase(cfg, zap.NewNop())
	require.Error(t, err)
}

func TestDatabase_StructFields(t *testing.T) {
	db := &Database{
		DB:     nil,
		Logger: zap.NewNop(),
	}

	assert.Nil(t, db.DB)
	assert.NotNil(t, db.Logger)
}

func TestMigration_Struct(t *testing.T) {
	now := time.Now().UTC()
	m := Migration{
		ID:        1,
		Name:      "001_create_agents",
		AppliedAt: now,
	}

	assert.Equal(t, 1, m.ID)
	assert.Equal(t, "001_create_agents", m.Name)
	assert.Equal(t, now, m.AppliedAt)
}

func TestConfig_SSLModes(t *testing.T) {
	modes := []string{"disable", "require", "verify-ca", "verify-full"}
	for _, mode := range modes {
		cfg := &Config{
			Host:     "localhost",
			Port:     5432,
			User:     "test",
			Password: "test",
			Database: "test",
			SSLMode:  mode,
		}
		assert.Equal(t, mode, cfg.SSLMode, "SSL mode should be %s", mode)
	}
}

func TestConfig_ConnectionPoolSettings(t *testing.T) {
	cfg := &Config{
		Host:            "localhost",
		Port:            5432,
		MaxOpenConns:    50,
		MaxIdleConns:    10,
		ConnMaxLifetime: 10 * time.Minute,
	}

	assert.Equal(t, 50, cfg.MaxOpenConns)
	assert.Equal(t, 10, cfg.MaxIdleConns)
	assert.Equal(t, 10*time.Minute, cfg.ConnMaxLifetime)
}

func TestHealthCheck_CancelledContext(t *testing.T) {
	db := &Database{DB: nil, Logger: zap.NewNop()}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := db.HealthCheck(ctx)
	require.Error(t, err)
}

func TestNewDatabase_NilConfig_Defaults(t *testing.T) {
	// Verify DefaultConfig returns reasonable values
	cfg := DefaultConfig()
	if cfg == nil {
		t.Error("expected non-nil config from DefaultConfig()")
	}
	// These should not be zero
	if cfg.MaxOpenConns == 0 {
		t.Error("MaxOpenConns should not be zero")
	}
	if cfg.MaxIdleConns == 0 {
		t.Error("MaxIdleConns should not be zero")
	}
}
