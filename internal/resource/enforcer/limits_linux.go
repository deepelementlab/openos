//go:build linux

package enforcer

import (
	"fmt"
	"os"
	"path/filepath"
)

// CgroupLimits holds cgroup v2 controller knob values (see cgroup v2 docs).
type CgroupLimits struct {
	// CPUMax content for cpu.max, e.g. "50000 100000" or "max"
	CPUMax string
	// MemoryMax bytes as decimal string for memory.max
	MemoryMax string
	// IOMax line for io.max, e.g. "8:0 rbps=1048576 wbps=1048576"
	IOMax string
}

// ApplyCgroupV2Limits creates groupPath under the unified cgroup root and writes limits.
// groupPath is relative (e.g. "aos.slice/tenant-a/agent-1").
func ApplyCgroupV2Limits(groupPath string, lim CgroupLimits) error {
	if groupPath == "" {
		return fmt.Errorf("enforcer: empty cgroup path")
	}
	if !CgroupV2Unified() {
		return fmt.Errorf("enforcer: cgroup v2 unified hierarchy not available")
	}
	base := filepath.Join(CgroupRoot(), groupPath)
	if err := os.MkdirAll(base, 0o755); err != nil {
		return err
	}
	if lim.CPUMax != "" {
		if err := os.WriteFile(filepath.Join(base, "cpu.max"), []byte(lim.CPUMax+"\n"), 0o644); err != nil {
			return fmt.Errorf("cpu.max: %w", err)
		}
	}
	if lim.MemoryMax != "" {
		if err := os.WriteFile(filepath.Join(base, "memory.max"), []byte(lim.MemoryMax+"\n"), 0o644); err != nil {
			return fmt.Errorf("memory.max: %w", err)
		}
	}
	if lim.IOMax != "" {
		if err := os.WriteFile(filepath.Join(base, "io.max"), []byte(lim.IOMax+"\n"), 0o644); err != nil {
			return fmt.Errorf("io.max: %w", err)
		}
	}
	return nil
}
