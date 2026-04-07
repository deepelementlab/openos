// Package types defines data types and structures for the Agent OS runtime.
package types

import (
	"time"
)

// RuntimeType defines supported runtime types
type RuntimeType string

const (
	// RuntimeContainerd uses containerd as the container runtime
	RuntimeContainerd RuntimeType = "containerd"
	// RuntimeGVisor uses gVisor for enhanced security isolation
	RuntimeGVisor RuntimeType = "gvisor"
	// RuntimeKata uses Kata Containers for VM-level isolation
	RuntimeKata RuntimeType = "kata"
	// RuntimeNative uses native container runtime with cgroups/namespaces
	RuntimeNative RuntimeType = "native"
)

// RuntimeInfo contains information about the runtime implementation
type RuntimeInfo struct {
	// Type of runtime
	Type RuntimeType `json:"type"`
	// Version of the runtime
	Version string `json:"version"`
	// Name of the runtime implementation
	Name string `json:"name"`
	// API version supported
	APIVersion string `json:"api_version"`
	// Features supported by the runtime
	Features []string `json:"features"`
	// Capabilities supported by the runtime
	Capabilities []string `json:"capabilities"`
}

// RuntimeConfig contains configuration for runtime initialization
type RuntimeConfig struct {
	// Type of runtime to use
	Type RuntimeType `json:"type"`
	// Runtime-specific configuration options
	Options map[string]interface{} `json:"options"`
	// Root directory for runtime data
	RootDir string `json:"root_dir"`
	// State directory for runtime state
	StateDir string `json:"state_dir"`
	// Log directory for runtime logs
	LogDir string `json:"log_dir"`
	// Default sandbox configuration
	DefaultSandbox *SandboxSpec `json:"default_sandbox"`
	// Resource limits for the runtime
	ResourceLimits *ResourceLimits `json:"resource_limits"`
	// Security policy for the runtime
	SecurityPolicy *SecurityPolicy `json:"security_policy"`
	// Network configuration
	NetworkConfig *NetworkConfig `json:"network_config"`
	// Storage configuration
	StorageConfig *StorageConfig `json:"storage_config"`
}

// AgentSpec defines the specification for creating an agent
type AgentSpec struct {
	// Unique identifier for the agent
	ID string `json:"id"`
	// Name of the agent
	Name string `json:"name"`
	// Image to use for the agent container
	Image string `json:"image"`
	// Command to run in the agent container
	Command []string `json:"command,omitempty"`
	// Arguments for the command
	Args []string `json:"args,omitempty"`
	// Environment variables
	Env []string `json:"env,omitempty"`
	// Working directory inside container
	WorkingDir string `json:"working_dir,omitempty"`
	// Resource limits for the agent
	Resources *ResourceRequirements `json:"resources"`
	// Security context for the agent
	SecurityContext *SecurityContext `json:"security_context,omitempty"`
	// Network configuration
	NetworkConfig *AgentNetworkConfig `json:"network_config,omitempty"`
	// Storage configuration
	StorageConfig *AgentStorageConfig `json:"storage_config,omitempty"`
	// Labels for the agent
	Labels map[string]string `json:"labels,omitempty"`
	// Annotations for the agent
	Annotations map[string]string `json:"annotations,omitempty"`
	// Sandbox to run the agent in
	SandboxID string `json:"sandbox_id,omitempty"`
	// Restart policy for the agent
	RestartPolicy *RestartPolicy `json:"restart_policy,omitempty"`
	// Health check configuration
	HealthCheck *HealthCheckConfig `json:"health_check,omitempty"`
	// Lifecycle hooks
	LifecycleHooks *LifecycleHooks `json:"lifecycle_hooks,omitempty"`
}

// ResourceRequirements defines resource requirements for an agent
type ResourceRequirements struct {
	// CPU limit in millicores
	CPULimit int64 `json:"cpu_limit,omitempty"`
	// CPU request in millicores (minimum guaranteed)
	CPURequest int64 `json:"cpu_request,omitempty"`
	// Memory limit in bytes
	MemoryLimit int64 `json:"memory_limit,omitempty"`
	// Memory request in bytes (minimum guaranteed)
	MemoryRequest int64 `json:"memory_request,omitempty"`
	// GPU configuration
	GPU *GPUConfig `json:"gpu,omitempty"`
	// IO bandwidth limits
	IOBandwidth *IOBandwidthLimit `json:"io_bandwidth,omitempty"`
	// Network bandwidth limits
	NetworkBandwidth *NetworkBandwidthLimit `json:"network_bandwidth,omitempty"`
	// Process limit
	ProcessLimit int64 `json:"process_limit,omitempty"`
	// File descriptor limit
	FileDescriptorLimit int64 `json:"file_descriptor_limit,omitempty"`
}

