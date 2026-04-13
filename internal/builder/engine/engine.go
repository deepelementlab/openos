package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"

	"github.com/agentos/aos/internal/builder/spec"
)

// Engine builds agent packages from AgentPackageSpec.
type Engine struct{}

// NewEngine creates a build engine.
func NewEngine() *Engine {
	return &Engine{}
}

// Parse reads JSON or YAML Agentfile from path.
func (e *Engine) Parse(src BuildSource) (*spec.AgentPackageSpec, error) {
	data, err := os.ReadFile(src.Path)
	if err != nil {
		return nil, err
	}
	ext := strings.ToLower(filepath.Ext(src.Path))
	var s spec.AgentPackageSpec
	switch ext {
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, &s); err != nil {
			return nil, fmt.Errorf("engine: yaml %s: %w", src.Path, err)
		}
	default:
		if err := json.Unmarshal(data, &s); err != nil {
			return nil, fmt.Errorf("engine: json %s: %w", src.Path, err)
		}
	}
	if s.APIVersion == "" {
		return nil, fmt.Errorf("engine: missing apiVersion in %s", src.Path)
	}
	return &s, nil
}

// Plan computes layer hashes (content-addressable cache keys) using multi-stage resolution.
func (e *Engine) Plan(s *spec.AgentPackageSpec) (*BuildPlan, error) {
	return PlanMultiStage(s)
}

// Build produces a logical package (MVP: no container execution; records digests only).
func (e *Engine) Build(ctx context.Context, plan *BuildPlan, opts BuildOptions) (*BuildResult, error) {
	if plan == nil {
		return nil, fmt.Errorf("engine: nil plan")
	}
	maxP := opts.Parallelism
	if maxP <= 0 {
		maxP = 4
	}
	// Parallel per-stage hook (MVP no-op; real execution would run here).
	if err := RunPlanStagesParallel(ctx, plan, maxP, func(ctx context.Context, stage spec.BuildStage) error {
		_ = stage
		return nil
	}); err != nil {
		return nil, err
	}

	var cache *LayerCache
	if !opts.SkipCache {
		var err error
		cache, err = OpenLayerCache(opts.CacheRoot)
		if err != nil {
			cache = nil
		}
	}
	pkgHint := plan.Spec.Metadata.Name + "@" + plan.Spec.Metadata.Version
	for i, d := range plan.StageHashes {
		if cache == nil {
			continue
		}
		if cache.Has(d) && !opts.SkipCache {
			continue
		}
		meta := LayerMeta{PackageHint: pkgHint}
		if opts.KernelTestMode && opts.MemoryManager != nil && i == 0 {
			if cp, err := CheckpointBuildState(ctx, opts.MemoryManager, "", plan); err == nil {
				meta.CheckpointID = cp.ID
			}
		}
		_ = cache.Put(d, meta)
	}

	manifest, err := json.MarshalIndent(plan.Spec, "", "  ")
	if err != nil {
		return nil, err
	}
	return &BuildResult{
		PackageID:    uuid.NewString(),
		LayerDigests: plan.StageHashes,
		Manifest:     manifest,
	}, nil
}

// WriteLocalAAP writes a minimal bundle directory (manifest + layers index).
func WriteLocalAAP(outDir string, res *BuildResult, name, version string) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(outDir, "manifest.json"), res.Manifest, 0o644); err != nil {
		return err
	}
	layers := map[string]interface{}{
		"package_id":    res.PackageID,
		"name":          name,
		"version":       version,
		"layer_digests": res.LayerDigests,
	}
	b, _ := json.MarshalIndent(layers, "", "  ")
	return os.WriteFile(filepath.Join(outDir, "layers.json"), b, 0o644)
}
