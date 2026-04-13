package integration

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/agentos/aos/internal/builder/engine"
	"github.com/agentos/aos/internal/builder/registry"
	"github.com/agentos/aos/pkg/runtime/facade"
)

func TestMultistageAAP_BuildPushPullAndFacadeSpec(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	af := filepath.Join(dir, "Agentfile.yaml")
	yaml := `apiVersion: aos.io/v1
kind: AgentPackage
metadata:
  name: msapp
  version: "1.2.3"
config:
  image: msapp/runtime:1.2.3
entrypoint:
  command: ["/app/agent"]
  args: ["--config", "/etc/agent.yaml"]
stages:
  - name: build
    steps:
      - type: run
        command: compile
  - name: final
    from: build
    steps:
      - type: run
        command: bundle
`
	require.NoError(t, os.WriteFile(af, []byte(yaml), 0o644))

	eng := engine.NewEngine()
	spec, err := eng.Parse(engine.BuildSource{Path: af})
	require.NoError(t, err)
	plan, err := eng.Plan(spec)
	require.NoError(t, err)
	require.Len(t, plan.StageHashes, 2)

	res, err := eng.Build(ctx, plan, engine.BuildOptions{CacheRoot: filepath.Join(dir, "cache")})
	require.NoError(t, err)
	aap := filepath.Join(dir, "aap")
	require.NoError(t, engine.WriteLocalAAP(aap, res, spec.Metadata.Name, spec.Metadata.Version))

	regRoot := filepath.Join(dir, "reg")
	reg := registry.NewLocalRegistry(regRoot)
	require.NoError(t, reg.Push(aap, spec.Metadata.Name, spec.Metadata.Version, registry.PushOptions{}))
	pulled, err := reg.Pull(spec.Metadata.Name, spec.Metadata.Version, registry.PullOptions{})
	require.NoError(t, err)

	pkg, err := registry.LoadAgentPackage(pulled)
	require.NoError(t, err)
	require.Equal(t, "msapp", pkg.Name)

	agentSpec, _, err := facade.AgentSpecFromPackage(pkg)
	require.NoError(t, err)
	require.Equal(t, "msapp", agentSpec.Name)
	require.Equal(t, "msapp/runtime:1.2.3", agentSpec.Image)
	require.Equal(t, []string{"/app/agent"}, agentSpec.Command)
	require.Equal(t, []string{"--config", "/etc/agent.yaml"}, agentSpec.Args)
}
