package deployment

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPipeline_PrepareFromManifest(t *testing.T) {
	ctx := context.Background()
	j := `{
	  "apiVersion": "openos.agent/v0alpha1",
	  "kind": "AgentPackage",
	  "metadata": {"name": "x", "version": "1"},
	  "spec": {"image": "docker.io/library/alpine:latest", "command": ["/bin/sh"]}
	}`
	m, err := NewPuller().PullFromReader(ctx, strings.NewReader(j))
	require.NoError(t, err)
	p := NewPipeline()
	res, err := p.PrepareFromManifest(ctx, m)
	require.NoError(t, err)
	require.Contains(t, res.Spec.Image, "alpine")
}
