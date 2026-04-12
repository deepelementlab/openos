// Package integration provides integration tests for Agent OS
package integration

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/agentos/aos/internal/config"
	"github.com/agentos/aos/internal/server"
	"go.uber.org/zap"
)

// TestServerIntegration tests basic server integration
func TestServerIntegration(t *testing.T) {
	// Create test configuration matching actual config structure
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host:                    "127.0.0.1",
			Port:                    0,
			Mode:                    "test",
			GracefulShutdownTimeout: time.Second,
		},
		Database: config.DatabaseConfig{
			Host:           "localhost",
			Port:           5432,
			Username:       "test_user",
			Password:       "test_password",
			Name:           "agentos_test",
			SSLMode:        "disable",
			MaxOpenConns:   10,
			MaxIdleConns:   5,
			ConnMaxLifetime: time.Hour,
		},
		Security: config.SecurityConfig{
			JWTSecret:            "test-secret-key-for-integration-tests-only",
			JWTExpiry:            24,
			CORSEnabled:          true,
			CORSAllowedOrigins:   []string{"*"},
			RateLimitEnabled:     false,
			RateLimitRequests:    100,
			RateLimitBurst:       20,
		},
		Agent: config.AgentConfig{
			DefaultImage:      "alpine:latest",
			DefaultWorkingDir: "/app",
			AllowedRegistries: []string{"docker.io"},
			MaxImageSize:      "1GB",
			MaxExecutionTime:  time.Hour,
			AutoRestart:       false,
			RestartPolicy:     "never",
		},
		Monitoring: config.MonitoringConfig{
			Enabled:                  true,
			PrometheusEnabled:        true,
			PrometheusPath:           "/metrics",
			HealthCheckPath:          "/health",
			MetricsCollectionInterval: 30 * time.Second,
			AlertingEnabled:          false,
			AlertManagerURL:          "",
		},
		Logging: config.LoggingConfig{
			Level:      "debug",
			Format:     "console",
			Output:     "stdout",
		},
		Container: config.ContainerConfig{
			Runtime:                "containerd",
			Socket:                 "/run/containerd/containerd.sock",
			NetworkDriver:          "bridge",
			StorageDriver:          "overlay2",
			DefaultMemoryLimit:     "512m",
			DefaultCPULimit:        "0.5",
			DefaultPidsLimit:       100,
			DefaultNetworkBandwidth: "100m",
		},
		Scheduler: config.SchedulerConfig{
			Algorithm:           "round-robin",
			MaxConcurrentAgents: 10,
			MaxAgentsPerHost:    5,
			CheckInterval:       30 * time.Second,
			AutoScale:           false,
			ScaleUpThreshold:    80,
			ScaleDownThreshold:  20,
		},
		API: config.APIConfig{
			SwaggerEnabled: false,
			SwaggerPath:    "/swagger/",
			APIPrefix:      "/api/v1",
			EnableCaching:  false,
			CacheDuration:  5 * time.Minute,
		},
		Redis: config.RedisConfig{
			Host:         "localhost",
			Port:         6379,
			Password:     "",
			DB:           0,
			PoolSize:     10,
			MinIdleConns: 5,
		},
		Mode: "test",
		Debug: true,
	}

	// Create test logger
	logger := zap.NewNop()

	// Create server
	srv, err := server.NewServer(cfg, logger)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	// Test health endpoint
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	
	// Use server's handler
	srv.GetHandler().ServeHTTP(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("health endpoint: expected status 200, got %d", w.Code)
	}
	
	// Test metrics endpoint
	if cfg.Monitoring.PrometheusEnabled {
		req = httptest.NewRequest(http.MethodGet, "/metrics", nil)
		w = httptest.NewRecorder()
		srv.GetHandler().ServeHTTP(w, req)
		
		if w.Code != http.StatusOK {
			t.Errorf("metrics endpoint: expected status 200, got %d", w.Code)
		}
	}
}

// TestServerConfig tests server configuration
func TestServerConfig(t *testing.T) {
	// Test minimal config
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host:                    "localhost",
			Port:                    8080,
			Mode:                    "development",
			GracefulShutdownTimeout: 10 * time.Second,
		},
		Logging: config.LoggingConfig{
			Level:  "info",
			Format: "json",
			Output: "stdout",
		},
		Mode: "development",
		Debug: true,
	}

	logger := zap.NewNop()
	srv, err := server.NewServer(cfg, logger)
	if err != nil {
		t.Fatalf("failed to create server with minimal config: %v", err)
	}

	// Verify server config
	srvConfig := srv.GetConfig()
	if srvConfig.Server.Host != "localhost" {
		t.Errorf("expected host localhost, got %s", srvConfig.Server.Host)
	}
	if srvConfig.Server.Port != 8080 {
		t.Errorf("expected port 8080, got %d", srvConfig.Server.Port)
	}
	if srvConfig.Mode != "development" {
		t.Errorf("expected mode development, got %s", srvConfig.Mode)
	}
}

// TestSecurityConfig tests security configuration
func TestSecurityConfig(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 8080,
		},
		Security: config.SecurityConfig{
			JWTSecret:            "test-secret",
			JWTExpiry:            24,
			CORSEnabled:          true,
			CORSAllowedOrigins:   []string{"http://localhost:3000"},
			RateLimitEnabled:     true,
			RateLimitRequests:    100,
			RateLimitBurst:       20,
		},
		Mode: "test",
	}

	logger := zap.NewNop()
	srv, err := server.NewServer(cfg, logger)
	if err != nil {
		t.Fatalf("failed to create server with security config: %v", err)
	}

	// Verify security config
	srvConfig := srv.GetConfig()
	if srvConfig.Security.JWTSecret != "test-secret" {
		t.Errorf("expected JWT secret test-secret, got %s", srvConfig.Security.JWTSecret)
	}
	if !srvConfig.Security.CORSEnabled {
		t.Error("CORS should be enabled")
	}
	if !srvConfig.Security.RateLimitEnabled {
		t.Error("rate limiting should be enabled")
	}
}

// TestContainerConfig tests container runtime configuration
func TestContainerConfig(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 8080,
		},
		Container: config.ContainerConfig{
			Runtime:                "containerd",
			Socket:                 "/var/run/containerd/containerd.sock",
			NetworkDriver:          "bridge",
			StorageDriver:          "overlay2",
			DefaultMemoryLimit:     "1GB",
			DefaultCPULimit:        "1.0",
			DefaultPidsLimit:       1024,
			DefaultNetworkBandwidth: "1Gbps",
		},
		Mode: "test",
	}

	logger := zap.NewNop()
	srv, err := server.NewServer(cfg, logger)
	if err != nil {
		t.Fatalf("failed to create server with container config: %v", err)
	}

	// Verify container config
	srvConfig := srv.GetConfig()
	if srvConfig.Container.Runtime != "containerd" {
		t.Errorf("expected runtime containerd, got %s", srvConfig.Container.Runtime)
	}
	if srvConfig.Container.DefaultMemoryLimit != "1GB" {
		t.Errorf("expected default memory limit 1GB, got %s", srvConfig.Container.DefaultMemoryLimit)
	}
	if srvConfig.Container.DefaultCPULimit != "1.0" {
		t.Errorf("expected default CPU limit 1.0, got %s", srvConfig.Container.DefaultCPULimit)
	}
}