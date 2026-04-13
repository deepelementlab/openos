package federation

import (
	"context"
	"fmt"
)

// AffinityMode drives cross-cluster placement.
type AffinityMode string

const (
	AffinitySameRegion  AffinityMode = "same_region"
	AffinitySpread      AffinityMode = "spread"
	AffinityCost        AffinityMode = "cost"
	AffinityTenantPack  AffinityMode = "tenant_pack"
)

// ScheduleHint is input to the federal scheduler.
type ScheduleHint struct {
	TenantID     string
	AgentClass   string
	PreferredGPU bool
	Mode         AffinityMode
}

// FederalScheduler picks a target cluster ID given hints (skeleton; extend with real scoring).
type FederalScheduler struct {
	reg *Registry
}

// NewFederalScheduler creates a scheduler backed by registry state.
func NewFederalScheduler(reg *Registry) *FederalScheduler {
	return &FederalScheduler{reg: reg}
}

// PickCluster returns a cluster ID or error if none suitable.
func (f *FederalScheduler) PickCluster(ctx context.Context, h ScheduleHint) (string, error) {
	clusters := f.reg.List()
	if len(clusters) == 0 {
		return "", fmt.Errorf("federation: no clusters registered")
	}
	var best *ClusterRecord
	for _, c := range clusters {
		if !c.Healthy {
			continue
		}
		if h.PreferredGPU && !c.Capabilities.GPU {
			continue
		}
		switch h.Mode {
		case AffinityCost:
			if best == nil || c.Capabilities.CostTier < best.Capabilities.CostTier {
				best = c
			}
		default:
			if best == nil {
				best = c
			}
		}
	}
	if best == nil {
		return "", fmt.Errorf("federation: no healthy cluster matches hint")
	}
	return best.ID, nil
}
