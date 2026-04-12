package benchmarks

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/agentos/aos/internal/data/repository"
)

func BenchmarkInMemoryRepo_Create(b *testing.B) {
	repo := repository.NewInMemoryAgentRepository()
	ctx := context.Background()
	now := time.Now().UTC()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = repo.Create(ctx, &repository.Agent{
			ID:        fmt.Sprintf("agent-%d", i),
			Name:      fmt.Sprintf("agent-%d", i),
			Image:     "alpine:latest",
			Status:    repository.AgentStatusCreating,
			CreatedAt: now,
			UpdatedAt: now,
		})
	}
}

func BenchmarkInMemoryRepo_Get(b *testing.B) {
	repo := repository.NewInMemoryAgentRepository()
	ctx := context.Background()
	now := time.Now().UTC()

	const count = 1000
	for i := 0; i < count; i++ {
		_ = repo.Create(ctx, &repository.Agent{
			ID:        fmt.Sprintf("agent-%d", i),
			Name:      fmt.Sprintf("agent-%d", i),
			Image:     "alpine:latest",
			Status:    repository.AgentStatusCreating,
			CreatedAt: now,
			UpdatedAt: now,
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = repo.Get(ctx, fmt.Sprintf("agent-%d", i%count))
	}
}

func BenchmarkInMemoryRepo_List_100Agents(b *testing.B) {
	repo := repository.NewInMemoryAgentRepository()
	ctx := context.Background()
	now := time.Now().UTC()

	for i := 0; i < 100; i++ {
		_ = repo.Create(ctx, &repository.Agent{
			ID:        fmt.Sprintf("agent-%d", i),
			Name:      fmt.Sprintf("agent-%d", i),
			Image:     "alpine:latest",
			Status:    repository.AgentStatusRunning,
			CreatedAt: now,
			UpdatedAt: now,
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = repo.List(ctx)
	}
}

func BenchmarkInMemoryRepo_Update(b *testing.B) {
	repo := repository.NewInMemoryAgentRepository()
	ctx := context.Background()
	now := time.Now().UTC()

	const count = 1000
	for i := 0; i < count; i++ {
		_ = repo.Create(ctx, &repository.Agent{
			ID:        fmt.Sprintf("agent-%d", i),
			Name:      fmt.Sprintf("agent-%d", i),
			Image:     "alpine:latest",
			Status:    repository.AgentStatusCreating,
			CreatedAt: now,
			UpdatedAt: now,
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = repo.Update(ctx, &repository.Agent{
			ID:        fmt.Sprintf("agent-%d", i%count),
			Name:      "updated-agent",
			Image:     "alpine:latest",
			Status:    repository.AgentStatusRunning,
			CreatedAt: now,
			UpdatedAt: time.Now().UTC(),
		})
	}
}