// ResourceLimits defines resource limits for the runtime
type ResourceLimits struct {
	// Maximum total CPU available to all agents (millicores)
	TotalCPULimit int64 `json:"total_cpu_limit,omitempty"`
	// Maximum total memory available to all agents (bytes)
	TotalMemoryLimit int64 `json:"total_memory_limit,omitempty"`
	// Maximum number of agents
	MaxAgents int `json:"max_agents,omitempty"`
	// Maximum total disk space (bytes)
	TotalDiskLimit int64 `json:"total_disk_limit,omitempty"`
}

// Agent represents a running agent instance
type Agent struct {
	// Unique identifier
	ID string `json:"id"`
	// Name of the agent
	Name string `json:"name"`
	// Current state of the agent
	State AgentState `json:"state"`
	// Status message
	Status string `json:"status,omitempty"`
	// Image used by the agent
	Image string `json:"image"`
	// Creation timestamp
	CreatedAt time.Time `json:"created_at"`
	// Start timestamp (if running)
	StartedAt *time.Time `json:"started_at,omitempty"`
	// Exit timestamp (if exited)
	FinishedAt *time.Time `json:"finished_at,omitempty"`
	// Exit code (if exited)
	ExitCode *int32 `json:"exit_code,omitempty"`
	// Resource usage statistics
	Stats *ResourceStats `json:"stats,omitempty"`
	// Labels
	Labels map[string]string `json:"labels,omitempty"`
	// Annotations
	Annotations map[string]string `json:"annotations,omitempty"`
	// Sandbox ID
	SandboxID string `json:"sandbox_id,omitempty"`
	// Network configuration
	Network *AgentNetworkInfo `json:"network,omitempty"`
	// Storage mounts
	Mounts []MountInfo `json:"mounts,omitempty"`
}

// AgentState defines possible states of an agent
type AgentState string

const (
	// AgentStateCreated means the agent has been created but not started
	AgentStateCreated AgentState = "created"
	// AgentStateRunning means the agent is running
	AgentStateRunning AgentState = "running"
	// AgentStatePaused means the agent is paused
	AgentStatePaused AgentState = "paused"
	// AgentStateStopped means the agent has been stopped
	AgentStateStopped AgentState = "stopped"
	// AgentStateExited means the agent has exited
	AgentStateExited AgentState = "exited"
	// AgentStateError means the agent is in an error state
	AgentStateError AgentState = "error"
	// AgentStateUnknown means the agent state is unknown
	AgentStateUnknown AgentState = "unknown"
)

// AgentFilter defines filters for listing agents
type AgentFilter struct {
	// Filter by agent state
	State AgentState `json:"state,omitempty"`
	// Filter by sandbox ID
	SandboxID string `json:"sandbox_id,omitempty"`
	// Filter by label selector
	LabelSelector map[string]string `json:"label_selector,omitempty"`
	// Maximum number of agents to return
	Limit int `json:"limit,omitempty"`
}

// Command defines a command to execute in an agent container
type Command struct {
	// Command to execute
	Command []string `json:"command"`
	// Arguments for the command
	Args []string `json:"args,omitempty"`
	// Working directory
	WorkingDir string `json:"working_dir,omitempty"`
	// Environment variables
	Env []string `json:"env,omitempty"`
	// Whether to run in TTY mode
	TTY bool `json:"tty,omitempty"`
	// Stdin content
	Stdin []byte `json:"stdin,omitempty"`
	// Timeout for command execution
	Timeout time.Duration `json:"timeout,omitempty"`
}

// CommandResult contains the result of command execution
type CommandResult struct {
	// Exit code
	ExitCode int32 `json:"exit_code"`
	// Standard output
	Stdout []byte `json:"stdout,omitempty"`
	// Standard error
	Stderr []byte `json:"stderr,omitempty"`
	// Execution time
	Duration time.Duration `json:"duration"`
	// Error if command failed to execute
	Error string `json:"error,omitempty"`
}

