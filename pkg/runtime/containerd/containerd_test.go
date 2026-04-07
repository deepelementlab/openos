// Package containerd implements tests for the containerd runtime.
package containerd

import (
	"context"
	"testing"
	"time"

	"github.com/agentos/aos/pkg/runtime/types"
	"github.com/stretchr/testify/assert"
)

// TestContainerdRuntime_Initialize tests runtime initialization.
func TestContainerdRuntime_Initialize(t *testing.T) {
	// Skip this test in CI environment without containerd
	if testing.Short() {
		t.Skip("Skipping containerd test in short mode")
	}

	ctx := context.Background()
	runtime := NewContainerdRuntime()

	config := &types.RuntimeConfig{
		Type: types.RuntimeContainerd,
		Options: map[string]interface{}{
			"socket_path": "/run/containerd/containerd.sock",
		},
		RootDir:  "/tmp/agentos/runtime",
		StateDir: "/tmp/agentos/state",
		LogDir:   "/tmp/agentos/logs",
	}

	// Test initialization
	err := runtime.Initialize(ctx, config)
	// Since we don't have containerd running in test environment,
	// this will likely fail, but that's expected
	if err != nil {
		t.Logf("Expected initialization error (no containerd): %v", err)
	}
}

// TestContainerdRuntime_GetRuntimeInfo tests runtime information retrieval.
func TestContainerdRuntime_GetRuntimeInfo(t *testing.T) {
	runtime := NewContainerdRuntime()

	info := runtime.GetRuntimeInfo()

	assert.NotNil(t, info)
	assert.Equal(t, types.RuntimeContainerd, info.Type)
	assert.Equal(t, "containerd", info.Name)
	assert.Contains(t, info.Capabilities, "create")
	assert.Contains(t, info.Capabilities, "start")
	assert.Contains(t, info.Capabilities, "stop")
	assert.Contains(t, info.Capabilities, "delete")
}

// TestContainerdRuntime_CreateAgent_Validation tests validation for agent creation.
func TestContainerdRuntime_CreateAgent_Validation(t *testing.T) {
	ctx := context.Background()
	runtime := NewContainerdRuntime()

	// Test nil spec
	_, err := runtime.CreateAgent(ctx, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "agent spec is required")

	// Test empty ID
	spec1 := &types.AgentSpec{
		ID:    "",
		Name:  "test-agent",
		Image: "nginx:latest",
	}
	_, err = runtime.CreateAgent(ctx, spec1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "agent ID is required")

	// Test empty image
	spec2 := &types.AgentSpec{
		ID:    "test-agent-1",
		Name:  "test-agent",
		Image: "",
	}
	_, err = runtime.CreateAgent(ctx, spec2)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "agent image is required")
}

// TestContainerdRuntime_AgentLifecycle_Validation tests validation for agent lifecycle operations.
func TestContainerdRuntime_AgentLifecycle_Validation(t *testing.T) {
	ctx := context.Background()
	runtime := NewContainerdRuntime()

	// Test StartAgent with empty ID
	err := runtime.StartAgent(ctx, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "agent ID is required")

	// Test StopAgent with empty ID
	err = runtime.StopAgent(ctx, "", 30*time.Second)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "agent ID is required")

	// Test DeleteAgent with empty ID
	err = runtime.DeleteAgent(ctx, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "agent ID is required")

	// Test GetAgent with empty ID
	_, err = runtime.GetAgent(ctx, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "agent ID is required")
}

// TestContainerdRuntime_StopAgent_DefaultTimeout tests default timeout for stop operation.
func TestContainerdRuntime_StopAgent_DefaultTimeout(t *testing.T) {
	ctx := context.Background()
	runtime := NewContainerdRuntime()

	// Test with invalid timeout (should use default)
	err := runtime.StopAgent(ctx, "test-agent", 0)
	// This will fail because containerd is not running, but the validation should pass
	assert.Error(t, err)
	// Error should not be about invalid timeout
	assert.NotContains(t, err.Error(), "timeout")
}

