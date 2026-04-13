//go:build !linux

package enforcer

import "fmt"

// CgroupBatchItem mirrors linux build.
type CgroupBatchItem struct {
	GroupPath string
	Limits    CgroupLimits
}

// ApplyCgroupV2LimitsBatch applies limits sequentially on non-Linux (delegates to single-item API).
func ApplyCgroupV2LimitsBatch(items []CgroupBatchItem, maxParallel int) error {
	for _, it := range items {
		if err := ApplyCgroupV2Limits(it.GroupPath, it.Limits); err != nil {
			return fmt.Errorf("batch item %q: %w", it.GroupPath, err)
		}
	}
	return nil
}
