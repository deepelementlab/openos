package facade

import (
	"encoding/json"
	"fmt"

	"github.com/agentos/aos/internal/builder/engine"
	"github.com/agentos/aos/internal/builder/spec"
	"github.com/agentos/aos/pkg/runtime/types"
)

// AgentSpecFromPackage maps a loaded AAP manifest into a runtime AgentSpec and returns the parsed spec.
func AgentSpecFromPackage(pkg *engine.AgentPackage) (*types.AgentSpec, *spec.AgentPackageSpec, error) {
	if pkg == nil {
		return nil, nil, fmt.Errorf("facade: nil package")
	}
	var s spec.AgentPackageSpec
	if err := json.Unmarshal(pkg.ManifestJSON, &s); err != nil {
		return nil, nil, fmt.Errorf("facade: manifest: %w", err)
	}
	id := s.Metadata.Name + "-" + s.Metadata.Version
	if id == "-" {
		id = "agent-unknown"
	}
	as := &types.AgentSpec{
		ID:          id,
		Name:        s.Metadata.Name,
		Image:       s.Config.Image,
		Labels:      map[string]string{},
		Annotations: map[string]string{},
	}
	for k, v := range s.Metadata.Labels {
		as.Labels[k] = v
	}
	as.Labels["aos.openos.dev/package"] = s.Metadata.Name + ":" + s.Metadata.Version
	if len(s.Entrypoint.Command) > 0 {
		as.Command = append([]string(nil), s.Entrypoint.Command...)
	}
	as.Args = append([]string(nil), s.Entrypoint.Args...)
	if s.Config.WorkDir != "" {
		as.WorkingDir = s.Config.WorkDir
	}
	for k, v := range s.Config.Env {
		as.Env = append(as.Env, k+"="+v)
	}
	return as, &s, nil
}
