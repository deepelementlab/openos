package security

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Sandbox Manager Tests ---

func TestSandboxManager_CreateSandbox(t *testing.T) {
	mgr := NewInMemorySandboxManager()

	config := &SandboxConfig{
		AgentID:        "agent-001",
		Type:           SandboxTypeContainerd,
		ReadOnlyRootFS: true,
		RunAsUser:      1000,
		ResourceLimits: &ResourceLimits{
			CPULimit:    1000,
			MemoryLimit: 512 * 1024 * 1024,
			PidsLimit:   100,
		},
	}

	state, err := mgr.CreateSandbox(context.Background(), config)
	require.NoError(t, err)
	assert.NotEmpty(t, state.ID)
	assert.Equal(t, "agent-001", state.AgentID)
	assert.Equal(t, "running", state.Status)
	assert.Greater(t, state.PID, 0)
}

func TestSandboxManager_CreateSandbox_DefaultType(t *testing.T) {
	mgr := NewInMemorySandboxManager()

	config := &SandboxConfig{
		AgentID: "agent-002",
	}

	state, err := mgr.CreateSandbox(context.Background(), config)
	require.NoError(t, err)
	assert.Equal(t, SandboxTypeContainerd, config.Type) // default
	assert.Equal(t, "running", state.Status)
}

