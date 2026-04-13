package engine

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"

	"github.com/agentos/aos/internal/builder/spec"
)

// MultiStageEngine coordinates multi-stage builds with optional layer cache.
type MultiStageEngine struct {
	stages []spec.BuildStage
	cache  *LayerCache
}

// NewMultiStageEngine creates a multi-stage engine; cache may be nil.
func NewMultiStageEngine(cache *LayerCache) *MultiStageEngine {
	return &MultiStageEngine{cache: cache}
}

// NormalizeStages returns explicit stages or a single synthetic stage from legacy From+Steps.
func NormalizeStages(s *spec.AgentPackageSpec) []spec.BuildStage {
	if s == nil {
		return nil
	}
	if len(s.Stages) > 0 {
		out := make([]spec.BuildStage, len(s.Stages))
		copy(out, s.Stages)
		return out
	}
	name := "default"
	if s.Metadata.Name != "" {
		name = s.Metadata.Name + "-default"
	}
	return []spec.BuildStage{{
		Name:  name,
		From:  s.From,
		Steps: append([]spec.BuildStep(nil), s.Steps...),
	}}
}

// hashStep returns a content-addressable digest for one build step (stable with engine.Plan).
func hashStep(step spec.BuildStep) string {
	h := sha256.New()
	_, _ = h.Write([]byte(string(step.Type)))
	_, _ = h.Write([]byte(step.Command))
	keys := make([]string, 0, len(step.Args))
	for k := range step.Args {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		_, _ = h.Write([]byte(k))
		_, _ = h.Write([]byte(step.Args[k]))
	}
	return hex.EncodeToString(h.Sum(nil))
}

// stageNames builds a set of stage names for From resolution.
func stageNameSet(stages []spec.BuildStage) map[string]struct{} {
	m := make(map[string]struct{}, len(stages))
	for _, st := range stages {
		if st.Name != "" {
			m[st.Name] = struct{}{}
		}
	}
	return m
}

// resolveStageSeed picks the hash seed for a stage: prior stage digest or external From string.
func resolveStageSeed(from string, stageDigests map[string]string, knownStages map[string]struct{}) string {
	if from == "" {
		return ""
	}
	if _, ok := knownStages[from]; ok {
		if d, ok := stageDigests[from]; ok {
			return d
		}
		// Dependency not computed yet — caller must topo-order first.
		return ""
	}
	h := sha256.Sum256([]byte("external-from:" + from))
	return hex.EncodeToString(h[:])
}

// aggregateStageDigest folds seed + ordered step hashes into one stage identifier.
func aggregateStageDigest(stageName, seed string, stepHashes []string) string {
	h := sha256.New()
	_, _ = h.Write([]byte("stage:"))
	_, _ = h.Write([]byte(stageName))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write([]byte(seed))
	for _, sh := range stepHashes {
		_, _ = h.Write([]byte{0})
		_, _ = h.Write([]byte(sh))
	}
	return hex.EncodeToString(h.Sum(nil))
}

// PlanMultiStage computes layer hashes and per-stage aggregates following topo order.
func PlanMultiStage(s *spec.AgentPackageSpec) (*BuildPlan, error) {
	if s == nil {
		return nil, fmt.Errorf("multistage: nil spec")
	}
	stages := NormalizeStages(s)
	ordered, err := TopoSortStages(stages)
	if err != nil {
		return nil, err
	}
	known := stageNameSet(stages)
	stageDigests := make(map[string]string, len(ordered))
	var flat []string
	var stagePerLayer []string
	for _, st := range ordered {
		seed := resolveStageSeed(st.From, stageDigests, known)
		if st.From != "" {
			if _, isStage := known[st.From]; isStage {
				if seed == "" {
					return nil, fmt.Errorf("multistage: unresolved from stage %q for stage %q", st.From, st.Name)
				}
			}
		}
		var stepHashes []string
		for _, step := range st.Steps {
			d := hashStep(step)
			stepHashes = append(stepHashes, d)
			flat = append(flat, d)
			stagePerLayer = append(stagePerLayer, st.Name)
		}
		agg := aggregateStageDigest(st.Name, seed, stepHashes)
		stageDigests[st.Name] = agg
	}
	root := sha256.Sum256([]byte(s.Metadata.Name + "@" + s.Metadata.Version))
	return &BuildPlan{
		Spec:             *s,
		StageHashes:      flat,
		StageNames:       stagePerLayer,
		CacheKeyRoot:     hex.EncodeToString(root[:]),
		StageOrder:       stageOrderNames(ordered),
		StageDigests:     stageDigests,
		NormalizedStages: ordered,
	}, nil
}

func stageOrderNames(stages []spec.BuildStage) []string {
	out := make([]string, 0, len(stages))
	for _, st := range stages {
		out = append(out, st.Name)
	}
	return out
}

// BuildMultiStage is the multi-stage entrypoint; delegates to PlanMultiStage + Engine.Build.
func (m *MultiStageEngine) BuildMultiStage(ctx context.Context, s *spec.AgentPackageSpec, opts BuildOptions) (*BuildResult, error) {
	plan, err := PlanMultiStage(s)
	if err != nil {
		return nil, err
	}
	eng := NewEngine()
	return eng.Build(ctx, plan, opts)
}
