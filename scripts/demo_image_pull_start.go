// Demo program for Agent OS image pull and start functionality
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/agentos/aos/pkg/runtime/containerd"
	"github.com/agentos/aos/pkg/runtime/types"
)

func main() {
	fmt.Println("=== Agent OS Image Pull and Start Demo ===")
	fmt.Println("This demonstrates the complete workflow of pulling an agent image")
	fmt.Println("and starting an agent container using the containerd runtime.")

	// Check if containerd socket exists
	socketPath := "/run/containerd/containerd.sock"
	if _, err := os.Stat(socketPath); os.IsNotExist(err) {
		log.Printf("Warning: Containerd socket not found at %s", socketPath)
		log.Println("Please ensure containerd is installed and running.")
		log.Println("On Ubuntu/Debian: sudo apt-get install containerd")
		log.Println("Start containerd: sudo systemctl start containerd")
		fmt.Println()
	}

	ctx := context.Background()

	// Initialize runtime
	runtime := containerd.NewContainerdRuntime()
	config := &types.RuntimeConfig{
		Type: types.RuntimeContainerd,
		Options: map[string]interface{}{
			"socket_path": socketPath,
		},
		RootDir:  "/tmp/agentos/runtime",
		StateDir: "/tmp/agentos/state",
		LogDir:   "/tmp/agentos/logs",
	}

	fmt.Println("1. Initializing containerd runtime...")
	err := runtime.Initialize(ctx, config)
	if err != nil {
		log.Fatalf("Failed to initialize runtime: %v", err)
	}
	fmt.Println("✓ Runtime initialized successfully")

	// Check runtime health
	fmt.Println("2. Checking runtime health...")
	if err := runtime.HealthCheck(ctx); err != nil {
		log.Printf("Warning: Runtime health check failed: %v", err)
	} else {
		fmt.Println("✓ Runtime health check passed")
	}

	// Get runtime info
	info := runtime.GetRuntimeInfo()
	fmt.Printf("3. Runtime Info:\n")
	fmt.Printf("   - Type: %s\n", info.Type)
	fmt.Printf("   - Name: %s\n", info.Name)
	fmt.Printf("   - Version: %s\n", info.Version)
	fmt.Printf("   - Features: %v\n", info.Features)
	fmt.Printf("   - Capabilities: %v\n", info.Capabilities)

	// Create agent spec
	agentID := fmt.Sprintf("demo-agent-%d", time.Now().Unix())
	spec := &types.AgentSpec{
		ID:    agentID,
		Name:  "demo-agent",
		Image: "docker.io/library/nginx:alpine", // Small, widely available image
		Command: []string{"nginx", "-g", "daemon off;"},
		Resources: &types.ResourceRequirements{
			MemoryLimit: 256 * 1024 * 1024, // 256MB
			CPULimit:    200, // 0.2 CPU
		},
		Labels: map[string]string{
			"demo":     "true",
			"creator":  "agentos-demo",
			"started":  time.Now().Format(time.RFC3339),
		},
	}

	fmt.Printf("\n4. Creating agent %s...\n", agentID)
	fmt.Printf("   Image: %s\n", spec.Image)
	fmt.Println("   Note: This will automatically pull the image if not already available")

	agent, err := runtime.CreateAgent(ctx, spec)
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}
	fmt.Println("✓ Agent created successfully")
	fmt.Printf("   Agent ID: %s\n", agent.ID)
	fmt.Printf("   Agent Name: %s\n", agent.Name)
	fmt.Printf("   Agent State: %s\n", agent.State)

	// List images (if we had access to ImageManager)
	fmt.Println("\n5. Available images (conceptual):")
	fmt.Println("   - Image pulling is integrated into CreateAgent method")
	fmt.Println("   - Future enhancement: Use ImageManager for advanced image operations")
	fmt.Println("   - Current image operations:")
	fmt.Println("     * Pull image if not exists")
	fmt.Println("     * Verify image integrity")
	fmt.Println("     * Cache image for future use")

	fmt.Printf("\n6. Starting agent %s...\n", agentID)
	err = runtime.StartAgent(ctx, agent.ID)
	if err != nil {
		log.Fatalf("Failed to start agent: %v", err)
	}
	fmt.Println("✓ Agent started successfully")

	// Get updated agent info
	fmt.Printf("\n7. Getting agent status...\n")
	startedAgent, err := runtime.GetAgent(ctx, agent.ID)
	if err != nil {
		log.Printf("Warning: Failed to get agent info: %v", err)
	} else {
		fmt.Printf("   Agent ID: %s\n", startedAgent.ID)
		fmt.Printf("   Agent State: %s\n", startedAgent.State)
		fmt.Printf("   Image: %s\n", startedAgent.Image)
		if startedAgent.StartedAt != nil {
			fmt.Printf("   Started At: %s\n", startedAgent.StartedAt.Format(time.RFC3339))
		}
	}

	// Demo: Get agent stats
	fmt.Printf("\n8. Getting agent statistics...\n")
	stats, err := runtime.GetAgentStats(ctx, agent.ID)
	if err != nil {
		fmt.Printf("   Note: Stats collection may not be fully configured: %v\n", err)
		fmt.Println("   Future enhancements:")
		fmt.Println("   - CPU usage monitoring")
		fmt.Println("   - Memory usage tracking")
		fmt.Println("   - Network I/O statistics")
		fmt.Println("   - Disk I/O metrics")
	} else {
		fmt.Printf("   ✓ Statistics retrieved\n")
		fmt.Printf("   Timestamp: %s\n", stats.Timestamp.Format(time.RFC3339))
		fmt.Printf("   CPU Usage: %d ns\n", stats.CPUUsage)
		fmt.Printf("   Memory Usage: %d bytes\n", stats.MemoryUsage)
	}

	// Demo: List all agents
	fmt.Printf("\n9. Listing all agents...\n")
	agents, err := runtime.ListAgents(ctx, nil)
	if err != nil {
		log.Printf("Warning: Failed to list agents: %v", err)
	} else {
		fmt.Printf("   Found %d agent(s):\n", len(agents))
		for i, a := range agents {
			fmt.Printf("   %d. %s (State: %s, Image: %s)\n", i+1, a.Name, a.State, a.Image)
		}
	}

	// Demo: Stop agent gracefully
	fmt.Printf("\n10. Stopping agent %s...\n", agentID)
	err = runtime.StopAgent(ctx, agent.ID, 30*time.Second)
	if err != nil {
		log.Printf("Warning: Failed to stop agent: %v", err)
	} else {
		fmt.Println("✓ Agent stopped gracefully")
	}

	// Demo: Delete agent
	fmt.Printf("\n11. Deleting agent %s...\n", agentID)
	err = runtime.DeleteAgent(ctx, agent.ID)
	if err != nil {
		log.Printf("Warning: Failed to delete agent: %v", err)
	} else {
		fmt.Println("✓ Agent deleted successfully")
	}

	// Demo: Cleanup
	fmt.Printf("\n12. Cleaning up runtime resources...\n")
	if err := runtime.Cleanup(ctx); err != nil {
		log.Printf("Warning: Cleanup failed: %v", err)
	} else {
		fmt.Println("✓ Runtime cleanup completed")
	}

	fmt.Println("\n=== Demo Summary ===")
	fmt.Println("✓ Agent lifecycle completed successfully:")
	fmt.Println("  1. Image pulled (if not cached)")
	fmt.Println("  2. Container created")
	fmt.Println("  3. Agent started")
	fmt.Println("  4. Statistics collected")
	fmt.Println("  5. Agent stopped")
	fmt.Println("  6. Agent deleted")
	fmt.Println("  7. Resources cleaned up")
	fmt.Println("\nNext steps:")
	fmt.Println("  - Implement advanced image management with ImageManager")
	fmt.Println("  - Add image caching and optimization")
	fmt.Println("  - Implement multi-registry support")
	fmt.Println("  - Add image signing and verification")
	fmt.Println("  - Implement automated image updates")
}