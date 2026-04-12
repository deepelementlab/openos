//go:build !linux

package enforcer

// CgroupV2Unified is false on non-Linux platforms.
func CgroupV2Unified() bool {
	return false
}

// CgroupRoot returns empty on non-Linux; cgroups not applicable.
func CgroupRoot() string {
	return ""
}
