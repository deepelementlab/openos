package spec

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestAgentPackageSpec_JSON(t *testing.T) {
	s := AgentPackageSpec{
		APIVersion: "aos.io/v1",
		Kind:       "AgentPackage",
		Metadata:   PackageMetadata{Name: "demo", Version: "1.0.0"},
		Steps: []BuildStep{
			{Type: StepRun, Command: "echo hello", Cache: true},
		},
	}
	b, err := json.Marshal(s)
	require.NoError(t, err)
	var out AgentPackageSpec
	require.NoError(t, json.Unmarshal(b, &out))
	require.Equal(t, "demo", out.Metadata.Name)
}

func TestAgentPackageSpec_YAMLStagesAndDeps(t *testing.T) {
	y := `
apiVersion: aos.io/v1
kind: AgentPackage
metadata:
  name: svc
  version: "2.0.0"
stages:
  - name: build
    steps:
      - type: run
        command: make
  - name: final
    from: build
    steps:
      - type: run
        command: pack
dependencies:
  agents:
    - name: database
      ref: aos/postgres:14.2
  services:
    - name: redis
`
	var s AgentPackageSpec
	require.NoError(t, yaml.Unmarshal([]byte(y), &s))
	require.Equal(t, "svc", s.Metadata.Name)
	require.Len(t, s.Stages, 2)
	require.Equal(t, "build", s.Stages[0].Name)
	require.Equal(t, "build", s.Stages[1].From)
	require.Len(t, s.Deps.Agents, 1)
	require.Equal(t, "aos/postgres:14.2", s.Deps.Agents[0].Ref)
	require.Len(t, s.Deps.Services, 1)
	require.Equal(t, "redis", s.Deps.Services[0].Name)
}
