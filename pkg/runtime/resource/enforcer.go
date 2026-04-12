package resource

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/agentos/aos/pkg/runtime/types"
)

type EnforcementAction string

const (
	EnforcementAllow    EnforcementAction = "allow"
	EnforcementWarn     EnforcementAction = "warn"
	EnforcementThrottle EnforcementAction = "throttle"
	EnforcementKill     EnforcementAction = "kill"
)

type QuotaViolation struct {
	AgentID   string            `json:"agent_id"`
	Resource  string            `json:"resource"`
	Requested int64             `json:"requested"`
	Limit     int64             `json:"limit"`
	Action    EnforcementAction `json:"action"`
	Timestamp time.Time         `json:"timestamp"`
}

type ResourceEnforcer struct {
	mu         sync.RWMutex
	quotaMap   map[string]*ResourceQuota
	violations []QuotaViolation
	tracker    ResourceTracker
}

type ResourceQuota struct {
	AgentID        string `json:"agent_id"`
	MaxCPUShares   int64  `json:"max_cpu_shares"`
	MaxMemoryMB    int64  `json:"max_memory_mb"`
	MaxDiskMB      int64  `json:"max_disk_mb"`
	MaxPIDs        int64  `json:"max_pids"`
	MaxNetworkKbps int64  `json:"max_network_kbps"`
}

func NewResourceEnforcer(tracker ResourceTracker) *ResourceEnforcer {
	return &ResourceEnforcer{
		quotaMap:   make(map[string]*ResourceQuota),
		violations: make([]QuotaViolation, 0),
		tracker:    tracker,
	}
}

func (re *ResourceEnforcer) SetQuota(quota *ResourceQuota) error {
	if quota.AgentID == "" {
		return fmt.Errorf("agent ID is required")
	}
	re.mu.Lock()
	defer re.mu.Unlock()
	re.quotaMap[quota.AgentID] = quota
	return nil
}

func (re *ResourceEnforcer) GetQuota(agentID string) (*ResourceQuota, error) {
	re.mu.RLock()
	defer re.mu.RUnlock()
	quota, ok := re.quotaMap[agentID]
	if !ok {
		return nil, fmt.Errorf("quota not found for agent %s", agentID)
	}
	return quota, nil
}

func (re *ResourceEnforcer) RemoveQuota(agentID string) {
	re.mu.Lock()
	defer re.mu.Unlock()
	delete(re.quotaMap, agentID)
}

func (re *ResourceEnforcer) CheckAndEnforce(ctx context.Context, agentID string, spec *types.AgentSpec) (EnforcementAction, error) {
	re.mu.Lock()
	defer re.mu.Unlock()

	quota, ok := re.quotaMap[agentID]
	if !ok {
		return EnforcementAllow, nil
	}

	if spec.Resources != nil {
		if quota.MaxMemoryMB > 0 && spec.Resources.MemoryLimit > 0 {
			memMB := spec.Resources.MemoryLimit / (1024 * 1024)
			if memMB > quota.MaxMemoryMB {
				violation := QuotaViolation{
					AgentID:   agentID,
					Resource:  "memory",
					Requested: memMB,
					Limit:     quota.MaxMemoryMB,
					Action:    EnforcementKill,
					Timestamp: time.Now(),
				}
				re.violations = append(re.violations, violation)
				return EnforcementKill, fmt.Errorf("memory limit %dMB exceeds quota %dMB", memMB, quota.MaxMemoryMB)
			}
		}

		if quota.MaxCPUShares > 0 && spec.Resources.CPULimit > 0 {
			if spec.Resources.CPULimit > quota.MaxCPUShares {
				violation := QuotaViolation{
					AgentID:   agentID,
					Resource:  "cpu",
					Requested: spec.Resources.CPULimit,
					Limit:     quota.MaxCPUShares,
					Action:    EnforcementThrottle,
					Timestamp: time.Now(),
				}
				re.violations = append(re.violations, violation)
				return EnforcementThrottle, fmt.Errorf("cpu limit %d exceeds quota %d", spec.Resources.CPULimit, quota.MaxCPUShares)
			}
		}
	}

	return EnforcementAllow, nil
}

func (re *ResourceEnforcer) GetViolations(agentID string) []QuotaViolation {
	re.mu.RLock()
	defer re.mu.RUnlock()

	var result []QuotaViolation
	for _, v := range re.violations {
		if v.AgentID == agentID {
			result = append(result, v)
		}
	}
	return result
}

func (re *ResourceEnforcer) ClearViolations(agentID string) {
	re.mu.Lock()
	defer re.mu.Unlock()

	var filtered []QuotaViolation
	for _, v := range re.violations {
		if v.AgentID != agentID {
			filtered = append(filtered, v)
		}
	}
	re.violations = filtered
}

func (re *ResourceEnforcer) ListQuotas() []*ResourceQuota {
	re.mu.RLock()
	defer re.mu.RUnlock()

	quotas := make([]*ResourceQuota, 0, len(re.quotaMap))
	for _, q := range re.quotaMap {
		quotas = append(quotas, q)
	}
	return quotas
}
