package slo

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestGate_Evaluate(t *testing.T) {
	c := NewCollector()
	c.RecordAgentStart(AgentStartSample{Success: true, LatencyMS: 50})
	g := NewGate()
	g.MinStartSuccessRate = 0.9
	require.NoError(t, g.Evaluate(c))

	c2 := NewCollector()
	c2.RecordAgentStart(AgentStartSample{Success: false})
	require.Error(t, g.Evaluate(c2))
}

func TestCollector_P99AndAPI(t *testing.T) {
	c := NewCollector()
	for i := 0; i < 10; i++ {
		c.RecordAgentStart(AgentStartSample{Success: true, LatencyMS: int64(i * 10)})
	}
	_ = c.StartLatencyP99()
	c.RecordAPI(true)
	c.RecordAPI(false)
	require.Greater(t, c.APIErrorRatio(), 0.0)
}

func TestGate_Latency(t *testing.T) {
	c := NewCollector()
	c.RecordAgentStart(AgentStartSample{Success: true, LatencyMS: 9000})
	g := &Gate{MinStartSuccessRate: 0, MaxP99StartLatency: time.Second}
	require.Error(t, g.Evaluate(c))
}
