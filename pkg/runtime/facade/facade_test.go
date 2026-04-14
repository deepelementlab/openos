package facade

import (
	"context"
	"testing"
	"time"

	"github.com/agentos/aos/internal/builder/engine"
	"github.com/agentos/aos/pkg/runtime/types"
	"github.com/stretchr/testify/require"
)

func TestSelectBackendForSpec(t *testing.T) {
	require.Equal(t, BackendGVisor, SelectBackendForSpec(&types.AgentSpec{
		Labels: map[string]string{"aos.openos.dev/runtime": "gvisor"},
	}))
	require.Equal(t, BackendKata, SelectBackendForSpec(&types.AgentSpec{
		Labels: map[string]string{"aos.openos.dev/runtime": "kata"},
	}))
	require.Equal(t, BackendContainerd, SelectBackendForSpec(&types.AgentSpec{}))
}

func TestRuntimeFacade_ConnectForSpec(t *testing.T) {
	f := NewRuntimeFacade()
	cfg := &types.RuntimeConfig{Type: types.RuntimeGVisor}
	err := f.ConnectForSpec(context.Background(), &types.AgentSpec{
		Labels: map[string]string{"aos.openos.dev/runtime": "gvisor"},
	}, cfg)
	require.NoError(t, err)
	require.Equal(t, BackendGVisor, f.Backend())
	_ = f.Runtime().Cleanup(context.Background())
}

func TestAgentSpecFromPackage(t *testing.T) {
	man := []byte(`{
		"apiVersion": "aos.io/v1",
		"kind": "AgentPackage",
		"metadata": {"name": "demo", "version": "0.1.0", "labels": {"k":"v"}},
		"config": {"image": "demo:latest", "workDir": "/app", "env": {"FOO": "bar"}},
		"entrypoint": {"command": ["/bin/sh"], "args": ["-c", "true"]}
	}`)
	as, sp, err := AgentSpecFromPackage(&engine.AgentPackage{ManifestJSON: man})
	require.NoError(t, err)
	require.Equal(t, "demo", as.Name)
	require.Equal(t, "demo:latest", as.Image)
	require.Equal(t, "/app", as.WorkingDir)
	require.Contains(t, as.Env, "FOO=bar")
	require.Equal(t, []string{"/bin/sh"}, as.Command)
	require.NotNil(t, sp)
}

func TestRuntimeFacade_Delegates(t *testing.T) {
	f := NewRuntimeFacade()
	_, err := f.CreateAgent(context.Background(), &types.AgentSpec{ID: "x"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "not connected")

	err = f.StartAgent(context.Background(), "x")
	require.Error(t, err)
	err = f.StopAgent(context.Background(), "x", time.Second)
	require.Error(t, err)
	err = f.DeleteAgent(context.Background(), "x")
	require.Error(t, err)
}
