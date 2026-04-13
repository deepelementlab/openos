package engine

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/agentos/aos/internal/builder/spec"
	"github.com/agentos/aos/internal/kernel/memory"
	"github.com/stretchr/testify/require"
)

func TestEngine_ParsePlanBuild(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "Agentfile.json")
	content := `{
  "apiVersion": "aos.io/v1",
  "kind": "AgentPackage",
  "metadata": {"name": "demo", "version": "0.1.0"},
  "steps": [{"type": "run", "command": "echo ok", "cache": true}]
}`
	require.NoError(t, os.WriteFile(p, []byte(content), 0o644))

	eng := NewEngine()
	spec, err := eng.Parse(BuildSource{Path: p})
	require.NoError(t, err)
	plan, err := eng.Plan(spec)
	require.NoError(t, err)
	require.NotEmpty(t, plan.StageHashes)

	res, err := eng.Build(context.Background(), plan, BuildOptions{})
	require.NoError(t, err)
	out := filepath.Join(dir, "aap")
	require.NoError(t, WriteLocalAAP(out, res, spec.Metadata.Name, spec.Metadata.Version))
	_, err = os.Stat(filepath.Join(out, "manifest.json"))
	require.NoError(t, err)
}

func TestEngine_MultiStagePlan(t *testing.T) {
	s := &spec.AgentPackageSpec{
		APIVersion: "aos.io/v1",
		Kind:       "AgentPackage",
		Metadata:   spec.PackageMetadata{Name: "app", Version: "1.0.0"},
		Stages: []spec.BuildStage{
			{Name: "build", Steps: []spec.BuildStep{{Type: spec.StepRun, Command: "echo a"}}},
			{Name: "final", From: "build", Steps: []spec.BuildStep{{Type: spec.StepRun, Command: "echo b"}}},
		},
	}
	eng := NewEngine()
	plan, err := eng.Plan(s)
	require.NoError(t, err)
	require.Len(t, plan.StageHashes, 2)
	require.Equal(t, []string{"build", "final"}, plan.StageOrder)
	require.Contains(t, plan.StageDigests, "build")
	require.Contains(t, plan.StageDigests, "final")
}

func TestEngine_BuildCheckpointCache(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "Agentfile.json")
	content := `{
  "apiVersion": "aos.io/v1",
  "kind": "AgentPackage",
  "metadata": {"name": "demo", "version": "0.1.0"},
  "steps": [{"type": "run", "command": "echo ok", "cache": true}]
}`
	require.NoError(t, os.WriteFile(p, []byte(content), 0o644))

	eng := NewEngine()
	spec, err := eng.Parse(BuildSource{Path: p})
	require.NoError(t, err)
	plan, err := eng.Plan(spec)
	require.NoError(t, err)
	mm := memory.NewInMemoryManager()
	cacheDir := filepath.Join(dir, "cache")
	_, err = eng.Build(context.Background(), plan, BuildOptions{
		KernelTestMode: true,
		MemoryManager:  mm,
		CacheRoot:      cacheDir,
	})
	require.NoError(t, err)
	require.NotEmpty(t, plan.StageHashes)
	c, err := OpenLayerCache(cacheDir)
	require.NoError(t, err)
	b, err := c.Get(plan.StageHashes[0])
	require.NoError(t, err)
	require.Contains(t, string(b), "checkpoint_id")
}
