package grpc

import (
	"fmt"

	"github.com/agentos/aos/api/grpc/pb"
	"github.com/agentos/aos/internal/data/repository"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func agentRepoToPB(a *repository.Agent) *pb.Agent {
	if a == nil {
		return nil
	}
	tenantID, _ := a.Metadata["tenant_id"].(string)
	return &pb.Agent{
		Id:          a.ID,
		TenantId:    tenantID,
		Name:        a.Name,
		Image:       a.Image,
		Runtime:     a.Runtime,
		Status:      repoStatusToPB(a.Status),
		Resources:   resourceMapToPB(a.Resources),
		Labels:      labelsFromMetadata(a.Metadata),
		CreatedAt:   timestamppb.New(a.CreatedAt),
		UpdatedAt:   timestamppb.New(a.UpdatedAt),
		Annotations: map[string]string{},
	}
}

func agentPBToRepo(a *pb.Agent) *repository.Agent {
	if a == nil {
		return nil
	}
	md := map[string]interface{}{}
	if a.TenantId != "" {
		md["tenant_id"] = a.TenantId
	}
	if len(a.Labels) > 0 {
		md["labels"] = a.Labels
	}
	return &repository.Agent{
		ID:        a.Id,
		Name:      a.Name,
		Image:     a.Image,
		Runtime:   a.Runtime,
		Status:    pbStatusToRepo(a.Status),
		Resources: resourcePBToMap(a.Resources),
		Metadata:  md,
		CreatedAt: a.GetCreatedAt().AsTime(),
		UpdatedAt: a.GetUpdatedAt().AsTime(),
	}
}

func repoStatusToPB(s repository.AgentStatus) pb.Status {
	switch s {
	case repository.AgentStatusPending:
		return pb.Status_PENDING
	case repository.AgentStatusCreating:
		return pb.Status_CREATING
	case repository.AgentStatusRunning:
		return pb.Status_RUNNING
	case repository.AgentStatusStopping:
		return pb.Status_STOPPING
	case repository.AgentStatusStopped:
		return pb.Status_STOPPED
	case repository.AgentStatusError:
		return pb.Status_ERROR
	default:
		return pb.Status_STATUS_UNSPECIFIED
	}
}

func pbStatusToRepo(s pb.Status) repository.AgentStatus {
	switch s {
	case pb.Status_PENDING:
		return repository.AgentStatusPending
	case pb.Status_CREATING:
		return repository.AgentStatusCreating
	case pb.Status_RUNNING:
		return repository.AgentStatusRunning
	case pb.Status_STOPPING:
		return repository.AgentStatusStopping
	case pb.Status_STOPPED:
		return repository.AgentStatusStopped
	case pb.Status_ERROR:
		return repository.AgentStatusError
	default:
		return repository.AgentStatusCreating
	}
}

func resourceMapToPB(m map[string]string) *pb.Resource {
	if len(m) == 0 {
		return nil
	}
	return &pb.Resource{
		Cpu:     m["cpu"],
		Memory:  m["memory"],
		Storage: m["storage"],
		Gpu:     m["gpu"],
	}
}

func resourcePBToMap(r *pb.Resource) map[string]string {
	if r == nil {
		return nil
	}
	out := map[string]string{}
	if r.Cpu != "" {
		out["cpu"] = r.Cpu
	}
	if r.Memory != "" {
		out["memory"] = r.Memory
	}
	if r.Storage != "" {
		out["storage"] = r.Storage
	}
	if r.Gpu != "" {
		out["gpu"] = r.Gpu
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func labelsFromMetadata(md map[string]interface{}) map[string]string {
	if md == nil {
		return nil
	}
	if v, ok := md["labels"].(map[string]string); ok {
		return v
	}
	return nil
}

func mergeRepoFromPB(dst *repository.Agent, src *pb.Agent) error {
	if dst == nil || src == nil {
		return fmt.Errorf("nil agent")
	}
	if src.Name != "" {
		dst.Name = src.Name
	}
	if src.Image != "" {
		dst.Image = src.Image
	}
	if src.Runtime != "" {
		dst.Runtime = src.Runtime
	}
	if src.Status != pb.Status_STATUS_UNSPECIFIED {
		dst.Status = pbStatusToRepo(src.Status)
	}
	if src.Resources != nil {
		dst.Resources = resourcePBToMap(src.Resources)
	}
	if src.Labels != nil {
		if dst.Metadata == nil {
			dst.Metadata = map[string]interface{}{}
		}
		dst.Metadata["labels"] = src.Labels
	}
	if src.TenantId != "" {
		if dst.Metadata == nil {
			dst.Metadata = map[string]interface{}{}
		}
		dst.Metadata["tenant_id"] = src.TenantId
	}
	dst.UpdatedAt = src.GetUpdatedAt().AsTime()
	return nil
}
