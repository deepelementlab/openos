// Package dns provides internal name resolution for agents (stub for full DNS integration).
package dns

import (
	"fmt"
	"net"
)

// FQDN returns a conventional internal FQDN for an agent.
func FQDN(agentName, tenantID string) string {
	return fmt.Sprintf("%s.%s.openos.local", agentName, tenantID)
}

// Resolve looks up an agent FQDN (stub: loopback for local dev).
func Resolve(fqdn string) ([]net.IP, error) {
	if fqdn == "" {
		return nil, fmt.Errorf("dns: empty fqdn")
	}
	return []net.IP{net.IPv4(127, 0, 0, 1)}, nil
}
