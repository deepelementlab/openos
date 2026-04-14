package gateway

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"go.uber.org/zap"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

// Gateway provides HTTP REST API by proxying to gRPC services
type Gateway struct {
	logger     *zap.Logger
	httpServer *http.Server
	mux        *runtime.ServeMux
	grpcAddr   string
	httpAddr   string
}

// Config holds gateway configuration
type Config struct {
	GRPCAddress string
	HTTPAddress string
	TLSEnabled  bool
	CertFile    string
	KeyFile     string
}

// NewGateway creates a new gRPC-Gateway instance
func NewGateway(config Config, logger *zap.Logger) (*Gateway, error) {
	// Create a new ServeMux with custom options
	mux := runtime.NewServeMux(
		runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.JSONPb{}),
		runtime.WithErrorHandler(customErrorHandler),
		runtime.WithMetadata(customMetadataHandler),
	)

	return &Gateway{
		logger:   logger,
		mux:      mux,
		grpcAddr: config.GRPCAddress,
		httpAddr: config.HTTPAddress,
	}, nil
}

// RegisterServices registers all gRPC services with the gateway.
//
// Full REST mapping requires HTTP annotations in .proto and code generated with
// protoc-gen-grpc-gateway (see api/proto/README.md). Until that is wired, this
// is a no-op so the control plane compiles; use gRPC clients or the JSON shim in api/grpc.
func (g *Gateway) RegisterServices(ctx context.Context) error {
	g.logger.Info("gRPC-Gateway: HTTP handler registration skipped (generate gateway stubs from annotated protos to enable)")
	return nil
}

// Start starts the HTTP gateway server
func (g *Gateway) Start() error {
	g.httpServer = &http.Server{
		Addr:    g.httpAddr,
		Handler: g.mux,
	}

	g.logger.Info("starting gRPC-Gateway", zap.String("addr", g.httpAddr))
	if err := g.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to serve gateway: %w", err)
	}

	return nil
}

// Stop gracefully stops the gateway
func (g *Gateway) Stop(ctx context.Context) error {
	g.logger.Info("stopping gRPC-Gateway")
	return g.httpServer.Shutdown(ctx)
}

// Handler returns the HTTP handler for integration with other servers
func (g *Gateway) Handler() http.Handler {
	return g.mux
}

// customErrorHandler provides custom error handling for the gateway
func customErrorHandler(ctx context.Context, mux *runtime.ServeMux, marshaler runtime.Marshaler, w http.ResponseWriter, r *http.Request, err error) {
	const fallback = `{"error": "failed to process request"}`

	w.Header().Set("Content-type", marshaler.ContentType(nil))

	st, _ := status.FromError(err)
	grpcCode := runtime.HTTPStatusFromCode(st.Code())
	w.WriteHeader(grpcCode)

	errorBody := map[string]interface{}{
		"error":   st.Code().String(),
		"message": st.Message(),
		"code":    grpcCode,
	}

	buf, merr := marshaler.Marshal(errorBody)
	if merr != nil {
		w.Write([]byte(fallback))
		return
	}

	w.Write(buf)
}

// customMetadataHandler extracts metadata from HTTP headers
func customMetadataHandler(ctx context.Context, r *http.Request) metadata.MD {
	var pairs []string
	if tenantID := r.Header.Get("X-Tenant-ID"); tenantID != "" {
		pairs = append(pairs, "tenant_id", tenantID)
	}
	if auth := r.Header.Get("Authorization"); auth != "" {
		pairs = append(pairs, "authorization", auth)
	}
	if len(pairs) == 0 {
		return nil
	}
	return metadata.Pairs(pairs...)
}

// ResponseModifier modifies responses for consistent formatting
func ResponseModifier(ctx context.Context, w http.ResponseWriter, p proto.Message) error {
	// Add custom headers if needed
	w.Header().Set("X-API-Version", "v1")
	return nil
}

// RequestValidator validates incoming requests
func RequestValidator(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip validation for health checks
		if r.URL.Path == "/health" || r.URL.Path == "/ready" {
			h.ServeHTTP(w, r)
			return
		}

		// Add request validation logic here
		h.ServeHTTP(w, r)
	})
}

// CORSHandler adds CORS headers
func CORSHandler(h http.Handler, allowedOrigins []string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		// Check if origin is allowed
		allowed := false
		for _, o := range allowedOrigins {
			if o == "*" || o == origin {
				allowed = true
				break
			}
		}

		if allowed {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Tenant-ID, X-Request-ID")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}

		// Handle preflight
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		h.ServeHTTP(w, r)
	})
}

// LoggingMiddleware logs HTTP requests
func LoggingMiddleware(logger *zap.Logger) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Create response wrapper to capture status code
			wrapped := &responseRecorder{ResponseWriter: w, statusCode: http.StatusOK}

			h.ServeHTTP(wrapped, r)

			duration := time.Since(start)

			logger.Info("HTTP request",
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.Int("status", wrapped.statusCode),
				zap.Duration("duration", duration),
				zap.String("remote_addr", r.RemoteAddr),
			)
		})
	}
}

type responseRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (r *responseRecorder) WriteHeader(code int) {
	r.statusCode = code
	r.ResponseWriter.WriteHeader(code)
}

// Note: This file uses grpc-gateway which needs to be added to go.mod:
// github.com/grpc-ecosystem/grpc-gateway/v2 v2.19.0
// This is optional and can be enabled if you want REST API bridge to gRPC
