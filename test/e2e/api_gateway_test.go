package e2e

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthEndpoint(t *testing.T) {
	srv := newTestServer(t)
	handler := srv.GetHandler()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("GET /health: expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode health response: %v", err)
	}
	if _, ok := resp["version"]; !ok {
		t.Error("health response missing version field")
	}
}

func TestReadyEndpoint(t *testing.T) {
	srv := newTestServer(t)
	handler := srv.GetHandler()

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("GET /ready: expected 200, got %d", w.Code)
	}
}

func TestLiveEndpoint(t *testing.T) {
	srv := newTestServer(t)
	handler := srv.GetHandler()

	req := httptest.NewRequest(http.MethodGet, "/live", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("GET /live: expected 200, got %d", w.Code)
	}
}

func TestMetricsEndpoint(t *testing.T) {
	srv := newTestServer(t)
	handler := srv.GetHandler()

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("GET /metrics: expected 200, got %d", w.Code)
	}
}

func TestRootEndpoint(t *testing.T) {
	srv := newTestServer(t)
	handler := srv.GetHandler()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("GET /: expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode root response: %v", err)
	}
	endpoints, ok := resp["endpoints"].(map[string]interface{})
	if !ok {
		t.Fatal("root response missing endpoints map")
	}
	for _, key := range []string{"health", "ready", "live", "metrics", "agents"} {
		if _, exists := endpoints[key]; !exists {
			t.Errorf("endpoints map missing key: %s", key)
		}
	}
}

func TestHealthMethodNotAllowed(t *testing.T) {
	srv := newTestServer(t)
	handler := srv.GetHandler()

	req := httptest.NewRequest(http.MethodPost, "/health", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST /health: expected 405, got %d", w.Code)
	}
}
