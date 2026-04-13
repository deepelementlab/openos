package validation

import (
	"context"
	"math/rand"
	"time"
)

// ChaosScenario injects controlled faults during tests.
type ChaosScenario struct {
	NodeKillProbability float64
	PartitionDuration   time.Duration
}

// RunFaultLoop randomly triggers fn until ctx done (use only in test/staging).
func RunFaultLoop(ctx context.Context, s ChaosScenario, kill func(node string), nodes []string) {
	t := time.NewTicker(500 * time.Millisecond)
	defer t.Stop()
	r := rand.New(rand.NewSource(42))
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if len(nodes) == 0 || r.Float64() > s.NodeKillProbability {
				continue
			}
			kill(nodes[r.Intn(len(nodes))])
		}
	}
}
