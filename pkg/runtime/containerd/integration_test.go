// Package containerd implements integration tests for the containerd runtime
package containerd

import (
	"context"
	"testing"
	"time"

	"github.com/agentos/aos/pkg/runtime/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAgentImagePullAndStart_Integration demonstrates the complete workflow
// of pulling an image and starting an agent container.
// This test requires containerd to be running.
func TestAgentImagePullAndStart_Integration(t *testing.T) {
	// Skip in short mode or CI environment without containerd
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	runtime := NewContainerdRuntime()

	// Initialize runtime with containerd configuration
	config := &types.RuntimeConfig{
		Type: types.RuntimeContainerd,
		Options: map[string]interface{}{
			"socket_path": "/run/containerd/containerd.sock",
		},
		RootDir:  "/tmp/agentos/runtime",
		StateDir: "/tmp/agentos/state",
		LogDir:   "/tmp/agentos/logs",
	}

	// Initialize runtime
	err := runtime.Initialize(ctx, config)
	if err != nil {
		t.Skipf("Skipping test: containerd not available or failed to initialize: %v", err)
	}

	// Clean up any existing test containers
	defer func() {
		// List and clean up test agents
		agents, _ := runtime.ListAgents(ctx, nil)
		for _, agent := range agents {
			if agent.Name == "test-agent" {
				runtime.StopAgent(ctx, agent.ID, 10*time.Second)
				runtime.DeleteAgent(ctx, agent.ID)
			}
		}
		runtime.Cleanup(ctx)
	}()

	// Test 1: Create agent with image pull
	t.Run("CreateAgentWithImage", func(t *testing.T) {
		spec := &types.AgentSpec{
			ID:    "test-agent-" + time.Now().Format("20060102150405"),
			Name:  "test-agent",
			Image: "docker.io/library/alpine:latest", // Small image for testing
			Command: []string{"/bin/sleep"},
			Args: []string{"300"}, // Sleep for 5 minutes
			Resources: &types.ResourceRequirements{
				MemoryLimit: 128 * 1024 * 1024, // 128MB
				CPULimit:    100, // 0.1 CPU
			},
			Labels: map[string]string{
				"test": "integration",
			},
		}

		// Create agent (this will pull image if not exists)
		agent, err := runtime.CreateAgent(ctx, spec)
		require.NoError(t, err, "Failed to create agent")
		assert.NotNil(t, agent, "Agent should not be nil")
		assert.Equal(t, spec.ID, agent.ID, "Agent ID should match")
		assert.Equal(t, spec.Image, agent.Image, "Agent image should match")
		assert.Equal(t, types.AgentStateCreated, agent.State, "Agent should be in created state")

		// Verify agent info can be retrieved
		retrievedAgent, err := runtime.GetAgent(ctx, agent.ID)
		require.NoError(t, err, "Failed to get agent")
		assert.Equal(t, agent.ID, retrievedAgent.ID, "Retrieved agent ID should match")

		// Test 2: Start the agent
		t.Run("StartAgent", func(t *testing.T) {
			err = runtime.StartAgent(ctx, agent.ID)
			require.NoError(t, err, "Failed to start agent")

			// Give container time to start
			time.Sleep(2 * time.Second)

			// Verify agent is running
			startedAgent, err := runtime.GetAgent(ctx, agent.ID)
			require.NoError(t, err, "Failed to get started agent")
			assert.Equal(t, types.AgentStateRunning, startedAgent.State, "Agent should be running")

			// Test 3: Get agent stats
			t.Run("GetAgentStats", func(t *testing.T) {
				stats, err := runtime.GetAgentStats(ctx, agent.ID)
				// Stats might fail if containerd doesn't have metrics configured
				if err == nil {
					assert.NotNil(t, stats, "Stats should not be nil")
					assert.True(t, stats.Timestamp.Before(time.Now()), "Stats timestamp should be in the past")
				} else {
					t.Logf("GetAgentStats failed (might be expected): %v", err)
				}
			})

			// Test 4: Stop agent
			t.Run("StopAgent", func(t *testing.T) {
				err = runtime.StopAgent(ctx, agent.ID, 30*time.Second)
				require.NoError(t, err, "Failed to stop agent")

				time.Sleep(1 * time.Second)

				stoppedAgent, err := runtime.GetAgent(ctx, agent.ID)
				require.NoError(t, err, "Failed to get stopped agent")
				assert.Equal(t, types.AgentStateStopped, stoppedAgent.State, "Agent should be stopped")
			})
		})

		// Test 5: Delete agent
		t.Run("DeleteAgent", func(t *testing.T) {
			err = runtime.DeleteAgent(ctx, agent.ID)
			require.NoError(t, err, "Failed to delete agent")

			// Verify agent is deleted
			_, err = runtime.GetAgent(ctx, agent.ID)
			assert.Error(t, err, "GetAgent should fail for deleted agent")
			assert.Contains(t, err.Error(), "not found", "Error should indicate agent not found")
		})
	})

	// Test 6: List agents
	t.Run("ListAgents", func(t *testing.T) {
		agents, err := runtime.ListAgents(ctx, nil)
		require.NoError(t, err, "Failed to list agents")
		assert.NotNil(t, agents, "Agents list should not be nil")
		
		// Check that our test agent is not in the list (should have been deleted)
		found := false
		for _, agent := range agents {
			if agent.Name == "test-agent" {
				found = true
				break
			}
		}
		assert.False(t, found, "Test agent should not be in the list after deletion")
	})
}

// TestImageManager_Integration demonstrates image management functionality
func TestImageManager_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	runtime := NewContainerdRuntime()

	config := &types.RuntimeConfig{
		Type: types.RuntimeContainerd,
		Options: map[string]interface{}{
			"socket_path": "/run/containerd/containerd.sock",
		},
	}

	err := runtime.Initialize(ctx, config)
	if err != nil {
		t.Skipf("Skipping test: containerd not available: %v", err)
	}

	// Get client from runtime (for demonstration, actual implementation would expose it)
	// This is just to show the image management workflow
	t.Run("ImageManagementWorkflow", func(t *testing.T) {
		// Note: In actual implementation, we would have access to the containerd client
		// and could create an ImageManager instance
		t.Log("Image management functionality is available via ImageManager struct")
		t.Log("Image pulling is integrated into CreateAgent method")
		t.Log("Use CreateAgent to pull images and start containers")
	})
}

