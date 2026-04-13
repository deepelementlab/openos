//go:build linux

package enforcer

import (
	"fmt"

	"golang.org/x/sync/errgroup"
)

// CgroupBatchItem is one cgroup path with limits for batch application.
type CgroupBatchItem struct {
	GroupPath string
	Limits    CgroupLimits
}

// ApplyCgroupV2LimitsBatch applies multiple cgroup v2 limit sets.
// Uses bounded parallelism to reduce total latency vs strictly sequential writes.
func ApplyCgroupV2LimitsBatch(items []CgroupBatchItem, maxParallel int) error {
	if len(items) == 0 {
		return nil
	}
	if maxParallel <= 0 {
		maxParallel = 4
	}
	var g errgroup.Group
	g.SetLimit(maxParallel)
	for _, it := range items {
		it := it
		g.Go(func() error {
			if err := ApplyCgroupV2Limits(it.GroupPath, it.Limits); err != nil {
				return fmt.Errorf("%q: %w", it.GroupPath, err)
			}
			return nil
		})
	}
	return g.Wait()
}
