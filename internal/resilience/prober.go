// Package resilience implements health probes and basic healing policies.
package resilience

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

// ProbeType mirrors k8s probe kinds.
type ProbeType string

const (
	ProbeLiveness  ProbeType = "Liveness"
	ProbeReadiness ProbeType = "Readiness"
	ProbeStartup   ProbeType = "Startup"
)

// ContainerdExecRunner runs commands inside a container (optional; wire containerd/task exec).
type ContainerdExecRunner interface {
	ExecInContainer(ctx context.Context, containerID string, argv []string) error
}

// Prober runs HTTP/TCP/exec probes. Target formats:
//   - http://host/path or https://...
//   - tcp host:port or host:port (TCP dial)
//   - exec:arg0 arg1... (local process; for containerd use ExecRunner)
//   - ctdexec:<containerID>|<argv joined by space> when ExecRunner is set
type Prober struct {
	Timeout time.Duration
	// HTTPClient override (tests)
	HTTPClient *http.Client
	// ExecRunner optional containerd exec
	ExecRunner ContainerdExecRunner
}

// NewProber creates a prober.
func NewProber() *Prober {
	return &Prober{Timeout: 5 * time.Second}
}

func (p *Prober) client() *http.Client {
	if p != nil && p.HTTPClient != nil {
		return p.HTTPClient
	}
	return &http.Client{Timeout: p.timeout()}
}

func (p *Prober) timeout() time.Duration {
	if p == nil || p.Timeout <= 0 {
		return 5 * time.Second
	}
	return p.Timeout
}

// Run executes a probe for the given target string.
func (p *Prober) Run(ctx context.Context, t ProbeType, target string) error {
	_ = t
	if target == "" {
		return fmt.Errorf("prober: empty target")
	}
	ctx, cancel := context.WithTimeout(ctx, p.timeout())
	defer cancel()

	switch {
	case strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://"):
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
		if err != nil {
			return err
		}
		resp, err := p.client().Do(req)
		if err != nil {
			return err
		}
		_ = resp.Body.Close()
		if resp.StatusCode >= 500 {
			return fmt.Errorf("prober: HTTP %d", resp.StatusCode)
		}
		return nil

	case strings.HasPrefix(target, "tcp:"):
		addr := strings.TrimPrefix(target, "tcp:")
		return probeTCP(ctx, addr)

	case strings.HasPrefix(target, "exec:"):
		line := strings.TrimPrefix(target, "exec:")
		return probeExec(ctx, line)

	case strings.HasPrefix(target, "ctdexec:"):
		if p == nil || p.ExecRunner == nil {
			return fmt.Errorf("prober: ctdexec requires ExecRunner")
		}
		rest := strings.TrimPrefix(target, "ctdexec:")
		parts := strings.SplitN(rest, "|", 2)
		if len(parts) != 2 {
			return fmt.Errorf("prober: ctdexec format ctdexec:<id>|<argv>")
		}
		cid := parts[0]
		argv := strings.Fields(parts[1])
		if len(argv) == 0 {
			return fmt.Errorf("prober: empty argv for ctdexec")
		}
		return p.ExecRunner.ExecInContainer(ctx, cid, argv)

	default:
		// Bare host:port → TCP
		if strings.Contains(target, ":") {
			return probeTCP(ctx, target)
		}
		return fmt.Errorf("prober: unsupported target %q", target)
	}
}

func probeTCP(ctx context.Context, addr string) error {
	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return err
	}
	return conn.Close()
}

func probeExec(ctx context.Context, line string) error {
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return fmt.Errorf("prober: empty exec")
	}
	c := exec.CommandContext(ctx, fields[0], fields[1:]...)
	out, err := c.CombinedOutput()
	if err != nil {
		return fmt.Errorf("exec failed: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}