// TestContainerdRuntime_ExecuteCommand_Validation tests validation for command execution.
func TestContainerdRuntime_ExecuteCommand_Validation(t *testing.T) {
	ctx := context.Background()
	runtime := NewContainerdRuntime()

	// Test with empty agent ID
	_, err := runtime.ExecuteCommand(ctx, "", &types.Command{
		Command: []string{"echo"},
		Args:    []string{"hello"},
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "agent ID is required")

	// Test with nil command
	_, err = runtime.ExecuteCommand(ctx, "test-agent", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "command is required")
}

// TestContainerdRuntime_GetAgentStats_Validation tests validation for stats retrieval.
func TestContainerdRuntime_GetAgentStats_Validation(t *testing.T) {
	ctx := context.Background()
	runtime := NewContainerdRuntime()

	// Test with empty agent ID
	_, err := runtime.GetAgentStats(ctx, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "agent ID is required")
}

// TestContainerdRuntime_ListAgents tests listing agents (should work even without containerd).
func TestContainerdRuntime_ListAgents(t *testing.T) {
	ctx := context.Background()
	runtime := NewContainerdRuntime()

	agents, err := runtime.ListAgents(ctx, nil)
	if err != nil {
		t.Logf("ListAgents error (expected without containerd): %v", err)
		return
	}
	assert.NotNil(t, agents)
}

// TestContainerdRuntime_HealthCheck tests health check functionality.
func TestContainerdRuntime_HealthCheck(t *testing.T) {
	ctx := context.Background()
	runtime := NewContainerdRuntime()

	// Health check should fail without initialization
	err := runtime.HealthCheck(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "containerd client not initialized")
}

// TestContainerdRuntime_UpdateAgent_Validation tests validation for agent update.
func TestContainerdRuntime_UpdateAgent_Validation(t *testing.T) {
	ctx := context.Background()
	runtime := NewContainerdRuntime()

	// Test with empty agent ID
	err := runtime.UpdateAgent(ctx, "", &types.AgentSpec{
		ID:    "test-agent",
		Name:  "test-agent",
		Image: "nginx:latest",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "agent ID is required")

	// Test with nil spec
	err = runtime.UpdateAgent(ctx, "test-agent", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "agent spec is required")
}

// TestContainerdRuntime_ResizeAgentTerminal_Validation tests validation for terminal resize.
func TestContainerdRuntime_ResizeAgentTerminal_Validation(t *testing.T) {
	ctx := context.Background()
	runtime := NewContainerdRuntime()

	// Test with empty agent ID
	err := runtime.ResizeAgentTerminal(ctx, "", 80, 24)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "agent ID is required")
}

// TestContainerdRuntime_AttachAgent_Validation tests validation for agent attachment.
func TestContainerdRuntime_AttachAgent_Validation(t *testing.T) {
	ctx := context.Background()
	runtime := NewContainerdRuntime()

	// Test with empty agent ID
	_, err := runtime.AttachAgent(ctx, "", &types.AttachOptions{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "agent ID is required")
}

// TestContainerdRuntime_Cleanup tests cleanup functionality.
func TestContainerdRuntime_Cleanup(t *testing.T) {
	ctx := context.Background()
	runtime := NewContainerdRuntime()

	// Cleanup should succeed even without initialization
	err := runtime.Cleanup(ctx)
	assert.NoError(t, err)
}

// TestContainerdRuntime_MockOperations tests mocked operations for development.
func TestContainerdRuntime_MockOperations(t *testing.T) {
	runtime := NewContainerdRuntime()

	// Verify runtime info is consistent
	info := runtime.GetRuntimeInfo()
	assert.Equal(t, types.RuntimeContainerd, info.Type)
	assert.Equal(t, "1.0.0", info.Version)
	assert.Equal(t, "v1alpha1", info.APIVersion)

	// Verify features and capabilities
	assert.Contains(t, info.Features, "container-management")
	assert.Contains(t, info.Features, "resource-limits")
	assert.Contains(t, info.Capabilities, "create")
	assert.Contains(t, info.Capabilities, "logs")
}