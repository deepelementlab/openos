package packaging

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseManifest(t *testing.T) {
	j := `{
  "apiVersion": "openos.agent/v0alpha1",
  "kind": "AgentPackage",
  "metadata": {"name": "demo", "version": "1.0.0"},
  "spec": {"image": "docker.io/library/alpine:latest"}
}`
	m, err := ParseManifest(strings.NewReader(j))
	require.NoError(t, err)
	require.Equal(t, "demo", m.Metadata.Name)
	require.Contains(t, m.Spec.Image, "alpine")
}
