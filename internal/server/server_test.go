package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/agentos/aos/internal/config"
	"go.uber.org/zap"
)

func TestCreateAgentAndList(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host:                    "127.0.0.1",
			Port:                    8080,
			Mode:                    "test",
			GracefulShutdownTimeout: 1,
		},
	}

	srv, err := NewServer(cfg, zap.NewNop())
	if err != nil {
		t.Fatalf("new server: %v", err)
	}

	createBody := `{"name":"demo-agent","image":"ghcr.io/agentos/demo:latest","runtime":"python"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/agents", bytes.NewBufferString(createBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.handleAgents(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("unexpected create status: %d body=%s", rec.Code, rec.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/agents", nil)
	listRec := httptest.NewRecorder()
	srv.handleAgents(listRec, listReq)

	if listRec.Code != http.StatusOK {
		t.Fatalf("unexpected list status: %d body=%s", listRec.Code, listRec.Body.String())
	}
	if !bytes.Contains(listRec.Body.Bytes(), []byte("demo-agent")) {
		t.Fatalf("expected created agent in list, got: %s", listRec.Body.String())
	}
}

func TestCreateAgentValidation(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host:                    "127.0.0.1",
			Port:                    8080,
			Mode:                    "test",
			GracefulShutdownTimeout: 1,
		},
	}
	srv, err := NewServer(cfg, zap.NewNop())
	if err != nil {
		t.Fatalf("new server: %v", err)
	}

	createBody := `{"name":"","image":""}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/agents", bytes.NewBufferString(createBody))
	rec := httptest.NewRecorder()
	srv.handleAgents(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected bad request, got %d", rec.Code)
	}
}


