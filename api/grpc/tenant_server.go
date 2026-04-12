package grpc

import (
	"context"
	"fmt"
	"time"

	"github.com/agentos/aos/internal/tenant"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TenantServiceServer implements the gRPC TenantService
type TenantServiceServer struct {
	logger       *zap.Logger
	tenantRepo   tenant.TenantRepository
	quotaManager tenant.QuotaManager
}

// NewTenantServiceServer creates a new tenant service server
func NewTenantServiceServer(logger *zap.Logger, tenantRepo tenant.TenantRepository, quotaManager tenant.QuotaManager) *TenantServiceServer {
	return &TenantServiceServer{
		logger:       logger,
		tenantRepo:   tenantRepo,
		quotaManager: quotaManager,
	}
}

// CreateTenant creates a new tenant
func (s *TenantServiceServer) CreateTenant(ctx context.Context, req *CreateTenantRequest) (*Tenant, error) {
	logger := s.logger.With(zap.String("method", "CreateTenant"))

	// Validate request
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "tenant name is required")
	}

	// Check if user has permission to create tenants
	// This would typically be done via RBAC
	if claims, ok := AuthClaimsFromContext(ctx); ok {
		logger = logger.With(zap.String("user_id", claims.UserID))
	}

	// Create tenant model
	t := &tenant.Tenant{
		ID:        uuid.New().String(),
		Name:      req.Name,
		Status:    tenant.TenantActive,
		Plan:      req.Plan,
		Labels:    req.Labels,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Set default quota if not provided
	if req.Quota != nil {
		t.Quota = tenant.ResourceQuota{
			MaxAgents:   int(req.Quota.MaxAgents),
			MaxCPU:      int(req.Quota.MaxCpuCores),
			MaxMemoryGB: int(req.Quota.MaxMemoryGb),
		}
	} else {
		// Set default quota based on plan
		t.Quota = s.getDefaultQuotaForPlan(req.Plan)
	}

	// Save to repository
	if err := s.tenantRepo.Create(ctx, t); err != nil {
		logger.Error("failed to create tenant", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to create tenant")
	}

	// Add owner as member
	if req.OwnerEmail != "" {
		owner := &tenant.TenantMember{
			TenantID: t.ID,
			UserID:   req.OwnerEmail, // Using email as user ID for now
			Email:    req.OwnerEmail,
			Role:     "owner",
		}
		if err := s.tenantRepo.AddMember(ctx, owner); err != nil {
			logger.Warn("failed to add owner as member", zap.Error(err))
		}
	}

	logger.Info("tenant created", zap.String("tenant_id", t.ID))

	return s.tenantToProto(t), nil
}

// GetTenant retrieves a tenant by ID
func (s *TenantServiceServer) GetTenant(ctx context.Context, req *GetTenantRequest) (*Tenant, error) {
	logger := s.logger.With(zap.String("method", "GetTenant"), zap.String("tenant_id", req.TenantId))

	// Get tenant from repository
	t, err := s.tenantRepo.Get(ctx, req.TenantId)
	if err != nil {
		logger.Error("failed to get tenant", zap.Error(err))
		return nil, status.Error(codes.NotFound, "tenant not found")
	}

	// Check if user has access to this tenant
	if tenantID, ok := tenant.TenantFromContext(ctx); ok {
		if tenantID != req.TenantId {
			// Check if user is a member
			members, err := s.tenantRepo.GetMembers(ctx, req.TenantId)
			if err != nil {
				return nil, status.Error(codes.PermissionDenied, "access denied")
			}

			hasAccess := false
			if claims, ok := AuthClaimsFromContext(ctx); ok {
				for _, m := range members {
					if m.UserID == claims.UserID {
						hasAccess = true
						break
					}
				}
			}

			if !hasAccess {
				return nil, status.Error(codes.PermissionDenied, "access denied")
			}
		}
	}

	return s.tenantToProto(t), nil
}

// ListTenants lists all tenants (admin only)
func (s *TenantServiceServer) ListTenants(ctx context.Context, req *ListTenantsRequest) (*ListTenantsResponse, error) {
	logger := s.logger.With(zap.String("method", "ListTenants"))

	// Check admin permission
	if claims, ok := AuthClaimsFromContext(ctx); ok {
		if !hasAdminRole(claims.Roles) {
			return nil, status.Error(codes.PermissionDenied, "admin access required")
		}
		logger = logger.With(zap.String("user_id", claims.UserID))
	} else {
		return nil, status.Error(codes.Unauthenticated, "authentication required")
	}

	// Get all tenants
	tenants, err := s.tenantRepo.List(ctx)
	if err != nil {
		logger.Error("failed to list tenants", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to list tenants")
	}

	// Apply status filter
	var filtered []*tenant.Tenant
	if req.Status != TenantStatus_TENANT_STATUS_UNSPECIFIED {
		for _, t := range tenants {
			if string(t.Status) == req.Status.String() {
				filtered = append(filtered, t)
			}
		}
	} else {
		filtered = tenants
	}

	// Apply plan filter
	if req.Plan != "" {
		var planFiltered []*tenant.Tenant
		for _, t := range filtered {
			if t.Plan == req.Plan {
				planFiltered = append(planFiltered, t)
			}
		}
		filtered = planFiltered
	}

	// Calculate pagination
	total := len(filtered)
	page := int(req.Pagination.Page)
	if page < 1 {
		page = 1
	}
	pageSize := int(req.Pagination.PageSize)
	if pageSize < 1 {
		pageSize = 20
	}

	start := (page - 1) * pageSize
	end := start + pageSize
	if start > total {
		start = total
		end = total
	}
	if end > total {
		end = total
	}

	paginated := filtered[start:end]

	// Convert to proto
	protoTenants := make([]*Tenant, len(paginated))
	for i, t := range paginated {
		protoTenants[i] = s.tenantToProto(t)
	}

	pages := (total + pageSize - 1) / pageSize
	if pages == 0 {
		pages = 1
	}

	return &ListTenantsResponse{
		Tenants: protoTenants,
		Pagination: &PaginationResponse{
			Page:     int32(page),
			PageSize: int32(pageSize),
			Total:    int32(total),
			Pages:    int32(pages),
		},
	}, nil
}

// UpdateTenant updates an existing tenant
func (s *TenantServiceServer) UpdateTenant(ctx context.Context, req *UpdateTenantRequest) (*Tenant, error) {
	logger := s.logger.With(zap.String("method", "UpdateTenant"), zap.String("tenant_id", req.TenantId))

	// Get existing tenant
	t, err := s.tenantRepo.Get(ctx, req.TenantId)
	if err != nil {
		logger.Error("failed to get tenant", zap.Error(err))
		return nil, status.Error(codes.NotFound, "tenant not found")
	}

	// Check permission - only admin or tenant owner can update
	if !s.hasTenantUpdatePermission(ctx, t) {
		return nil, status.Error(codes.PermissionDenied, "access denied")
	}

	// Update fields
	if req.Name != "" {
		t.Name = req.Name
	}
	if req.Plan != "" {
		t.Plan = req.Plan
	}
	if req.Labels != nil {
		t.Labels = req.Labels
	}

	t.UpdatedAt = time.Now()

	// Save changes
	if err := s.tenantRepo.Update(ctx, t); err != nil {
		logger.Error("failed to update tenant", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to update tenant")
	}

	logger.Info("tenant updated")

	return s.tenantToProto(t), nil
}

// DeleteTenant deletes a tenant
func (s *TenantServiceServer) DeleteTenant(ctx context.Context, req *DeleteTenantRequest) (*Empty, error) {
	logger := s.logger.With(zap.String("method", "DeleteTenant"), zap.String("tenant_id", req.TenantId))

	// Get existing tenant
	t, err := s.tenantRepo.Get(ctx, req.TenantId)
	if err != nil {
		logger.Error("failed to get tenant", zap.Error(err))
		return nil, status.Error(codes.NotFound, "tenant not found")
	}

	// Check permission - only admin can delete tenants
	if claims, ok := AuthClaimsFromContext(ctx); ok {
		if !hasAdminRole(claims.Roles) {
			return nil, status.Error(codes.PermissionDenied, "admin access required")
		}
	} else {
		return nil, status.Error(codes.Unauthenticated, "authentication required")
	}

	// Check if tenant has resources
	if !req.Force {
		usage, err := s.quotaManager.GetUsage(ctx, req.TenantId)
		if err == nil && usage.AgentsCount > 0 {
			return nil, status.Error(codes.FailedPrecondition,
				fmt.Sprintf("tenant has %d active resources. Use force=true to delete anyway.", usage.AgentsCount))
		}
	}

	// Delete tenant
	if err := s.tenantRepo.Delete(ctx, req.TenantId); err != nil {
		logger.Error("failed to delete tenant", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to delete tenant")
	}

	logger.Info("tenant deleted")

	return &Empty{}, nil
}

// GetTenantQuota retrieves tenant quota
func (s *TenantServiceServer) GetTenantQuota(ctx context.Context, req *GetTenantQuotaRequest) (*TenantQuota, error) {
	logger := s.logger.With(zap.String("method", "GetTenantQuota"), zap.String("tenant_id", req.TenantId))

	// Get tenant
	t, err := s.tenantRepo.Get(ctx, req.TenantId)
	if err != nil {
		logger.Error("failed to get tenant", zap.Error(err))
		return nil, status.Error(codes.NotFound, "tenant not found")
	}

	// Check permission
	if !s.hasTenantAccess(ctx, t) {
		return nil, status.Error(codes.PermissionDenied, "access denied")
	}

	// Get current usage
	usage, err := s.quotaManager.GetUsage(ctx, req.TenantId)
	if err != nil {
		logger.Warn("failed to get usage", zap.Error(err))
		usage = &tenant.ResourceUsage{}
	}

	// Calculate overall usage percentage
	var overallPercent float64
	agentsPercent := calculatePercent(usage.AgentsCount, t.Quota.MaxAgents)
	cpuPercent := calculatePercent(usage.CPUCoresUsed, t.Quota.MaxCPU)
	memPercent := calculatePercent(usage.MemoryGBUsed, t.Quota.MaxMemoryGB)

	overallPercent = max(agentsPercent, cpuPercent, memPercent)

	return &TenantQuota{
		TenantId: req.TenantId,
		Allowed: &ResourceQuota{
			MaxAgents:     int32(t.Quota.MaxAgents),
			MaxCpuCores:   int32(t.Quota.MaxCPU),
			MaxMemoryGb:   int32(t.Quota.MaxMemoryGB),
			MaxStorageGb:  int32(t.Quota.MaxStorageGB),
			MaxGpu:        int32(t.Quota.MaxGPU),
		},
		Used: &ResourceUsage{
			AgentsCount:   int32(usage.AgentsCount),
			CpuCoresUsed:  int32(usage.CPUCoresUsed),
			MemoryGbUsed:  int32(usage.MemoryGBUsed),
			StorageGbUsed: int32(usage.StorageGBUsed),
			GpuUsed:       int32(usage.GPUUsed),
		},
		UsagePercent: overallPercent,
	}, nil
}

// UpdateTenantQuota updates tenant quota (admin only)
func (s *TenantServiceServer) UpdateTenantQuota(ctx context.Context, req *UpdateTenantQuotaRequest) (*TenantQuota, error) {
	logger := s.logger.With(zap.String("method", "UpdateTenantQuota"), zap.String("tenant_id", req.TenantId))

	// Check admin permission
	if claims, ok := AuthClaimsFromContext(ctx); ok {
		if !hasAdminRole(claims.Roles) {
			return nil, status.Error(codes.PermissionDenied, "admin access required")
		}
	} else {
		return nil, status.Error(codes.Unauthenticated, "authentication required")
	}

	// Get tenant
	t, err := s.tenantRepo.Get(ctx, req.TenantId)
	if err != nil {
		logger.Error("failed to get tenant", zap.Error(err))
		return nil, status.Error(codes.NotFound, "tenant not found")
	}

	// Update quota
	if req.Quota != nil {
		t.Quota = tenant.ResourceQuota{
			MaxAgents:   int(req.Quota.MaxAgents),
			MaxCPU:      int(req.Quota.MaxCpuCores),
			MaxMemoryGB: int(req.Quota.MaxMemoryGb),
			MaxStorageGB: int(req.Quota.MaxStorageGb),
			MaxGPU:      int(req.Quota.MaxGpu),
		}
	}

	t.UpdatedAt = time.Now()

	// Save changes
	if err := s.tenantRepo.Update(ctx, t); err != nil {
		logger.Error("failed to update tenant quota", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to update quota")
	}

	logger.Info("tenant quota updated")

	// Return updated quota
	return s.GetTenantQuota(ctx, &GetTenantQuotaRequest{TenantId: req.TenantId})
}

// GetTenantUsage retrieves tenant resource usage
func (s *TenantServiceServer) GetTenantUsage(ctx context.Context, req *GetTenantUsageRequest) (*TenantUsage, error) {
	logger := s.logger.With(zap.String("method", "GetTenantUsage"), zap.String("tenant_id", req.TenantId))

	// Check access
	t, err := s.tenantRepo.Get(ctx, req.TenantId)
	if err != nil {
		logger.Error("failed to get tenant", zap.Error(err))
		return nil, status.Error(codes.NotFound, "tenant not found")
	}

	if !s.hasTenantAccess(ctx, t) {
		return nil, status.Error(codes.PermissionDenied, "access denied")
	}

	// Get usage from quota manager
	usage, err := s.quotaManager.GetUsage(ctx, req.TenantId)
	if err != nil {
		logger.Error("failed to get usage", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to get usage")
	}

	return &TenantUsage{
		TenantId:        req.TenantId,
		TotalAgents:     int32(usage.AgentsCount),
		Resources: &ResourceUsage{
			AgentsCount:   int32(usage.AgentsCount),
			CpuCoresUsed:  int32(usage.CPUCoresUsed),
			MemoryGbUsed:  int32(usage.MemoryGBUsed),
		},
	}, nil
}

// AddTenantMember adds a member to a tenant
func (s *TenantServiceServer) AddTenantMember(ctx context.Context, req *AddTenantMemberRequest) (*TenantMember, error) {
	logger := s.logger.With(zap.String("method", "AddTenantMember"), zap.String("tenant_id", req.TenantId))

	// Check permission
	t, err := s.tenantRepo.Get(ctx, req.TenantId)
	if err != nil {
		logger.Error("failed to get tenant", zap.Error(err))
		return nil, status.Error(codes.NotFound, "tenant not found")
	}

	if !s.hasTenantAdminPermission(ctx, t) {
		return nil, status.Error(codes.PermissionDenied, "admin access required")
	}

	// Create member
	member := &tenant.TenantMember{
		TenantID: req.TenantId,
		UserID:   req.Email, // Using email as user ID
		Email:    req.Email,
		Role:     req.Role.String(),
	}

	if claims, ok := AuthClaimsFromContext(ctx); ok {
		member.UserID = claims.UserID
	}

	if err := s.tenantRepo.AddMember(ctx, member); err != nil {
		logger.Error("failed to add member", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to add member")
	}

	logger.Info("member added", zap.String("email", req.Email))

	return &TenantMember{
		TenantId: req.TenantId,
		UserId:   member.UserID,
		Email:    member.Email,
		Role:     MemberRole(MemberRole_value[member.Role]),
	}, nil
}

// RemoveTenantMember removes a member from a tenant
func (s *TenantServiceServer) RemoveTenantMember(ctx context.Context, req *RemoveTenantMemberRequest) (*Empty, error) {
	logger := s.logger.With(zap.String("method", "RemoveTenantMember"), zap.String("tenant_id", req.TenantId))

	// Check permission
	t, err := s.tenantRepo.Get(ctx, req.TenantId)
	if err != nil {
		logger.Error("failed to get tenant", zap.Error(err))
		return nil, status.Error(codes.NotFound, "tenant not found")
	}

	if !s.hasTenantAdminPermission(ctx, t) {
		return nil, status.Error(codes.PermissionDenied, "admin access required")
	}

	// Cannot remove owner
	members, _ := s.tenantRepo.GetMembers(ctx, req.TenantId)
	for _, m := range members {
		if m.UserID == req.UserId && m.Role == "owner" {
			return nil, status.Error(codes.FailedPrecondition, "cannot remove owner")
		}
	}

	if err := s.tenantRepo.RemoveMember(ctx, req.TenantId, req.UserId); err != nil {
		logger.Error("failed to remove member", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to remove member")
	}

	logger.Info("member removed", zap.String("user_id", req.UserId))

	return &Empty{}, nil
}

// ListTenantMembers lists tenant members
func (s *TenantServiceServer) ListTenantMembers(ctx context.Context, req *ListTenantMembersRequest) (*ListTenantMembersResponse, error) {
	logger := s.logger.With(zap.String("method", "ListTenantMembers"), zap.String("tenant_id", req.TenantId))

	// Check access
	t, err := s.tenantRepo.Get(ctx, req.TenantId)
	if err != nil {
		logger.Error("failed to get tenant", zap.Error(err))
		return nil, status.Error(codes.NotFound, "tenant not found")
	}

	if !s.hasTenantAccess(ctx, t) {
		return nil, status.Error(codes.PermissionDenied, "access denied")
	}

	// Get members
	members, err := s.tenantRepo.GetMembers(ctx, req.TenantId)
	if err != nil {
		logger.Error("failed to get members", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to get members")
	}

	// Apply role filter
	var filtered []*tenant.TenantMember
	if req.Role != MemberRole_MEMBER_ROLE_UNSPECIFIED {
		for _, m := range members {
			if m.Role == req.Role.String() {
				filtered = append(filtered, m)
			}
		}
	} else {
		filtered = members
	}

	// Convert to proto
	protoMembers := make([]*TenantMember, len(filtered))
	for i, m := range filtered {
		protoMembers[i] = &TenantMember{
			TenantId: m.TenantID,
			UserId:   m.UserID,
			Email:    m.Email,
			Name:     m.Name,
			Role:     MemberRole(MemberRole_value[m.Role]),
		}
	}

	return &ListTenantMembersResponse{
		TenantId: req.TenantId,
		Members:  protoMembers,
	}, nil
}

// UpdateTenantMember updates a member's role
func (s *TenantServiceServer) UpdateTenantMember(ctx context.Context, req *UpdateTenantMemberRequest) (*TenantMember, error) {
	logger := s.logger.With(zap.String("method", "UpdateTenantMember"))

	// Check permission
	t, err := s.tenantRepo.Get(ctx, req.TenantId)
	if err != nil {
		logger.Error("failed to get tenant", zap.Error(err))
		return nil, status.Error(codes.NotFound, "tenant not found")
	}

	if !s.hasTenantAdminPermission(ctx, t) {
		return nil, status.Error(codes.PermissionDenied, "admin access required")
	}

	// Cannot change owner role
	members, _ := s.tenantRepo.GetMembers(ctx, req.TenantId)
	for _, m := range members {
		if m.UserID == req.UserId && m.Role == "owner" {
			return nil, status.Error(codes.FailedPrecondition, "cannot change owner role")
		}
	}

	// Update member (remove and re-add with new role)
	if err := s.tenantRepo.RemoveMember(ctx, req.TenantId, req.UserId); err != nil {
		logger.Error("failed to remove member for update", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to update member")
	}

	updatedMember := &tenant.TenantMember{
		TenantID: req.TenantId,
		UserID:   req.UserId,
		Role:     req.Role.String(),
	}

	if err := s.tenantRepo.AddMember(ctx, updatedMember); err != nil {
		logger.Error("failed to add member with new role", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to update member")
	}

	logger.Info("member role updated", zap.String("user_id", req.UserId))

	return &TenantMember{
		TenantId: req.TenantId,
		UserId:   req.UserId,
		Role:     req.Role,
	}, nil
}

// SuspendTenant suspends a tenant
func (s *TenantServiceServer) SuspendTenant(ctx context.Context, req *SuspendTenantRequest) (*Tenant, error) {
	logger := s.logger.With(zap.String("method", "SuspendTenant"), zap.String("tenant_id", req.TenantId))

	// Check admin permission
	if claims, ok := AuthClaimsFromContext(ctx); ok {
		if !hasAdminRole(claims.Roles) {
			return nil, status.Error(codes.PermissionDenied, "admin access required")
		}
	} else {
		return nil, status.Error(codes.Unauthenticated, "authentication required")
	}

	// Get tenant
	t, err := s.tenantRepo.Get(ctx, req.TenantId)
	if err != nil {
		logger.Error("failed to get tenant", zap.Error(err))
		return nil, status.Error(codes.NotFound, "tenant not found")
	}

	// Update status
	t.Status = tenant.TenantSuspended
	t.UpdatedAt = time.Now()

	// Save changes
	if err := s.tenantRepo.Update(ctx, t); err != nil {
		logger.Error("failed to suspend tenant", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to suspend tenant")
	}

	logger.Info("tenant suspended", zap.String("reason", req.Reason))

	return s.tenantToProto(t), nil
}

// ActivateTenant activates a suspended tenant
func (s *TenantServiceServer) ActivateTenant(ctx context.Context, req *ActivateTenantRequest) (*Tenant, error) {
	logger := s.logger.With(zap.String("method", "ActivateTenant"), zap.String("tenant_id", req.TenantId))

	// Check admin permission
	if claims, ok := AuthClaimsFromContext(ctx); ok {
		if !hasAdminRole(claims.Roles) {
			return nil, status.Error(codes.PermissionDenied, "admin access required")
		}
	} else {
		return nil, status.Error(codes.Unauthenticated, "authentication required")
	}

	// Get tenant
	t, err := s.tenantRepo.Get(ctx, req.TenantId)
	if err != nil {
		logger.Error("failed to get tenant", zap.Error(err))
		return nil, status.Error(codes.NotFound, "tenant not found")
	}

	// Update status
	t.Status = tenant.TenantActive
	t.UpdatedAt = time.Now()

	// Save changes
	if err := s.tenantRepo.Update(ctx, t); err != nil {
		logger.Error("failed to activate tenant", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to activate tenant")
	}

	logger.Info("tenant activated")

	return s.tenantToProto(t), nil
}

// GetTenantStats retrieves tenant statistics
func (s *TenantServiceServer) GetTenantStats(ctx context.Context, req *GetTenantStatsRequest) (*TenantStats, error) {
	// Check access
	t, err := s.tenantRepo.Get(ctx, req.TenantId)
	if err != nil {
		return nil, status.Error(codes.NotFound, "tenant not found")
	}

	if !s.hasTenantAccess(ctx, t) {
		return nil, status.Error(codes.PermissionDenied, "access denied")
	}

	// This would typically query analytics data
	// For now, return placeholder stats
	return &TenantStats{
		TenantId: req.TenantId,
		Period:   req.Period,
	}, nil
}

// Helper methods

func (s *TenantServiceServer) tenantToProto(t *tenant.Tenant) *Tenant {
	return &Tenant{
		Id:        t.ID,
		Name:      t.Name,
		Status:    TenantStatus(TenantStatus_value[string(t.Status)]),
		Plan:      t.Plan,
		Labels:    t.Labels,
		CreatedAt: timeToProto(t.CreatedAt),
		UpdatedAt: timeToProto(t.UpdatedAt),
	}
}

func (s *TenantServiceServer) getDefaultQuotaForPlan(plan string) tenant.ResourceQuota {
	switch plan {
	case "free":
		return tenant.ResourceQuota{
			MaxAgents:   3,
			MaxCPU:      2,
			MaxMemoryGB: 4,
			MaxStorageGB: 10,
			MaxGPU:      0,
		}
	case "basic":
		return tenant.ResourceQuota{
			MaxAgents:   10,
			MaxCPU:      8,
			MaxMemoryGB: 16,
			MaxStorageGB: 100,
			MaxGPU:      1,
		}
	case "pro":
		return tenant.ResourceQuota{
			MaxAgents:   50,
			MaxCPU:      32,
			MaxMemoryGB: 64,
			MaxStorageGB: 500,
			MaxGPU:      4,
		}
	case "enterprise":
		return tenant.ResourceQuota{
			MaxAgents:   -1, // unlimited
			MaxCPU:      -1,
			MaxMemoryGB: -1,
			MaxStorageGB: -1,
			MaxGPU:      -1,
		}
	default:
		return tenant.ResourceQuota{
			MaxAgents:   3,
			MaxCPU:      2,
			MaxMemoryGB: 4,
			MaxStorageGB: 10,
			MaxGPU:      0,
		}
	}
}

func (s *TenantServiceServer) hasTenantUpdatePermission(ctx context.Context, t *tenant.Tenant) bool {
	// Check if user is admin
	if claims, ok := AuthClaimsFromContext(ctx); ok {
		if hasAdminRole(claims.Roles) {
			return true
		}

		// Check if user is owner of this tenant
		members, _ := s.tenantRepo.GetMembers(ctx, t.ID)
		for _, m := range members {
			if m.UserID == claims.UserID && (m.Role == "owner" || m.Role == "admin") {
				return true
			}
		}
	}
	return false
}

func (s *TenantServiceServer) hasTenantAdminPermission(ctx context.Context, t *tenant.Tenant) bool {
	return s.hasTenantUpdatePermission(ctx, t)
}

func (s *TenantServiceServer) hasTenantAccess(ctx context.Context, t *tenant.Tenant) bool {
	// Admin has access to all tenants
	if claims, ok := AuthClaimsFromContext(ctx); ok {
		if hasAdminRole(claims.Roles) {
			return true
		}

		// Check if user is member of this tenant
		members, _ := s.tenantRepo.GetMembers(ctx, t.ID)
		for _, m := range members {
			if m.UserID == claims.UserID {
				return true
			}
		}
	}
	return false
}

func hasAdminRole(roles []string) bool {
	for _, r := range roles {
		if r == "admin" || r == "system:admin" {
			return true
		}
	}
	return false
}

func calculatePercent(used, limit int) float64 {
	if limit <= 0 {
		return 0
	}
	return float64(used) / float64(limit) * 100
}

func max(values ...float64) float64 {
	if len(values) == 0 {
		return 0
	}
	m := values[0]
	for _, v := range values[1:] {
		if v > m {
			m = v
		}
	}
	return m
}
