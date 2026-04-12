// Package policy provides network policy evaluation hooks (iptables/nftables integration TBD).
package policy

import "context"

// Rule is a simplified allow/deny rule between tenants or labels.
type Rule struct {
	SrcTenant string
	DstTenant string
	Allow     bool
}

// Enforcer applies policy rules (in-memory stub).
type Enforcer struct {
	rules []Rule
}

// NewEnforcer creates an enforcer with default deny across tenants.
func NewEnforcer() *Enforcer {
	return &Enforcer{}
}

// AllowTenantPair registers that traffic from src to dst is allowed.
func (e *Enforcer) AllowTenantPair(src, dst string) {
	e.rules = append(e.rules, Rule{SrcTenant: src, DstTenant: dst, Allow: true})
}

// Allowed returns whether src tenant may reach dst tenant.
func (e *Enforcer) Allowed(ctx context.Context, src, dst string) bool {
	_ = ctx
	if src == dst {
		return true
	}
	for _, r := range e.rules {
		if r.SrcTenant == src && r.DstTenant == dst && r.Allow {
			return true
		}
	}
	return false
}
