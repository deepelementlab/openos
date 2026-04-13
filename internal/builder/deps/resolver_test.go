package deps

import (
	"context"
	"testing"

	"github.com/agentos/aos/internal/builder/spec"
	"github.com/stretchr/testify/require"
)

func TestResolve(t *testing.T) {
	g, err := Resolve(context.Background(), spec.DependencySpec{
		Agents: []spec.AgentDependency{
			{Ref: "aos/postgres:14.2"},
			{Name: "redis", Version: "7"},
		},
	})
	require.NoError(t, err)
	require.Len(t, g.Agents, 2)
	require.Equal(t, "14.2", g.Agents[0].Version)
}

func TestParseRef(t *testing.T) {
	n, v, err := ParseRef("myorg/app:v1")
	require.NoError(t, err)
	require.Equal(t, "myorg/app", n)
	require.Equal(t, "v1", v)
}

func TestResolve_EmptyAgentDependency(t *testing.T) {
	_, err := Resolve(context.Background(), spec.DependencySpec{
		Agents: []spec.AgentDependency{{}},
	})
	require.Error(t, err)
}
