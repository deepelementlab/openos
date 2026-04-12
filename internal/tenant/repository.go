package tenant

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"
)

type TenantRepository interface {
	Create(ctx context.Context, t *Tenant) error
	Get(ctx context.Context, id string) (*Tenant, error)
	Update(ctx context.Context, t *Tenant) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context) ([]*Tenant, error)

	AddMember(ctx context.Context, m *TenantMember) error
	RemoveMember(ctx context.Context, tenantID, userID string) error
	GetMembers(ctx context.Context, tenantID string) ([]*TenantMember, error)
}

type InMemoryTenantRepository struct {
	mu      sync.RWMutex
	tenants map[string]*Tenant
	members map[string][]*TenantMember
}

func NewInMemoryTenantRepository() *InMemoryTenantRepository {
	return &InMemoryTenantRepository{
		tenants: make(map[string]*Tenant),
		members: make(map[string][]*TenantMember),
	}
}

func (r *InMemoryTenantRepository) Create(_ context.Context, t *Tenant) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.tenants[t.ID]; exists {
		return fmt.Errorf("tenant already exists")
	}
	cp := *t
	if cp.Labels == nil {
		cp.Labels = make(map[string]string)
	}
	r.tenants[t.ID] = &cp
	return nil
}

func (r *InMemoryTenantRepository) Get(_ context.Context, id string) (*Tenant, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.tenants[id]
	if !ok {
		return nil, fmt.Errorf("tenant not found")
	}
	cp := *t
	cp.Labels = make(map[string]string)
	for k, v := range t.Labels {
		cp.Labels[k] = v
	}
	return &cp, nil
}

func (r *InMemoryTenantRepository) Update(_ context.Context, t *Tenant) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.tenants[t.ID]; !ok {
		return fmt.Errorf("tenant not found")
	}
	cp := *t
	if cp.Labels == nil {
		cp.Labels = make(map[string]string)
	}
	r.tenants[t.ID] = &cp
	return nil
}

func (r *InMemoryTenantRepository) Delete(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.tenants[id]; !ok {
		return fmt.Errorf("tenant not found")
	}
	delete(r.tenants, id)
	delete(r.members, id)
	return nil
}

func (r *InMemoryTenantRepository) List(_ context.Context) ([]*Tenant, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]*Tenant, 0, len(r.tenants))
	for _, t := range r.tenants {
		cp := *t
		cp.Labels = make(map[string]string)
		for k, v := range t.Labels {
			cp.Labels[k] = v
		}
		items = append(items, &cp)
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].CreatedAt.Equal(items[j].CreatedAt) {
			return items[i].ID < items[j].ID
		}
		return items[i].CreatedAt.Before(items[j].CreatedAt)
	})
	return items, nil
}

func (r *InMemoryTenantRepository) AddMember(_ context.Context, m *TenantMember) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.tenants[m.TenantID]; !ok {
		return fmt.Errorf("tenant not found")
	}
	for _, existing := range r.members[m.TenantID] {
		if existing.UserID == m.UserID {
			return fmt.Errorf("member already exists")
		}
	}
	cp := *m
	r.members[m.TenantID] = append(r.members[m.TenantID], &cp)
	return nil
}

func (r *InMemoryTenantRepository) RemoveMember(_ context.Context, tenantID, userID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.tenants[tenantID]; !ok {
		return fmt.Errorf("tenant not found")
	}
	members := r.members[tenantID]
	found := false
	for i, m := range members {
		if m.UserID == userID {
			r.members[tenantID] = append(members[:i], members[i+1:]...)
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("member not found")
	}
	return nil
}

func (r *InMemoryTenantRepository) GetMembers(_ context.Context, tenantID string) ([]*TenantMember, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if _, ok := r.tenants[tenantID]; !ok {
		return nil, fmt.Errorf("tenant not found")
	}
	members := r.members[tenantID]
	result := make([]*TenantMember, 0, len(members))
	for _, m := range members {
		cp := *m
		result = append(result, &cp)
	}
	return result, nil
}

func NewTenant(id, name string, plan string) *Tenant {
	now := time.Now()
	return &Tenant{
		ID:        id,
		Name:      name,
		Status:    TenantActive,
		Plan:      plan,
		Labels:    make(map[string]string),
		CreatedAt: now,
		UpdatedAt: now,
	}
}
