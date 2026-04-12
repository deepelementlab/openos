package grpc

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

var (
	// gRPC request duration histogram
	grpcDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "aos_grpc_request_duration_seconds",
			Help:    "gRPC request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"service", "method", "status"},
	)

	// gRPC request counter
	grpcRequests = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "aos_grpc_requests_total",
			Help: "Total number of gRPC requests",
		},
		[]string{"service", "method", "status"},
	)

	// gRPC request in-flight gauge
	grpcInFlight = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "aos_grpc_requests_in_flight",
			Help: "Number of gRPC requests currently being served",
		},
		[]string{"service", "method"},
	)
)

// MetricsInterceptor collects Prometheus metrics
type MetricsInterceptor struct{}

// NewMetricsInterceptor creates a new metrics interceptor
func NewMetricsInterceptor() *MetricsInterceptor {
	return &MetricsInterceptor{}
}

// UnaryInterceptor returns the unary interceptor
func (m *MetricsInterceptor) UnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		service, method := parseMethod(info.FullMethod)

		// Increment in-flight gauge
		grpcInFlight.WithLabelValues(service, method).Inc()
		defer grpcInFlight.WithLabelValues(service, method).Dec()

		// Record start time
		start := time.Now()

		// Call handler
		resp, err := handler(ctx, req)

		// Calculate duration
		duration := time.Since(start).Seconds()

		// Get status code
		statusCode := "OK"
		if err != nil {
			statusCode = status.Code(err).String()
		}

		// Record metrics
		grpcDuration.WithLabelValues(service, method, statusCode).Observe(duration)
		grpcRequests.WithLabelValues(service, method, statusCode).Inc()

		return resp, err
	}
}

// StreamInterceptor returns the stream interceptor
func (m *MetricsInterceptor) StreamInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		service, method := parseMethod(info.FullMethod)

		// Increment in-flight gauge
		grpcInFlight.WithLabelValues(service, method).Inc()
		defer grpcInFlight.WithLabelValues(service, method).Dec()

		// Record start time
		start := time.Now()

		// Call handler
		err := handler(srv, ss)

		// Calculate duration
		duration := time.Since(start).Seconds()

		// Get status code
		statusCode := "OK"
		if err != nil {
			statusCode = status.Code(err).String()
		}

		// Record metrics
		grpcDuration.WithLabelValues(service, method, statusCode).Observe(duration)
		grpcRequests.WithLabelValues(service, method, statusCode).Inc()

		return err
	}
}

// parseMethod parses a gRPC full method name into service and method
func parseMethod(fullMethod string) (string, string) {
	// Full method format: /package.service/Method
	parts := splitMethod(fullMethod)
	if len(parts) >= 2 {
		return parts[0], parts[1]
	}
	return "unknown", "unknown"
}

// splitMethod splits a gRPC method path
func splitMethod(fullMethod string) []string {
	// Remove leading slash and split
	if len(fullMethod) > 0 && fullMethod[0] == '/' {
		fullMethod = fullMethod[1:]
	}

	var parts []string
	var current string
	for _, c := range fullMethod {
		if c == '/' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(c)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}
