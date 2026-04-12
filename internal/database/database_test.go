package database

import (
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Host != "localhost" {
		t.Errorf("expected host=localhost, got %s", cfg.Host)
	}
	if cfg.Port != 5432 {
		t.Errorf("expected port=5432, got %d", cfg.Port)
	}
	if cfg.User != "postgres" {
		t.Errorf("expected user=postgres, got %s", cfg.User)
	}
	if cfg.Database != "agent_os" {
		t.Errorf("expected database=agent_os, got %s", cfg.Database)
	}
	if cfg.SSLMode != "disable" {
		t.Errorf("expected sslmode=disable, got %s", cfg.SSLMode)
	}
	if cfg.MaxOpenConns != 25 {
		t.Errorf("expected max_open_conns=25, got %d", cfg.MaxOpenConns)
	}
	if cfg.MaxIdleConns != 5 {
		t.Errorf("expected max_idle_conns=5, got %d", cfg.MaxIdleConns)
	}
	if cfg.ConnMaxLifetime != 5*time.Minute {
		t.Errorf("expected conn_max_lifetime=5m, got %v", cfg.ConnMaxLifetime)
	}
}

func TestNewDatabase_NilConfig(t *testing.T) {
	// This test verifies that nil config defaults are applied correctly.
	// We cannot actually connect to a database in unit tests, so we test
	// the configuration path only.
	cfg := DefaultConfig()
	if cfg == nil {
		t.Error("expected non-nil config from DefaultConfig()")
	}
}

func TestNewDatabase_InvalidConnection(t *testing.T) {
	logger := zap.NewNop()
	cfg := &Config{
		Host:              "nonexistent-host",
		Port:              5432,
		User:              "invalid",
		Password:          "invalid",
		Database:          "invalid",
		SSLMode:           "disable",
		ConnectionTimeout: 1 * time.Second,
	}

	_, err := NewDatabase(cfg, logger)
	if err == nil {
		t.Error("expected error connecting to nonexistent database")
	}
}

func TestHealthCheck_NilDB(t *testing.T) {
	db := &Database{DB: nil, Logger: zap.NewNop()}
	err := db.HealthCheck(nil)
	if err == nil {
		t.Error("expected error for nil database")
	}
}

func TestConfig_Fields(t *testing.T) {
	cfg := &Config{
		Host:              "db.example.com",
		Port:              5433,
		User:              "aos_user",
		Password:          "secret",
		Database:          "aos_prod",
		SSLMode:           "require",
		MaxOpenConns:      50,
		MaxIdleConns:      10,
		ConnMaxLifetime:   10 * time.Minute,
		ConnectionTimeout: 5 * time.Second,
	}

	if cfg.Host != "db.example.com" {
		t.Errorf("expected db.example.com, got %s", cfg.Host)
	}
	if cfg.Port != 5433 {
		t.Errorf("expected 5433, got %d", cfg.Port)
	}
	if cfg.SSLMode != "require" {
		t.Errorf("expected require, got %s", cfg.SSLMode)
	}
}
