// Package monitoring implements metrics collection and monitoring for Agent OS.
package monitoring

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/agentos/aos/internal/config"
)

// Metrics represents the monitoring metrics collector.
type Metrics struct {
	config *config.MonitoringConfig
	mu     sync.Mutex

	agentCreated int64
	agentStarted int64
	agentStopped int64
	agentDeleted int64

	agentCreationDurationSum float64
	agentCreationDurationCnt int64

	apiRequests map[string]int64
	apiErrors   map[string]int64
	resourceUse map[string]float64
}

// NewMetrics creates a new metrics collector.
func NewMetrics(cfg *config.MonitoringConfig) (*Metrics, error) {
	m := &Metrics{
		config: cfg,
		apiRequests: make(map[string]int64),
		apiErrors:   make(map[string]int64),
		resourceUse: make(map[string]float64),
	}
	return m, nil
}

// IncAgentCreated increments the count of created agents.
func (m *Metrics) IncAgentCreated() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.agentCreated++
}

// IncAgentStarted increments the count of started agents.
func (m *Metrics) IncAgentStarted() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.agentStarted++
}

// IncAgentStopped increments the count of stopped agents.
func (m *Metrics) IncAgentStopped() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.agentStopped++
}

// IncAgentDeleted increments the count of deleted agents.
func (m *Metrics) IncAgentDeleted() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.agentDeleted++
}

// ObserveAgentCreationDuration observes the duration of agent creation.
func (m *Metrics) ObserveAgentCreationDuration(duration float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.agentCreationDurationSum += duration
	m.agentCreationDurationCnt++
}

// IncAPIRequest increments the count of API requests.
func (m *Metrics) IncAPIRequest(method, path string, status int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := fmt.Sprintf("%s|%s|%d", method, path, status)
	m.apiRequests[key]++
}

// IncAPIError increments the count of API errors.
func (m *Metrics) IncAPIError(method, path string, errorType string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := fmt.Sprintf("%s|%s|%s", method, path, errorType)
	m.apiErrors[key]++
}

// SetResourceUsage sets the resource usage metrics.
func (m *Metrics) SetResourceUsage(resourceType string, usage float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.resourceUse[resourceType] = usage
}

// GetMetrics returns the current metrics in text format.
func (m *Metrics) GetMetrics() string {
	m.mu.Lock()
	defer m.mu.Unlock()

	lines := []string{
		"# HELP aos_agent_created_total Total created agents",
		"# TYPE aos_agent_created_total counter",
		fmt.Sprintf("aos_agent_created_total %d", m.agentCreated),
		"# HELP aos_agent_started_total Total started agents",
		"# TYPE aos_agent_started_total counter",
		fmt.Sprintf("aos_agent_started_total %d", m.agentStarted),
		"# HELP aos_agent_stopped_total Total stopped agents",
		"# TYPE aos_agent_stopped_total counter",
		fmt.Sprintf("aos_agent_stopped_total %d", m.agentStopped),
		"# HELP aos_agent_deleted_total Total deleted agents",
		"# TYPE aos_agent_deleted_total counter",
		fmt.Sprintf("aos_agent_deleted_total %d", m.agentDeleted),
		"# HELP aos_agent_creation_duration_seconds Agent creation duration summary",
		"# TYPE aos_agent_creation_duration_seconds summary",
		fmt.Sprintf("aos_agent_creation_duration_seconds_sum %.6f", m.agentCreationDurationSum),
		fmt.Sprintf("aos_agent_creation_duration_seconds_count %d", m.agentCreationDurationCnt),
	}

	reqKeys := make([]string, 0, len(m.apiRequests))
	for k := range m.apiRequests {
		reqKeys = append(reqKeys, k)
	}
	sort.Strings(reqKeys)
	lines = append(lines, "# HELP aos_api_requests_total Total API requests")
	lines = append(lines, "# TYPE aos_api_requests_total counter")
	for _, k := range reqKeys {
		parts := strings.Split(k, "|")
		lines = append(lines, fmt.Sprintf(
			`aos_api_requests_total{method="%s",path="%s",status="%s"} %d`,
			parts[0], parts[1], parts[2], m.apiRequests[k],
		))
	}

	errKeys := make([]string, 0, len(m.apiErrors))
	for k := range m.apiErrors {
		errKeys = append(errKeys, k)
	}
	sort.Strings(errKeys)
	lines = append(lines, "# HELP aos_api_errors_total Total API errors")
	lines = append(lines, "# TYPE aos_api_errors_total counter")
	for _, k := range errKeys {
		parts := strings.Split(k, "|")
		lines = append(lines, fmt.Sprintf(
			`aos_api_errors_total{method="%s",path="%s",type="%s"} %d`,
			parts[0], parts[1], parts[2], m.apiErrors[k],
		))
	}

	resKeys := make([]string, 0, len(m.resourceUse))
	for k := range m.resourceUse {
		resKeys = append(resKeys, k)
	}
	sort.Strings(resKeys)
	lines = append(lines, "# HELP aos_resource_usage Resource usage gauge")
	lines = append(lines, "# TYPE aos_resource_usage gauge")
	for _, k := range resKeys {
		lines = append(lines, fmt.Sprintf(`aos_resource_usage{resource="%s"} %.6f`, k, m.resourceUse[k]))
	}

	return strings.Join(lines, "\n") + "\n"
}

// Close closes the metrics collector.
func (m *Metrics) Close() error {
	return nil
}