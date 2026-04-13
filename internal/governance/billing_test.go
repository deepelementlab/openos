package governance

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAllocateCost(t *testing.T) {
	out := AllocateCost(1000, []UsageRecord{
		{TenantID: "a", Units: 30},
		{TenantID: "b", Units: 70},
	})
	require.InDelta(t, 300, out["a"], 0.01)
	require.InDelta(t, 700, out["b"], 0.01)
}
