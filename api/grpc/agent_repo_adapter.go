package grpc

import (
	"context"
	"strings"

	"github.com/agentos/aos/api/grpc/pb"
	"github.com/agentos/aos/internal/data/repository"
)

// agentListFilter mirrors the gRPC list filter (package-local to avoid exporting).
type agentListFilter struct {
	Status string
	Labels map[string]string
}

// grpcAgentRepo adapts repository.AgentRepository to tenant-scoped pb.Agent CRUD.
type grpcAgentRepo struct {
	inner repository.AgentRepository
}

func newGRPCAgentRepo(inner repository.AgentRepository) *grpcAgentRepo {
	return &grpcAgentRepo{inner: inner}
}

func (r *grpcAgentRepo) Create(ctx context.Context, agent *pb.Agent) error {
	return r.inner.Create(ctx, agentPBToRepo(agent))
}

func (r *grpcAgentRepo) Get(ctx context.Context, id string) (*pb.Agent, error) {
	a, err := r.inner.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	return agentRepoToPB(a), nil
}

func (r *grpcAgentRepo) List(ctx context.Context, tenantID string, filter *agentListFilter) ([]*pb.Agent, int, error) {
	all, err := r.inner.List(ctx)
	if err != nil {
		return nil, 0, err
	}
	var out []*pb.Agent
	for _, a := range all {
		if tenantID != "" {
			tid, _ := a.Metadata["tenant_id"].(string)
			if tid != tenantID {
				continue
			}
		}
		if filter != nil {
			if filter.Status != "" && !strings.EqualFold(string(a.Status), filter.Status) {
				continue
			}
			if len(filter.Labels) > 0 {
				lbls, _ := a.Metadata["labels"].(map[string]string)
				if !labelsMatch(lbls, filter.Labels) {
					continue
				}
			}
		}
		out = append(out, agentRepoToPB(a))
	}
	return out, len(out), nil
}

func labelsMatch(have, need map[string]string) bool {
	for k, v := range need {
		if have[k] != v {
			return false
		}
	}
	return true
}

func (r *grpcAgentRepo) Update(ctx context.Context, agent *pb.Agent) error {
	cur, err := r.inner.Get(ctx, agent.Id)
	if err != nil {
		return err
	}
	if err := mergeRepoFromPB(cur, agent); err != nil {
		return err
	}
	return r.inner.Update(ctx, cur)
}

// Replace overwrites persistence from a full protobuf agent (CreatedAt preserved).
func (r *grpcAgentRepo) Replace(ctx context.Context, agent *pb.Agent) error {
	dom := agentPBToRepo(agent)
	if old, err := r.inner.Get(ctx, agent.Id); err == nil {
		dom.CreatedAt = old.CreatedAt
	}
	return r.inner.Update(ctx, dom)
}

func (r *grpcAgentRepo) Delete(ctx context.Context, id string) error {
	return r.inner.Delete(ctx, id)
}

func (r *grpcAgentRepo) UpdateStatus(ctx context.Context, id string, st repository.AgentStatus, _ string) error {
	a, err := r.inner.Get(ctx, id)
	if err != nil {
		return err
	}
	a.Status = st
	return r.inner.Update(ctx, a)
}
