package types

import (
	"time"
)

// Lifecycle-related types for Agent OS runtime

// RestartPolicy defines the restart policy for an agent
type RestartPolicy struct {
	// Restart policy type
	Type RestartPolicyType `json:"type"`
	// Maximum number of retries
	MaximumRetryCount int `json:"maximum_retry_count,omitempty"`
	// Delay between restarts
	RestartDelay time.Duration `json:"restart_delay,omitempty"`
	// Maximum restart delay
	MaxRestartDelay time.Duration `json:"max_restart_delay,omitempty"`
	// Restart window (time window for restart counting)
	RestartWindow time.Duration `json:"restart_window,omitempty"`
}

// RestartPolicyType defines types of restart policies
type RestartPolicyType string

const (
	// RestartPolicyNever means never restart the agent
	RestartPolicyNever RestartPolicyType = "never"
	// RestartPolicyAlways means always restart the agent
	RestartPolicyAlways RestartPolicyType = "always"
	// RestartPolicyOnFailure means restart only on failure
	RestartPolicyOnFailure RestartPolicyType = "on-failure"
	// RestartPolicyUnlessStopped means restart unless explicitly stopped
	RestartPolicyUnlessStopped RestartPolicyType = "unless-stopped"
)

// HealthCheckConfig defines health check configuration for an agent
type HealthCheckConfig struct {
	// Health check test command
	Test []string `json:"test,omitempty"`
	// Health check interval
	Interval time.Duration `json:"interval,omitempty"`
	// Health check timeout
	Timeout time.Duration `json:"timeout,omitempty"`
	// Health check start period
	StartPeriod time.Duration `json:"start_period,omitempty"`
	// Number of retries before marking unhealthy
	Retries int `json:"retries,omitempty"`
	// Command health check
	Command *CommandHealthCheck `json:"command,omitempty"`
	// HTTP health check
	HTTP *HTTPHealthCheck `json:"http,omitempty"`
	// TCP health check
	TCP *TCPHealthCheck `json:"tcp,omitempty"`
	// Exec health check
	Exec *ExecHealthCheck `json:"exec,omitempty"`
}

// CommandHealthCheck defines command-based health check
type CommandHealthCheck struct {
	// Command to execute
	Command []string `json:"command"`
	// Arguments for the command
	Args []string `json:"args,omitempty"`
	// Working directory
	WorkingDir string `json:"working_dir,omitempty"`
	// Environment variables
	Env []string `json:"env,omitempty"`
}

// HTTPHealthCheck defines HTTP-based health check
type HTTPHealthCheck struct {
	// HTTP endpoint to check
	Endpoint string `json:"endpoint"`
	// HTTP method to use (GET, POST, etc.)
	Method string `json:"method,omitempty"`
	// HTTP headers to include
	Headers map[string]string `json:"headers,omitempty"`
	// Expected status codes
	ExpectedStatusCodes []int `json:"expected_status_codes,omitempty"`
	// Expected response body pattern
	ExpectedResponseBody string `json:"expected_response_body,omitempty"`
}

// TCPHealthCheck defines TCP-based health check
type TCPHealthCheck struct {
	// Host to connect to
	Host string `json:"host"`
	// Port to connect to
	Port int `json:"port"`
	// Timeout for connection
	Timeout time.Duration `json:"timeout,omitempty"`
}

// ExecHealthCheck defines exec-based health check
type ExecHealthCheck struct {
	// Command to execute
	Command []string `json:"command"`
	// Arguments for the command
	Args []string `json:"args,omitempty"`
	// Working directory
	WorkingDir string `json:"working_dir,omitempty"`
	// Environment variables
	Env []string `json:"env,omitempty"`
}

// LifecycleHooks defines lifecycle hooks for an agent
type LifecycleHooks struct {
	// Pre-create hook
	PreCreate *LifecycleHook `json:"pre_create,omitempty"`
	// Post-create hook
	PostCreate *LifecycleHook `json:"post_create,omitempty"`
	// Pre-start hook
	PreStart *LifecycleHook `json:"pre_start,omitempty"`
	// Post-start hook
	PostStart *LifecycleHook `json:"post_start,omitempty"`
	// Pre-stop hook
	PreStop *LifecycleHook `json:"pre_stop,omitempty"`
	// Post-stop hook
	PostStop *LifecycleHook `json:"post_stop,omitempty"`
	// Pre-delete hook
	PreDelete *LifecycleHook `json:"pre_delete,omitempty"`
	// Post-delete hook
	PostDelete *LifecycleHook `json:"post_delete,omitempty"`
}

