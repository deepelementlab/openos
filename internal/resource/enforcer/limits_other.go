//go:build !linux

package enforcer

import "fmt"

// CgroupLimits mirrors linux build for API parity.
type CgroupLimits struct {
	CPUMax    string
	MemoryMax string
	IOMax     string
}

// ApplyCgroupV2Limits is a no-op on non-Linux platforms.
func ApplyCgroupV2Limits(groupPath string, lim CgroupLimits) error {
	return fmt.Errorf("enforcer: cgroups v2 not supported on this platform")
}
