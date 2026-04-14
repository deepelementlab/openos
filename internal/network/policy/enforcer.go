// Package policy provides network policy evaluation hooks (iptables/nftables integration TBD).
package policy

import (
	"context"
	"fmt"
	"strings"
)

// Rule is a simplified allow/deny rule between tenants or labels.
type Rule struct {
	SrcTenant string
	DstTenant string
	Allow     bool
}

// Enforcer applies policy rules (in-memory) and can render firewall snippets.
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

// RenderIPTables generates iptables-save style lines for tenant pair rules (default DROP at end of chain).
func (e *Enforcer) RenderIPTables(chainName string) []string {
	var b strings.Builder
	fmt.Fprintf(&b, ":%s - [0:0]\n", chainName)
	for _, r := range e.rules {
		if !r.Allow {
			continue
		}
		fmt.Fprintf(&b, "-A %s -m comment --comment \"allow %s->%s\" -j ACCEPT\n", chainName, r.SrcTenant, r.DstTenant)
	}
	fmt.Fprintf(&b, "-A %s -j DROP\n", chainName)
	return strings.Split(strings.TrimSuffix(b.String(), "\n"), "\n")
}

// RenderNftables returns nftables table snippet for a simple tenant filter hook.
func (e *Enforcer) RenderNftables(table, chain string) []string {
	lines := []string{
		fmt.Sprintf("table %s {", table),
		fmt.Sprintf("  chain %s {", chain),
		"    type filter hook forward priority 0; policy drop;",
	}
	for _, r := range e.rules {
		if r.Allow {
			lines = append(lines, fmt.Sprintf("    # allow %s -> %s", r.SrcTenant, r.DstTenant))
		}
	}
	lines = append(lines, "  }", "}")
	return lines
}
