package monitoring

import (
	"strings"
	"testing"
)

func TestMetricsOutput(t *testing.T) {
	m, err := NewMetrics(nil)
	if err != nil {
		t.Fatalf("new metrics: %v", err)
	}

	m.IncAgentCreated()
	m.IncAPIRequest("POST", "/api/v1/agents", 201)
	m.IncAPIError("POST", "/api/v1/agents", "validation")
	m.ObserveAgentCreationDuration(0.42)
	m.SetResourceUsage("cpu", 0.66)

	out := m.GetMetrics()
	needles := []string{
		"aos_agent_created_total 1",
		`aos_api_requests_total{method="POST",path="/api/v1/agents",status="201"} 1`,
		`aos_api_errors_total{method="POST",path="/api/v1/agents",type="validation"} 1`,
		`aos_resource_usage{resource="cpu"} 0.660000`,
	}
	for _, n := range needles {
		if !strings.Contains(out, n) {
			t.Fatalf("expected metrics output to contain %q, got:\n%s", n, out)
		}
	}
}

