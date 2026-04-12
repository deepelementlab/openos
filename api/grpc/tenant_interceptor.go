package grpc

import (
	"context"
	"fmt"

	"github.com/agentos/aos/internal/tenant"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// TenantInterceptor handles tenant context extraction and validation
type TenantInterceptor struct {
	logger          *zap.Logger
	enabled         bool
	tenantHeader    string
	requireTenant   bool
	tenantRepo      tenant.TenantRepository
}

// NewTenantInterceptor creates a new tenant interceptor
func NewTenantInterceptor(logger *zap.Logger) *TenantInterceptor {
	return &TenantInterceptor{
		logger:        logger,
		enabled:       true,
		tenantHeader:  "x-tenant-id",
		requireTenant: true,
	}
}

// WithTenantRepository sets the tenant repository
func (t *TenantInterceptor) WithTenantRepository(repo tenant.TenantRepository) *TenantInterceptor {
	t.tenantRepo = repo
	return t
}

// WithEnabled enables/disables tenant checking
func (t *TenantInterceptor) WithEnabled(enabled bool) *TenantInterceptor {
	t.enabled = enabled
	return t
}

// WithRequireTenant sets whether tenant is required
func (t *TenantInterceptor) WithRequireTenant(required bool) *TenantInterceptor {
	t.requireTenant = required
	return t
}

// UnaryInterceptor returns the unary interceptor
func (t *TenantInterceptor) UnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if !t.enabled {
			return handler(ctx, req)
		}

		// Skip tenant check for public methods
		if isPublicMethod(info.FullMethod) {
			return handler(ctx, req)
		}

		// Extract tenant ID
		tenantID, err := t.extractTenantID(ctx)
		if err != nil {
			if t.requireTenant {
				return nil, status.Error(codes.InvalidArgument, err.Error())
			}
			// Tenant not required, continue without tenant context
			return handler(ctx, req)
		}

		// Validate tenant if repository is available
		if t.tenantRepo != nil {
			if err := t.validateTenant(ctx, tenantID); err != nil {
				return nil, status.Error(codes.PermissionDenied, err.Error())
			}
		}

		// Add tenant to context
		ctx = tenant.WithTenant(ctx, tenantID)

		// Also add to auth claims if available
		if claims, ok := AuthClaimsFromContext(ctx); ok {
			claims.TenantID = tenantID
		}

		t.logger.Debug("tenant context established",
			zap.String("method", info.FullMethod),
			zap.String("tenant_id", tenantID),
		)

		return handler(ctx, req)
	}
}

// StreamInterceptor returns the stream interceptor
func (t *TenantInterceptor) StreamInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if !t.enabled {
			return handler(srv, ss)
		}

		// Skip tenant check for public methods
		if isPublicMethod(info.FullMethod) {
			return handler(srv, ss)
		}

		// Extract tenant ID
		tenantID, err := t.extractTenantID(ss.Context())
		if err != nil {
			if t.requireTenant {
				return status.Error(codes.InvalidArgument, err.Error())
			}
			return handler(srv, ss)
		}

		// Validate tenant if repository is available
		if t.tenantRepo != nil {
			if err := t.validateTenant(ss.Context(), tenantID); err != nil {
				return status.Error(codes.PermissionDenied, err.Error())
			}
		}

		// Wrap stream with tenant context
		wrapped := &tenantWrappedStream{
			ServerStream: ss,
			ctx:          tenant.WithTenant(ss.Context(), tenantID),
		}

		return handler(srv, wrapped)
	}
}

// extractTenantID extracts tenant ID from context (metadata or auth claims)
func (t *TenantInterceptor) extractTenantID(ctx context.Context) (string, error) {
	// First try to get from auth claims
	if claims, ok := AuthClaimsFromContext(ctx); ok && claims.TenantID != "" {
		return claims.TenantID, nil
	}

	// Try to get from metadata
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", fmt.Errorf("missing metadata")
	}

	tenantIDs := md.Get(t.tenantHeader)
	if len(tenantIDs) == 0 {
		return "", fmt.Errorf("missing tenant ID")
	}

	return tenantIDs[0], nil
}

// validateTenant validates that the tenant exists and is active
func (t *TenantInterceptor) validateTenant(ctx context.Context, tenantID string) error {
	tenant, err := t.tenantRepo.Get(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("invalid tenant: %w", err)
	}

	if !tenant.IsActive() {
		return fmt.Errorf("tenant is not active (status: %s)", tenant.Status)
	}

	return nil
}

// tenantWrappedStream wraps ServerStream with modified context
type tenantWrappedStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *tenantWrappedStream) Context() context.Context {
	return w.ctx
}
