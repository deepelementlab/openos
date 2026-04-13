package health

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAggregator_Collect(t *testing.T) {
	c := NewChecker()
	c.Register("scheduler", "sched", func(ctx context.Context) error { return nil })
	a := NewAggregator(c)
	rep := a.Collect(context.Background())
	require.Equal(t, StatusHealthy, rep.Overall)
	require.NotEmpty(t, rep.ByLevel[LevelSystem])
}
