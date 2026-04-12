package tenant

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInMemoryTenantRepo_CreateAndGet(t *testing.T) {
	repo := NewInMemoryTenantRepository()
	ctx := context.Background()

	tn := &Tenant{
		ID:     "t-1",
		Name:   "Acme Corp",
		Status: TenantActive,
		Plan:   "enterprise",
		Labels: map[string]string{"env": "prod"},
	}

	err := repo.Create(ctx, tn)
	require.NoError(t, err)

	got, err := repo.Get(ctx, "t-1")
	require.NoError(t, err)
	assert.Equal(t, "t-1", got.ID)
	assert.Equal(t, "Acme Corp", got.Name)
	assert.Equal(t, TenantActive, got.Status)
	assert.Equal(t, "enterprise", got.Plan)
	assert.Equal(t, "prod", got.Labels["env"])
}

func TestInMemoryTenantRepo_CreateDuplicate(t *testing.T) {
	repo := NewInMemoryTenantRepository()
	ctx := context.Background()

	tn := &Tenant{ID: "t-1", Name: "Acme", Status: TenantActive}
	require.NoError(t, repo.Create(ctx, tn))

	err := repo.Create(ctx, &Tenant{ID: "t-1", Name: "Dup", Status: TenantActive})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestInMemoryTenantRepo_GetNotFound(t *testing.T) {
	repo := NewInMemoryTenantRepository()
	_, err := repo.Get(context.Background(), "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestInMemoryTenantRepo_Update(t *testing.T) {
	repo := NewInMemoryTenantRepository()
	ctx := context.Background()

	repo.Create(ctx, &Tenant{ID: "t-1", Name: "Original", Status: TenantActive, Plan: "free"})

	got, _ := repo.Get(ctx, "t-1")
	got.Name = "Updated"
	got.Status = TenantSuspended
	got.Plan = "enterprise"
	require.NoError(t, repo.Update(ctx, got))

	updated, _ := repo.Get(ctx, "t-1")
	assert.Equal(t, "Updated", updated.Name)
	assert.Equal(t, TenantSuspended, updated.Status)
	assert.Equal(t, "enterprise", updated.Plan)
}

func TestInMemoryTenantRepo_UpdateNotFound(t *testing.T) {
	repo := NewInMemoryTenantRepository()
	err := repo.Update(context.Background(), &Tenant{ID: "nonexistent", Name: "x", Status: TenantActive})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestInMemoryTenantRepo_Delete(t *testing.T) {
	repo := NewInMemoryTenantRepository()
	ctx := context.Background()

	repo.Create(ctx, &Tenant{ID: "t-1", Name: "Delete Me", Status: TenantActive})
	require.NoError(t, repo.Delete(ctx, "t-1"))

	_, err := repo.Get(ctx, "t-1")
	assert.Error(t, err)
}

func TestInMemoryTenantRepo_DeleteNotFound(t *testing.T) {
	repo := NewInMemoryTenantRepository()
	err := repo.Delete(context.Background(), "nonexistent")
	assert.Error(t, err)
}

func TestInMemoryTenantRepo_DeleteRemovesMembers(t *testing.T) {
	repo := NewInMemoryTenantRepository()
	ctx := context.Background()

	repo.Create(ctx, &Tenant{ID: "t-1", Name: "Acme", Status: TenantActive})
	repo.AddMember(ctx, &TenantMember{TenantID: "t-1", UserID: "u-1", Role: "admin"})
	repo.AddMember(ctx, &TenantMember{TenantID: "t-1", UserID: "u-2", Role: "member"})

	require.NoError(t, repo.Delete(ctx, "t-1"))

	_, err := repo.GetMembers(ctx, "t-1")
	assert.Error(t, err)
}

func TestInMemoryTenantRepo_List(t *testing.T) {
	repo := NewInMemoryTenantRepository()
	ctx := context.Background()

	agents, err := repo.List(ctx)
	require.NoError(t, err)
	assert.Empty(t, agents)

	repo.Create(ctx, &Tenant{ID: "t-1", Name: "First", Status: TenantActive, CreatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)})
	repo.Create(ctx, &Tenant{ID: "t-2", Name: "Second", Status: TenantActive, CreatedAt: time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC)})
	repo.Create(ctx, &Tenant{ID: "t-3", Name: "Third", Status: TenantActive, CreatedAt: time.Date(2025, 1, 3, 0, 0, 0, 0, time.UTC)})

	list, err := repo.List(ctx)
	require.NoError(t, err)
	assert.Len(t, list, 3)
	assert.Equal(t, "First", list[0].Name)
	assert.Equal(t, "Second", list[1].Name)
	assert.Equal(t, "Third", list[2].Name)
}

func TestInMemoryTenantRepo_ListSortedByCreationTime(t *testing.T) {
	repo := NewInMemoryTenantRepository()
	ctx := context.Background()

	repo.Create(ctx, &Tenant{ID: "t-3", Name: "Third", Status: TenantActive, CreatedAt: time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)})
	repo.Create(ctx, &Tenant{ID: "t-1", Name: "First", Status: TenantActive, CreatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)})
	repo.Create(ctx, &Tenant{ID: "t-2", Name: "Second", Status: TenantActive, CreatedAt: time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC)})

	list, _ := repo.List(ctx)
	assert.Equal(t, "First", list[0].Name)
	assert.Equal(t, "Second", list[1].Name)
	assert.Equal(t, "Third", list[2].Name)
}

