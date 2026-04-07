package types

// Security-related types for Agent OS runtime

// SELinuxOptions defines SELinux context options
type SELinuxOptions struct {
	// SELinux user label
	User string `json:"user,omitempty"`
	// SELinux role label
	Role string `json:"role,omitempty"`
	// SELinux type label
	Type string `json:"type,omitempty"`
	// SELinux level label
	Level string `json:"level,omitempty"`
}

// SeccompProfile defines seccomp profile configuration
type SeccompProfile struct {
	// Profile name or type
	ProfileType string `json:"profile_type"`
	// Custom profile data
	Data string `json:"data,omitempty"`
}

// ResourceAccessRule defines access control rules for resources
type ResourceAccessRule struct {
	// Resource type (cpu, memory, disk, network, etc.)
	ResourceType string `json:"resource_type"`
	// Action to take (allow, deny, limit)
	Action string `json:"action"`
	// Limit value (for limit action)
	Limit *int64 `json:"limit,omitempty"`
	// Condition for applying the rule
	Condition *RuleCondition `json:"condition,omitempty"`
}

// RuleCondition defines conditions for applying security rules
type RuleCondition struct {
	// Agent labels to match
	LabelSelector map[string]string `json:"label_selector,omitempty"`
	// Time window for the condition
	TimeWindow *TimeWindow `json:"time_window,omitempty"`
	// Resource usage threshold
	ResourceThreshold *ResourceThreshold `json:"resource_threshold,omitempty"`
}

// TimeWindow defines a time window for security rules
type TimeWindow struct {
	// Start time
	Start string `json:"start"` // Format: "HH:MM"
	// End time
	End string `json:"end"` // Format: "HH:MM"
	// Days of week (0-6, Sunday=0)
	Days []int `json:"days,omitempty"`
}

// ResourceThreshold defines threshold for resource usage conditions
type ResourceThreshold struct {
	// CPU usage percentage threshold
	CPUPercent *float64 `json:"cpu_percent,omitempty"`
	// Memory usage percentage threshold
	MemoryPercent *float64 `json:"memory_percent,omitempty"`
	// Disk usage percentage threshold
	DiskPercent *float64 `json:"disk_percent,omitempty"`
}

// FilesystemPolicy defines filesystem access policy
type FilesystemPolicy struct {
	// Read-only paths
	ReadOnlyPaths []string `json:"read_only_paths,omitempty"`
	// Writable paths
	WritablePaths []string `json:"writable_paths,omitempty"`
	// Forbidden paths
	ForbiddenPaths []string `json:"forbidden_paths,omitempty"`
	// Executable paths (allowed to execute)
	ExecutablePaths []string `json:"executable_paths,omitempty"`
	// Mount points allowed
	AllowedMounts []string `json:"allowed_mounts,omitempty"`
}

// GPUConfig defines GPU resource configuration
type GPUConfig struct {
	// Type of GPU
	Type string `json:"type,omitempty"`
	// GPU device count
	Count int `json:"count,omitempty"`
	// GPU memory limit in bytes
	MemoryLimit int64 `json:"memory_limit,omitempty"`
	// GPU model requirements
	Model string `json:"model,omitempty"`
}

// IOBandwidthLimit defines IO bandwidth limits
type IOBandwidthLimit struct {
	// Read bandwidth limit in bytes per second
	ReadBPS int64 `json:"read_bps,omitempty"`
	// Write bandwidth limit in bytes per second
	WriteBPS int64 `json:"write_bps,omitempty"`
	// Read IOPS limit
	ReadIOPS int64 `json:"read_iops,omitempty"`
	// Write IOPS limit
	WriteIOPS int64 `json:"write_iops,omitempty"`
}

// NetworkBandwidthLimit defines network bandwidth limits
type NetworkBandwidthLimit struct {
	// Ingress bandwidth limit in bytes per second
	IngressBPS int64 `json:"ingress_bps,omitempty"`
	// Egress bandwidth limit in bytes per second
	EgressBPS int64 `json:"egress_bps,omitempty"`
}

// NetworkConfig defines runtime network configuration
type NetworkConfig struct {
	// Network type (bridge, overlay, host, none)
	Type string `json:"type"`
	// Network bridge name
	BridgeName string `json:"bridge_name,omitempty"`
	// Bridge IP address
	BridgeIP string `json:"bridge_ip,omitempty"`
	// Subnet for the network
	Subnet string `json:"subnet,omitempty"`
	// Gateway address
	Gateway string `json:"gateway,omitempty"`
	// DNS servers
	DNSServers []string `json:"dns_servers,omitempty"`
	// DNS search domains
	DNSSearchDomains []string `json:"dns_search_domains,omitempty"`
	// MTU size
	MTU int `json:"mtu,omitempty"`
	// Network plugin to use
	Plugin string `json:"plugin,omitempty"`
}

// StorageConfig defines runtime storage configuration
type StorageConfig struct {
	// Storage driver to use
	Driver string `json:"driver"`
	// Root directory for storage
	RootDir string `json:"root_dir,omitempty"`
	// Image store configuration
	ImageStore *ImageStoreConfig `json:"image_store,omitempty"`
	// Volume store configuration
	VolumeStore *VolumeStoreConfig `json:"volume_store,omitempty"`
	// Snapshot configuration
	SnapshotConfig *SnapshotConfig `json:"snapshot_config,omitempty"`
}

// ImageStoreConfig defines image store configuration
type ImageStoreConfig struct {
	// Type of image store (filesystem, s3, registry)
	Type string `json:"type"`
	// Image store endpoint
	Endpoint string `json:"endpoint,omitempty"`
	// Authentication credentials
	Credentials *AuthCredentials `json:"credentials,omitempty"`
	// Cache size limit
	CacheSizeLimit int64 `json:"cache_size_limit,omitempty"`
}

// VolumeStoreConfig defines volume store configuration
type VolumeStoreConfig struct {
	// Type of volume store (local, nfs, ceph, s3)
	Type string `json:"type"`
	// Volume store endpoint
	Endpoint string `json:"endpoint,omitempty"`
	// Authentication credentials
	Credentials *AuthCredentials `json:"credentials,omitempty"`
	// Volume size limit
	SizeLimit int64 `json:"size_limit,omitempty"`
}

// SnapshotConfig defines snapshot configuration
type SnapshotConfig struct {
	// Snapshot driver to use
	Driver string `json:"driver"`
	// Snapshot retention policy
	RetentionPolicy *RetentionPolicy `json:"retention_policy,omitempty"`
}

// RetentionPolicy defines snapshot retention policy
type RetentionPolicy struct {
	// Maximum number of snapshots to keep
	MaxSnapshots int `json:"max_snapshots,omitempty"`
	// Maximum age of snapshots to keep
	MaxAge string `json:"max_age,omitempty"`
	// Whether to keep at least one snapshot
	KeepAtLeastOne bool `json:"keep_at_least_one,omitempty"`
}

// AuthCredentials defines authentication credentials
type AuthCredentials struct {
	// Username
	Username string `json:"username,omitempty"`
	// Password
	Password string `json:"password,omitempty"`
	// Token
	Token string `json:"token,omitempty"`
	// Certificate file path
	CertFile string `json:"cert_file,omitempty"`
	// Key file path
	KeyFile string `json:"key_file,omitempty"`
	// CA certificate file path
	CAFile string `json:"ca_file,omitempty"`
}