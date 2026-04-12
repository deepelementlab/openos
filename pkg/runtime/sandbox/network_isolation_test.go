package sandbox

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewNetworkIsolation(t *testing.T) {
	ni := NewNetworkIsolation()
	assert.NotNil(t, ni)
}

func TestNetworkIsolation_SetPolicy(t *testing.T) {
	ni := NewNetworkIsolation()
	ctx := context.Background()

	err := ni.SetPolicy(ctx, &NetworkIsolationPolicy{
		AgentID: "agent-1",
		Level:   IsolationBasic,
	})
	require.NoError(t, err)

	policy, err := ni.GetPolicy(ctx, "agent-1")
	require.NoError(t, err)
	assert.Equal(t, IsolationBasic, policy.Level)
}

func TestNetworkIsolation_SetPolicy_EmptyAgentID(t *testing.T) {
	ni := NewNetworkIsolation()
	err := ni.SetPolicy(context.Background(), &NetworkIsolationPolicy{AgentID: ""})
	assert.Error(t, err)
}

func TestNetworkIsolation_GetPolicy_NotSet(t *testing.T) {
	ni := NewNetworkIsolation()
	policy, err := ni.GetPolicy(context.Background(), "agent-no-policy")
	require.NoError(t, err)
	assert.Equal(t, IsolationNone, policy.Level)
}

func TestNetworkIsolation_RemovePolicy(t *testing.T) {
	ni := NewNetworkIsolation()
	ctx := context.Background()

	ni.SetPolicy(ctx, &NetworkIsolationPolicy{AgentID: "agent-1", Level: IsolationStrict})
	ni.RemovePolicy(ctx, "agent-1")

	policy, _ := ni.GetPolicy(ctx, "agent-1")
	assert.Equal(t, IsolationNone, policy.Level)
}

func TestNetworkIsolation_CheckConnection_None(t *testing.T) {
	ni := NewNetworkIsolation()
	ctx := context.Background()

	ni.SetPolicy(ctx, &NetworkIsolationPolicy{AgentID: "agent-1", Level: IsolationNone})

	allowed, reason := ni.CheckConnection(ctx, "agent-1", "tcp", "1.2.3.4", 80)
	assert.True(t, allowed)
	assert.Contains(t, reason, "disabled")
}

func TestNetworkIsolation_CheckConnection_BasicAllowed(t *testing.T) {
	ni := NewNetworkIsolation()
	ctx := context.Background()

	ni.SetPolicy(ctx, &NetworkIsolationPolicy{
		AgentID:      "agent-1",
		Level:        IsolationBasic,
		BlockedPorts: []int{22, 3389},
	})

	allowed, _ := ni.CheckConnection(ctx, "agent-1", "tcp", "1.2.3.4", 80)
	assert.True(t, allowed)
}

func TestNetworkIsolation_CheckConnection_BasicBlocked(t *testing.T) {
	ni := NewNetworkIsolation()
	ctx := context.Background()

	ni.SetPolicy(ctx, &NetworkIsolationPolicy{
		AgentID:      "agent-1",
		Level:        IsolationBasic,
		BlockedPorts: []int{22, 3389},
	})

	allowed, reason := ni.CheckConnection(ctx, "agent-1", "tcp", "1.2.3.4", 22)
	assert.False(t, allowed)
	assert.Contains(t, reason, "blocked")
}

func TestNetworkIsolation_CheckConnection_StrictAllowed(t *testing.T) {
	ni := NewNetworkIsolation()
	ctx := context.Background()

	ni.SetPolicy(ctx, &NetworkIsolationPolicy{
		AgentID:      "agent-1",
		Level:        IsolationStrict,
		AllowedPorts: []int{80, 443},
	})

	allowed, _ := ni.CheckConnection(ctx, "agent-1", "tcp", "1.2.3.4", 443)
	assert.True(t, allowed)
}

func TestNetworkIsolation_CheckConnection_StrictBlocked(t *testing.T) {
	ni := NewNetworkIsolation()
	ctx := context.Background()

	ni.SetPolicy(ctx, &NetworkIsolationPolicy{
		AgentID:      "agent-1",
		Level:        IsolationStrict,
		AllowedPorts: []int{80, 443},
	})

	allowed, _ := ni.CheckConnection(ctx, "agent-1", "tcp", "1.2.3.4", 8080)
	assert.False(t, allowed)
}

func TestNetworkIsolation_CheckConnection_CustomHostAllowed(t *testing.T) {
	ni := NewNetworkIsolation()
	ctx := context.Background()

	ni.SetPolicy(ctx, &NetworkIsolationPolicy{
		AgentID:      "agent-1",
		Level:        IsolationCustom,
		AllowedHosts: []string{"api.example.com", "cdn.example.com"},
	})

	allowed, _ := ni.CheckConnection(ctx, "agent-1", "tcp", "api.example.com", 443)
	assert.True(t, allowed)
}

