package kata

import (
	"context"
	"testing"

	"github.com/agentos/aos/pkg/runtime/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRuntime_NotSupported(t *testing.T) {
	_, err := NewRuntime(context.Background(), &types.RuntimeConfig{Type: types.RuntimeKata})
	assert.EqualError(t, err, "kata runtime is not yet supported")
}

func TestKataRuntime_Initialize(t *testing.T) {
	r := &KataRuntime{}
	err := r.Initialize(context.Background(), &types.RuntimeConfig{Type: types.RuntimeKata})
	assert.EqualError(t, err, "kata runtime is not yet supported")
}

func TestKataRuntime_GetRuntimeInfo(t *testing.T) {
	r := &KataRuntime{}
	info := r.GetRuntimeInfo()
	assert.Equal(t, types.RuntimeKata, info.Type)
	assert.Equal(t, "kata", info.Name)
	assert.Equal(t, "0.0.1", info.Version)
}

func TestKataRuntime_CreateAgent(t *testing.T) {
	r := &KataRuntime{}
	_, err := r.CreateAgent(context.Background(), nil)
	assert.EqualError(t, err, "kata runtime is not yet supported")
}

func TestKataRuntime_StartAgent(t *testing.T) {
	r := &KataRuntime{}
	err := r.StartAgent(context.Background(), "agent-1")
	assert.EqualError(t, err, "kata runtime is not yet supported")
}

func TestKataRuntime_StopAgent(t *testing.T) {
	r := &KataRuntime{}
	err := r.StopAgent(context.Background(), "agent-1", 0)
	assert.EqualError(t, err, "kata runtime is not yet supported")
}

func TestKataRuntime_DeleteAgent(t *testing.T) {
	r := &KataRuntime{}
	err := r.DeleteAgent(context.Background(), "agent-1")
	assert.EqualError(t, err, "kata runtime is not yet supported")
}

func TestKataRuntime_GetAgent(t *testing.T) {
	r := &KataRuntime{}
	_, err := r.GetAgent(context.Background(), "agent-1")
	assert.EqualError(t, err, "kata runtime is not yet supported")
}

func TestKataRuntime_ListAgents(t *testing.T) {
	r := &KataRuntime{}
	_, err := r.ListAgents(context.Background(), nil)
	assert.EqualError(t, err, "kata runtime is not yet supported")
}

func TestKataRuntime_ExecuteCommand(t *testing.T) {
	r := &KataRuntime{}
	_, err := r.ExecuteCommand(context.Background(), "agent-1", nil)
	assert.EqualError(t, err, "kata runtime is not yet supported")
}

func TestKataRuntime_GetAgentLogs(t *testing.T) {
	r := &KataRuntime{}
	_, err := r.GetAgentLogs(context.Background(), "agent-1", nil)
	assert.EqualError(t, err, "kata runtime is not yet supported")
}

func TestKataRuntime_GetAgentStats(t *testing.T) {
	r := &KataRuntime{}
	_, err := r.GetAgentStats(context.Background(), "agent-1")
	assert.EqualError(t, err, "kata runtime is not yet supported")
}

func TestKataRuntime_UpdateAgent(t *testing.T) {
	r := &KataRuntime{}
	err := r.UpdateAgent(context.Background(), "agent-1", nil)
	assert.EqualError(t, err, "kata runtime is not yet supported")
}

func TestKataRuntime_ResizeAgentTerminal(t *testing.T) {
	r := &KataRuntime{}
	err := r.ResizeAgentTerminal(context.Background(), "agent-1", 80, 24)
	assert.EqualError(t, err, "kata runtime is not yet supported")
}

func TestKataRuntime_AttachAgent(t *testing.T) {
	r := &KataRuntime{}
	_, err := r.AttachAgent(context.Background(), "agent-1", nil)
	assert.EqualError(t, err, "kata runtime is not yet supported")
}

func TestKataRuntime_HealthCheck(t *testing.T) {
	r := &KataRuntime{}
	err := r.HealthCheck(context.Background())
	assert.EqualError(t, err, "kata runtime is not yet supported")
}

func TestKataRuntime_Cleanup(t *testing.T) {
	r := &KataRuntime{}
	err := r.Cleanup(context.Background())
	require.NoError(t, err)
}
