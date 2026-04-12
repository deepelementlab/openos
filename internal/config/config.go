package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/viper"
)

// Config represents the application configuration
type Config struct {
	Server     ServerConfig     `mapstructure:"server"`
	Logging    LoggingConfig    `mapstructure:"logging"`
	Database   DatabaseConfig   `mapstructure:"database"`
	Redis      RedisConfig      `mapstructure:"redis"`
	Container  ContainerConfig  `mapstructure:"container"`
	Scheduler  SchedulerConfig  `mapstructure:"scheduler"`
	Security   SecurityConfig   `mapstructure:"security"`
	Monitoring MonitoringConfig `mapstructure:"monitoring"`
	Agent      AgentConfig      `mapstructure:"agent"`
	API        APIConfig        `mapstructure:"api"`
	Cluster    ClusterConfig    `mapstructure:"cluster"`
	Storage    StorageConfig    `mapstructure:"storage"`
	Network    NetworkConfig    `mapstructure:"network"`
	Features   FeaturesConfig   `mapstructure:"features"`
	Messaging  MessagingConfig  `mapstructure:"messaging"`

	// Global settings
	Mode  string `mapstructure:"mode"`
	Debug bool   `mapstructure:"debug"`
}

// ServerConfig holds server-specific configuration
type ServerConfig struct {
	Host                     string        `mapstructure:"host"`
	Port                     int           `mapstructure:"port"`
	Mode                     string        `mapstructure:"mode"`
	GracefulShutdownTimeout  time.Duration `mapstructure:"graceful_shutdown_timeout"`
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level      string `mapstructure:"level"`
	Format     string `mapstructure:"format"`
	Output     string `mapstructure:"output"`
	FilePath   string `mapstructure:"file_path"`
	MaxSize    int    `mapstructure:"max_size"`
	MaxBackups int    `mapstructure:"max_backups"`
	MaxAge     int    `mapstructure:"max_age"`
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Driver         string `mapstructure:"driver"`
	Host           string `mapstructure:"host"`
	Port           int    `mapstructure:"port"`
	Name           string `mapstructure:"name"`
	Username       string `mapstructure:"username"`
	Password       string `mapstructure:"password"`
	SSLMode        string `mapstructure:"ssl_mode"`
	MaxOpenConns   int    `mapstructure:"max_open_conns"`
	MaxIdleConns   int    `mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
}

// RedisConfig holds Redis configuration
type RedisConfig struct {
	Host         string `mapstructure:"host"`
	Port         int    `mapstructure:"port"`
	Password     string `mapstructure:"password"`
	DB           int    `mapstructure:"db"`
	PoolSize     int    `mapstructure:"pool_size"`
	MinIdleConns int    `mapstructure:"min_idle_conns"`
}

// ContainerConfig holds container runtime configuration
type ContainerConfig struct {
	Runtime                string `mapstructure:"runtime"`
	Socket                 string `mapstructure:"socket"`
	NetworkDriver          string `mapstructure:"network_driver"`
	StorageDriver          string `mapstructure:"storage_driver"`
	DefaultMemoryLimit     string `mapstructure:"default_memory_limit"`
	DefaultCPULimit        string `mapstructure:"default_cpu_limit"`
	DefaultPidsLimit       int    `mapstructure:"default_pids_limit"`
	DefaultNetworkBandwidth string `mapstructure:"default_network_bandwidth"`
}

// SchedulerConfig holds scheduler configuration
type SchedulerConfig struct {
	Algorithm           string `mapstructure:"algorithm"`
	MaxConcurrentAgents int    `mapstructure:"max_concurrent_agents"`
	MaxAgentsPerHost    int    `mapstructure:"max_agents_per_host"`
	CheckInterval       time.Duration `mapstructure:"check_interval"`
	AutoScale           bool   `mapstructure:"auto_scale"`
	ScaleUpThreshold    int    `mapstructure:"scale_up_threshold"`
	ScaleDownThreshold  int    `mapstructure:"scale_down_threshold"`
}

// SecurityConfig holds security configuration
type SecurityConfig struct {
	JWTSecret            string   `mapstructure:"jwt_secret"`
	JWTExpiry            int      `mapstructure:"jwt_expiry"`
	CORSEnabled          bool     `mapstructure:"cors_enabled"`
	CORSAllowedOrigins   []string `mapstructure:"cors_allowed_origins"`
	RateLimitEnabled     bool     `mapstructure:"rate_limit_enabled"`
	RateLimitRequests    int      `mapstructure:"rate_limit_requests"`
	RateLimitBurst       int      `mapstructure:"rate_limit_burst"`
}

// MonitoringConfig holds monitoring configuration
type MonitoringConfig struct {
	Enabled                  bool   `mapstructure:"enabled"`
	PrometheusEnabled        bool   `mapstructure:"prometheus_enabled"`
	PrometheusPath           string `mapstructure:"prometheus_path"`
	HealthCheckPath          string `mapstructure:"health_check_path"`
	MetricsCollectionInterval time.Duration `mapstructure:"metrics_collection_interval"`
	AlertingEnabled          bool   `mapstructure:"alerting_enabled"`
	AlertManagerURL          string `mapstructure:"alert_manager_url"`
}

// AgentConfig holds agent-specific configuration
type AgentConfig struct {
	DefaultImage         string   `mapstructure:"default_image"`
	DefaultWorkingDir    string   `mapstructure:"default_working_dir"`
	AllowedRegistries    []string `mapstructure:"allowed_registries"`
	MaxImageSize         string   `mapstructure:"max_image_size"`
	MaxExecutionTime     time.Duration `mapstructure:"max_execution_time"`
	AutoRestart          bool     `mapstructure:"auto_restart"`
	RestartPolicy        string   `mapstructure:"restart_policy"`
}

// APIConfig holds API configuration
type APIConfig struct {
	SwaggerEnabled bool   `mapstructure:"swagger_enabled"`
	SwaggerPath    string `mapstructure:"swagger_path"`
	APIPrefix      string `mapstructure:"api_prefix"`
	EnableCaching  bool   `mapstructure:"enable_caching"`
	CacheDuration  time.Duration `mapstructure:"cache_duration"`
}

// ClusterConfig holds cluster configuration
type ClusterConfig struct {
	Enabled                bool     `mapstructure:"enabled"`
	NodeName               string   `mapstructure:"node_name"`
	DiscoveryMode          string   `mapstructure:"discovery_mode"`
	StaticNodes            []string `mapstructure:"static_nodes"`
	LeaderElectionEnabled  bool     `mapstructure:"leader_election_enabled"`
	LeaderElectionLease    time.Duration `mapstructure:"leader_election_lease"`
}

// StorageConfig holds storage configuration
type StorageConfig struct {
	Type              string `mapstructure:"type"`
	LocalPath         string `mapstructure:"local_path"`
	S3Endpoint        string `mapstructure:"s3_endpoint"`
	S3AccessKey       string `mapstructure:"s3_access_key"`
	S3SecretKey       string `mapstructure:"s3_secret_key"`
	S3Bucket          string `mapstructure:"s3_bucket"`
	S3Region          string `mapstructure:"s3_region"`
	MaxStoragePerAgent string `mapstructure:"max_storage_per_agent"`
}

// NetworkConfig holds network configuration
type NetworkConfig struct {
	DefaultSubnet       string   `mapstructure:"default_subnet"`
	EnableIPv6          bool     `mapstructure:"enable_ipv6"`
	DNSServers          []string `mapstructure:"dns_servers"`
	DNSSearchDomains    []string `mapstructure:"dns_search_domains"`
	EnableHostNetwork   bool     `mapstructure:"enable_host_network"`
	EnablePortMapping   bool     `mapstructure:"enable_port_mapping"`
}

// FeaturesConfig holds feature flags
type FeaturesConfig struct {
	MultiTenant   bool `mapstructure:"multi_tenant"`
	GPUSupport    bool `mapstructure:"gpu_support"`
	EdgeMode      bool `mapstructure:"edge_mode"`
	BackupEnabled bool `mapstructure:"backup_enabled"`
	AuditLogging  bool `mapstructure:"audit_logging"`
}

// MessagingConfig holds messaging and event bus configuration
type MessagingConfig struct {
	Enabled           bool          `mapstructure:"enabled"`
	Driver            string        `mapstructure:"driver"`
	NATS              NATSConfig    `mapstructure:"nats"`
	EventBufferSize   int           `mapstructure:"event_buffer_size"`
	PublishTimeout    time.Duration `mapstructure:"publish_timeout"`
	MaxRetries        int           `mapstructure:"max_retries"`
	RetryBackoff      time.Duration `mapstructure:"retry_backoff"`
}

// NATSConfig holds NATS-specific configuration
type NATSConfig struct {
	URL               string        `mapstructure:"url"`
	Token             string        `mapstructure:"token"`
	CredentialsFile     string        `mapstructure:"credentials_file"`
	TLSCert           string        `mapstructure:"tls_cert"`
	TLSKey            string        `mapstructure:"tls_key"`
	TLSCA             string        `mapstructure:"tls_ca"`
	ReconnectWait     time.Duration `mapstructure:"reconnect_wait"`
	MaxReconnects     int           `mapstructure:"max_reconnects"`
	JetStreamEnabled  bool          `mapstructure:"jetstream_enabled"`
	JetStreamDomain   string        `mapstructure:"jetstream_domain"`
}

// LoadConfig loads configuration from file and environment variables
func LoadConfig(configPath string) (*Config, error) {
	v := viper.New()

	// Set default values
	setDefaults(v)

	// Read configuration file
	if configPath != "" {
		absPath, err := filepath.Abs(configPath)
		if err != nil {
			return nil, fmt.Errorf("failed to get absolute config path: %w", err)
		}
		
		v.SetConfigFile(absPath)
		if err := v.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	// Read environment variables
	v.AutomaticEnv()
	v.SetEnvPrefix("AOS")

	// Unmarshal config
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate configuration
	if err := validateConfig(&cfg); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &cfg, nil
}

// setDefaults sets default configuration values
func setDefaults(v *viper.Viper) {
	// Server defaults
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.mode", "release")
	v.SetDefault("server.graceful_shutdown_timeout", 30)

	// Logging defaults
	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.format", "json")
	v.SetDefault("logging.output", "stdout")
	v.SetDefault("logging.file_path", "logs/aos.log")
	v.SetDefault("logging.max_size", 100)
	v.SetDefault("logging.max_backups", 10)
	v.SetDefault("logging.max_age", 30)

	// Container defaults
	v.SetDefault("container.runtime", "containerd")
	v.SetDefault("container.socket", "/run/containerd/containerd.sock")
	v.SetDefault("container.network_driver", "bridge")
	v.SetDefault("container.storage_driver", "overlay2")
	v.SetDefault("container.default_memory_limit", "512Mi")
	v.SetDefault("container.default_cpu_limit", "0.5")
	v.SetDefault("container.default_pids_limit", 100)
	v.SetDefault("container.default_network_bandwidth", "10M")

	// Scheduler defaults
	v.SetDefault("scheduler.algorithm", "round_robin")
	v.SetDefault("scheduler.max_concurrent_agents", 100)
	v.SetDefault("scheduler.max_agents_per_host", 20)
	v.SetDefault("scheduler.check_interval", 10)
	v.SetDefault("scheduler.auto_scale", false)
	v.SetDefault("scheduler.scale_up_threshold", 80)
	v.SetDefault("scheduler.scale_down_threshold", 20)

	// Security defaults
	v.SetDefault("security.jwt_secret", "change-this-in-production")
	v.SetDefault("security.jwt_expiry", 24)
	v.SetDefault("security.cors_enabled", true)
	v.SetDefault("security.cors_allowed_origins", []string{"*"})
	v.SetDefault("security.rate_limit_enabled", true)
	v.SetDefault("security.rate_limit_requests", 100)
	v.SetDefault("security.rate_limit_burst", 10)

	// Monitoring defaults
	v.SetDefault("monitoring.enabled", true)
	v.SetDefault("monitoring.prometheus_enabled", true)
	v.SetDefault("monitoring.prometheus_path", "/metrics")
	v.SetDefault("monitoring.health_check_path", "/health")
	v.SetDefault("monitoring.metrics_collection_interval", 15)
	v.SetDefault("monitoring.alerting_enabled", false)

	// Agent defaults
	v.SetDefault("agent.default_image", "agentos/base:latest")
	v.SetDefault("agent.default_working_dir", "/app")
	v.SetDefault("agent.allowed_registries", []string{"docker.io", "ghcr.io", "quay.io"})
	v.SetDefault("agent.max_image_size", "1Gi")
	v.SetDefault("agent.max_execution_time", 3600)
	v.SetDefault("agent.auto_restart", true)
	v.SetDefault("agent.restart_policy", "on-failure")

	// API defaults
	v.SetDefault("api.swagger_enabled", true)
	v.SetDefault("api.swagger_path", "/swagger/*")
	v.SetDefault("api.api_prefix", "/api/v1")
	v.SetDefault("api.enable_caching", true)
	v.SetDefault("api.cache_duration", 300)

	// Global defaults
	v.SetDefault("mode", "release")
	v.SetDefault("debug", false)

	// Messaging defaults
	v.SetDefault("messaging.enabled", true)
	v.SetDefault("messaging.driver", "nats")
	v.SetDefault("messaging.event_buffer_size", 1000)
	v.SetDefault("messaging.publish_timeout", 5)
	v.SetDefault("messaging.max_retries", 3)
	v.SetDefault("messaging.retry_backoff", 1)

	// NATS defaults
	v.SetDefault("messaging.nats.url", "nats://localhost:4222")
	v.SetDefault("messaging.nats.reconnect_wait", 1)
	v.SetDefault("messaging.nats.max_reconnects", 10)
	v.SetDefault("messaging.nats.jetstream_enabled", false)
}

const defaultConfigPath = "config.yaml"

// NewConfig returns a Config populated with default values (no file read).
func NewConfig() *Config {
	cfg := &Config{}
	v := viper.New()
	setDefaults(v)
	if err := v.Unmarshal(cfg); err != nil {
		panic(fmt.Sprintf("failed to unmarshal default config: %v", err))
	}
	cfg.Mode = "development"
	return cfg
}

// getDefaultConfigPath returns path if non-empty, otherwise the built-in default.
func getDefaultConfigPath(path string) string {
	if path != "" {
		return path
	}
	return defaultConfigPath
}

// validateConfig validates the configuration
func validateConfig(cfg *Config) error {
	// Validate server configuration
	if cfg.Server.Port <= 0 || cfg.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", cfg.Server.Port)
	}

	// Validate container runtime
	if cfg.Container.Runtime != "containerd" && cfg.Container.Runtime != "docker" {
		return fmt.Errorf("unsupported container runtime: %s", cfg.Container.Runtime)
	}

	// Validate scheduler algorithm
	validAlgorithms := []string{"round_robin", "least_loaded", "priority"}
	valid := false
	for _, algo := range validAlgorithms {
		if cfg.Scheduler.Algorithm == algo {
			valid = true
			break
		}
	}
	if !valid {
		return fmt.Errorf("unsupported scheduler algorithm: %s", cfg.Scheduler.Algorithm)
	}

	// Validate agent restart policy
	validPolicies := []string{"on-failure", "always", "never"}
	valid = false
	for _, policy := range validPolicies {
		if cfg.Agent.RestartPolicy == policy {
			valid = true
			break
		}
	}
	if !valid {
		return fmt.Errorf("unsupported agent restart policy: %s", cfg.Agent.RestartPolicy)
	}

	// Validate storage configuration
	if cfg.Storage.LocalPath != "" {
		if _, err := os.Stat(cfg.Storage.LocalPath); os.IsNotExist(err) {
			// Try to create the directory
			if err := os.MkdirAll(cfg.Storage.LocalPath, 0755); err != nil {
				return fmt.Errorf("failed to create storage directory: %w", err)
			}
		}
	}

	return nil
}