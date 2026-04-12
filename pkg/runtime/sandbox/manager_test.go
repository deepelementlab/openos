package sandbox

import (
	"context"
	"testing"

	"github.com/agentos/aos/pkg/runtime/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSandboxManager(t *testing.T) {
	m := NewSandboxManager()
	assert.NotNil(t, m)
}

func TestSandboxManager_Create(t *testing.T) {
	m := NewSandboxManager()
	spec := &types.SandboxSpec{
		ID:   "sb-1",
		Name: "test-sandbox",
		Type: "gvisor",
	}

	sb, err := m.Create(context.Background(), spec)
	require.NoError(t, err)
	assert.Equal(t, "sb-1", sb.ID)
	assert.Equal(t, "test-sandbox", sb.Name)
	assert.Equal(t, 0, sb.AgentCount)
}

func TestSandboxManager_Create_MissingID(t *testing.T) {
	m := NewSandboxManager()
	_, err := m.Create(context.Background(), &types.SandboxSpec{Name: "test"})
	assert.EqualError(t, err, "sandbox ID is required")
}

func TestSandboxManager_Create_Duplicate(t *testing.T) {
	m := NewSandboxManager()
	spec := &types.SandboxSpec{ID: "sb-1", Name: "test", Type: "gvisor"}
	_, err := m.Create(context.Background(), spec)
	require.NoError(t, err)
	_, err = m.Create(context.Background(), spec)
	assert.EqualError(t, err, "sandbox sb-1 already exists")
}

func TestSandboxManager_Remove(t *testing.T) {
	m := NewSandboxManager()
	_, err := m.Create(context.Background(), &types.SandboxSpec{ID: "sb-1", Name: "test", Type: "gvisor"})
	require.NoError(t, err)
	require.NoError(t, m.Remove(context.Background(), "sb-1"))
}

func TestSandboxManager_Remove_NotFound(t *testing.T) {
	m := NewSandboxManager()
	err := m.Remove(context.Background(), "nonexistent")
	assert.EqualError(t, err, "sandbox nonexistent not found")
}

