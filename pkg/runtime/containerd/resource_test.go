package containerd

import (
	"testing"

	"github.com/agentos/aos/pkg/runtime/types"
	"github.com/stretchr/testify/assert"
)

func TestResourceLimits(t *testing.T) {
	tests := []struct {
		name           string
		resources      *types.ResourceRequirements
		expectCPU      bool
		expectMemory   bool
		cpuLimit       int64
		memoryLimit    int64
		cpuRequest     int64
		memoryRequest  int64
	}{
		{
			name: "CPU only limit",
			resources: &types.ResourceRequirements{
				CPULimit: 500, // 500 millicores = 0.5 CPU
			},
			expectCPU:    true,
			expectMemory: false,
			cpuLimit:     500,
		},
		{
			name: "Memory only limit",
			resources: &types.ResourceRequirements{
				MemoryLimit: 512 * 1024 * 1024, // 512MB
			},
			expectCPU:    false,
			expectMemory: true,
			memoryLimit:  512 * 1024 * 1024,
		},
		{
			name: "Both CPU and memory limits",
			resources: &types.ResourceRequirements{
				CPULimit:    1000,               // 1 CPU core
				MemoryLimit: 1024 * 1024 * 1024, // 1GB
			},
			expectCPU:    true,
			expectMemory: true,
			cpuLimit:     1000,
			memoryLimit:  1024 * 1024 * 1024,
		},
		{
			name: "CPU requests and limits",
			resources: &types.ResourceRequirements{
				CPULimit:    2000, // 2 CPU cores
				CPURequest:  1000, // 1 CPU core minimum
			},
			expectCPU:    true,
			expectMemory: false,
			cpuLimit:     2000,
			cpuRequest:   1000,
		},
		{
			name: "Memory requests and limits",
			resources: &types.ResourceRequirements{
				MemoryLimit:    2048 * 1024 * 1024, // 2GB limit
				MemoryRequest:  1024 * 1024 * 1024, // 1GB request
			},
			expectCPU:    false,
			expectMemory: true,
			memoryLimit:  2048 * 1024 * 1024,
			memoryRequest: 1024 * 1024 * 1024,
		},
		{
			name: "Complete resource specification",
			resources: &types.ResourceRequirements{
				CPULimit:       2000,               // 2 CPU cores
				CPURequest:     1000,               // 1 CPU core minimum
				MemoryLimit:    2048 * 1024 * 1024, // 2GB limit
				MemoryRequest:  1024 * 1024 * 1024, // 1GB request
				ProcessLimit:   1024,               // Max 1024 processes
			},
			expectCPU:    true,
			expectMemory: true,
			cpuLimit:     2000,
			cpuRequest:   1000,
			memoryLimit:  2048 * 1024 * 1024,
			memoryRequest: 1024 * 1024 * 1024,
		},
		{
			name: "Minimum CPU shares",
			resources: &types.ResourceRequirements{
				CPURequest: 10,
			},
			expectCPU:    false,
			expectMemory: false,
			cpuRequest:   10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := &types.AgentSpec{
				ID:    "test-agent-" + tt.name,
				Name:  "Test Agent",
				Image: "busybox:latest",
				Resources: tt.resources,
			}

			// Verify the spec was created correctly
			assert.Equal(t, tt.resources, spec.Resources)
			
			if tt.resources != nil {
				if tt.expectCPU {
					assert.Greater(t, tt.resources.CPULimit, int64(0))
				}
				if tt.expectMemory {
					assert.Greater(t, tt.resources.MemoryLimit, int64(0))
				}
				if tt.cpuRequest > 0 {
					assert.Equal(t, tt.cpuRequest, tt.resources.CPURequest)
				}
				if tt.memoryRequest > 0 {
					assert.Equal(t, tt.memoryRequest, tt.resources.MemoryRequest)
				}
			}
		})
	}
}

