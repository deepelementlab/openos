package gvisor

import (
	"context"
	"io"
	"testing"

	"github.com/agentos/aos/pkg/runtime/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRuntime(t *testing.T) {
	rt, err := NewRuntime(context.Background(), &types.RuntimeConfig{Type: types.RuntimeGVisor})
	require.NoError(t, err)
	require.NotNil(t, rt)
}

func TestNewGVisorRuntime(t *testing.T) {
	rt := NewGVisorRuntime()
	require.NotNil(t, rt)
}

func TestGVisorRuntime_Initialize(t *testing.T) {
	rt := NewGVisorRuntime()
	config := &types.RuntimeConfig{
		Type:    types.RuntimeGVisor,
		RootDir: "/tmp/gvisor",
	}
	err := rt.Initialize(context.Background(), config)
	require.NoError(t, err)
}

func TestGVisorRuntime_GetRuntimeInfo(t *testing.T) {
	rt := NewGVisorRuntime()
	info := rt.GetRuntimeInfo()

	assert.Equal(t, types.RuntimeGVisor, info.Type)
	assert.Equal(t, "gvisor", info.Name)
	assert.Equal(t, "1.0.0", info.Version)
	assert.Equal(t, "v1alpha1", info.APIVersion)
	assert.Contains(t, info.Features, "enhanced-security")
	assert.Contains(t, info.Capabilities, "create")
}

func TestGVisorRuntime_CreateAgent(t *testing.T) {
	rt := NewGVisorRuntime()
	spec := &types.AgentSpec{
		ID:    "agent-1",
		Name:  "test-agent",
		Image: "nginx:latest",
	}

	agent, err := rt.CreateAgent(context.Background(), spec)
	require.NoError(t, err)
	assert.Equal(t, "agent-1", agent.ID)
	assert.Equal(t, "test-agent", agent.Name)
	assert.Equal(t, "nginx:latest", agent.Image)
	assert.Equal(t, types.AgentStateCreated, agent.State)
}

func TestGVisorRuntime_CreateAgent_NilSpec(t *testing.T) {
	rt := NewGVisorRuntime()
	_, err := rt.CreateAgent(context.Background(), nil)
	assert.EqualError(t, err, "agent spec is required")
}

func TestGVisorRuntime_CreateAgent_MissingID(t *testing.T) {
	rt := NewGVisorRuntime()
	_, err := rt.CreateAgent(context.Background(), &types.AgentSpec{Image: "nginx"})
	assert.EqualError(t, err, "agent ID is required")
}

func TestGVisorRuntime_CreateAgent_MissingImage(t *testing.T) {
	rt := NewGVisorRuntime()
	_, err := rt.CreateAgent(context.Background(), &types.AgentSpec{ID: "a1"})
	assert.EqualError(t, err, "agent image is required")
}

func TestGVisorRuntime_StartAgent(t *testing.T) {
	rt := NewGVisorRuntime()
	err := rt.StartAgent(context.Background(), "agent-1")
	require.NoError(t, err)
}

func TestGVisorRuntime_StartAgent_EmptyID(t *testing.T) {
	rt := NewGVisorRuntime()
	err := rt.StartAgent(context.Background(), "")
	assert.EqualError(t, err, "agent ID is required")
}

func TestGVisorRuntime_StopAgent(t *testing.T) {
	rt := NewGVisorRuntime()
	err := rt.StopAgent(context.Background(), "agent-1", 0)
	require.NoError(t, err)
}

func TestGVisorRuntime_StopAgent_EmptyID(t *testing.T) {
	rt := NewGVisorRuntime()
	err := rt.StopAgent(context.Background(), "", 0)
	assert.EqualError(t, err, "agent ID is required")
}

func TestGVisorRuntime_DeleteAgent(t *testing.T) {
	rt := NewGVisorRuntime()
	err := rt.DeleteAgent(context.Background(), "agent-1")
	require.NoError(t, err)
}

func TestGVisorRuntime_DeleteAgent_EmptyID(t *testing.T) {
	rt := NewGVisorRuntime()
	err := rt.DeleteAgent(context.Background(), "")
	assert.EqualError(t, err, "agent ID is required")
}

