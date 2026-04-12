package grpc

import (
	"context"
	"fmt"
	"net"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

// ServerConfig holds gRPC server configuration
type ServerConfig struct {
	Host                  string        `yaml:"host"`
	Port                  int           `yaml:"port"`
	TLSEnabled            bool          `yaml:"tls_enabled"`
	CertFile              string        `yaml:"cert_file"`
	KeyFile               string        `yaml:"key_file"`
	MaxConnectionIdle     time.Duration `yaml:"max_connection_idle"`
	MaxConnectionAge      time.Duration `yaml:"max_connection_age"`
	MaxConnectionAgeGrace time.Duration `yaml:"max_connection_age_grace"`
	Time                  time.Duration `yaml:"time"`
	Timeout               time.Duration `yaml:"timeout"`
}

// DefaultServerConfig returns default server configuration
func DefaultServerConfig() ServerConfig {
	return ServerConfig{
		Host:                  "0.0.0.0",
		Port:                  9090,
		TLSEnabled:            false,
		MaxConnectionIdle:     time.Minute * 10,
		MaxConnectionAge:      time.Hour,
		MaxConnectionAgeGrace: time.Minute * 5,
		Time:                  time.Second * 30,
		Timeout:               time.Second * 20,
	}
}

// Server wraps the gRPC server
type Server struct {
	grpcServer *grpc.Server
	config     ServerConfig
	logger     *zap.Logger
	listener   net.Listener

	// Services
	agentService      AgentServiceServer
	tenantService     TenantServiceServer
	runtimeService    RuntimeServiceServer
	monitoringService MonitoringServiceServer
}

// ServerOption is a functional option for Server
type ServerOption func(*Server)

// WithAgentService sets the agent service
func WithAgentService(svc AgentServiceServer) ServerOption {
	return func(s *Server) {
		s.agentService = svc
	}
}

// WithTenantService sets the tenant service
func WithTenantService(svc TenantServiceServer) ServerOption {
	return func(s *Server) {
		s.tenantService = svc
	}
}

// WithRuntimeService sets the runtime service
func WithRuntimeService(svc RuntimeServiceServer) ServerOption {
	return func(s *Server) {
		s.runtimeService = svc
	}
}

// WithMonitoringService sets the monitoring service
func WithMonitoringService(svc MonitoringServiceServer) ServerOption {
	return func(s *Server) {
		s.monitoringService = svc
	}
}

// NewServer creates a new gRPC server
func NewServer(config ServerConfig, logger *zap.Logger, opts ...ServerOption) (*Server, error) {
	s := &Server{
		config: config,
		logger: logger,
	}

	// Apply options
	for _, opt := range opts {
		opt(s)
	}

	// Create listener
	addr := fmt.Sprintf("%s:%d", config.Host, config.Port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to create listener: %w", err)
	}
	s.listener = listener

	// Build interceptor chain
	chain := NewInterceptorChain(logger)
	chain.Add(NewRecoveryInterceptor(logger))
	chain.Add(NewLoggingInterceptor(logger))
	chain.Add(NewAuthInterceptor(logger))
	chain.Add(NewTenantInterceptor(logger))
	chain.Add(NewQuotaInterceptor(logger))
	chain.Add(NewMetricsInterceptor())

	// Create gRPC server options
	serverOpts := []grpc.ServerOption{
		grpc.ChainUnaryInterceptor(chain.UnaryInterceptors()...),
		grpc.ChainStreamInterceptor(chain.StreamInterceptors()...),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle:     config.MaxConnectionIdle,
			MaxConnectionAge:      config.MaxConnectionAge,
			MaxConnectionAgeGrace: config.MaxConnectionAgeGrace,
			Time:                  config.Time,
			Timeout:               config.Timeout,
		}),
	}

	// Configure TLS if enabled
	if config.TLSEnabled {
		creds, err := credentials.NewServerTLSFromFile(config.CertFile, config.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load TLS credentials: %w", err)
		}
		serverOpts = append(serverOpts, grpc.Creds(creds))
	}

	s.grpcServer = grpc.NewServer(serverOpts...)

	// Register health check service
	healthServer := health.NewServer()
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	grpc_health_v1.RegisterHealthServer(s.grpcServer, healthServer)

	// Register services
	if s.agentService != nil {
		RegisterAgentServiceServer(s.grpcServer, s.agentService)
		logger.Info("registered AgentService")
	}
	if s.tenantService != nil {
		RegisterTenantServiceServer(s.grpcServer, s.tenantService)
		logger.Info("registered TenantService")
	}
	if s.runtimeService != nil {
		RegisterRuntimeServiceServer(s.grpcServer, s.runtimeService)
		logger.Info("registered RuntimeService")
	}
	if s.monitoringService != nil {
		RegisterMonitoringServiceServer(s.grpcServer, s.monitoringService)
		logger.Info("registered MonitoringService")
	}

	// Register reflection service for debugging
	reflection.Register(s.grpcServer)

	logger.Info("gRPC server created",
		zap.String("addr", addr),
		zap.Bool("tls", config.TLSEnabled),
	)

	return s, nil
}