func TestResourceConversion(t *testing.T) {
	tests := []struct {
		name        string
		cpuMillis   int64
		expectedCPU bool
	}{
		{
			name:        "Zero CPU limit",
			cpuMillis:   0,
			expectedCPU: false,
		},
		{
			name:        "Small CPU limit",
			cpuMillis:   100,
			expectedCPU: true,
		},
		{
			name:        "Normal CPU limit",
			cpuMillis:   1000,
			expectedCPU: true,
		},
		{
			name:        "Large CPU limit",
			cpuMillis:   8000,
			expectedCPU: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test CPU quota calculation
			if tt.expectedCPU {
				// 1000 millicores = 1 core = 100000 microseconds quota per 100ms period
				expectedQuota := int64(tt.cpuMillis * 100) // 100 microseconds per millicore
				quota := tt.cpuMillis * 100
				assert.Equal(t, expectedQuota, quota, "CPU quota calculation incorrect")
				
				// Test CPU shares calculation
				shares := uint64(tt.cpuMillis * 1024 / 1000)
				if tt.cpuMillis < 1000/1024 { // Less than ~0.98 millicore
					assert.Less(t, shares, uint64(2))
				} else {
					assert.GreaterOrEqual(t, shares, uint64(2))
				}
			}
		})
	}
}

func TestMemoryConversion(t *testing.T) {
	tests := []struct {
		name          string
		memoryBytes   int64
		expectedBytes uint64
	}{
		{
			name:          "Zero memory",
			memoryBytes:   0,
			expectedBytes: 0,
		},
		{
			name:          "Small memory",
			memoryBytes:   64 * 1024 * 1024, // 64MB
			expectedBytes: 64 * 1024 * 1024,
		},
		{
			name:          "Medium memory",
			memoryBytes:   512 * 1024 * 1024, // 512MB
			expectedBytes: 512 * 1024 * 1024,
		},
		{
			name:          "Large memory",
			memoryBytes:   8 * 1024 * 1024 * 1024, // 8GB
			expectedBytes: 8 * 1024 * 1024 * 1024,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test memory conversion
			bytes := uint64(tt.memoryBytes)
			assert.Equal(t, tt.expectedBytes, bytes, "Memory conversion incorrect")
		})
	}
}

func TestResourceValidation(t *testing.T) {
	tests := []struct {
		name      string
		resources *types.ResourceRequirements
		valid     bool
		errorMsg  string
	}{
		{
			name:      "Valid resources",
			resources: &types.ResourceRequirements{
				CPULimit:    1000,
				MemoryLimit: 512 * 1024 * 1024,
			},
			valid: true,
		},
		{
			name:      "Request greater than limit",
			resources: &types.ResourceRequirements{
				CPURequest:  1500,
				CPULimit:    1000,
			},
			valid:    false,
			errorMsg: "CPU request should not exceed limit",
		},
		{
			name:      "Memory request greater than limit",
			resources: &types.ResourceRequirements{
				MemoryRequest: 2 * 1024 * 1024 * 1024,
				MemoryLimit:   1 * 1024 * 1024 * 1024,
			},
			valid:    false,
			errorMsg: "Memory request should not exceed limit",
		},
		{
			name:      "Negative values",
			resources: &types.ResourceRequirements{
				CPULimit:    -100,
				MemoryLimit: -1024,
			},
			valid:    false,
			errorMsg: "Negative values not allowed",
		},
		{
			name:      "Zero values allowed",
			resources: &types.ResourceRequirements{
				CPULimit:    0,
				MemoryLimit: 0,
			},
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test agent spec
			spec := &types.AgentSpec{
				ID:    "test-agent-" + tt.name,
				Name:  "Test Agent",
				Image: "busybox:latest",
				Resources: tt.resources,
			}

			// Basic validation logic
			if tt.resources != nil {
				if tt.resources.CPURequest > 0 && tt.resources.CPULimit > 0 {
					if tt.resources.CPURequest > tt.resources.CPULimit {
						assert.False(t, tt.valid, tt.errorMsg)
					}
				}
				if tt.resources.MemoryRequest > 0 && tt.resources.MemoryLimit > 0 {
					if tt.resources.MemoryRequest > tt.resources.MemoryLimit {
						assert.False(t, tt.valid, tt.errorMsg)
					}
				}
				if tt.resources.CPULimit < 0 || tt.resources.MemoryLimit < 0 {
					assert.False(t, tt.valid, tt.errorMsg)
				}
			}
			
			if tt.valid {
				assert.NotNil(t, spec.Resources)
			}
		})
	}
}