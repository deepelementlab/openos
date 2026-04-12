package discovery

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ServiceInstance represents a registered service instance.
type ServiceInstance struct {
	ID           string            `json:"id"`
	ServiceName  string            `json:"service_name"`
	Host         string            `json:"host"`
	Port         int               `json:"port"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	Tags         []string          `json:"tags,omitempty"`
	HealthStatus HealthStatus      `json:"health_status"`
	Weight       int               `json:"weight"`           // For weighted load balancing
	Zone         string            `json:"zone,omitempty"`   // For zone-aware routing
	Region       string            `json:"region,omitempty"` // For region-aware routing
	LastHeartbeat time.Time        `json:"last_heartbeat"`
	RegisteredAt  time.Time       `json:"registered_at"`
	Version      string            `json:"version,omitempty"`
}

// HealthStatus represents the health status of a service instance.
type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
	HealthStatusUnknown   HealthStatus = "unknown"
)

// IsHealthy checks if the instance is healthy.
func (si *ServiceInstance) IsHealthy() bool {
	return si.HealthStatus == HealthStatusHealthy
}

// IsExpired checks if the instance has missed heartbeats.
func (si *ServiceInstance) IsExpired(timeout time.Duration) bool {
	return time.Since(si.LastHeartbeat) > timeout
}

// Address returns the full address of the instance.
func (si *ServiceInstance) Address() string {
	return formatAddress(si.Host, si.Port)
}

// formatAddress formats host and port.
func formatAddress(host string, port int) string {
	if port == 0 {
		return host
	}
	return fmt.Sprintf("%s:%d", host, port)
}

// NewServiceInstance creates a new service instance.
func NewServiceInstance(serviceName, host string, port int) *ServiceInstance {
	return &ServiceInstance{
		ID:            uuid.New().String(),
		ServiceName:   serviceName,
		Host:          host,
		Port:          port,
		Metadata:      make(map[string]string),
		Tags:          make([]string, 0),
		HealthStatus:  HealthStatusHealthy,
		Weight:        100,
		LastHeartbeat: time.Now().UTC(),
		RegisteredAt:  time.Now().UTC(),
	}
}

// ServiceDefinition represents a service type definition.
type ServiceDefinition struct {
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Protocol    string   `json:"protocol"` // http, grpc, etc.
	Required    bool     `json:"required"`
	Tags        []string `json:"tags,omitempty"`
}

// ServiceQuery provides query parameters for service discovery.
type ServiceQuery struct {
	ServiceName string            `json:"service_name"`
	Tags        []string          `json:"tags,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	HealthyOnly bool              `json:"healthy_only"`
	Zone        string            `json:"zone,omitempty"`
	Region      string            `json:"region,omitempty"`
}

// ServiceSet represents a set of service instances.
type ServiceSet struct {
	ServiceName string             `json:"service_name"`
	Instances   []*ServiceInstance `json:"instances"`
	Version     int64              `json:"version"`
	UpdatedAt   time.Time          `json:"updated_at"`
}

// Count returns the number of instances.
func (ss *ServiceSet) Count() int {
	return len(ss.Instances)
}

// HealthyCount returns the number of healthy instances.
func (ss *ServiceSet) HealthyCount() int {
	count := 0
	for _, instance := range ss.Instances {
		if instance.IsHealthy() {
			count++
		}
	}
	return count
}

// GetHealthy returns all healthy instances.
func (ss *ServiceSet) GetHealthy() []*ServiceInstance {
	var healthy []*ServiceInstance
	for _, instance := range ss.Instances {
		if instance.IsHealthy() {
			healthy = append(healthy, instance)
		}
	}
	return healthy
}

// FilterByZone filters instances by zone.
func (ss *ServiceSet) FilterByZone(zone string) []*ServiceInstance {
	var filtered []*ServiceInstance
	for _, instance := range ss.Instances {
		if instance.Zone == zone {
			filtered = append(filtered, instance)
		}
	}
	return filtered
}

// FilterByRegion filters instances by region.
func (ss *ServiceSet) FilterByRegion(region string) []*ServiceInstance {
	var filtered []*ServiceInstance
	for _, instance := range ss.Instances {
		if instance.Region == region {
			filtered = append(filtered, instance)
		}
	}
	return filtered
}

// ServiceInstanceStatus tracks the dynamic status of an instance.
type ServiceInstanceStatus struct {
	InstanceID    string        `json:"instance_id"`
	ConsecutiveFailures int     `json:"consecutive_failures"`
	ConsecutiveSuccesses int    `json:"consecutive_successes"`
	LastCheckTime time.Time     `json:"last_check_time"`
	LastError     string        `json:"last_error,omitempty"`
	TotalRequests int64         `json:"total_requests"`
	FailedRequests int64        `json:"failed_requests"`
	LatencyAvg    time.Duration `json:"latency_avg"`
}