func TestInMemoryTenantRepo_Isolation(t *testing.T) {
	repo := NewInMemoryTenantRepository()
	ctx := context.Background()

	original := &Tenant{
		ID:     "t-1",
		Name:   "Original",
		Status: TenantActive,
		Labels: map[string]string{"key": "value"},
	}
	repo.Create(ctx, original)

	got, _ := repo.Get(ctx, "t-1")
	got.Name = "Modified"
	got.Labels["key"] = "changed"

	stored, _ := repo.Get(ctx, "t-1")
	assert.Equal(t, "Original", stored.Name)
	assert.Equal(t, "value", stored.Labels["key"])
}

func TestInMemoryTenantRepo_NilLabelsHandling(t *testing.T) {
	repo := NewInMemoryTenantRepository()
	ctx := context.Background()

	tn := &Tenant{ID: "t-1", Name: "No Labels", Status: TenantActive}
	require.NoError(t, repo.Create(ctx, tn))

	got, _ := repo.Get(ctx, "t-1")
	assert.NotNil(t, got.Labels)
}

func TestInMemoryTenantRepo_AddMember(t *testing.T) {
	repo := NewInMemoryTenantRepository()
	ctx := context.Background()

	repo.Create(ctx, &Tenant{ID: "t-1", Name: "Acme", Status: TenantActive})

	err := repo.AddMember(ctx, &TenantMember{TenantID: "t-1", UserID: "u-1", Role: "admin"})
	require.NoError(t, err)

	members, err := repo.GetMembers(ctx, "t-1")
	require.NoError(t, err)
	assert.Len(t, members, 1)
	assert.Equal(t, "u-1", members[0].UserID)
	assert.Equal(t, "admin", members[0].Role)
}

