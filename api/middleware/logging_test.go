package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestLoggingMiddleware_NewLoggingMiddleware(t *testing.T) {
	m := NewLoggingMiddleware(zap.NewNop())
	require.NotNil(t, m)
	assert.Equal(t, zap.NewNop(), m.logger)
}

func TestLoggingMiddleware_Handler_SuccessfulRequest(t *testing.T) {
	logger := zap.NewNop()
	m := NewLoggingMiddleware(logger)

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("hello"))
	})

	handler := m.Handler(inner)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/agents?limit=10", nil)
	req.Header.Set("User-Agent", "test-client")
	req.RemoteAddr = "192.168.1.1:12345"
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "hello", rec.Body.String())
}

func TestLoggingMiddleware_Handler_ErrorStatus(t *testing.T) {
	logger := zap.NewNop()
	m := NewLoggingMiddleware(logger)

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("error"))
	})

	handler := m.Handler(inner)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/agents", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestLoggingMiddleware_Handler_StatusForbidden(t *testing.T) {
	logger := zap.NewNop()
	m := NewLoggingMiddleware(logger)

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	})

	handler := m.Handler(inner)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestLoggingMiddleware_Handler_NoExplicitStatus(t *testing.T) {
	logger := zap.NewNop()
	m := NewLoggingMiddleware(logger)

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("implicit 200"))
	})

	handler := m.Handler(inner)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "implicit 200", rec.Body.String())
}

func TestLoggingMiddleware_Handler_MultipleWrites(t *testing.T) {
	logger := zap.NewNop()
	m := NewLoggingMiddleware(logger)

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("part1"))
		w.Write([]byte("part2"))
		w.Write([]byte("part3"))
	})

	handler := m.Handler(inner)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "part1part2part3", rec.Body.String())
}

func TestLoggingMiddleware_Handler_DifferentMethods(t *testing.T) {
	logger := zap.NewNop()
	m := NewLoggingMiddleware(logger)

	methods := []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			handler := m.Handler(inner)
			req := httptest.NewRequest(method, "/api/v1/test", nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			assert.Equal(t, http.StatusOK, rec.Code)
		})
	}
}

func TestResponseWriterWrapper_WriteHeaderOnce(t *testing.T) {
	rec := httptest.NewRecorder()
	wrapper := &responseWriterWrapper{ResponseWriter: rec, statusCode: http.StatusOK}

	wrapper.WriteHeader(http.StatusNotFound)
	assert.Equal(t, http.StatusNotFound, wrapper.statusCode)
	assert.True(t, wrapper.wroteHeader)

	// Second WriteHeader should be ignored
	wrapper.WriteHeader(http.StatusOK)
	assert.Equal(t, http.StatusNotFound, wrapper.statusCode)
}

func TestResponseWriterWrapper_WriteWithoutHeader(t *testing.T) {
	rec := httptest.NewRecorder()
	wrapper := &responseWriterWrapper{ResponseWriter: rec, statusCode: http.StatusOK}

	n, err := wrapper.Write([]byte("test"))
	assert.NoError(t, err)
	assert.Equal(t, 4, n)
	assert.True(t, wrapper.wroteHeader)
	assert.Equal(t, int64(4), wrapper.responseSize)
	assert.Equal(t, http.StatusOK, wrapper.statusCode)
}

func TestResponseWriterWrapper_WriteAfterHeader(t *testing.T) {
	rec := httptest.NewRecorder()
	wrapper := &responseWriterWrapper{ResponseWriter: rec, statusCode: http.StatusOK}

	wrapper.WriteHeader(http.StatusCreated)
	n, err := wrapper.Write([]byte("created"))
	assert.NoError(t, err)
	assert.Equal(t, 7, n)
	assert.Equal(t, int64(7), wrapper.responseSize)
	assert.Equal(t, http.StatusCreated, wrapper.statusCode)
}

func TestResponseWriterWrapper_MultipleWrites(t *testing.T) {
	rec := httptest.NewRecorder()
	wrapper := &responseWriterWrapper{ResponseWriter: rec, statusCode: http.StatusOK}

	wrapper.Write([]byte("abc"))
	wrapper.Write([]byte("def"))
	wrapper.Write([]byte("ghi"))

	assert.Equal(t, int64(9), wrapper.responseSize)
}

func TestLoggingMiddleware_Handler_WithQuery(t *testing.T) {
	logger := zap.NewNop()
	m := NewLoggingMiddleware(logger)

	called := false
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	handler := m.Handler(inner)

	req := httptest.NewRequest(http.MethodGet, "/search?q=test&limit=20", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.True(t, called)
}
