package monitoring

import (
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// PrometheusCollector implements Prometheus metrics collection
type PrometheusCollector struct {
	// System metrics
	cpuUsage        *prometheus.GaugeVec
	memoryUsage     *prometheus.GaugeVec
	diskUsage       *prometheus.GaugeVec
	networkTx       *prometheus.CounterVec
	networkRx       *prometheus.CounterVec

	// Agent metrics
	agentCount      *prometheus.GaugeVec
	agentCreated    *prometheus.CounterVec
	agentStarted    *prometheus.CounterVec
	agentStopped    *prometheus.CounterVec
	agentDeleted    *prometheus.CounterVec
	agentCreationDuration *prometheus.HistogramVec

	// API metrics
	apiRequests     *prometheus.CounterVec
	apiErrors       *prometheus.CounterVec
	apiLatency      *prometheus.HistogramVec

	// Scheduler metrics
	schedulerTasks  *prometheus.CounterVec
	schedulerQueue  *prometheus.GaugeVec
	schedulerErrors *prometheus.CounterVec

	// Security metrics
	authSuccess     *prometheus.CounterVec
	authFailures    *prometheus.CounterVec
	authLatency     *prometheus.HistogramVec
	authorizationAttempts *prometheus.CounterVec
	authorizationSuccess  *prometheus.CounterVec

	// Resource metrics
	resourceAllocated *prometheus.GaugeVec
	resourceAvailable *prometheus.GaugeVec
	resourceUsage     *prometheus.GaugeVec

	mu sync.RWMutex
}

// NewPrometheusCollector creates a new Prometheus collector
func NewPrometheusCollector() *PrometheusCollector {
	pc := &PrometheusCollector{}
	pc.initMetrics()
	return pc
}

// initMetrics initializes all Prometheus metrics
func (pc *PrometheusCollector) initMetrics() {
	// System metrics
	pc.cpuUsage = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "aos_system_cpu_usage_percent",
			Help: "CPU usage percentage by node",
		},
		[]string{"node", "type"},
	)

	pc.memoryUsage = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "aos_system_memory_usage_bytes",
			Help: "Memory usage in bytes by node",
		},
		[]string{"node", "type"},
	)

	pc.diskUsage = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "aos_system_disk_usage_bytes",
			Help: "Disk usage in bytes by node and mountpoint",
		},
		[]string{"node", "mountpoint", "type"},
	)

	pc.networkTx = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "aos_system_network_tx_bytes_total",
			Help: "Total network transmitted bytes",
		},
		[]string{"node", "interface"},
	)

	pc.networkRx = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "aos_system_network_rx_bytes_total",
			Help: "Total network received bytes",
		},
		[]string{"node", "interface"},
	)

	// Agent metrics
	pc.agentCount = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "aos_agent_count",
			Help: "Current number of agents by status",
		},
		[]string{"status", "runtime"},
	)

	pc.agentCreated = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "aos_agent_created_total",
			Help: "Total number of agents created",
		},
		[]string{"runtime", "image"},
	)

	pc.agentStarted = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "aos_agent_started_total",
			Help: "Total number of agents started",
		},
		[]string{"runtime"},
	)

	pc.agentStopped = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "aos_agent_stopped_total",
			Help: "Total number of agents stopped",
		},
		[]string{"runtime", "reason"},
	)

	pc.agentDeleted = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "aos_agent_deleted_total",
			Help: "Total number of agents deleted",
		},
		[]string{"runtime", "reason"},
	)

	pc.agentCreationDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "aos_agent_creation_duration_seconds",
			Help:    "Time taken to create an agent",
			Buckets: []float64{0.1, 0.5, 1.0, 2.0, 5.0, 10.0},
		},
		[]string{"runtime", "image"},
	)

	// API metrics
	pc.apiRequests = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "aos_api_requests_total",
			Help: "Total API requests",
		},
		[]string{"method", "path", "status"},
	)

	pc.apiErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "aos_api_errors_total",
			Help: "Total API errors",
		},
		[]string{"method", "path", "type"},
	)

	pc.apiLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "aos_api_latency_seconds",
			Help:    "API request latency",
			Buckets: []float64{0.01, 0.05, 0.1, 0.5, 1.0, 5.0},
		},
		[]string{"method", "path"},
	)

	// Scheduler metrics
	pc.schedulerTasks = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "aos_scheduler_tasks_total",
			Help: "Total tasks scheduled",
		},
		[]string{"strategy", "status"},
	)

	pc.schedulerQueue = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "aos_scheduler_queue_size",
			Help: "Current scheduler queue size",
		},
		[]string{"type"},
	)

	pc.schedulerErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "aos_scheduler_errors_total",
			Help: "Total scheduler errors",
		},
		[]string{"type", "reason"},
	)

	// Security metrics
	pc.authSuccess = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "aos_security_auth_success_total",
			Help: "Total successful authentications",
		},
		[]string{"method", "user"},
	)

	pc.authFailures = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "aos_security_auth_failures_total",
			Help: "Total authentication failures",
		},
		[]string{"method", "reason"},
	)

	pc.authLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "aos_security_auth_latency_seconds",
			Help:    "Authentication latency",
			Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1},
		},
		[]string{"method"},
	)

	pc.authorizationAttempts = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "aos_security_authorization_attempts_total",
			Help: "Total authorization attempts",
		},
		[]string{"resource", "action"},
	)

	pc.authorizationSuccess = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "aos_security_authorization_success_total",
			Help: "Total successful authorizations",
		},
		[]string{"resource", "action"},
	)

	// Resource metrics
	pc.resourceAllocated = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "aos_resource_allocated",
			Help: "Allocated resources by type",
		},
		[]string{"node", "type", "unit"},
	)

	pc.resourceAvailable = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "aos_resource_available",
			Help: "Available resources by type",
		},
		[]string{"node", "type", "unit"},
	)

	pc.resourceUsage = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "aos_resource_usage_percent",
			Help: "Resource usage percentage",
		},
		[]string{"node", "type"},
	)
}