func TestNetworkIsolation_CheckConnection_CustomHostBlocked(t *testing.T) {
	ni := NewNetworkIsolation()
	ctx := context.Background()

	ni.SetPolicy(ctx, &NetworkIsolationPolicy{
		AgentID:      "agent-1",
		Level:        IsolationCustom,
		AllowedHosts: []string{"api.example.com"},
	})

	allowed, _ := ni.CheckConnection(ctx, "agent-1", "tcp", "evil.com", 443)
	assert.False(t, allowed)
}

func TestNetworkIsolation_CheckConnection_Wildcard(t *testing.T) {
	ni := NewNetworkIsolation()
	ctx := context.Background()

	ni.SetPolicy(ctx, &NetworkIsolationPolicy{
		AgentID:      "agent-1",
		Level:        IsolationCustom,
		AllowedHosts: []string{"*"},
	})

	allowed, _ := ni.CheckConnection(ctx, "agent-1", "tcp", "anything.com", 443)
	assert.True(t, allowed)
}

func TestNetworkIsolation_CheckConnection_NoPolicy(t *testing.T) {
	ni := NewNetworkIsolation()
	allowed, reason := ni.CheckConnection(context.Background(), "agent-no-policy", "tcp", "1.2.3.4", 80)
	assert.True(t, allowed)
	assert.Contains(t, reason, "default allow")
}

func TestNetworkIsolation_RecordConnection(t *testing.T) {
	ni := NewNetworkIsolation()
	ctx := context.Background()

	err := ni.RecordConnection(ctx, NetworkConnection{
		ID:         "conn-1",
		AgentID:    "agent-1",
		Protocol:   "tcp",
		RemoteAddr: "1.2.3.4",
		RemotePort: 443,
	})
	require.NoError(t, err)

	conns, err := ni.GetConnections(ctx, "agent-1")
	require.NoError(t, err)
	assert.Len(t, conns, 1)
	assert.Equal(t, "conn-1", conns[0].ID)
}

func TestNetworkIsolation_RecordConnection_EmptyAgentID(t *testing.T) {
	ni := NewNetworkIsolation()
	err := ni.RecordConnection(context.Background(), NetworkConnection{ID: "conn-1"})
	assert.Error(t, err)
}

func TestNetworkIsolation_RecordConnection_EmptyConnID(t *testing.T) {
	ni := NewNetworkIsolation()
	err := ni.RecordConnection(context.Background(), NetworkConnection{AgentID: "agent-1"})
	assert.Error(t, err)
}

func TestNetworkIsolation_GetConnections_Empty(t *testing.T) {
	ni := NewNetworkIsolation()
	conns, err := ni.GetConnections(context.Background(), "nonexistent")
	require.NoError(t, err)
	assert.Empty(t, conns)
}

func TestNetworkIsolation_CloseConnection(t *testing.T) {
	ni := NewNetworkIsolation()
	ctx := context.Background()

	ni.RecordConnection(ctx, NetworkConnection{ID: "conn-1", AgentID: "agent-1"})
	ni.RecordConnection(ctx, NetworkConnection{ID: "conn-2", AgentID: "agent-1"})

	err := ni.CloseConnection(ctx, "agent-1", "conn-1")
	require.NoError(t, err)

	conns, _ := ni.GetConnections(ctx, "agent-1")
	assert.Len(t, conns, 1)
	assert.Equal(t, "conn-2", conns[0].ID)
}

func TestNetworkIsolation_CloseConnection_NotFound(t *testing.T) {
	ni := NewNetworkIsolation()
	err := ni.CloseConnection(context.Background(), "agent-1", "nonexistent")
	assert.Error(t, err)
}

func TestNetworkIsolation_ListPolicies(t *testing.T) {
	ni := NewNetworkIsolation()
	ctx := context.Background()

	ni.SetPolicy(ctx, &NetworkIsolationPolicy{AgentID: "a1", Level: IsolationBasic})
	ni.SetPolicy(ctx, &NetworkIsolationPolicy{AgentID: "a2", Level: IsolationStrict})

	policies := ni.ListPolicies(ctx)
	assert.Len(t, policies, 2)
}

func TestNetworkConnection_Fields(t *testing.T) {
	now := time.Now()
	conn := NetworkConnection{
		ID:            "conn-1",
		AgentID:       "agent-1",
		Protocol:      "tcp",
		RemoteAddr:    "10.0.0.1",
		RemotePort:    8080,
		EstablishedAt: now,
		BytesSent:     1024,
		BytesRecv:     2048,
	}
	assert.Equal(t, "conn-1", conn.ID)
	assert.Equal(t, "tcp", conn.Protocol)
	assert.Equal(t, int64(1024), conn.BytesSent)
	assert.Equal(t, int64(2048), conn.BytesRecv)
}
