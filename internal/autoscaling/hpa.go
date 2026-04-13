// Package autoscaling provides horizontal/vertical scaling policies for AOS workers.
package autoscaling

import (
	"context"
	"time"
)

// MetricKind identifies an autoscaling signal.
type MetricKind string

const (
	MetricCPU          MetricKind = "cpu"
	MetricMemory       MetricKind = "memory"
	MetricQueueDepth   MetricKind = "queue_depth"
	MetricScheduleP95  MetricKind = "schedule_p95"
)

// Target describes desired utilization or absolute value.
type Target struct {
	AverageUtilization *int32   // 0-100 for Resource metrics
	AverageValue       *float64 // for custom metrics
}

// MetricSpec is one scaling metric (Kubernetes HPA-shaped).
type MetricSpec struct {
	Type MetricKind
	Name string
	Target
}

// Behavior tuning windows.
type Behavior struct {
	ScaleUpStabilization   time.Duration
	ScaleDownStabilization time.Duration
}

// Policy is the aos_autoscaling YAML equivalent (in-memory).
type Policy struct {
	MinReplicas int
	MaxReplicas int
	Metrics     []MetricSpec
	Behavior
}

// Evaluator decides desired replica count from current metrics snapshot.
type Evaluator struct{}

// Evaluate returns next replica count (simplified reactive controller).
func (e *Evaluator) Evaluate(ctx context.Context, p Policy, current int, cpuUtil float64, queueLen float64) int {
	next := current
	for _, m := range p.Metrics {
		switch m.Type {
		case MetricCPU:
			if m.AverageUtilization != nil && cpuUtil > float64(*m.AverageUtilization)/100.0 {
				next++
			}
		case MetricQueueDepth:
			if m.AverageValue != nil && queueLen > *m.AverageValue {
				next++
			}
		}
	}
	if next < p.MinReplicas {
		next = p.MinReplicas
	}
	if next > p.MaxReplicas {
		next = p.MaxReplicas
	}
	return next
}
