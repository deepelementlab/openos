package smoke

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/agentos/aos/internal/config"
	"github.com/agentos/aos/internal/server"
	"go.uber.org/zap"
)

func newSmokeServer(t *testing.T) *server.Server {
	t.Helper()
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host:                    "127.0.0.1",
			Port:                    0,
			Mode:                    "test",
			GracefulShutdownTimeout: 2 * time.Second,
		},
		Mode: "test",
	}
	srv, err := server.NewServer(cfg, zap.NewNop())
	if err != nil {
		t.Fatalf("server failed to start: %v", err)
	}
	return srv
}

func TestServerStartup(t *testing.T) {
	srv := newSmokeServer(t)
	if srv == nil {
		t.Fatal("server is nil")
	}
	handler := srv.GetHandler()
	if handler == nil {
		t.Fatal("server handler is nil")
	}
}

func TestHealthEndpointResponds(t *testing.T) {
	srv := newSmokeServer(t)
	handler := srv.GetHandler()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("health endpoint: expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode health response: %v", err)
	}
	if _, ok := resp["status"]; !ok {
		t.Error("health response missing status field")
	}
}

func TestGracefulShutdown(t *testing.T) {
	srv := newSmokeServer(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		t.Errorf("graceful shutdown failed: %v", err)
	}
}
