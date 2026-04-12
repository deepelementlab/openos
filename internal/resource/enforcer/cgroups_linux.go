//go:build linux

package enforcer

import "os"

// CgroupV2Unified reports whether the unified cgroup v2 hierarchy is available.
func CgroupV2Unified() bool {
	st, err := os.Stat("/sys/fs/cgroup/cgroup.controllers")
	return err == nil && !st.IsDir()
}

// CgroupRoot returns the default cgroup mount path (v2 unified).
func CgroupRoot() string {
	return "/sys/fs/cgroup"
}
