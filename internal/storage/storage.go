package storage

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// ObjectInfo describes a stored object.
type ObjectInfo struct {
	Key         string    `json:"key"`
	Size        int64     `json:"size"`
	ContentType string    `json:"content_type,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// ListOptions controls object listing behaviour.
type ListOptions struct {
	Prefix string
	Limit  int
	Offset int
}

// StorageService defines a minimal object-storage interface.
type StorageService interface {
	Put(ctx context.Context, key string, data []byte, contentType string) error
	Get(ctx context.Context, key string) ([]byte, error)
	Delete(ctx context.Context, key string) error
	List(ctx context.Context, opts ListOptions) ([]ObjectInfo, error)
	Stat(ctx context.Context, key string) (*ObjectInfo, error)
}

// InMemoryStorage is an in-memory StorageService for MVP / testing.
type InMemoryStorage struct {
	mu      sync.RWMutex
	objects map[string]*storedObject
}

type storedObject struct {
	info ObjectInfo
	data []byte
}

func NewInMemoryStorage() *InMemoryStorage {
	return &InMemoryStorage{
		objects: make(map[string]*storedObject),
	}
}

func (s *InMemoryStorage) Put(_ context.Context, key string, data []byte, contentType string) error {
	if key == "" {
		return fmt.Errorf("key is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	cp := make([]byte, len(data))
	copy(cp, data)

	s.objects[key] = &storedObject{
		info: ObjectInfo{
			Key:         key,
			Size:        int64(len(data)),
			ContentType: contentType,
			CreatedAt:   time.Now().UTC(),
		},
		data: cp,
	}
	return nil
}

func (s *InMemoryStorage) Get(_ context.Context, key string) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	obj, ok := s.objects[key]
	if !ok {
		return nil, fmt.Errorf("object %q not found", key)
	}
	cp := make([]byte, len(obj.data))
	copy(cp, obj.data)
	return cp, nil
}

func (s *InMemoryStorage) Delete(_ context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.objects[key]; !ok {
		return fmt.Errorf("object %q not found", key)
	}
	delete(s.objects, key)
	return nil
}

func (s *InMemoryStorage) List(_ context.Context, opts ListOptions) ([]ObjectInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var items []ObjectInfo
	for _, obj := range s.objects {
		if opts.Prefix != "" && !strings.HasPrefix(obj.info.Key, opts.Prefix) {
			continue
		}
		items = append(items, obj.info)
	}

	sort.Slice(items, func(i, j int) bool { return items[i].Key < items[j].Key })

	if opts.Offset > 0 && opts.Offset < len(items) {
		items = items[opts.Offset:]
	} else if opts.Offset >= len(items) {
		return nil, nil
	}
	if opts.Limit > 0 && opts.Limit < len(items) {
		items = items[:opts.Limit]
	}
	return items, nil
}

func (s *InMemoryStorage) Stat(_ context.Context, key string) (*ObjectInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	obj, ok := s.objects[key]
	if !ok {
		return nil, fmt.Errorf("object %q not found", key)
	}
	cp := obj.info
	return &cp, nil
}

var _ StorageService = (*InMemoryStorage)(nil)
