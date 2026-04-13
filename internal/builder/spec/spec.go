// Package spec defines the AOS Agent Package (AAP) manifest schema.
package spec

// AgentPackageSpec is the top-level manifest (Agentfile / manifest.json).
type AgentPackageSpec struct {
	APIVersion string          `json:"apiVersion" yaml:"apiVersion"`
	Kind       string          `json:"kind" yaml:"kind"`
	Metadata   PackageMetadata `json:"metadata" yaml:"metadata"`
	From       string          `json:"from,omitempty" yaml:"from,omitempty"`
	Config     AgentConfig     `json:"config,omitempty" yaml:"config,omitempty"`
	Steps      []BuildStep     `json:"steps,omitempty" yaml:"steps,omitempty"`
	// Stages defines multi-stage builds (Docker-style). When empty, From+Steps form a single implicit stage.
	Stages     []BuildStage         `json:"stages,omitempty" yaml:"stages,omitempty"`
	Deps       DependencySpec       `json:"dependencies,omitempty" yaml:"dependencies,omitempty"`
	Resources  ResourceRequirements `json:"resources,omitempty" yaml:"resources,omitempty"`
	Entrypoint EntrypointSpec       `json:"entrypoint,omitempty" yaml:"entrypoint,omitempty"`
}

// PackageMetadata identifies the package.
type PackageMetadata struct {
	Name        string            `json:"name" yaml:"name"`
	Version     string            `json:"version" yaml:"version"`
	Labels      map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
	Description string            `json:"description,omitempty" yaml:"description,omitempty"`
}

// AgentConfig is runtime-oriented defaults baked into the package.
type AgentConfig struct {
	Env     map[string]string `json:"env,omitempty" yaml:"env,omitempty"`
	WorkDir string            `json:"workDir,omitempty" yaml:"workDir,omitempty"`
	Image   string            `json:"image,omitempty" yaml:"image,omitempty"`
}

// StepType enumerates build step kinds.
type StepType string

const (
	StepInstall StepType = "install"
	StepCopy    StepType = "copy"
	StepRun     StepType = "run"
	StepPlugin  StepType = "plugin"
)

// BuildStage is one named stage in a multi-stage Agentfile.
type BuildStage struct {
	Name      string      `json:"name" yaml:"name"`
	From      string      `json:"from,omitempty" yaml:"from,omitempty"`
	DependsOn []string    `json:"dependsOn,omitempty" yaml:"dependsOn,omitempty"`
	Steps     []BuildStep `json:"steps,omitempty" yaml:"steps,omitempty"`
}

// BuildStep is one layer-producing instruction.
type BuildStep struct {
	Type    StepType          `json:"type" yaml:"type"`
	Command string            `json:"command,omitempty" yaml:"command,omitempty"`
	Args    map[string]string `json:"args,omitempty" yaml:"args,omitempty"`
	Cache   bool              `json:"cache,omitempty" yaml:"cache,omitempty"`
}

// DependencySpec declares external requirements.
type DependencySpec struct {
	Agents   []AgentDependency   `json:"agents,omitempty" yaml:"agents,omitempty"`
	Services []ServiceDependency `json:"services,omitempty" yaml:"services,omitempty"`
	Volumes  []VolumeDependency  `json:"volumes,omitempty" yaml:"volumes,omitempty"`
}

// AgentDependency references another agent package.
type AgentDependency struct {
	Name    string `json:"name" yaml:"name"`
	Version string `json:"version,omitempty" yaml:"version,omitempty"`
	// Ref is a full package reference, e.g. aos/postgres:14.2 (takes precedence when set).
	Ref string `json:"ref,omitempty" yaml:"ref,omitempty"`
}

// ServiceDependency references a platform service.
type ServiceDependency struct {
	Name string `json:"name" yaml:"name"`
}

// VolumeDependency references persistent storage.
type VolumeDependency struct {
	Name string `json:"name" yaml:"name"`
	Size string `json:"size,omitempty" yaml:"size,omitempty"`
}

// ResourceRequirements for scheduling hints.
type ResourceRequirements struct {
	CPU    string `json:"cpu,omitempty" yaml:"cpu,omitempty"`
	Memory string `json:"memory,omitempty" yaml:"memory,omitempty"`
	GPU    int    `json:"gpu,omitempty" yaml:"gpu,omitempty"`
}

// EntrypointSpec defines how the agent starts.
type EntrypointSpec struct {
	Command []string `json:"command,omitempty" yaml:"command,omitempty"`
	Args    []string `json:"args,omitempty" yaml:"args,omitempty"`
}
