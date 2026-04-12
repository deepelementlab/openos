// Package cni wraps CNI ADD/DEL operations for agent network namespaces.
package cni

// NetworkConfig is a minimal CNI JSON config subset.
type NetworkConfig struct {
	Name       string `json:"name"`
	CNIVersion string `json:"cniVersion"`
	Type       string `json:"type"`
}
