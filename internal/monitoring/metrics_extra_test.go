package monitoring

import (
	"strings"
	"testing"
)

func TestMetricsAllCounters(t *testing.T) {
	m, err := NewMetrics(nil)
	if err != nil {
		t.Fatalf("new metrics: %v", err)
	}

	// Increment all counters
	m.IncAgentCreated()
	m.IncAgentCreated()
	m.IncAgentStarted()
	m.IncAgentStopped()
	m.IncAgentDeleted()

	out := m.GetMetrics()
	checks := []struct {
		needle string
		desc   string
	}{
		{"aos_agent_created_total 2", "created counter"},
		{"aos_agent_started_total 1", "started counter"},
		{"aos_agent_stopped_total 1", "stopped counter"},
		{"aos_agent_deleted_total 1", "deleted counter"},
	}
	for _, c := range checks {
		if !strings.Contains(out, c.needle) {
			t.Fatalf("%s: expected %q in output, got:\n%s", c.desc, c.needle, out)
		}
	}
}

func TestMetricsCreationDuration(t *testing.T) {
	m, _ := NewMetrics(nil)

	m.ObserveAgentCreationDuration(0.5)
	m.ObserveAgentCreationDuration(1.5)

	out := m.GetMetrics()
	if !strings.Contains(out, "aos_agent_creation_duration_seconds_sum 2.000000") {
		t.Fatalf("expected sum=2.0, got:\n%s", out)
	}
	if !strings.Contains(out, "aos_agent_creation_duration_seconds_count 2") {
		t.Fatalf("expected count=2, got:\n%s", out)
	}
}

func TestMetricsMultipleAPIRequests(t *testing.T) {
	m, _ := NewMetrics(nil)

	m.IncAPIRequest("GET", "/api/v1/agents", 200)
	m.IncAPIRequest("GET", "/api/v1/agents", 200)
	m.IncAPIRequest("POST", "/api/v1/agents", 201)

	out := m.GetMetrics()
	if !strings.Contains(out, `aos_api_requests_total{method="GET",path="/api/v1/agents",status="200"} 2`) {
		t.Fatalf("expected GET count=2, got:\n%s", out)
	}
	if !strings.Contains(out, `aos_api_requests_total{method="POST",path="/api/v1/agents",status="201"} 1`) {
		t.Fatalf("expected POST count=1, got:\n%s", out)
	}
}

func TestMetricsMultipleResourceUsage(t *testing.T) {
	m, _ := NewMetrics(nil)

	m.SetResourceUsage("cpu", 0.75)
	m.SetResourceUsage("memory", 0.50)

	out := m.GetMetrics()
	if !strings.Contains(out, `aos_resource_usage{resource="cpu"} 0.750000`) {
		t.Fatalf("expected cpu=0.75, got:\n%s", out)
	}
	if !strings.Contains(out, `aos_resource_usage{resource="memory"} 0.500000`) {
		t.Fatalf("expected memory=0.50, got:\n%s", out)
	}
}

func TestMetricsEmptyOutput(t *testing.T) {
	m, _ := NewMetrics(nil)

	out := m.GetMetrics()
	// All counters should be zero but headers should be present
	if !strings.Contains(out, "aos_agent_created_total 0") {
		t.Fatalf("expected zero created counter, got:\n%s", out)
	}
	if !strings.Contains(out, "# TYPE aos_agent_created_total counter") {
		t.Fatalf("expected TYPE header, got:\n%s", out)
	}
}

func TestMetricsClose(t *testing.T) {
	m, _ := NewMetrics(nil)
	if err := m.Close(); err != nil {
		t.Fatalf("close should not error: %v", err)
	}
}

func TestMetricsMultipleErrors(t *testing.T) {
	m, _ := NewMetrics(nil)

	m.IncAPIError("POST", "/api/v1/agents", "validation")
	m.IncAPIError("POST", "/api/v1/agents", "validation")
	m.IncAPIError("GET", "/api/v1/agents", "not_found")

	out := m.GetMetrics()
	if !strings.Contains(out, `aos_api_errors_total{method="POST",path="/api/v1/agents",type="validation"} 2`) {
		t.Fatalf("expected validation error count=2, got:\n%s", out)
	}
	if !strings.Contains(out, `aos_api_errors_total{method="GET",path="/api/v1/agents",type="not_found"} 1`) {
		t.Fatalf("expected not_found error count=1, got:\n%s", out)
	}
}
