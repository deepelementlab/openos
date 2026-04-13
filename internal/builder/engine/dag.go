package engine

import (
	"context"
	"fmt"

	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"

	"github.com/agentos/aos/internal/builder/spec"
)

// DAGBuilder holds stage -> dependency edges (each stage depends on listed stages).
type DAGBuilder struct {
	graph map[string][]string
}

// NewDAGBuilder builds edges from explicit DependsOn and From references to other stage names.
func NewDAGBuilder(stages []spec.BuildStage) *DAGBuilder {
	names := make(map[string]struct{})
	for _, st := range stages {
		if st.Name != "" {
			names[st.Name] = struct{}{}
		}
	}
	g := make(map[string][]string)
	for _, st := range stages {
		if st.Name == "" {
			continue
		}
		var deps []string
		seen := make(map[string]struct{})
		for _, d := range st.DependsOn {
			if d == "" || d == st.Name {
				continue
			}
			if _, ok := names[d]; !ok {
				continue
			}
			if _, ok := seen[d]; ok {
				continue
			}
			seen[d] = struct{}{}
			deps = append(deps, d)
		}
		if st.From != "" {
			if _, ok := names[st.From]; ok && st.From != st.Name {
				if _, ok := seen[st.From]; !ok {
					deps = append(deps, st.From)
				}
			}
		}
		g[st.Name] = deps
	}
	return &DAGBuilder{graph: g}
}

// TopoSortStages returns stages in dependency order (dependencies first).
func TopoSortStages(stages []spec.BuildStage) ([]spec.BuildStage, error) {
	db := NewDAGBuilder(stages)
	byName := make(map[string]spec.BuildStage, len(stages))
	for _, st := range stages {
		if st.Name == "" {
			return nil, fmt.Errorf("dag: stage missing name")
		}
		if _, dup := byName[st.Name]; dup {
			return nil, fmt.Errorf("dag: duplicate stage name %q", st.Name)
		}
		byName[st.Name] = st
	}
	inDegree := make(map[string]int, len(byName))
	rev := make(map[string][]string)
	for n := range byName {
		inDegree[n] = 0
	}
	for child, deps := range db.graph {
		cnt := 0
		for _, d := range deps {
			if _, ok := byName[d]; !ok {
				continue
			}
			cnt++
			rev[d] = append(rev[d], child)
		}
		inDegree[child] = cnt
	}
	var queue []string
	for n, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, n)
		}
	}
	var order []string
	for len(queue) > 0 {
		n := queue[0]
		queue = queue[1:]
		order = append(order, n)
		for _, m := range rev[n] {
			inDegree[m]--
			if inDegree[m] == 0 {
				queue = append(queue, m)
			}
		}
	}
	if len(order) != len(byName) {
		return nil, fmt.Errorf("dag: cycle or unresolved dependencies in stages")
	}
	out := make([]spec.BuildStage, 0, len(order))
	for _, name := range order {
		out = append(out, byName[name])
	}
	return out, nil
}

// Waves returns batches of stage names that can run in parallel (same depth).
func (d *DAGBuilder) Waves(stages []spec.BuildStage) ([][]string, error) {
	ordered, err := TopoSortStages(stages)
	if err != nil {
		return nil, err
	}
	depth := make(map[string]int, len(ordered))
	for _, st := range ordered {
		maxD := 0
		for _, dep := range d.graph[st.Name] {
			if v, ok := depth[dep]; ok && v+1 > maxD {
				maxD = v + 1
			}
		}
		depth[st.Name] = maxD
	}
	maxDepth := 0
	for _, v := range depth {
		if v > maxDepth {
			maxDepth = v
		}
	}
	buckets := make([][]string, maxDepth+1)
	for _, st := range ordered {
		buckets[depth[st.Name]] = append(buckets[depth[st.Name]], st.Name)
	}
	return buckets, nil
}

// ExecuteParallel runs fn for each stage name in wave order; stages within the same wave run concurrently.
func (d *DAGBuilder) ExecuteParallel(ctx context.Context, maxConcurrency int, stages []spec.BuildStage, fn func(ctx context.Context, stage spec.BuildStage) error) error {
	if maxConcurrency < 1 {
		maxConcurrency = 4
	}
	waves, err := d.Waves(stages)
	if err != nil {
		return err
	}
	byName := make(map[string]spec.BuildStage, len(stages))
	for _, st := range stages {
		byName[st.Name] = st
	}
	sem := semaphore.NewWeighted(int64(maxConcurrency))
	for _, wave := range waves {
		g, ctx := errgroup.WithContext(ctx)
		for _, name := range wave {
			name := name
			st := byName[name]
			g.Go(func() error {
				if err := sem.Acquire(ctx, 1); err != nil {
					return err
				}
				defer sem.Release(1)
				return fn(ctx, st)
			})
		}
		if err := g.Wait(); err != nil {
			return err
		}
	}
	return nil
}

// RunPlanStagesParallel optionally recomputes or validates layers per stage in parallel (MVP hooks).
func RunPlanStagesParallel(ctx context.Context, plan *BuildPlan, maxConcurrency int, fn func(ctx context.Context, stage spec.BuildStage) error) error {
	if plan == nil || len(plan.NormalizedStages) == 0 {
		return nil
	}
	db := NewDAGBuilder(plan.NormalizedStages)
	return db.ExecuteParallel(ctx, maxConcurrency, plan.NormalizedStages, fn)
}