func TestInMemoryTenantRepo_AddMember_Duplicate(t *testing.T) {
	repo := NewInMemoryTenantRepository()
	ctx := context.Background()

	repo.Create(ctx, &Tenant{ID: "t-1", Name: "Acme", Status: TenantActive})
	repo.AddMember(ctx, &TenantMember{TenantID: "t-1", UserID: "u-1", Role: "admin"})

	err := repo.AddMember(ctx, &TenantMember{TenantID: "t-1", UserID: "u-1", Role: "member"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestInMemoryTenantRepo_AddMember_TenantNotFound(t *testing.T) {
	repo := NewInMemoryTenantRepository()
	err := repo.AddMember(context.Background(), &TenantMember{TenantID: "nonexistent", UserID: "u-1", Role: "admin"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "tenant not found")
}

func TestInMemoryTenantRepo_RemoveMember(t *testing.T) {
	repo := NewInMemoryTenantRepository()
	ctx := context.Background()

	repo.Create(ctx, &Tenant{ID: "t-1", Name: "Acme", Status: TenantActive})
	repo.AddMember(ctx, &TenantMember{TenantID: "t-1", UserID: "u-1", Role: "admin"})
	repo.AddMember(ctx, &TenantMember{TenantID: "t-1", UserID: "u-2", Role: "member"})

	err := repo.RemoveMember(ctx, "t-1", "u-1")
	require.NoError(t, err)

	members, _ := repo.GetMembers(ctx, "t-1")
	assert.Len(t, members, 1)
	assert.Equal(t, "u-2", members[0].UserID)
}

func TestInMemoryTenantRepo_RemoveMember_NotFound(t *testing.T) {
	repo := NewInMemoryTenantRepository()
	ctx := context.Background()

	repo.Create(ctx, &Tenant{ID: "t-1", Name: "Acme", Status: TenantActive})

	err := repo.RemoveMember(ctx, "t-1", "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "member not found")
}

func TestInMemoryTenantRepo_RemoveMember_TenantNotFound(t *testing.T) {
	repo := NewInMemoryTenantRepository()
	err := repo.RemoveMember(context.Background(), "nonexistent", "u-1")
	assert.Error(t, err)
}

func TestInMemoryTenantRepo_GetMembers_Empty(t *testing.T) {
	repo := NewInMemoryTenantRepository()
	ctx := context.Background()

	repo.Create(ctx, &Tenant{ID: "t-1", Name: "Acme", Status: TenantActive})

	members, err := repo.GetMembers(ctx, "t-1")
	require.NoError(t, err)
	assert.Empty(t, members)
}

func TestInMemoryTenantRepo_GetMembers_TenantNotFound(t *testing.T) {
	repo := NewInMemoryTenantRepository()
	_, err := repo.GetMembers(context.Background(), "nonexistent")
	assert.Error(t, err)
}

func TestInMemoryTenantRepo_MembersIsolation(t *testing.T) {
	repo := NewInMemoryTenantRepository()
	ctx := context.Background()

	repo.Create(ctx, &Tenant{ID: "t-1", Name: "Acme", Status: TenantActive})
	repo.AddMember(ctx, &TenantMember{TenantID: "t-1", UserID: "u-1", Role: "admin"})

	members, _ := repo.GetMembers(ctx, "t-1")
	members[0].Role = "hacked"

	original, _ := repo.GetMembers(ctx, "t-1")
	assert.Equal(t, "admin", original[0].Role)
}

func TestInMemoryTenantRepo_ConcurrentAccess(t *testing.T) {
	repo := NewInMemoryTenantRepository()
	ctx := context.Background()

	var wg sync.WaitGroup
	errors := make(chan error, 100)

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			err := repo.Create(ctx, &Tenant{
				ID:        fmt.Sprintf("tenant-%d", idx),
				Name:      "Concurrent Tenant",
				Status:    TenantActive,
				CreatedAt: time.Now(),
			})
			if err != nil {
				errors <- err
			}
		}(i)
	}

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = repo.List(ctx)
		}()
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Errorf("concurrent access error: %v", err)
	}
}

func TestNewTenant_Helper(t *testing.T) {
	tn := NewTenant("t-new", "New Corp", "pro")
	assert.Equal(t, "t-new", tn.ID)
	assert.Equal(t, "New Corp", tn.Name)
	assert.Equal(t, TenantActive, tn.Status)
	assert.Equal(t, "pro", tn.Plan)
	assert.NotNil(t, tn.Labels)
}
