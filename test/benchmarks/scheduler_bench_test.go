package benchmarks

import (
	"context"
	"fmt"
	"testing"

	"github.com/agentos/aos/internal/scheduler"
)

func addNodes(b *testing.B, s *scheduler.DefaultScheduler, n int) {
	b.Helper()
	for i := 0; i < n; i++ {
		_ = s.AddNode(context.Background(), scheduler.NodeState{
			NodeID:      fmt.Sprintf("node-%d", i),
			NodeName:    fmt.Sprintf("node-%d", i),
			CPUCores:    4,
			MemoryBytes: 8 * 1024 * 1024 * 1024,
			Health:      "healthy",
		})
	}
}

func BenchmarkScheduler_Schedule_100Nodes(b *testing.B) {
	s := scheduler.NewDefaultScheduler()
	addNodes(b, s, 100)

	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = s.Schedule(ctx, scheduler.TaskRequest{
			TaskID:   fmt.Sprintf("task-%d", i),
			TaskType: "agent",
		})
	}
}

func BenchmarkScheduler_Schedule_1000Nodes(b *testing.B) {
	s := scheduler.NewDefaultScheduler()
	addNodes(b, s, 1000)

	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = s.Schedule(ctx, scheduler.TaskRequest{
			TaskID:   fmt.Sprintf("task-%d", i),
			TaskType: "agent",
		})
	}
}

func BenchmarkDefaultScheduler_AddNode(b *testing.B) {
	s := scheduler.NewDefaultScheduler()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = s.AddNode(ctx, scheduler.NodeState{
			NodeID:      fmt.Sprintf("node-%d", i),
			NodeName:    fmt.Sprintf("node-%d", i),
			CPUCores:    4,
			MemoryBytes: 8 * 1024 * 1024 * 1024,
			Health:      "healthy",
		})
	}
}

func BenchmarkBestFit_Select(b *testing.B) {
	bf := scheduler.NewBestFitAlgorithm()
	ctx := context.Background()

	const numNodes = 100
	candidates := make([]scheduler.NodeState, numNodes)
	scores := make(map[string]int, numNodes)
	for i := range candidates {
		candidates[i] = scheduler.NodeState{
			NodeID:      fmt.Sprintf("node-%d", i),
			CPUCores:    8,
			MemoryBytes: 16 * 1024 * 1024 * 1024,
			Health:      "healthy",
		}
		scores[candidates[i].NodeID] = i
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = bf.SelectNode(ctx, candidates, scores)
	}
}
