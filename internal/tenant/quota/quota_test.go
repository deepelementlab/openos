package quota

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEnforcer_HardLimit(t *testing.T) {
	e := NewEnforcer()
	e.SetHardLimit(&HardLimit{TenantID: "t1", MaxMemory: 100})
	require.NoError(t, e.CheckWithinHardLimit(context.Background(), "t1", 0, 50))
	err := e.CheckWithinHardLimit(context.Background(), "t1", 0, 200)
	require.Error(t, err)
	require.True(t, IsResourceExhausted(err))
}
