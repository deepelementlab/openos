package autoscaling

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEvaluator_Evaluate(t *testing.T) {
	u := int32(70)
	qv := 150.0
	e := Evaluator{}
	p := Policy{
		MinReplicas: 1,
		MaxReplicas: 10,
		Metrics: []MetricSpec{
			{Type: MetricCPU, Target: Target{AverageUtilization: &u}},
			{Type: MetricQueueDepth, Target: Target{AverageValue: &qv}},
		},
	}
	next := e.Evaluate(context.Background(), p, 2, 0.85, 200)
	require.GreaterOrEqual(t, next, p.MinReplicas)
	require.LessOrEqual(t, next, p.MaxReplicas)
}
