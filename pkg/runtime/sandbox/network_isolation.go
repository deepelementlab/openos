package sandbox

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type NetworkIsolationLevel string

const (
	IsolationNone   NetworkIsolationLevel = "none"
	IsolationBasic  NetworkIsolationLevel = "basic"
	IsolationStrict NetworkIsolationLevel = "strict"
	IsolationCustom NetworkIsolationLevel = "custom"
)

type NetworkIsolation struct {
	mu          sync.RWMutex
	policies    map[string]*NetworkIsolationPolicy
	connections map[string][]NetworkConnection
}

type NetworkIsolationPolicy struct {
	AgentID        string                `json:"agent_id"`
	Level          NetworkIsolationLevel `json:"level"`
	AllowedHosts   []string              `json:"allowed_hosts,omitempty"`
	BlockedPorts   []int                 `json:"blocked_ports,omitempty"`
	AllowedPorts   []int                 `json:"allowed_ports,omitempty"`
	BandwidthLimit int64                 `json:"bandwidth_limit_kbps,omitempty"`
	DNSOverride    map[string]string     `json:"dns_override,omitempty"`
	CreatedAt      time.Time             `json:"created_at"`
	UpdatedAt      time.Time             `json:"updated_at"`
}

type NetworkConnection struct {
	ID            string    `json:"id"`
	AgentID       string    `json:"agent_id"`
	Protocol      string    `json:"protocol"`
	RemoteAddr    string    `json:"remote_addr"`
	RemotePort    int       `json:"remote_port"`
	EstablishedAt time.Time `json:"established_at"`
	BytesSent     int64     `json:"bytes_sent"`
	BytesRecv     int64     `json:"bytes_recv"`
}

func NewNetworkIsolation() *NetworkIsolation {
	return &NetworkIsolation{
		policies:    make(map[string]*NetworkIsolationPolicy),
		connections: make(map[string][]NetworkConnection),
	}
}

func (ni *NetworkIsolation) SetPolicy(ctx context.Context, policy *NetworkIsolationPolicy) error {
	if policy.AgentID == "" {
		return fmt.Errorf("agent ID is required")
	}

	now := time.Now()
	if policy.CreatedAt.IsZero() {
		policy.CreatedAt = now
	}
	policy.UpdatedAt = now

	ni.mu.Lock()
	defer ni.mu.Unlock()
	ni.policies[policy.AgentID] = policy
	return nil
}

func (ni *NetworkIsolation) GetPolicy(ctx context.Context, agentID string) (*NetworkIsolationPolicy, error) {
	ni.mu.RLock()
	defer ni.mu.RUnlock()

	policy, ok := ni.policies[agentID]
	if !ok {
		return &NetworkIsolationPolicy{
			AgentID: agentID,
			Level:   IsolationNone,
		}, nil
	}
	return policy, nil
}

func (ni *NetworkIsolation) RemovePolicy(ctx context.Context, agentID string) error {
	ni.mu.Lock()
	defer ni.mu.Unlock()
	delete(ni.policies, agentID)
	delete(ni.connections, agentID)
	return nil
}

func (ni *NetworkIsolation) CheckConnection(ctx context.Context, agentID, protocol string, remoteAddr string, port int) (bool, string) {
	ni.mu.RLock()
	defer ni.mu.RUnlock()

	policy, ok := ni.policies[agentID]
	if !ok {
		return true, "no policy - default allow"
	}

	switch policy.Level {
	case IsolationNone:
		return true, "isolation disabled"
	case IsolationBasic:
		for _, blocked := range policy.BlockedPorts {
			if port == blocked {
				return false, fmt.Sprintf("port %d is blocked", port)
			}
		}
		return true, "basic isolation passed"
	case IsolationStrict:
		for _, allowed := range policy.AllowedPorts {
			if port == allowed {
				return true, "port explicitly allowed"
			}
		}
		return false, fmt.Sprintf("port %d not in allowed list", port)
	case IsolationCustom:
		if len(policy.AllowedHosts) > 0 {
			hostAllowed := false
			for _, host := range policy.AllowedHosts {
				if host == remoteAddr || host == "*" {
					hostAllowed = true
					break
				}
			}
			if !hostAllowed {
				return false, fmt.Sprintf("host %s not allowed", remoteAddr)
			}
		}
		return true, "custom policy passed"
	default:
		return false, "unknown isolation level"
	}
}

func (ni *NetworkIsolation) RecordConnection(ctx context.Context, conn NetworkConnection) error {
	if conn.AgentID == "" {
		return fmt.Errorf("agent ID is required")
	}
	if conn.ID == "" {
		return fmt.Errorf("connection ID is required")
	}

	ni.mu.Lock()
	defer ni.mu.Unlock()
	ni.connections[conn.AgentID] = append(ni.connections[conn.AgentID], conn)
	return nil
}

func (ni *NetworkIsolation) GetConnections(ctx context.Context, agentID string) ([]NetworkConnection, error) {
	ni.mu.RLock()
	defer ni.mu.RUnlock()
	conns := ni.connections[agentID]
	if conns == nil {
		return []NetworkConnection{}, nil
	}
	result := make([]NetworkConnection, len(conns))
	copy(result, conns)
	return result, nil
}

func (ni *NetworkIsolation) CloseConnection(ctx context.Context, agentID, connID string) error {
	ni.mu.Lock()
	defer ni.mu.Unlock()

	conns := ni.connections[agentID]
	for i, c := range conns {
		if c.ID == connID {
			ni.connections[agentID] = append(conns[:i], conns[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("connection %s not found", connID)
}

func (ni *NetworkIsolation) ListPolicies(ctx context.Context) []*NetworkIsolationPolicy {
	ni.mu.RLock()
	defer ni.mu.RUnlock()

	policies := make([]*NetworkIsolationPolicy, 0, len(ni.policies))
	for _, p := range ni.policies {
		policies = append(policies, p)
	}
	return policies
}
