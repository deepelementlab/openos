// Package packaging implements OpenOS Agent package manifest (v0 spec).
package packaging

import (
	"encoding/json"
	"fmt"
	"io"
)

const SpecVersionV0 = "openos.agent/v0alpha1"

// Manifest is the portable description of an agent workload.
type Manifest struct {
	APIVersion string            `json:"apiVersion"`
	Kind       string            `json:"kind"` // AgentPackage
	Metadata   ManifestMetadata  `json:"metadata"`
	Spec       ManifestSpec      `json:"spec"`
}

// ManifestMetadata identifies the package.
type ManifestMetadata struct {
	Name        string            `json:"name"`
	Version     string            `json:"version"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// ManifestSpec describes runtime and image references.
type ManifestSpec struct {
	Image   string            `json:"image,omitempty"`   // OCI image reference
	Command []string          `json:"command,omitempty"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
	// Resources uses Kubernetes-style quantities in JSON as strings or numbers per implementation.
	Resources map[string]string `json:"resources,omitempty"`
}

// ParseManifest reads and validates a JSON manifest.
func ParseManifest(r io.Reader) (*Manifest, error) {
	var m Manifest
	if err := json.NewDecoder(r).Decode(&m); err != nil {
		return nil, err
	}
	if m.APIVersion == "" {
		return nil, fmt.Errorf("packaging: apiVersion required")
	}
	if m.Kind != "AgentPackage" {
		return nil, fmt.Errorf("packaging: kind must be AgentPackage")
	}
	if m.Metadata.Name == "" || m.Metadata.Version == "" {
		return nil, fmt.Errorf("packaging: metadata.name and metadata.version required")
	}
	return &m, nil
}