func TestSandboxManager_CreateSandbox_InvalidType(t *testing.T) {
	mgr := NewInMemorySandboxManager()

	config := &SandboxConfig{
		AgentID: "agent-003",
		Type:    "invalid",
	}

	_, err := mgr.CreateSandbox(context.Background(), config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported sandbox type")
}

func TestSandboxManager_CreateSandbox_MissingAgentID(t *testing.T) {
	mgr := NewInMemorySandboxManager()

	config := &SandboxConfig{}
	_, err := mgr.CreateSandbox(context.Background(), config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "agent_id is required")
}

func TestSandboxManager_CreateSandbox_InvalidCPU(t *testing.T) {
	mgr := NewInMemorySandboxManager()

	config := &SandboxConfig{
		AgentID: "agent-004",
		Type:    SandboxTypeGVisor,
		ResourceLimits: &ResourceLimits{
			CPULimit: 50, // too low
		},
	}

	_, err := mgr.CreateSandbox(context.Background(), config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cpu_limit")
}

func TestSandboxManager_DestroySandbox(t *testing.T) {
	mgr := NewInMemorySandboxManager()

	config := &SandboxConfig{AgentID: "agent-005", Type: SandboxTypeContainerd}
	state, _ := mgr.CreateSandbox(context.Background(), config)

	err := mgr.DestroySandbox(context.Background(), state.ID)
	assert.NoError(t, err)

	updated, _ := mgr.GetSandboxState(context.Background(), state.ID)
	assert.Equal(t, "stopped", updated.Status)
}

func TestSandboxManager_DestroySandbox_NotFound(t *testing.T) {
	mgr := NewInMemorySandboxManager()

	err := mgr.DestroySandbox(context.Background(), "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestSandboxManager_ListSandboxes(t *testing.T) {
	mgr := NewInMemorySandboxManager()

	for i := 0; i < 3; i++ {
		config := &SandboxConfig{
			AgentID: "agent-" + string(rune('A'+i)),
			Type:    SandboxTypeContainerd,
		}
		mgr.CreateSandbox(context.Background(), config)
	}

	sandboxes, err := mgr.ListSandboxes(context.Background())
	require.NoError(t, err)
	assert.Len(t, sandboxes, 3)
}

func TestSandboxManager_ExecInSandbox(t *testing.T) {
	mgr := NewInMemorySandboxManager()

	config := &SandboxConfig{AgentID: "agent-exec", Type: SandboxTypeContainerd}
	state, _ := mgr.CreateSandbox(context.Background(), config)

	exitCode, err := mgr.ExecInSandbox(context.Background(), state.ID, []string{"/bin/echo", "hello"})
	assert.NoError(t, err)
	assert.Equal(t, 0, exitCode)
}

func TestSandboxManager_ExecInSandbox_NotRunning(t *testing.T) {
	mgr := NewInMemorySandboxManager()

	config := &SandboxConfig{AgentID: "agent-stopped", Type: SandboxTypeContainerd}
	state, _ := mgr.CreateSandbox(context.Background(), config)
	mgr.DestroySandbox(context.Background(), state.ID)

	_, err := mgr.ExecInSandbox(context.Background(), state.ID, []string{"/bin/ls"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not running")
}

func TestSandboxManager_ExecInSandbox_EmptyCmd(t *testing.T) {
	mgr := NewInMemorySandboxManager()

	config := &SandboxConfig{AgentID: "agent-cmd", Type: SandboxTypeContainerd}
	state, _ := mgr.CreateSandbox(context.Background(), config)

	_, err := mgr.ExecInSandbox(context.Background(), state.ID, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "command cannot be empty")
}

// --- Network Policy Manager Tests ---

func TestNetworkPolicyManager_CreatePolicy(t *testing.T) {
	mgr := NewInMemoryNetworkPolicyManager()

	policy := &NetworkPolicy{
		AgentID:        "agent-001",
		Name:           "default-policy",
		DefaultIngress: NetActionDeny,
		DefaultEgress:  NetActionAllow,
		Rules: []NetworkRule{
			{
				ID:        "rule-1",
				Direction: "ingress",
				Protocol:  ProtocolTCP,
				FromPort:  80,
				ToPort:    80,
				Action:    NetActionAllow,
			},
		},
	}

	err := mgr.CreatePolicy(context.Background(), policy)
	require.NoError(t, err)
	assert.Equal(t, "agent-001", policy.AgentID)
	assert.NotZero(t, policy.CreatedAt)
}

func TestNetworkPolicyManager_CreatePolicy_Defaults(t *testing.T) {
	mgr := NewInMemoryNetworkPolicyManager()

	policy := &NetworkPolicy{
		AgentID: "agent-002",
		Name:    "minimal",
	}

	err := mgr.CreatePolicy(context.Background(), policy)
	require.NoError(t, err)

	// Should have default deny ingress, allow egress
	retrieved, _ := mgr.GetPolicy(context.Background(), "agent-002")
	assert.Equal(t, NetActionDeny, retrieved.DefaultIngress)
	assert.Equal(t, NetActionAllow, retrieved.DefaultEgress)
}

func TestNetworkPolicyManager_CreatePolicy_MissingAgentID(t *testing.T) {
	mgr := NewInMemoryNetworkPolicyManager()

	policy := &NetworkPolicy{Name: "no-agent"}
	err := mgr.CreatePolicy(context.Background(), policy)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "agent_id is required")
}

func TestNetworkPolicyManager_CreatePolicy_Duplicate(t *testing.T) {
	mgr := NewInMemoryNetworkPolicyManager()

	policy := &NetworkPolicy{AgentID: "agent-dup", Name: "first"}
	mgr.CreatePolicy(context.Background(), policy)

	policy2 := &NetworkPolicy{AgentID: "agent-dup", Name: "second"}
	err := mgr.CreatePolicy(context.Background(), policy2)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestNetworkPolicyManager_EvaluateConnection_Allowed(t *testing.T) {
	mgr := NewInMemoryNetworkPolicyManager()

	policy := &NetworkPolicy{
		AgentID:        "agent-conn",
		Name:           "test",
		DefaultIngress: NetActionDeny,
		DefaultEgress:  NetActionAllow,
		Rules: []NetworkRule{
			{
				ID:        "http-in",
				Direction: "ingress",
				Protocol:  ProtocolTCP,
				FromPort:  80,
				ToPort:    80,
				Action:    NetActionAllow,
			},
		},
	}
	mgr.CreatePolicy(context.Background(), policy)

	conn := &ConnectionRequest{
		Direction: "ingress",
		Protocol:  ProtocolTCP,
		Port:      80,
	}

	verdict, err := mgr.EvaluateConnection(context.Background(), "agent-conn", conn)
	require.NoError(t, err)
	assert.True(t, verdict.Allowed)
	assert.Equal(t, "http-in", verdict.RuleID)
}

func TestNetworkPolicyManager_EvaluateConnection_DeniedByRule(t *testing.T) {
	mgr := NewInMemoryNetworkPolicyManager()

	policy := &NetworkPolicy{
		AgentID:        "agent-deny",
		Name:           "test",
		DefaultIngress: NetActionDeny,
		DefaultEgress:  NetActionAllow,
		Rules: []NetworkRule{
			{
				ID:        "block-ssh",
				Direction: "ingress",
				Protocol:  ProtocolTCP,
				FromPort:  22,
				ToPort:    22,
				Action:    NetActionDeny,
			},
		},
	}
	mgr.CreatePolicy(context.Background(), policy)

	conn := &ConnectionRequest{
		Direction: "ingress",
		Protocol:  ProtocolTCP,
		Port:      22,
	}

	verdict, err := mgr.EvaluateConnection(context.Background(), "agent-deny", conn)
	require.NoError(t, err)
	assert.False(t, verdict.Allowed)
	assert.Equal(t, "block-ssh", verdict.RuleID)
}

func TestNetworkPolicyManager_EvaluateConnection_DefaultDenyIngress(t *testing.T) {
	mgr := NewInMemoryNetworkPolicyManager()

	policy := &NetworkPolicy{
		AgentID:        "agent-default",
		Name:           "test",
		DefaultIngress: NetActionDeny,
		DefaultEgress:  NetActionAllow,
	}
	mgr.CreatePolicy(context.Background(), policy)

	conn := &ConnectionRequest{
		Direction: "ingress",
		Protocol:  ProtocolTCP,
		Port:      9999,
	}

	verdict, err := mgr.EvaluateConnection(context.Background(), "agent-default", conn)
	require.NoError(t, err)
	assert.False(t, verdict.Allowed)
}

func TestNetworkPolicyManager_EvaluateConnection_NoPolicy(t *testing.T) {
	mgr := NewInMemoryNetworkPolicyManager()

	conn := &ConnectionRequest{
		Direction: "egress",
		Protocol:  ProtocolTCP,
		Port:      443,
	}

	verdict, err := mgr.EvaluateConnection(context.Background(), "no-policy-agent", conn)
	require.NoError(t, err)
	assert.False(t, verdict.Allowed)
	assert.Contains(t, verdict.Reason, "no network policy defined")
}

func TestNetworkPolicyManager_UpdatePolicy(t *testing.T) {
	mgr := NewInMemoryNetworkPolicyManager()

	policy := &NetworkPolicy{
		AgentID: "agent-update",
		Name:    "original",
	}
	mgr.CreatePolicy(context.Background(), policy)

	policy.Name = "updated"
	policy.Rules = []NetworkRule{
		{ID: "new-rule", Direction: "egress", Protocol: ProtocolAny, Action: NetActionAllow},
	}
	err := mgr.UpdatePolicy(context.Background(), policy)
	require.NoError(t, err)

	updated, _ := mgr.GetPolicy(context.Background(), "agent-update")
	assert.Equal(t, "updated", updated.Name)
	assert.Len(t, updated.Rules, 1)
}

func TestNetworkPolicyManager_DeletePolicy(t *testing.T) {
	mgr := NewInMemoryNetworkPolicyManager()

	policy := &NetworkPolicy{AgentID: "agent-del", Name: "to-delete"}
	mgr.CreatePolicy(context.Background(), policy)

	err := mgr.DeletePolicy(context.Background(), "agent-del")
	require.NoError(t, err)

	_, err = mgr.GetPolicy(context.Background(), "agent-del")
	assert.Error(t, err)
}

func TestNetworkPolicyManager_ListPolicies(t *testing.T) {
	mgr := NewInMemoryNetworkPolicyManager()

	for i := 0; i < 3; i++ {
		policy := &NetworkPolicy{
			AgentID: "agent-list-" + string(rune('A'+i)),
			Name:    "policy",
		}
		mgr.CreatePolicy(context.Background(), policy)
	}

	policies, err := mgr.ListPolicies(context.Background())
	require.NoError(t, err)
	assert.Len(t, policies, 3)
}

// --- Policy Evaluator Tests (RBAC) ---

func TestPolicyEvaluator_AllowRule(t *testing.T) {
	eval := NewDefaultPolicyEvaluator()

	eval.AddRule(PolicyRule{
		ID:       "admin-read",
		Roles:    []string{"admin"},
		Resource: "agents",
		Actions:  []string{"read", "write"},
		Effect:   ActionAllow,
	})

	result, err := eval.Evaluate(context.Background(), EvalRequest{
		UserID:   "user1",
		Roles:    []string{"admin"},
		Resource: "agents",
		Action:   "read",
	})
	require.NoError(t, err)
	assert.True(t, result.Allowed)
}

func TestPolicyEvaluator_DenyRule(t *testing.T) {
	eval := NewDefaultPolicyEvaluator()

	eval.AddRule(PolicyRule{
		ID:       "block-delete",
		Roles:    []string{"viewer"},
		Resource: "agents",
		Actions:  []string{"delete"},
		Effect:   ActionDeny,
	})

	result, err := eval.Evaluate(context.Background(), EvalRequest{
		UserID:   "user1",
		Roles:    []string{"viewer"},
		Resource: "agents",
		Action:   "delete",
	})
	require.NoError(t, err)
	assert.False(t, result.Allowed)
}

func TestPolicyEvaluator_WildcardResource(t *testing.T) {
	eval := NewDefaultPolicyEvaluator()

	eval.AddRule(PolicyRule{
		ID:       "super-admin",
		Roles:    []string{"super"},
		Resource: "*",
		Actions:  []string{"*"},
		Effect:   ActionAllow,
	})

	result, err := eval.Evaluate(context.Background(), EvalRequest{
		Roles:    []string{"super"},
		Resource: "anything",
		Action:   "whatever",
	})
	require.NoError(t, err)
	assert.True(t, result.Allowed)
}

func TestPolicyEvaluator_DefaultDeny(t *testing.T) {
	eval := NewDefaultPolicyEvaluator()

	result, err := eval.Evaluate(context.Background(), EvalRequest{
		Roles:    []string{"unknown"},
		Resource: "agents",
		Action:   "read",
	})
	require.NoError(t, err)
	assert.False(t, result.Allowed)
	assert.Contains(t, result.Reason, "default deny")
}

func TestPolicyEvaluator_RemoveRule(t *testing.T) {
	eval := NewDefaultPolicyEvaluator()

	eval.AddRule(PolicyRule{
		ID:       "temp-rule",
		Roles:    []string{"temp"},
		Resource: "agents",
		Actions:  []string{"read"},
		Effect:   ActionAllow,
	})

	// Verify rule exists
	result, _ := eval.Evaluate(context.Background(), EvalRequest{
		Roles:    []string{"temp"},
		Resource: "agents",
		Action:   "read",
	})
	assert.True(t, result.Allowed)

	// Remove rule
	err := eval.RemoveRule("temp-rule")
	assert.NoError(t, err)

	// Verify rule is gone
	result, _ = eval.Evaluate(context.Background(), EvalRequest{
		Roles:    []string{"temp"},
		Resource: "agents",
		Action:   "read",
	})
	assert.False(t, result.Allowed)
}

func TestPolicyEvaluator_DuplicateRule(t *testing.T) {
	eval := NewDefaultPolicyEvaluator()

	err := eval.AddRule(PolicyRule{ID: "dup", Resource: "test", Actions: []string{"read"}, Effect: ActionAllow})
	assert.NoError(t, err)

	err = eval.AddRule(PolicyRule{ID: "dup", Resource: "test", Actions: []string{"write"}, Effect: ActionAllow})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestPolicyEvaluator_ListRules(t *testing.T) {
	eval := NewDefaultPolicyEvaluator()

	eval.AddRule(PolicyRule{ID: "r1", Resource: "a", Actions: []string{"read"}, Effect: ActionAllow})
	eval.AddRule(PolicyRule{ID: "r2", Resource: "b", Actions: []string{"write"}, Effect: ActionAllow})

	rules := eval.ListRules()
	assert.Len(t, rules, 2)
}
