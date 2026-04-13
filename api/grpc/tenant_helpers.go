package grpc

import (
	"strings"

	"github.com/agentos/aos/api/grpc/pb"
	"github.com/agentos/aos/internal/tenant"
)

func tenantDomainStatusToPB(s tenant.TenantStatus) pb.TenantStatus {
	switch s {
	case tenant.TenantActive:
		return pb.TenantStatus_TENANT_STATUS_ACTIVE
	case tenant.TenantSuspended:
		return pb.TenantStatus_TENANT_STATUS_SUSPENDED
	default:
		return pb.TenantStatus_TENANT_STATUS_UNSPECIFIED
	}
}

func memberRolePBToString(r pb.MemberRole) string {
	switch r {
	case pb.MemberRole_MEMBER_ROLE_OWNER:
		return "owner"
	case pb.MemberRole_MEMBER_ROLE_ADMIN:
		return "admin"
	case pb.MemberRole_MEMBER_ROLE_MEMBER:
		return "member"
	case pb.MemberRole_MEMBER_ROLE_VIEWER:
		return "viewer"
	default:
		return "member"
	}
}

func memberRoleStringToPB(role string) pb.MemberRole {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "owner":
		return pb.MemberRole_MEMBER_ROLE_OWNER
	case "admin":
		return pb.MemberRole_MEMBER_ROLE_ADMIN
	case "member":
		return pb.MemberRole_MEMBER_ROLE_MEMBER
	case "viewer":
		return pb.MemberRole_MEMBER_ROLE_VIEWER
	default:
		return pb.MemberRole_MEMBER_ROLE_UNSPECIFIED
	}
}
