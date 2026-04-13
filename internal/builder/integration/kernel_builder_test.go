// Package integration tests Kernel + Builder hooks (M8).
package integration

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/agentos/aos/internal/builder/engine"
	"github.com/agentos/aos/internal/builder/registry"
	"github.com/agentos/aos/internal/kernel"
	"github.com/agentos/aos/internal/kernel/ipc"
)

func TestKernelBuilder_CheckpointAndIPC(t *testing.T) {
	ctx := context.Background()
	k := kernel.NewDefaultFacade()

	// Builder cache analogue: checkpoint a synthetic build agent id
	cp, err := k.Memory.Checkpoint(ctx, "build-agent-1")
	require.NoError(t, err)
	_, err = k.Memory.Restore(ctx, cp)
	require.NoError(t, err)

	q, err := k.IPC.CreateMessageQueue(ctx, "build-test", ipc.MQConfig{})
	require.NoError(t, err)
	require.NoError(t, q.Publish(ctx, ipc.Message{Topic: "build", Payload: []byte("ok")}))

	eng := engine.NewEngine()
	dir := t.TempDir()
	af := filepath.Join(dir, "Agentfile.json")
	require.NoError(t, os.WriteFile(af, []byte(`{
  "apiVersion": "aos.io/v1",
  "kind": "AgentPackage",
  "metadata": {"name": "int", "version": "0.0.1"},
  "steps": [{"type": "run", "command": "noop"}]
}`), 0o644))

	spec, err := eng.Parse(engine.BuildSource{Path: af})
	require.NoError(t, err)
	plan, err := eng.Plan(spec)
	require.NoError(t, err)
	res, err := eng.Build(ctx, plan, engine.BuildOptions{KernelTestMode: true})
	require.NoError(t, err)
	aap := filepath.Join(dir, "out")
	require.NoError(t, engine.WriteLocalAAP(aap, res, spec.Metadata.Name, spec.Metadata.Version))

	regRoot := filepath.Join(dir, "reg")
	reg := registry.NewLocalRegistry(regRoot)
	require.NoError(t, reg.Push(aap, spec.Metadata.Name, spec.Metadata.Version, registry.PushOptions{}))
	pulled, err := reg.Pull(spec.Metadata.Name, spec.Metadata.Version, registry.PullOptions{})
	require.NoError(t, err)
	ap, err := registry.LoadAgentPackage(pulled)
	require.NoError(t, err)
	require.Equal(t, "int", ap.Name)
}
