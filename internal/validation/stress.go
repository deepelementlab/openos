// Package validation contains production verification harnesses (load, chaos, DR drills).
package validation

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// StressConfig drives synthetic agent creation throughput tests.
type StressConfig struct {
	Concurrency int
	Iterations  int
}

// StressResult captures observed throughput.
type StressResult struct {
	Started     time.Time
	Completed   time.Time
	Iterations  int64
	Errors      int64
}

// SimulateAgentBurst models scheduling pressure (integrate with real API in CI).
func SimulateAgentBurst(ctx context.Context, cfg StressConfig, work func(ctx context.Context, id int) error) StressResult {
	start := time.Now()
	var iters, errs int64
	if cfg.Concurrency < 1 {
		cfg.Concurrency = 1
	}
	if cfg.Iterations < 1 {
		cfg.Iterations = 1
	}
	sem := make(chan struct{}, cfg.Concurrency)
	var wg sync.WaitGroup
	for i := 0; i < cfg.Iterations; i++ {
		select {
		case <-ctx.Done():
			wg.Wait()
			return StressResult{Started: start, Completed: time.Now(), Iterations: atomic.LoadInt64(&iters), Errors: atomic.LoadInt64(&errs)}
		default:
		}
		wg.Add(1)
		sem <- struct{}{}
		go func(id int) {
			defer func() { <-sem; wg.Done() }()
			if err := work(ctx, id); err != nil {
				atomic.AddInt64(&errs, 1)
			}
			atomic.AddInt64(&iters, 1)
		}(i)
	}
	wg.Wait()
	return StressResult{Started: start, Completed: time.Now(), Iterations: atomic.LoadInt64(&iters), Errors: atomic.LoadInt64(&errs)}
}
