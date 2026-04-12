package grpc

import (
	"context"
	"fmt"
	"strings"

	"github.com/agentos/aos/internal/tenant"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// QuotaInterceptor enforces tenant resource quotas
type QuotaInterceptor struct {
	logger       *zap.Logger
	enabled      bool
	quotaManager tenant.QuotaManager
	// Method -> resource type mapping
	resourceMapping map[string]string
}

// NewQuotaInterceptor creates a new quota interceptor
func NewQuotaInterceptor(logger *zap.Logger) *QuotaInterceptor {
	return &QuotaInterceptor{
		logger:  logger,
		enabled: true,
		resourceMapping: map[string]string{
			"/aos.api.v1.AgentService/CreateAgent": "agent",
			"/aos.api.v1.TenantService/CreateTenant": "tenant",
		},
	}
}

// WithQuotaManager sets the quota manager
func (q *QuotaInterceptor) WithQuotaManager(qm tenant.QuotaManager) *QuotaInterceptor {
	q.quotaManager = qm
	return q
}

// WithEnabled enables/disables quota checking
func (q *QuotaInterceptor) WithEnabled(enabled bool) *QuotaInterceptor {
	q.enabled = enabled
	return q
}
}

// UnaryInterceptor returns the unary interceptor
func (q *QuotaInterceptor) UnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if !q.enabled || q.quotaManager == nil {
			return handler(ctx, req)
		}

		// Skip quota check for methods that don't consume resources
		resourceType := q.getResourceType(info.FullMethod)
		if resourceType == "" {
			return handler(ctx, req)
		}

		// Get tenant ID from context
		tenantID, ok := tenant.TenantFromContext(ctx)
		if !ok {
			// No tenant context, skip quota check
			return handler(ctx, req)
		}

		// Check if this is a create operation that consumes quota
		if !q.isCreateOperation(info.FullMethod) {
			return handler(ctx, req)
		}

		// Check quota
		if err := q.checkQuota(ctx, tenantID, resourceType); err != nil {
			q.logger.Warn("quota exceeded",
				zap.String("tenant_id", tenantID),
				zap.String("resource_type", resourceType),
				zap.String("method", info.FullMethod),
			)
			return nil, status.Error(codes.ResourceExhausted, err.Error())
		}

		// Call handler
		resp, err := handler(ctx, req)

		// If successful, increment usage
		if err == nil {
			if incrementErr := q.incrementUsage(ctx, tenantID, resourceType); incrementErr != nil {
				q.logger.Error("failed to increment quota usage",
					zap.String("tenant_id", tenantID),
					zap.String("resource_type", resourceType),
					zap.Error(incrementErr),
				)
			}
		}

		return resp, err
	}
}

// StreamInterceptor returns the stream interceptor
func (q *QuotaInterceptor) StreamInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		// Stream operations typically don't consume resources in the same way
		// but we could add rate limiting here if needed
		return handler(srv, ss)
	}
}

// getResourceType returns the resource type for a given method
func (q *QuotaInterceptor) getResourceType(method string) string {
	for pattern, resourceType := range q.resourceMapping {
		if strings.HasPrefix(method, pattern) {
			return resourceType
		}
	}
	return ""
}

// isCreateOperation checks if the method is a create operation
func (q *QuotaInterceptor) isCreateOperation(method string) bool {
	return strings.Contains(method, "Create") || strings.Contains(method, "Pull")
}

// checkQuota checks if the tenant has quota available
func (q *QuotaInterceptor) checkQuota(ctx context.Context, tenantID, resourceType string) error {
	switch resourceType {
	case "agent":
		return q.quotaManager.CheckAgentQuota(ctx, tenantID)
	case "tenant":
		// System-level operation, typically not quota-limited
		return nil
	default:
		return fmt.Errorf("unknown resource type: %s", resourceType)
	}
}

// incrementUsage increments the quota usage for the tenant
func (q *QuotaInterceptor) incrementUsage(ctx context.Context, tenantID, resourceType string) error {
	switch resourceType {
	case "agent":
		return q.quotaManager.IncrementAgentUsage(ctx, tenantID)
	default:
		return nil
	}
}