// Start starts the gRPC server
func (s *Server) Start() error {
	s.logger.Info("starting gRPC server", zap.String("addr", s.listener.Addr().String()))
	if err := s.grpcServer.Serve(s.listener); err != nil {
		return fmt.Errorf("failed to serve: %w", err)
	}
	return nil
}

// Stop gracefully stops the gRPC server
func (s *Server) Stop() {
	s.logger.Info("stopping gRPC server")
	s.grpcServer.GracefulStop()
}

// ForceStop force stops the gRPC server
func (s *Server) ForceStop() {
	s.logger.Info("force stopping gRPC server")
	s.grpcServer.Stop()
}

// InterceptorChain manages a chain of interceptors
type InterceptorChain struct {
	unaryInterceptors  []grpc.UnaryServerInterceptor
	streamInterceptors []grpc.StreamServerInterceptor
}

// NewInterceptorChain creates a new interceptor chain
func NewInterceptorChain(logger *zap.Logger) *InterceptorChain {
	return &InterceptorChain{
		unaryInterceptors:  make([]grpc.UnaryServerInterceptor, 0),
		streamInterceptors: make([]grpc.StreamServerInterceptor, 0),
	}
}

// Interceptor is a combined unary and stream interceptor interface
type Interceptor interface {
	UnaryInterceptor() grpc.UnaryServerInterceptor
	StreamInterceptor() grpc.StreamServerInterceptor
}

// Add adds an interceptor to the chain
func (c *InterceptorChain) Add(i Interceptor) {
	c.unaryInterceptors = append(c.unaryInterceptors, i.UnaryInterceptor())
	c.streamInterceptors = append(c.streamInterceptors, i.StreamInterceptor())
}

// UnaryInterceptors returns all unary interceptors
func (c *InterceptorChain) UnaryInterceptors() []grpc.UnaryServerInterceptor {
	return c.unaryInterceptors
}

// StreamInterceptors returns all stream interceptors
func (c *InterceptorChain) StreamInterceptors() []grpc.StreamServerInterceptor {
	return c.streamInterceptors
}

// RecoveryInterceptor recovers from panics
type RecoveryInterceptor struct {
	logger *zap.Logger
}

// NewRecoveryInterceptor creates a new recovery interceptor
func NewRecoveryInterceptor(logger *zap.Logger) *RecoveryInterceptor {
	return &RecoveryInterceptor{logger: logger}
}

// UnaryInterceptor returns the unary interceptor
func (r *RecoveryInterceptor) UnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		defer func() {
			if rec := recover(); rec != nil {
				r.logger.Error("panic recovered in unary handler",
					zap.String("method", info.FullMethod),
					zap.Any("panic", rec),
				)
				err = fmt.Errorf("internal server error")
			}
		}()
		return handler(ctx, req)
	}
}

// StreamInterceptor returns the stream interceptor
func (r *RecoveryInterceptor) StreamInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		defer func() {
			if rec := recover(); rec != nil {
				r.logger.Error("panic recovered in stream handler",
					zap.String("method", info.FullMethod),
					zap.Any("panic", rec),
				)
			}
		}()
		return handler(srv, ss)
	}
}

// LoggingInterceptor logs requests and responses
type LoggingInterceptor struct {
	logger *zap.Logger
}

// NewLoggingInterceptor creates a new logging interceptor
func NewLoggingInterceptor(logger *zap.Logger) *LoggingInterceptor {
	return &LoggingInterceptor{logger: logger}
}

// UnaryInterceptor returns the unary interceptor
func (l *LoggingInterceptor) UnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()
		resp, err := handler(ctx, req)
		duration := time.Since(start)

		fields := []zap.Field{
			zap.String("method", info.FullMethod),
			zap.Duration("duration", duration),
		}

		if err != nil {
			fields = append(fields, zap.Error(err))
			l.logger.Warn("gRPC request failed", fields...)
		} else {
			l.logger.Debug("gRPC request completed", fields...)
		}

		return resp, err
	}
}

// StreamInterceptor returns the stream interceptor
func (l *LoggingInterceptor) StreamInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		l.logger.Debug("stream started", zap.String("method", info.FullMethod))
		err := handler(srv, ss)
		if err != nil {
			l.logger.Warn("stream ended with error", zap.String("method", info.FullMethod), zap.Error(err))
		} else {
			l.logger.Debug("stream completed", zap.String("method", info.FullMethod))
		}
		return err
	}
}
