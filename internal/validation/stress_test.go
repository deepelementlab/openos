package validation

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSimulateAgentBurst(t *testing.T) {
	res := SimulateAgentBurst(context.Background(), StressConfig{Concurrency: 4, Iterations: 20}, func(ctx context.Context, id int) error {
		return nil
	})
	require.Equal(t, int64(20), res.Iterations)
	require.Equal(t, int64(0), res.Errors)
}
