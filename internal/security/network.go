package security

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// NetworkPolicyAction defines the action for a network rule.
type NetworkPolicyAction string

const (
	NetActionAllow NetworkPolicyAction = "allow"
	NetActionDeny  NetworkPolicyAction = "deny"
)

// NetworkProtocol defines the network protocol.
type NetworkProtocol string

const (
	ProtocolTCP NetworkProtocol = "tcp"
	ProtocolUDP NetworkProtocol = "udp"
	ProtocolICMP NetworkProtocol = "icmp"
	ProtocolAny  NetworkProtocol = "any"
)

// NetworkRule defines a single network policy rule.
type NetworkRule struct {
	ID          string               `json:"id"`
	Direction   string               `json:"direction"` // "ingress" or "egress"
	Protocol    NetworkProtocol      `json:"protocol"`
	FromPort    int                  `json:"from_port,omitempty"`
	ToPort      int                  `json:"to_port,omitempty"`
	CIDR        string               `json:"cidr,omitempty"`       // e.g., "10.0.0.0/8"
	DNSName     string               `json:"dns_name,omitempty"`   // e.g., "api.example.com"
	Action      NetworkPolicyAction  `json:"action"`
	Description string               `json:"description,omitempty"`
}

// NetworkPolicy defines the network policy for an agent.
type NetworkPolicy struct {
	ID          string         `json:"id"`
	AgentID     string         `json:"agent_id"`
	Name        string         `json:"name"`
	DefaultIngress NetworkPolicyAction `json:"default_ingress"` // default ingress action
	DefaultEgress  NetworkPolicyAction `json:"default_egress"`  // default egress action
	Rules       []NetworkRule  `json:"rules"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

// NetworkPolicyManager manages network policies for agents.
type NetworkPolicyManager interface {
	CreatePolicy(ctx context.Context, policy *NetworkPolicy) error
	GetPolicy(ctx context.Context, agentID string) (*NetworkPolicy, error)
	UpdatePolicy(ctx context.Context, policy *NetworkPolicy) error
	DeletePolicy(ctx context.Context, agentID string) error
	EvaluateConnection(ctx context.Context, agentID string, conn *ConnectionRequest) (*ConnectionVerdict, error)
	ListPolicies(ctx context.Context) ([]*NetworkPolicy, error)
}

// ConnectionRequest represents a network connection request.
type ConnectionRequest struct {
	Direction string          `json:"direction"` // "ingress" or "egress"
	Protocol  NetworkProtocol `json:"protocol"`
	Port      int             `json:"port,omitempty"`
	RemoteIP  string          `json:"remote_ip,omitempty"`
	RemoteDNS string          `json:"remote_dns,omitempty"`
}

// ConnectionVerdict represents the decision for a connection request.
type ConnectionVerdict struct {
	Allowed bool   `json:"allowed"`
	Reason  string `json:"reason"`
	RuleID  string `json:"rule_id,omitempty"`
}

// InMemoryNetworkPolicyManager provides an in-memory network policy manager.
type InMemoryNetworkPolicyManager struct {
	mu       sync.RWMutex
	policies map[string]*NetworkPolicy // agentID -> policy
}

// NewInMemoryNetworkPolicyManager creates a new in-memory network policy manager.
func NewInMemoryNetworkPolicyManager() *InMemoryNetworkPolicyManager {
	return &InMemoryNetworkPolicyManager{
		policies: make(map[string]*NetworkPolicy),
	}
}

// CreatePolicy creates a new network policy for an agent.
func (m *InMemoryNetworkPolicyManager) CreatePolicy(_ context.Context, policy *NetworkPolicy) error {
	if policy.AgentID == "" {
		return fmt.Errorf("agent_id is required")
	}
	if policy.Name == "" {
		return fmt.Errorf("policy name is required")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.policies[policy.AgentID]; exists {
		return fmt.Errorf("policy already exists for agent %s", policy.AgentID)
	}

	now := time.Now().UTC()
	policy.CreatedAt = now
	policy.UpdatedAt = now

	if policy.DefaultIngress == "" {
		policy.DefaultIngress = NetActionDeny
	}
	if policy.DefaultEgress == "" {
		policy.DefaultEgress = NetActionAllow
	}
	if policy.Rules == nil {
		policy.Rules = []NetworkRule{}
	}

	cp := *policy
	m.policies[policy.AgentID] = &cp
	return nil
}

// GetPolicy returns the network policy for an agent.
func (m *InMemoryNetworkPolicyManager) GetPolicy(_ context.Context, agentID string) (*NetworkPolicy, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	policy, ok := m.policies[agentID]
	if !ok {
		return nil, fmt.Errorf("no network policy found for agent %s", agentID)
	}

	cp := *policy
	return &cp, nil
}

// UpdatePolicy updates an existing network policy.
func (m *InMemoryNetworkPolicyManager) UpdatePolicy(_ context.Context, policy *NetworkPolicy) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.policies[policy.AgentID]; !ok {
		return fmt.Errorf("no network policy found for agent %s", policy.AgentID)
	}

	policy.UpdatedAt = time.Now().UTC()
	cp := *policy
	m.policies[policy.AgentID] = &cp
	return nil
}

// DeletePolicy removes the network policy for an agent.
func (m *InMemoryNetworkPolicyManager) DeletePolicy(_ context.Context, agentID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.policies[agentID]; !ok {
		return fmt.Errorf("no network policy found for agent %s", agentID)
	}

	delete(m.policies, agentID)
	return nil
}

// EvaluateConnection evaluates whether a network connection should be allowed.
func (m *InMemoryNetworkPolicyManager) EvaluateConnection(_ context.Context, agentID string, conn *ConnectionRequest) (*ConnectionVerdict, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	policy, ok := m.policies[agentID]
	if !ok {
		// No policy = default deny
		return &ConnectionVerdict{
			Allowed: false,
			Reason:  "no network policy defined (default deny)",
		}, nil
	}

	// Check rules in order
	for _, rule := range policy.Rules {
		if rule.Direction != conn.Direction {
			continue
		}
		if !matchProtocol(rule.Protocol, conn.Protocol) {
			continue
		}
		if !matchPortRange(rule.FromPort, rule.ToPort, conn.Port) {
			continue
		}

		// Rule matched
		if rule.Action == NetActionAllow {
			return &ConnectionVerdict{
				Allowed: true,
				Reason:  fmt.Sprintf("allowed by rule %s", rule.ID),
				RuleID:  rule.ID,
			}, nil
		}
		return &ConnectionVerdict{
			Allowed: false,
			Reason:  fmt.Sprintf("denied by rule %s", rule.ID),
			RuleID:  rule.ID,
		}, nil
	}

	// No rule matched, apply default
	defaultAction := policy.DefaultEgress
	if conn.Direction == "ingress" {
		defaultAction = policy.DefaultIngress
	}

	return &ConnectionVerdict{
		Allowed: defaultAction == NetActionAllow,
		Reason:  fmt.Sprintf("no matching rule, applying default %s policy", conn.Direction),
	}, nil
}

// ListPolicies returns all network policies.
func (m *InMemoryNetworkPolicyManager) ListPolicies(_ context.Context) ([]*NetworkPolicy, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*NetworkPolicy, 0, len(m.policies))
	for _, policy := range m.policies {
		cp := *policy
		result = append(result, &cp)
	}
	return result, nil
}

func matchProtocol(rule, conn NetworkProtocol) bool {
	if rule == ProtocolAny || rule == conn {
		return true
	}
	return false
}

func matchPortRange(fromPort, toPort, port int) bool {
	if fromPort == 0 && toPort == 0 {
		return true // match any port
	}
	if port >= fromPort && port <= toPort {
		return true
	}
	return false
}

var _ NetworkPolicyManager = (*InMemoryNetworkPolicyManager)(nil)
