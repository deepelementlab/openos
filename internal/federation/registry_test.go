package federation

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRegistry_RegisterPick(t *testing.T) {
	r := NewRegistry()
	ctx := context.Background()
	require.NoError(t, r.Register(ctx, &ClusterRecord{
		ID:       "c1",
		Name:     "aws-1",
		Endpoint: "grpc://c1",
		Capabilities: ClusterCapabilities{
			GPU:       true,
			CostTier:  "low",
			Region:    "us-east-1",
		},
	}))
	fs := NewFederalScheduler(r)
	id, err := fs.PickCluster(ctx, ScheduleHint{PreferredGPU: true, Mode: AffinityCost})
	require.NoError(t, err)
	require.Equal(t, "c1", id)
}
