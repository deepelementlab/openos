// Package metrics registers production AOS Prometheus metrics.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// SchedulerLatencySeconds tracks end-to-end scheduling latency (histogram).
	SchedulerLatencySeconds = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "aos_scheduler_latency_seconds",
			Help:    "Scheduler operation latency in seconds",
			Buckets: prometheus.ExponentialBuckets(0.001, 2, 14),
		},
		[]string{"stage"}, // e.g. filter, score, bind
	)

	// AgentLifecycleDuration tracks agent create/start/stop durations.
	AgentLifecycleDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "aos_agent_lifecycle_duration_seconds",
			Help:    "Agent lifecycle phase duration",
			Buckets: prometheus.ExponentialBuckets(0.1, 2, 12),
		},
		[]string{"phase"},
	)

	// TenantQuotaUsageRatio is a gauge for quota consumption (0-1).
	TenantQuotaUsageRatio = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "aos_tenant_quota_usage_ratio",
			Help: "Observed quota usage ratio per tenant and resource class",
		},
		[]string{"tenant_id", "resource"},
	)

	// RuntimeOperationsTotal counts runtime operations (containerd/runsc).
	RuntimeOperationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "aos_runtime_operations_total",
			Help: "Total runtime operations by backend and result",
		},
		[]string{"backend", "operation", "result"},
	)
)

// Registry returns the default prometheus registry (for tests that need to unregister).
func Registry() prometheus.Registerer {
	return prometheus.DefaultRegisterer
}
