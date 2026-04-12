package deployment

import (
	"context"
	"fmt"
	"sort"

	"github.com/agentos/aos/pkg/packaging"
	"github.com/agentos/aos/pkg/runtime/types"
)

// PipelineResult is the output of a successful prepare step before runtime.CreateAgent.
type PipelineResult struct {
	Manifest *packaging.Manifest
	Spec     *types.AgentSpec
}

// Pipeline coordinates pull → validate → build AgentSpec.
type Pipeline struct {
	puller *Puller
}

// NewPipeline creates a deployment pipeline.
func NewPipeline() *Pipeline {
	return &Pipeline{puller: NewPuller()}
}

// PrepareFromManifest builds a types.AgentSpec minimal stub from a manifest.
func (p *Pipeline) PrepareFromManifest(ctx context.Context, m *packaging.Manifest) (*PipelineResult, error) {
	_ = ctx
	if m == nil {
		return nil, fmt.Errorf("deployment: manifest is nil")
	}
	if m.Spec.Image == "" {
		return nil, fmt.Errorf("deployment: spec.image required")
	}
	env := make([]string, 0, len(m.Spec.Env))
	for k, v := range m.Spec.Env {
		env = append(env, k+"="+v)
	}
	sort.Strings(env)
	spec := &types.AgentSpec{
		Name:    m.Metadata.Name,
		Image:   m.Spec.Image,
		Command: append([]string(nil), m.Spec.Command...),
		Args:    append([]string(nil), m.Spec.Args...),
		Env:     env,
		Labels:  m.Metadata.Labels,
	}
	return &PipelineResult{Manifest: m, Spec: spec}, nil
}