// HTTPHandler returns Prometheus metrics HTTP handler
func (pc *PrometheusCollector) HTTPHandler() http.Handler {
	return promhttp.Handler()
}

// RecordAgentCreated records agent creation metrics
func (pc *PrometheusCollector) RecordAgentCreated(runtime, image string, duration time.Duration) {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	pc.agentCreated.WithLabelValues(runtime, image).Inc()
	pc.agentCreationDuration.WithLabelValues(runtime, image).Observe(duration.Seconds())
}

// RecordAgentStarted records agent start metrics
func (pc *PrometheusCollector) RecordAgentStarted(runtime string) {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	pc.agentStarted.WithLabelValues(runtime).Inc()
}

// RecordAgentStopped records agent stop metrics
func (pc *PrometheusCollector) RecordAgentStopped(runtime, reason string) {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	pc.agentStopped.WithLabelValues(runtime, reason).Inc()
}

// RecordAgentDeleted records agent deletion metrics
func (pc *PrometheusCollector) RecordAgentDeleted(runtime, reason string) {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	pc.agentDeleted.WithLabelValues(runtime, reason).Inc()
}

// UpdateAgentCount updates current agent count by status
func (pc *PrometheusCollector) UpdateAgentCount(status, runtime string, count float64) {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	pc.agentCount.WithLabelValues(status, runtime).Set(count)
}

// RecordAPIRequest records API request metrics
func (pc *PrometheusCollector) RecordAPIRequest(method, path string, status int, latency time.Duration) {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	pc.apiRequests.WithLabelValues(method, path, http.StatusText(status)).Inc()
	pc.apiLatency.WithLabelValues(method, path).Observe(latency.Seconds())
}

// RecordAPIError records API error metrics
func (pc *PrometheusCollector) RecordAPIError(method, path, errorType string) {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	pc.apiErrors.WithLabelValues(method, path, errorType).Inc()
}

// RecordSchedulerTask records scheduler task metrics
func (pc *PrometheusCollector) RecordSchedulerTask(strategy, status string) {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	pc.schedulerTasks.WithLabelValues(strategy, status).Inc()
}

// UpdateSchedulerQueue updates scheduler queue size
func (pc *PrometheusCollector) UpdateSchedulerQueue(queueType string, size float64) {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	pc.schedulerQueue.WithLabelValues(queueType).Set(size)
}

// RecordSchedulerError records scheduler error metrics
func (pc *PrometheusCollector) RecordSchedulerError(errorType, reason string) {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	pc.schedulerErrors.WithLabelValues(errorType, reason).Inc()
}

// RecordAuthSuccess records authentication success metrics
func (pc *PrometheusCollector) RecordAuthSuccess(method, user string, latency time.Duration) {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	pc.authSuccess.WithLabelValues(method, user).Inc()
	pc.authLatency.WithLabelValues(method).Observe(latency.Seconds())
}

// RecordAuthFailure records authentication failure metrics
func (pc *PrometheusCollector) RecordAuthFailure(method, reason string) {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	pc.authFailures.WithLabelValues(method, reason).Inc()
}

// RecordAuthorizationAttempt records authorization attempt metrics
func (pc *PrometheusCollector) RecordAuthorizationAttempt(resource, action string, success bool) {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	pc.authorizationAttempts.WithLabelValues(resource, action).Inc()
	if success {
		pc.authorizationSuccess.WithLabelValues(resource, action).Inc()
	}
}

// UpdateResourceMetrics updates resource allocation and usage metrics
func (pc *PrometheusCollector) UpdateResourceMetrics(node, resourceType, unit string, allocated, available, usage float64) {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	pc.resourceAllocated.WithLabelValues(node, resourceType, unit).Set(allocated)
	pc.resourceAvailable.WithLabelValues(node, resourceType, unit).Set(available)
	
	if unit == "percent" {
		pc.resourceUsage.WithLabelValues(node, resourceType).Set(usage)
	}
}

// UpdateSystemMetrics updates system metrics
func (pc *PrometheusCollector) UpdateSystemMetrics(node string, cpuUsage, memoryUsage, diskUsage float64) {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	pc.cpuUsage.WithLabelValues(node, "system").Set(cpuUsage)
	pc.memoryUsage.WithLabelValues(node, "system").Set(memoryUsage)
	pc.diskUsage.WithLabelValues(node, "/", "system").Set(diskUsage)
}

// UpdateNetworkMetrics updates network metrics
func (pc *PrometheusCollector) UpdateNetworkMetrics(node, interfaceName string, txBytes, rxBytes uint64) {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	pc.networkTx.WithLabelValues(node, interfaceName).Add(float64(txBytes))
	pc.networkRx.WithLabelValues(node, interfaceName).Add(float64(rxBytes))
}
