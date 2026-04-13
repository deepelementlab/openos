package engine

import (
	"context"
	"testing"

	"github.com/agentos/aos/internal/builder/spec"
	"github.com/stretchr/testify/require"
)

func TestTopoSortStages_linear(t *testing.T) {
	stages := []spec.BuildStage{
		{Name: "a", Steps: []spec.BuildStep{{Type: spec.StepRun, Command: "1"}}},
		{Name: "b", From: "a", Steps: []spec.BuildStep{{Type: spec.StepRun, Command: "2"}}},
	}
	out, err := TopoSortStages(stages)
	require.NoError(t, err)
	require.Equal(t, []string{"a", "b"}, stageOrderNames(out))
}

func TestTopoSortStages_cycle(t *testing.T) {
	stages := []spec.BuildStage{
		{Name: "a", DependsOn: []string{"b"}},
		{Name: "b", DependsOn: []string{"a"}},
	}
	_, err := TopoSortStages(stages)
	require.Error(t, err)
}

func TestDAGBuilder_ExecuteParallel(t *testing.T) {
	stages := []spec.BuildStage{
		{Name: "a"},
		{Name: "b"},
		{Name: "c", DependsOn: []string{"a", "b"}},
	}
	db := NewDAGBuilder(stages)
	var seen int
	err := db.ExecuteParallel(t.Context(), 4, stages, func(ctx context.Context, st spec.BuildStage) error {
		seen++
		return nil
	})
	require.NoError(t, err)
	require.Equal(t, 3, seen)
}
