package opa

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	c := NewClient("http://localhost:8181")
	assert.Equal(t, "http://localhost:8181", c.baseURL)
	assert.Equal(t, 5*time.Second, c.timeout)
	assert.NotNil(t, c.httpClient)
}

func TestClient_Evaluate_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Contains(t, r.URL.Path, "policy/allow")

		var reqBody EvaluateRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&reqBody))
		assert.Equal(t, "alice", reqBody.Input["user"])

		resp := EvaluateResponse{Result: true}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	result, err := c.Evaluate(context.Background(), "policy/allow", map[string]interface{}{
		"user": "alice",
	})
	require.NoError(t, err)
	assert.True(t, result.Result)
}

func TestClient_Evaluate_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	_, err := c.Evaluate(context.Background(), "policy/allow", map[string]interface{}{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected status code: 500")
}

func TestClient_Evaluate_ConnectionError(t *testing.T) {
	c := NewClient("http://127.0.0.1:0")
	_, err := c.Evaluate(context.Background(), "policy/allow", map[string]interface{}{})
	assert.Error(t, err)
}

func TestClient_QueryData_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Contains(t, r.URL.Path, "policies")

		result := map[string]interface{}{"count": float64(5)}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	result, err := c.QueryData(context.Background(), "policies")
	require.NoError(t, err)
	assert.Equal(t, float64(5), result["count"])
}

func TestClient_QueryData_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	_, err := c.QueryData(context.Background(), "nonexistent")
	assert.Error(t, err)
}

func TestClient_QueryData_ConnectionError(t *testing.T) {
	c := NewClient("http://127.0.0.1:0")
	_, err := c.QueryData(context.Background(), "policies")
	assert.Error(t, err)
}

func TestPolicyEvaluator_Evaluate_NoCache(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := EvaluateResponse{Result: true}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	pe := NewPolicyEvaluator(NewClient(srv.URL))
	result, err := pe.Evaluate(context.Background(), "policy/allow", map[string]interface{}{"user": "bob"}, false)
	require.NoError(t, err)
	assert.True(t, result)
}

func TestPolicyEvaluator_Evaluate_WithCache(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		resp := EvaluateResponse{Result: true}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	pe := NewPolicyEvaluator(NewClient(srv.URL))

	result, err := pe.Evaluate(context.Background(), "policy/allow", map[string]interface{}{"user": "alice"}, true)
	require.NoError(t, err)
	assert.True(t, result)

	result, err = pe.Evaluate(context.Background(), "policy/allow", map[string]interface{}{"user": "alice"}, true)
	require.NoError(t, err)
	assert.True(t, result)

	assert.Equal(t, 1, callCount, "second call should hit cache")
}

func TestPolicyEvaluator_ClearCache(t *testing.T) {
	pe := NewPolicyEvaluator(NewClient("http://localhost:8181"))
	pe.cache["test"] = &cacheEntry{result: &EvaluateResponse{Result: true}, cachedAt: time.Now()}
	pe.ClearCache()
	assert.Empty(t, pe.cache)
}

func TestGenerateCacheKey(t *testing.T) {
	key1 := generateCacheKey("policy/a", map[string]interface{}{"user": "alice"})
	key2 := generateCacheKey("policy/a", map[string]interface{}{"user": "bob"})
	key3 := generateCacheKey("policy/a", map[string]interface{}{"user": "alice"})

	assert.Equal(t, key1, key3)
	assert.NotEqual(t, key1, key2)
}

func TestEvaluateRequest_Marshal(t *testing.T) {
	req := EvaluateRequest{Input: map[string]interface{}{"action": "read"}}
	data, err := json.Marshal(req)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"action":"read"`)
}

func TestEvaluateResponse_Unmarshal(t *testing.T) {
	body := `{"result": true, "decision": {"reason": "allowed"}}`
	var resp EvaluateResponse
	require.NoError(t, json.Unmarshal([]byte(body), &resp))
	assert.True(t, resp.Result)
}

func TestClient_URLConstruction(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/data/my/policy/path", r.URL.Path)
		resp := EvaluateResponse{Result: false}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	result, err := c.Evaluate(context.Background(), "my/policy/path", map[string]interface{}{})
	require.NoError(t, err)
	assert.False(t, result.Result)
}

func TestClient_CancelledContext(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := EvaluateResponse{Result: true}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := c.Evaluate(ctx, "policy/allow", map[string]interface{}{})
	assert.Error(t, err)
}