// TestAgentLifecycle_Demo demonstrates a complete agent lifecycle
func TestAgentLifecycle_Demo(t *testing.T) {
	// This is a demonstration test showing the complete workflow
	t.Run("CompleteAgentLifecycle", func(t *testing.T) {
		t.Log("=== Agent OS Agent Lifecycle Demo ===")
		t.Log("1. Initialize containerd runtime")
		t.Log("2. Create agent spec with image reference")
		t.Log("3. Call CreateAgent - automatically pulls image if needed")
		t.Log("4. Call StartAgent - starts the container")
		t.Log("5. Monitor agent with GetAgentStats")
		t.Log("6. Execute commands in agent with ExecuteCommand")
		t.Log("7. Stop agent gracefully")
		t.Log("8. Delete agent and clean up resources")
		t.Log("=====================================")
	})
}

// TestValidation demonstrates input validation for agent operations
func TestValidation(t *testing.T) {
	ctx := context.Background()
	runtime := NewContainerdRuntime()

	testCases := []struct {
		name        string
		spec        *types.AgentSpec
		expectError bool
		errorMsg    string
	}{
		{
			name:        "EmptyImage",
			spec:        &types.AgentSpec{ID: "test-1", Name: "test", Image: ""},
			expectError: true,
			errorMsg:    "agent image is required",
		},
		{
			name:        "EmptyID",
			spec:        &types.AgentSpec{ID: "", Name: "test", Image: "alpine:latest"},
			expectError: true,
			errorMsg:    "agent ID is required",
		},
		{
			name:        "NilSpec",
			spec:        nil,
			expectError: true,
			errorMsg:    "agent spec is required",
		},
		{
			name:        "ValidSpec",
			spec:        &types.AgentSpec{ID: "valid-test", Name: "test", Image: "alpine:latest"},
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			agent, err := runtime.CreateAgent(ctx, tc.spec)

			if tc.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.errorMsg)
			} else {
				// Without a running containerd, CreateAgent will fail at image pull.
				if err != nil {
					t.Skipf("Skipping: containerd not available: %v", err)
				}
				assert.NotNil(t, agent)
				assert.Equal(t, tc.spec.ID, agent.ID)
			}
		})
	}
}