// LifecycleHook defines a single lifecycle hook
type LifecycleHook struct {
	// Command to execute
	Command []string `json:"command"`
	// Arguments for the command
	Args []string `json:"args,omitempty"`
	// Working directory
	WorkingDir string `json:"working_dir,omitempty"`
	// Environment variables
	Env []string `json:"env,omitempty"`
	// Timeout for the hook
	Timeout time.Duration `json:"timeout,omitempty"`
	// Whether to ignore hook failures
	IgnoreFailure bool `json:"ignore_failure,omitempty"`
}

// AgentNetworkConfig defines network configuration for an agent
type AgentNetworkConfig struct {
	// Network mode (bridge, host, none, container:<id>)
	Mode string `json:"mode"`
	// Network to join
	NetworkID string `json:"network_id,omitempty"`
	// IP address to assign
	IPAddress string `json:"ip_address,omitempty"`
	// MAC address to assign
	MACAddress string `json:"mac_address,omitempty"`
	// Port mappings
	PortMappings []PortMapping `json:"port_mappings,omitempty"`
	// DNS servers
	DNSServers []string `json:"dns_servers,omitempty"`
	// DNS search domains
	DNSSearchDomains []string `json:"dns_search_domains,omitempty"`
	// Extra hosts
	ExtraHosts []string `json:"extra_hosts,omitempty"`
	// Network aliases
	Aliases []string `json:"aliases,omitempty"`
}

// PortMapping defines port mapping configuration
type PortMapping struct {
	// Protocol (tcp, udp)
	Protocol string `json:"protocol"`
	// Host IP address to bind to
	HostIP string `json:"host_ip,omitempty"`
	// Host port number
	HostPort int `json:"host_port"`
	// Container port number
	ContainerPort int `json:"container_port"`
	// Port range
	PortRange *PortRange `json:"port_range,omitempty"`
}

// AgentStorageConfig defines storage configuration for an agent
type AgentStorageConfig struct {
	// Storage mounts
	Mounts []MountSpec `json:"mounts,omitempty"`
	// Volumes to create
	Volumes []VolumeSpec `json:"volumes,omitempty"`
	// Tmpfs mounts
	Tmpfs []TmpfsMount `json:"tmpfs,omitempty"`
	// Storage driver options
	DriverOptions map[string]string `json:"driver_options,omitempty"`
}

// MountSpec defines a filesystem mount specification
type MountSpec struct {
	// Source of the mount
	Source string `json:"source"`
	// Target mount point
	Target string `json:"target"`
	// Mount type (bind, volume, tmpfs, nfs, etc.)
	Type string `json:"type"`
	// Mount options
	Options []string `json:"options,omitempty"`
	// Whether the mount is read-only
	ReadOnly bool `json:"read_only,omitempty"`
	// Bind propagation mode
	BindPropagation string `json:"bind_propagation,omitempty"`
	// Consistency mode
	Consistency string `json:"consistency,omitempty"`
}

// VolumeSpec defines a volume specification
type VolumeSpec struct {
	// Volume name
	Name string `json:"name"`
	// Volume driver
	Driver string `json:"driver,omitempty"`
	// Driver options
	DriverOptions map[string]string `json:"driver_options,omitempty"`
	// Labels
	Labels map[string]string `json:"labels,omitempty"`
	// Size limit
	SizeLimit int64 `json:"size_limit,omitempty"`
	// Whether to create the volume if it doesn't exist
	CreateIfMissing bool `json:"create_if_missing,omitempty"`
}

// TmpfsMount defines a tmpfs mount specification
type TmpfsMount struct {
	// Mount point
	Target string `json:"target"`
	// Size limit in bytes
	SizeBytes int64 `json:"size_bytes,omitempty"`
	// Mount mode
	Mode uint32 `json:"mode,omitempty"`
	// Mount options
	Options []string `json:"options,omitempty"`
}

// LogOptions defines options for retrieving logs
type LogOptions struct {
	// Whether to follow logs
	Follow bool `json:"follow,omitempty"`
	// Whether to show timestamps
	Timestamps bool `json:"timestamps,omitempty"`
	// Number of lines to show from the end
	Tail int `json:"tail,omitempty"`
	// Show logs since timestamp
	Since time.Time `json:"since,omitempty"`
	// Show logs until timestamp
	Until time.Time `json:"until,omitempty"`
}

// AttachOptions defines options for attaching to an agent
type AttachOptions struct {
	// Whether to attach stdin
	Stdin bool `json:"stdin,omitempty"`
	// Whether to attach stdout
	Stdout bool `json:"stdout,omitempty"`
	// Whether to attach stderr
	Stderr bool `json:"stderr,omitempty"`
	// Whether to attach in TTY mode
	TTY bool `json:"tty,omitempty"`
	// Terminal size
	TerminalSize *TerminalSize `json:"terminal_size,omitempty"`
}

