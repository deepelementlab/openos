// Package deps resolves Agentfile dependency declarations into a concrete graph.
package deps

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/agentos/aos/internal/builder/spec"
)

// ResolvedRef is a normalized package reference.
type ResolvedRef struct {
	Name    string
	Version string
	RawRef  string
}

// ResolvedGraph is the output of dependency resolution (in-memory; no registry IO).
type ResolvedGraph struct {
	Agents   []ResolvedRef
	Services []spec.ServiceDependency
	Volumes  []spec.VolumeDependency
}

// ParseRef splits "repo/name:version" using the last ':' as version delimiter.
func ParseRef(ref string) (name, version string, err error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return "", "", fmt.Errorf("deps: empty ref")
	}
	i := strings.LastIndex(ref, ":")
	if i <= 0 || i == len(ref)-1 {
		return ref, "latest", nil
	}
	return ref[:i], ref[i+1:], nil
}

// Resolve expands DependencySpec into ResolvedGraph.
func Resolve(ctx context.Context, d spec.DependencySpec) (ResolvedGraph, error) {
	_ = ctx
	var out ResolvedGraph
	for _, a := range d.Agents {
		raw := strings.TrimSpace(a.Ref)
		if raw == "" {
			if strings.TrimSpace(a.Name) == "" {
				return ResolvedGraph{}, fmt.Errorf("deps: agent dependency needs name or ref")
			}
			ver := a.Version
			if ver == "" {
				ver = "latest"
			}
			raw = fmt.Sprintf("%s:%s", a.Name, ver)
		}
		name, ver, err := ParseRef(raw)
		if err != nil {
			return ResolvedGraph{}, err
		}
		out.Agents = append(out.Agents, ResolvedRef{Name: name, Version: ver, RawRef: raw})
	}
	out.Services = append(out.Services, d.Services...)
	out.Volumes = append(out.Volumes, d.Volumes...)
	sort.Slice(out.Agents, func(i, j int) bool {
		return out.Agents[i].RawRef < out.Agents[j].RawRef
	})
	return out, nil
}