func TestSandboxManager_Remove_HasAgents(t *testing.T) {
	m := NewSandboxManager()
	_, err := m.Create(context.Background(), &types.SandboxSpec{ID: "sb-1", Name: "test", Type: "gvisor"})
	require.NoError(t, err)
	require.NoError(t, m.Join(context.Background(), "agent-1", "sb-1"))

	err = m.Remove(context.Background(), "sb-1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "still has 1 agents attached")
}

func TestSandboxManager_Join(t *testing.T) {
	m := NewSandboxManager()
	_, err := m.Create(context.Background(), &types.SandboxSpec{ID: "sb-1", Name: "test", Type: "gvisor"})
	require.NoError(t, err)

	err = m.Join(context.Background(), "agent-1", "sb-1")
	require.NoError(t, err)

	sb, _ := m.Get(context.Background(), "sb-1")
	assert.Equal(t, 1, sb.AgentCount)
}

func TestSandboxManager_Join_SandboxNotFound(t *testing.T) {
	m := NewSandboxManager()
	err := m.Join(context.Background(), "agent-1", "nonexistent")
	assert.EqualError(t, err, "sandbox nonexistent not found")
}

func TestSandboxManager_Join_AlreadyJoined(t *testing.T) {
	m := NewSandboxManager()
	_, err := m.Create(context.Background(), &types.SandboxSpec{ID: "sb-1", Name: "test", Type: "gvisor"})
	require.NoError(t, err)
	require.NoError(t, m.Join(context.Background(), "agent-1", "sb-1"))

	err = m.Join(context.Background(), "agent-1", "sb-1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already in sandbox")
}

func TestSandboxManager_Leave(t *testing.T) {
	m := NewSandboxManager()
	_, err := m.Create(context.Background(), &types.SandboxSpec{ID: "sb-1", Name: "test", Type: "gvisor"})
	require.NoError(t, err)
	require.NoError(t, m.Join(context.Background(), "agent-1", "sb-1"))

	err = m.Leave(context.Background(), "agent-1", "sb-1")
	require.NoError(t, err)

	sb, _ := m.Get(context.Background(), "sb-1")
	assert.Equal(t, 0, sb.AgentCount)
}

func TestSandboxManager_Leave_NotJoined(t *testing.T) {
	m := NewSandboxManager()
	err := m.Leave(context.Background(), "agent-1", "sb-1")
	assert.Error(t, err)
}

func TestSandboxManager_Leave_WrongSandbox(t *testing.T) {
	m := NewSandboxManager()
	_, err := m.Create(context.Background(), &types.SandboxSpec{ID: "sb-1", Name: "test", Type: "gvisor"})
	_, err = m.Create(context.Background(), &types.SandboxSpec{ID: "sb-2", Name: "test2", Type: "gvisor"})
	require.NoError(t, err)
	require.NoError(t, m.Join(context.Background(), "agent-1", "sb-1"))

	err = m.Leave(context.Background(), "agent-1", "sb-2")
	assert.Error(t, err)
}

func TestSandboxManager_Get(t *testing.T) {
	m := NewSandboxManager()
	_, err := m.Create(context.Background(), &types.SandboxSpec{ID: "sb-1", Name: "test", Type: "gvisor"})
	require.NoError(t, err)

	sb, err := m.Get(context.Background(), "sb-1")
	require.NoError(t, err)
	assert.Equal(t, "sb-1", sb.ID)
}

func TestSandboxManager_Get_NotFound(t *testing.T) {
	m := NewSandboxManager()
	_, err := m.Get(context.Background(), "nonexistent")
	assert.EqualError(t, err, "sandbox nonexistent not found")
}

func TestSandboxManager_Get_ReturnsCopy(t *testing.T) {
	m := NewSandboxManager()
	_, err := m.Create(context.Background(), &types.SandboxSpec{ID: "sb-1", Name: "test", Type: "gvisor"})
	require.NoError(t, err)

	sb1, _ := m.Get(context.Background(), "sb-1")
	sb1.Name = "modified"

	sb2, _ := m.Get(context.Background(), "sb-1")
	assert.Equal(t, "test", sb2.Name, "should return a copy")
}

func TestSandboxManager_List(t *testing.T) {
	m := NewSandboxManager()
	_, err := m.Create(context.Background(), &types.SandboxSpec{ID: "sb-1", Name: "test1", Type: "gvisor"})
	require.NoError(t, err)
	_, err = m.Create(context.Background(), &types.SandboxSpec{ID: "sb-2", Name: "test2", Type: "kata"})
	require.NoError(t, err)

	list, err := m.List(context.Background())
	require.NoError(t, err)
	assert.Len(t, list, 2)
}

func TestSandboxManager_List_Empty(t *testing.T) {
	m := NewSandboxManager()
	list, err := m.List(context.Background())
	require.NoError(t, err)
	assert.Empty(t, list)
}

func TestSandboxManager_UpdatePolicy(t *testing.T) {
	m := NewSandboxManager()
	_, err := m.Create(context.Background(), &types.SandboxSpec{ID: "sb-1", Name: "test", Type: "gvisor"})
	require.NoError(t, err)

	policy := &types.SecurityPolicy{
		AllowedCapabilities: []string{"NET_ADMIN"},
	}
	err = m.UpdatePolicy(context.Background(), "sb-1", policy)
	require.NoError(t, err)

	sb, _ := m.Get(context.Background(), "sb-1")
	assert.Equal(t, []string{"NET_ADMIN"}, sb.SecurityPolicy.AllowedCapabilities)
}

func TestSandboxManager_UpdatePolicy_NotFound(t *testing.T) {
	m := NewSandboxManager()
	err := m.UpdatePolicy(context.Background(), "nonexistent", nil)
	assert.EqualError(t, err, "sandbox nonexistent not found")
}