// ResourceStats contains resource usage statistics for an agent
type ResourceStats struct {
	// Timestamp of the stats
	Timestamp time.Time `json:"timestamp"`
	// CPU usage in nanoseconds
	CPUUsage uint64 `json:"cpu_usage"`
	// Memory usage in bytes
	MemoryUsage uint64 `json:"memory_usage"`
	// Memory limit in bytes
	MemoryLimit uint64 `json:"memory_limit,omitempty"`
	// Disk usage in bytes
	DiskUsage uint64 `json:"disk_usage,omitempty"`
	// Disk limit in bytes
	DiskLimit uint64 `json:"disk_limit,omitempty"`
	// Network received bytes
	NetworkRxBytes uint64 `json:"network_rx_bytes,omitempty"`
	// Network transmitted bytes
	NetworkTxBytes uint64 `json:"network_tx_bytes,omitempty"`
	// Number of processes
	ProcessCount uint64 `json:"process_count,omitempty"`
	// Number of file descriptors
	FileDescriptorCount uint64 `json:"file_descriptor_count,omitempty"`
}

// SecurityContext defines security context for an agent
type SecurityContext struct {
	// Whether to run in privileged mode
	Privileged bool `json:"privileged,omitempty"`
	// User ID to run as
	RunAsUser *int64 `json:"run_as_user,omitempty"`
	// Group ID to run as
	RunAsGroup *int64 `json:"run_as_group,omitempty"`
	// Additional groups
	RunAsGroups []int64 `json:"run_as_groups,omitempty"`
	// Read-only root filesystem
	ReadOnlyRootFS bool `json:"read_only_rootfs,omitempty"`
	// SELinux context
	SELinuxOptions *SELinuxOptions `json:"selinux_options,omitempty"`
	// AppArmor profile
	AppArmorProfile string `json:"apparmor_profile,omitempty"`
	// Seccomp profile
	SeccompProfile *SeccompProfile `json:"seccomp_profile,omitempty"`
	// Capabilities to add
	AddCapabilities []string `json:"add_capabilities,omitempty"`
	// Capabilities to drop
	DropCapabilities []string `json:"drop_capabilities,omitempty"`
	// No new privileges
	NoNewPrivileges bool `json:"no_new_privileges,omitempty"`
}

// SecurityPolicy defines security policy for sandbox or runtime
type SecurityPolicy struct {
	// Default security context
	DefaultContext *SecurityContext `json:"default_context,omitempty"`
	// Allowed capabilities
	AllowedCapabilities []string `json:"allowed_capabilities,omitempty"`
	// Required drop capabilities
	RequiredDropCapabilities []string `json:"required_drop_capabilities,omitempty"`
	// Allowed syscalls
	AllowedSyscalls []string `json:"allowed_syscalls,omitempty"`
	// Forbidden syscalls
	ForbiddenSyscalls []string `json:"forbidden_syscalls,omitempty"`
	// Resource access control rules
	ResourceAccessRules []ResourceAccessRule `json:"resource_access_rules,omitempty"`
	// Network policy
	NetworkPolicy *NetworkPolicy `json:"network_policy,omitempty"`
	// Filesystem access policy
	FilesystemPolicy *FilesystemPolicy `json:"filesystem_policy,omitempty"`
}

// NetworkPolicy defines network security policy
type NetworkPolicy struct {
	// Whether to allow all network access
	AllowAll bool `json:"allow_all,omitempty"`
	// Allowed ingress ports
	AllowedIngressPorts []PortRange `json:"allowed_ingress_ports,omitempty"`
	// Allowed egress ports
	AllowedEgressPorts []PortRange `json:"allowed_egress_ports,omitempty"`
	// Allowed IP ranges
	AllowedIPRanges []string `json:"allowed_ip_ranges,omitempty"`
	// Denied IP ranges
	DeniedIPRanges []string `json:"denied_ip_ranges,omitempty"`
}

// PortRange defines a range of ports
type PortRange struct {
	// Start port
	Start uint32 `json:"start"`
	// End port
	End uint32 `json:"end"`
}

// More type definitions continue in next files...