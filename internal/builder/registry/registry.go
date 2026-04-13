// Package registry implements local and logical Agent package registry (M7).
package registry

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/agentos/aos/internal/builder/engine"
)

// RegistryType identifies backend (extensible).
type RegistryType string

const (
	RegistryLocal RegistryType = "local"
)

// PushOptions for push.
type PushOptions struct {
	RegistryType RegistryType
	LocalRoot    string
}

// PullOptions for pull.
type PullOptions struct {
	RegistryType RegistryType
	LocalRoot    string
}

// PackageInfo is search/list metadata.
type PackageInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Path    string `json:"path"`
}

// LocalRegistry stores packages under a root directory.
type LocalRegistry struct {
	root string
	mu   sync.RWMutex
}

// NewLocalRegistry creates a registry rooted at root (e.g. ./var/aos-registry).
func NewLocalRegistry(root string) *LocalRegistry {
	return &LocalRegistry{root: root}
}

func (r *LocalRegistry) indexPath() string {
	return filepath.Join(r.root, "index.json")
}

// Push copies an AAP directory produced by engine.WriteLocalAAP.
func (r *LocalRegistry) Push(pkgDir string, name, version string, opts PushOptions) error {
	if opts.LocalRoot != "" {
		r.root = opts.LocalRoot
	}
	if err := os.MkdirAll(r.root, 0o755); err != nil {
		return err
	}
	dest := filepath.Join(r.root, safeName(name), safeName(version))
	if err := os.RemoveAll(dest); err != nil && !os.IsNotExist(err) {
		return err
	}
	if err := copyDir(pkgDir, dest); err != nil {
		return err
	}
	return r.addIndex(PackageInfo{Name: name, Version: version, Path: dest})
}

// Pull returns the path to a stored package directory.
func (r *LocalRegistry) Pull(name, version string, opts PullOptions) (string, error) {
	if opts.LocalRoot != "" {
		r.root = opts.LocalRoot
	}
	p := filepath.Join(r.root, safeName(name), safeName(version))
	if st, err := os.Stat(p); err != nil || !st.IsDir() {
		return "", fmt.Errorf("registry: package not found %s@%s", name, version)
	}
	return p, nil
}

// ListVersions returns known versions for a name from index.
func (r *LocalRegistry) ListVersions(name string) ([]string, error) {
	idx, err := r.readIndex()
	if err != nil {
		return nil, err
	}
	var vs []string
	for _, e := range idx {
		if e.Name == name {
			vs = append(vs, e.Version)
		}
	}
	return vs, nil
}

func (r *LocalRegistry) readIndex() ([]PackageInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	b, err := os.ReadFile(r.indexPath())
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []PackageInfo
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *LocalRegistry) addIndex(entry PackageInfo) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	var list []PackageInfo
	if b, err := os.ReadFile(r.indexPath()); err == nil {
		_ = json.Unmarshal(b, &list)
	}
	list = append(list, entry)
	b, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(r.indexPath(), b, 0o644)
}

func safeName(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, "/", "_"), "\\", "_")
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
}

// LoadAgentPackage reads manifest from a pulled directory into engine.AgentPackage.
func LoadAgentPackage(dir string) (*engine.AgentPackage, error) {
	manPath := filepath.Join(dir, "manifest.json")
	b, err := os.ReadFile(manPath)
	if err != nil {
		return nil, err
	}
	layersPath := filepath.Join(dir, "layers.json")
	var digests []string
	if lb, err := os.ReadFile(layersPath); err == nil {
		var meta struct {
			Name         string   `json:"name"`
			Version      string   `json:"version"`
			LayerDigests []string `json:"layer_digests"`
		}
		if json.Unmarshal(lb, &meta) == nil {
			digests = meta.LayerDigests
		}
	}
	var name, ver string
	var raw map[string]interface{}
	if json.Unmarshal(b, &raw) == nil {
		if m, ok := raw["metadata"].(map[string]interface{}); ok {
			if n, ok := m["name"].(string); ok {
				name = n
			}
			if v, ok := m["version"].(string); ok {
				ver = v
			}
		}
	}
	return &engine.AgentPackage{
		Name:         name,
		Version:      ver,
		ManifestJSON: b,
		LayerDigests: digests,
	}, nil
}
