// Package network contains tenant-scoped network namespace metadata (Linux netns integration TBD).
package network

// NamespaceID identifies an isolated network namespace for a tenant workload.
type NamespaceID string

// Spec describes desired network isolation for a tenant.
type Spec struct {
	TenantID string
	Veth     string
	Bridge   string
}
