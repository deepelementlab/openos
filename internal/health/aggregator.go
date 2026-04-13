package health

import (
	"context"
	"sync"
	"time"
)

// Level identifies aggregation scope for dashboards and SLO reporting.
type Level string

const (
	LevelSystem Level = "system"
	LevelTenant Level = "tenant"
	LevelNode   Level = "node"
	LevelAgent  Level = "agent"
)

// ComponentReport is a single check result with scope metadata.
type ComponentReport struct {
	Level     Level  `json:"level"`
	ScopeID   string `json:"scope_id,omitempty"`
	Name      string `json:"name"`
	Status    Status `json:"status"`
	Error     string `json:"error,omitempty"`
	LastCheck string `json:"last_check"`
}

// AggregatedHealth combines Checker output with hierarchical grouping.
type AggregatedHealth struct {
	Overall   Status                       `json:"overall"`
	Uptime    string                       `json:"uptime"`
	ByLevel   map[Level][]ComponentReport  `json:"by_level"`
	Raw       map[string]interface{}       `json:"raw_summary"`
	Timestamp string                       `json:"timestamp"`
}

// Aggregator wraps the base Checker and adds tenant/node/agent scoped components.
type Aggregator struct {
	mu       sync.RWMutex
	checker  *Checker
	tenants  map[string]func(ctx context.Context) error
	nodes    map[string]func(ctx context.Context) error
	agents   map[string]func(ctx context.Context) error
}

// NewAggregator creates an aggregator around an existing Checker.
func NewAggregator(c *Checker) *Aggregator {
	return &Aggregator{
		checker: c,
		tenants: make(map[string]func(ctx context.Context) error),
		nodes:   make(map[string]func(ctx context.Context) error),
		agents:  make(map[string]func(ctx context.Context) error),
	}
}

// RegisterTenantCheck adds a tenant-scoped probe (e.g. quota RPC).
func (a *Aggregator) RegisterTenantCheck(tenantID, name string, fn func(ctx context.Context) error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.checker.Register("tenant/"+tenantID+"/"+name, "tenant scope", fn)
	a.tenants[tenantID] = fn
}

// RegisterNodeCheck registers a node-level probe.
func (a *Aggregator) RegisterNodeCheck(nodeID, name string, fn func(ctx context.Context) error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.checker.Register("node/"+nodeID+"/"+name, "node scope", fn)
	a.nodes[nodeID] = fn
}

// Collect runs all checks and returns a hierarchical report.
func (a *Aggregator) Collect(ctx context.Context) *AggregatedHealth {
	a.mu.RLock()
	defer a.mu.RUnlock()

	results := a.checker.Check(ctx)
	byLevel := make(map[Level][]ComponentReport)
	var overall Status = StatusHealthy

	for name, comp := range results {
		lvl, scope := classifyComponentName(name)
		if comp.Status == StatusUnhealthy {
			overall = StatusUnhealthy
		} else if comp.Status == StatusUnknown && overall == StatusHealthy {
			overall = StatusUnknown
		}
		errStr := ""
		if comp.Error != nil {
			errStr = comp.Error.Error()
		}
		byLevel[lvl] = append(byLevel[lvl], ComponentReport{
			Level:     lvl,
			ScopeID:   scope,
			Name:      name,
			Status:    comp.Status,
			Error:     errStr,
			LastCheck: comp.LastCheck.Format(time.RFC3339),
		})
	}

	summary := a.checker.Status()
	return &AggregatedHealth{
		Overall:   overall,
		Uptime:    summary["uptime"].(string),
		ByLevel:   byLevel,
		Raw:       summary,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
}

func classifyComponentName(name string) (Level, string) {
	if len(name) > 7 && name[:7] == "tenant/" {
		parts := split3(name)
		if len(parts) >= 2 {
			return LevelTenant, parts[1]
		}
		return LevelTenant, ""
	}
	if len(name) > 5 && name[:5] == "node/" {
		parts := split3(name)
		if len(parts) >= 2 {
			return LevelNode, parts[1]
		}
		return LevelNode, ""
	}
	if len(name) > 6 && name[:6] == "agent/" {
		parts := split3(name)
		if len(parts) >= 2 {
			return LevelAgent, parts[1]
		}
		return LevelAgent, ""
	}
	return LevelSystem, ""
}

func split3(s string) []string {
	var out []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '/' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	out = append(out, s[start:])
	return out
}
