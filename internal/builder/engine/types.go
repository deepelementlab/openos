package engine

import (
	"github.com/agentos/aos/internal/builder/spec"
	"github.com/agentos/aos/internal/kernel/memory"
)

// BuildSource points to an Agentfile (JSON or YAML).
type BuildSource struct {
	Path string
}

// BuildOptions tunes the build.
type BuildOptions struct {
	Tag            string
	Parallelism    int
	SkipCache      bool
	KernelTestMode bool // when true with MemoryManager, record kernel checkpoint on first layer
	MemoryManager  memory.Manager
	CacheRoot      string // optional; default UserCacheDir/aos/cache
}

// BuildPlan is a resolved build (multi-stage aware).
type BuildPlan struct {
	Spec         spec.AgentPackageSpec
	StageHashes  []string // one digest per build step in topo stage order
	StageNames   []string // stage name for each entry in StageHashes (same length)
	CacheKeyRoot string
	// StageOrder is the topologically sorted stage names.
	StageOrder []string
	// StageDigests maps stage name -> aggregate digest after that stage completes.
	StageDigests map[string]string
	// NormalizedStages is the ordered stage list used for DAG execution.
	NormalizedStages []spec.BuildStage
}

// BuildResult is the output artifact metadata.
type BuildResult struct {
	PackageID    string
	LayerDigests []string
	Manifest     []byte
}

// AgentPackage is an in-memory representation before registry push.
type AgentPackage struct {
	Name         string
	Version      string
	ManifestJSON []byte
	LayerDigests []string
}