// TerminalSize defines terminal dimensions
type TerminalSize struct {
	// Width in characters
	Width uint `json:"width"`
	// Height in characters
	Height uint `json:"height"`
}

// SandboxSpec defines specification for creating a sandbox
type SandboxSpec struct {
	// Unique identifier
	ID string `json:"id"`
	// Name of the sandbox
	Name string `json:"name"`
	// Sandbox type
	Type string `json:"type"`
	// Security policy
	SecurityPolicy *SecurityPolicy `json:"security_policy,omitempty"`
	// Network configuration
	Network *NetworkSpec `json:"network,omitempty"`
	// Resource limits
	ResourceLimits *ResourceLimits `json:"resource_limits,omitempty"`
	// Labels
	Labels map[string]string `json:"labels,omitempty"`
	// Annotations
	Annotations map[string]string `json:"annotations,omitempty"`
}

// NetworkSpec defines specification for creating a network
type NetworkSpec struct {
	// Unique identifier
	ID string `json:"id"`
	// Name of the network
	Name string `json:"name"`
	// Network type
	Type string `json:"type"`
	// Subnet
	Subnet string `json:"subnet,omitempty"`
	// Gateway
	Gateway string `json:"gateway,omitempty"`
	// IP range
	IPRange string `json:"ip_range,omitempty"`
	// Labels
	Labels map[string]string `json:"labels,omitempty"`
	// Options
	Options map[string]string `json:"options,omitempty"`
}

// VolumeSpec defines specification for creating a volume (duplicate from above, but needed for interface)
// Already defined in this file

// Sandbox represents a sandbox environment
type Sandbox struct {
	// Unique identifier
	ID string `json:"id"`
	// Name of the sandbox
	Name string `json:"name"`
	// Sandbox type
	Type string `json:"type"`
	// Security policy
	SecurityPolicy *SecurityPolicy `json:"security_policy,omitempty"`
	// Network information
	Network *Network `json:"network,omitempty"`
	// Resource usage
	ResourceUsage *ResourceStats `json:"resource_usage,omitempty"`
	// Agent count
	AgentCount int `json:"agent_count"`
	// Creation timestamp
	CreatedAt time.Time `json:"created_at"`
	// Labels
	Labels map[string]string `json:"labels,omitempty"`
	// Annotations
	Annotations map[string]string `json:"annotations,omitempty"`
}

// Network represents a network
type Network struct {
	// Unique identifier
	ID string `json:"id"`
	// Name of the network
	Name string `json:"name"`
	// Network type
	Type string `json:"type"`
	// Subnet
	Subnet string `json:"subnet,omitempty"`
	// Gateway
	Gateway string `json:"gateway,omitempty"`
	// IP range
	IPRange string `json:"ip_range,omitempty"`
	// Labels
	Labels map[string]string `json:"labels,omitempty"`
	// Options
	Options map[string]string `json:"options,omitempty"`
}

// Volume represents a volume
type Volume struct {
	// Unique identifier
	ID string `json:"id"`
	// Name of the volume
	Name string `json:"name"`
	// Volume driver
	Driver string `json:"driver,omitempty"`
	// Mount point
	MountPoint string `json:"mount_point,omitempty"`
	// Size in bytes
	Size int64 `json:"size,omitempty"`
	// Used space in bytes
	Used int64 `json:"used,omitempty"`
	// Labels
	Labels map[string]string `json:"labels,omitempty"`
	// Options
	Options map[string]string `json:"options,omitempty"`
	// Creation timestamp
	CreatedAt time.Time `json:"created_at"`
}

// AgentNetworkInfo contains network information for an agent
type AgentNetworkInfo struct {
	// IP address
	IPAddress string `json:"ip_address,omitempty"`
	// MAC address
	MACAddress string `json:"mac_address,omitempty"`
	// Network ID
	NetworkID string `json:"network_id,omitempty"`
	// Network name
	NetworkName string `json:"network_name,omitempty"`
	// Port mappings
	PortMappings []PortMapping `json:"port_mappings,omitempty"`
}

// MountInfo contains information about a mount
type MountInfo struct {
	// Source of the mount
	Source string `json:"source"`
	// Target mount point
	Target string `json:"target"`
	// Mount type
	Type string `json:"type"`
	// Mount options
	Options []string `json:"options,omitempty"`
	// Whether the mount is read-only
	ReadOnly bool `json:"read_only"`
}