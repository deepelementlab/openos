package database

import (
	"context"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestDefaultConfig_Comprehensive(t *testing.T) {
	cfg := DefaultConfig()
	if cfg == nil {
		t.Fatal("DefaultConfig returned nil")
	}
	if cfg.Host != "localhost" {
		t.Errorf("Host = %q, want %q", cfg.Host, "localhost")
	}
	if cfg.Port != 5432 {
		t.Errorf("Port = %d, want %d", cfg.Port, 5432)
	}
	if cfg.User != "postgres" {
		t.Errorf("User = %q, want %q", cfg.User, "postgres")
	}
	if cfg.Password != "postgres" {
		t.Errorf("Password = %q, want %q", cfg.Password, "postgres")
	}
	if cfg.Database != "agent_os" {
		t.Errorf("Database = %q, want %q", cfg.Database, "agent_os")
	}
	if cfg.SSLMode != "disable" {
		t.Errorf("SSLMode = %q, want %q", cfg.SSLMode, "disable")
	}
	if cfg.MaxOpenConns != 25 {
		t.Errorf("MaxOpenConns = %d, want %d", cfg.MaxOpenConns, 25)
	}
	if cfg.MaxIdleConns != 5 {
		t.Errorf("MaxIdleConns = %d, want %d", cfg.MaxIdleConns, 5)
	}
	if cfg.ConnMaxLifetime != 5*time.Minute {
		t.Errorf("ConnMaxLifetime = %v, want %v", cfg.ConnMaxLifetime, 5*time.Minute)
	}
	if cfg.ConnectionTimeout != 30*time.Second {
		t.Errorf("ConnectionTimeout = %v, want %v", cfg.ConnectionTimeout, 30*time.Second)
	}
}

func TestConfig_Fields_Comprehensive(t *testing.T) {
	cfg := &Config{
		Host:              "db.example.com",
		Port:              5433,
		User:              "admin",
		Password:          "secret",
		Database:          "testdb",
		SSLMode:           "require",
		MaxOpenConns:      50,
		MaxIdleConns:      10,
		ConnMaxLifetime:   10 * time.Minute,
		ConnectionTimeout: 5 * time.Second,
	}
	if cfg.Host != "db.example.com" {
		t.Errorf("Host = %q, want %q", cfg.Host, "db.example.com")
	}
	if cfg.Port != 5433 {
		t.Errorf("Port = %d, want %d", cfg.Port, 5433)
	}
	if cfg.User != "admin" {
		t.Errorf("User = %q, want %q", cfg.User, "admin")
	}
	if cfg.Password != "secret" {
		t.Errorf("Password = %q, want %q", cfg.Password, "secret")
	}
	if cfg.Database != "testdb" {
		t.Errorf("Database = %q, want %q", cfg.Database, "testdb")
	}
	if cfg.SSLMode != "require" {
		t.Errorf("SSLMode = %q, want %q", cfg.SSLMode, "require")
	}
	if cfg.MaxOpenConns != 50 {
		t.Errorf("MaxOpenConns = %d, want %d", cfg.MaxOpenConns, 50)
	}
	if cfg.MaxIdleConns != 10 {
		t.Errorf("MaxIdleConns = %d, want %d", cfg.MaxIdleConns, 10)
	}
	if cfg.ConnMaxLifetime != 10*time.Minute {
		t.Errorf("ConnMaxLifetime = %v, want %v", cfg.ConnMaxLifetime, 10*time.Minute)
	}
	if cfg.ConnectionTimeout != 5*time.Second {
		t.Errorf("ConnectionTimeout = %v, want %v", cfg.ConnectionTimeout, 5*time.Second)
	}
}

func TestNewDatabase_NilConfig_UsesDefault(t *testing.T) {
	logger := zap.NewNop()
	db, err := NewDatabase(nil, logger)
	if err == nil {
		if db != nil {
			db.Close()
		}
		t.Fatal("expected error when no PostgreSQL is available")
	}
	if !strings.Contains(err.Error(), "failed to connect") {
		t.Errorf("error = %q, want to contain %q", err.Error(), "failed to connect")
	}
}

func TestNewDatabase_ConnectionFailed(t *testing.T) {
	logger := zap.NewNop()
	cfg := &Config{
		Host:              "nonexistent-host.local",
		Port:              9999,
		User:              "nobody",
		Password:          "nope",
		Database:          "nope",
		SSLMode:           "disable",
		MaxOpenConns:      1,
		MaxIdleConns:      1,
		ConnMaxLifetime:   time.Minute,
		ConnectionTimeout: 2 * time.Second,
	}
	db, err := NewDatabase(cfg, logger)
	if err == nil {
		if db != nil {
			db.Close()
		}
		t.Fatal("expected error for invalid host")
	}
	if !strings.Contains(err.Error(), "failed to connect") {
		t.Errorf("error = %q, want to contain %q", err.Error(), "failed to connect")
	}
}

func TestDatabase_Close_NilDB(t *testing.T) {
	d := &Database{DB: nil, Logger: zap.NewNop()}
	if err := d.Close(); err != nil {
		t.Errorf("Close() with nil DB returned error: %v", err)
	}
}

func TestDatabase_HealthCheck_NilDB_Comprehensive(t *testing.T) {
	d := &Database{DB: nil, Logger: zap.NewNop()}
	err := d.HealthCheck(context.Background())
	if err == nil {
		t.Fatal("expected error for nil DB")
	}
	if !strings.Contains(err.Error(), "nil") {
		t.Errorf("error = %q, want to contain %q", err.Error(), "nil")
	}
}

func TestDatabase_GetStats_NilDB(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic when calling GetStats with nil DB")
		}
	}()
	d := &Database{DB: nil, Logger: zap.NewNop()}
	_ = d.GetStats()
}

func TestMigration_Fields(t *testing.T) {
	now := time.Now()
	m := Migration{
		ID:        42,
		Name:      "create_users_table",
		AppliedAt: now,
	}
	if m.ID != 42 {
		t.Errorf("ID = %d, want %d", m.ID, 42)
	}
	if m.Name != "create_users_table" {
		t.Errorf("Name = %q, want %q", m.Name, "create_users_table")
	}
	if !m.AppliedAt.Equal(now) {
		t.Errorf("AppliedAt = %v, want %v", m.AppliedAt, now)
	}
}

func TestDatabase_Close_WithNonNilDB(t *testing.T) {
	logger := zap.NewNop()
	d := &Database{DB: nil, Logger: logger}
	if err := d.Close(); err != nil {
		t.Errorf("Close() returned error: %v", err)
	}
}
