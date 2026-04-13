package runtime_test

import (
	"context"
	"testing"
	"time"

	"github.com/agentos/aos/pkg/runtime/facade"
	"github.com/agentos/aos/pkg/runtime/types"
	"github.com/stretchr/testify/require"
)

// TestFacadeFlow exercises Connect + CreateAgent delegation on gVisor backend (no real runsc).
func TestFacadeFlow_GVisorMock(t *testing.T) {
	f := facade.NewRuntimeFacade()
	cfg := &types.RuntimeConfig{Type: types.RuntimeGVisor}
	err := f.Connect(context.Background(), facade.BackendGVisor, cfg)
	require.NoError(t, err)
	ag, err := f.CreateAgent(context.Background(), &types.AgentSpec{
		ID: "facade-1", Name: "n", Image: "alpine:latest",
	})
	require.NoError(t, err)
	require.Equal(t, "facade-1", ag.ID)
	require.NoError(t, f.StartAgent(context.Background(), ag.ID))
	require.NoError(t, f.StopAgent(context.Background(), ag.ID, time.Second))
	require.NoError(t, f.DeleteAgent(context.Background(), ag.ID))
	_ = f.Runtime().Cleanup(context.Background())
}
