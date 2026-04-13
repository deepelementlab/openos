// Package cilium provides a thin abstraction over Cilium NetworkPolicy-style rules.
package cilium

import (
	"context"
	"fmt"
)

// EndpointSelector identifies pods/agents (labels).
type EndpointSelector map[string]string

// PortRule is an L4 allow rule.
type PortRule struct {
	Port     int
	Protocol string // tcp, udp
}

// Policy models a tenant isolation policy (Kubernetes CRD-shaped).
type Policy struct {
	Name        string
	Namespace   string
	TenantID    string
	IngressFrom []EndpointSelector
	EgressTo    []EndpointSelector
	Ports       []PortRule
}

// Client is a pluggable backend (real cluster vs CI stub).
type Client interface {
	ApplyPolicy(ctx context.Context, p Policy) error
	DeletePolicy(ctx context.Context, name, namespace string) error
}

// StubClient records calls for tests.
type StubClient struct {
	Applied []Policy
}

// ApplyPolicy implements Client.
func (s *StubClient) ApplyPolicy(ctx context.Context, p Policy) error {
	s.Applied = append(s.Applied, p)
	return nil
}

// DeletePolicy implements Client.
func (s *StubClient) DeletePolicy(ctx context.Context, name, namespace string) error {
	return nil
}

// Validate checks minimal fields before apply.
func Validate(p Policy) error {
	if p.Name == "" || p.Namespace == "" {
		return fmt.Errorf("cilium: policy name/namespace required")
	}
	return nil
}
