package vfs

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
)

// Manager is the VFS facade (mount, open, snapshot).
type Manager interface {
	Mount(ctx context.Context, spec MountSpec) (*MountPoint, error)
	Unmount(ctx context.Context, target string) error
	Open(ctx context.Context, path string, flags OpenFlags) (FileHandle, error)
	Close(ctx context.Context, fd FileHandle) error
	Read(ctx context.Context, fd FileHandle, buf []byte, offset int64) (int, error)
	Write(ctx context.Context, fd FileHandle, buf []byte, offset int64) (int, error)
	Snapshot(ctx context.Context, path string) (*Snapshot, error)
	RestoreSnapshot(ctx context.Context, snap *Snapshot, target string) error
	DiffSnapshots(ctx context.Context, snap1, snap2 *Snapshot) ([]DiffEntry, error)
}

// InMemoryManager implements Manager with an in-memory file table.
type InMemoryManager struct {
	mu     sync.RWMutex
	mounts map[string]*MountPoint // target -> mount
	files  map[FileHandle][]byte  // fd -> content
}

// NewInMemoryManager creates a VFS manager.
func NewInMemoryManager() *InMemoryManager {
	return &InMemoryManager{
		mounts: make(map[string]*MountPoint),
		files:  make(map[FileHandle][]byte),
	}
}

func (m *InMemoryManager) Mount(ctx context.Context, spec MountSpec) (*MountPoint, error) {
	if spec.Target == "" {
		return nil, fmt.Errorf("vfs: empty mount target")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.mounts[spec.Target]; exists {
		return nil, fmt.Errorf("vfs: target already mounted %s", spec.Target)
	}
	mp := &MountPoint{
		ID:     uuid.NewString(),
		Spec:   spec,
		Active: true,
	}
	m.mounts[spec.Target] = mp
	return mp, nil
}

func (m *InMemoryManager) Unmount(ctx context.Context, target string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.mounts, target)
	return nil
}

func (m *InMemoryManager) Open(ctx context.Context, path string, flags OpenFlags) (FileHandle, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	fd := FileHandle(uuid.NewString())
	m.files[fd] = []byte{}
	return fd, nil
}

func (m *InMemoryManager) Close(ctx context.Context, fd FileHandle) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.files, fd)
	return nil
}

func (m *InMemoryManager) Read(ctx context.Context, fd FileHandle, buf []byte, offset int64) (int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	data, ok := m.files[fd]
	if !ok {
		return 0, fmt.Errorf("vfs: bad fd")
	}
	if int(offset) >= len(data) {
		return 0, nil
	}
	n := copy(buf, data[offset:])
	return n, nil
}

func (m *InMemoryManager) Write(ctx context.Context, fd FileHandle, buf []byte, offset int64) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	data, ok := m.files[fd]
	if !ok {
		return 0, fmt.Errorf("vfs: bad fd")
	}
	if int(offset) > len(data) {
		pad := int(offset) - len(data)
		data = append(data, make([]byte, pad)...)
	}
	end := int(offset) + len(buf)
	if end > len(data) {
		nd := make([]byte, end)
		copy(nd, data)
		data = nd
	}
	copy(data[int(offset):], buf)
	m.files[fd] = data
	return len(buf), nil
}

func (m *InMemoryManager) Snapshot(ctx context.Context, path string) (*Snapshot, error) {
	return &Snapshot{ID: uuid.NewString(), Path: path, Data: []byte(path)}, nil
}

func (m *InMemoryManager) RestoreSnapshot(ctx context.Context, snap *Snapshot, target string) error {
	if snap == nil {
		return fmt.Errorf("vfs: nil snapshot")
	}
	return nil
}

func (m *InMemoryManager) DiffSnapshots(ctx context.Context, snap1, snap2 *Snapshot) ([]DiffEntry, error) {
	if snap1 == nil || snap2 == nil {
		return nil, fmt.Errorf("vfs: nil snapshot")
	}
	if string(snap1.Data) == string(snap2.Data) {
		return nil, nil
	}
	return []DiffEntry{{Path: snap1.Path, Change: "modify"}}, nil
}