func TestGVisorRuntime_GetAgent(t *testing.T) {
	rt := NewGVisorRuntime()
	agent, err := rt.GetAgent(context.Background(), "agent-1")
	require.NoError(t, err)
	assert.Equal(t, "agent-1", agent.ID)
	assert.Equal(t, types.AgentStateRunning, agent.State)
}

func TestGVisorRuntime_GetAgent_EmptyID(t *testing.T) {
	rt := NewGVisorRuntime()
	_, err := rt.GetAgent(context.Background(), "")
	assert.EqualError(t, err, "agent ID is required")
}

func TestGVisorRuntime_ListAgents(t *testing.T) {
	rt := NewGVisorRuntime()
	agents, err := rt.ListAgents(context.Background(), nil)
	require.NoError(t, err)
	assert.Empty(t, agents)
}

func TestGVisorRuntime_ExecuteCommand(t *testing.T) {
	rt := NewGVisorRuntime()
	cmd := &types.Command{Command: []string{"echo", "hello"}}
	result, err := rt.ExecuteCommand(context.Background(), "agent-1", cmd)
	require.NoError(t, err)
	assert.Equal(t, int32(0), result.ExitCode)
}

func TestGVisorRuntime_ExecuteCommand_EmptyID(t *testing.T) {
	rt := NewGVisorRuntime()
	_, err := rt.ExecuteCommand(context.Background(), "", &types.Command{})
	assert.EqualError(t, err, "agent ID is required")
}

func TestGVisorRuntime_ExecuteCommand_NilCmd(t *testing.T) {
	rt := NewGVisorRuntime()
	_, err := rt.ExecuteCommand(context.Background(), "agent-1", nil)
	assert.EqualError(t, err, "command is required")
}

func TestGVisorRuntime_GetAgentLogs(t *testing.T) {
	rt := NewGVisorRuntime()
	rc, err := rt.GetAgentLogs(context.Background(), "agent-1", nil)
	require.NoError(t, err)
	data, err := io.ReadAll(rc)
	assert.NoError(t, err)
	assert.Empty(t, data)
}

func TestGVisorRuntime_GetAgentLogs_EmptyID(t *testing.T) {
	rt := NewGVisorRuntime()
	_, err := rt.GetAgentLogs(context.Background(), "", nil)
	assert.EqualError(t, err, "agent ID is required")
}

func TestGVisorRuntime_GetAgentStats(t *testing.T) {
	rt := NewGVisorRuntime()
	stats, err := rt.GetAgentStats(context.Background(), "agent-1")
	require.NoError(t, err)
	assert.False(t, stats.Timestamp.IsZero())
}

func TestGVisorRuntime_GetAgentStats_EmptyID(t *testing.T) {
	rt := NewGVisorRuntime()
	_, err := rt.GetAgentStats(context.Background(), "")
	assert.EqualError(t, err, "agent ID is required")
}

func TestGVisorRuntime_UpdateAgent(t *testing.T) {
	rt := NewGVisorRuntime()
	err := rt.UpdateAgent(context.Background(), "agent-1", nil)
	require.NoError(t, err)
}

func TestGVisorRuntime_ResizeAgentTerminal(t *testing.T) {
	rt := NewGVisorRuntime()
	err := rt.ResizeAgentTerminal(context.Background(), "agent-1", 80, 24)
	require.NoError(t, err)
}

func TestGVisorRuntime_AttachAgent(t *testing.T) {
	rt := NewGVisorRuntime()
	_, err := rt.AttachAgent(context.Background(), "agent-1", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not implemented")
}

func TestGVisorRuntime_HealthCheck(t *testing.T) {
	rt := NewGVisorRuntime()
	err := rt.HealthCheck(context.Background())
	require.NoError(t, err)
}

func TestGVisorRuntime_Cleanup(t *testing.T) {
	rt := NewGVisorRuntime()
	err := rt.Cleanup(context.Background())
	require.NoError(t, err)
}
