package ipc

import (
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/google/uuid"
)

// Manager aggregates IPC factory methods.
type Manager interface {
	CreatePipe(ctx context.Context, name string) (Pipe, error)
	CreateMessageQueue(ctx context.Context, name string, cfg MQConfig) (MessageQueue, error)
	CreateSharedMemory(ctx context.Context, name string, size int64) (SharedMemory, error)
	AttachShm(ctx context.Context, agentID, shmName string) (MemoryMapping, error)
	DetachShm(ctx context.Context, agentID, shmName string) error
	CreateSemaphore(ctx context.Context, name string, initValue int) (Semaphore, error)
	Acquire(ctx context.Context, semName string, blocking bool) error
	Release(ctx context.Context, semName string) error
}

// MQConfig configures a queue (reserved for NATS / JetStream).
type MQConfig struct {
	Durable bool
}

// InMemoryManager implements Manager for single-process / tests.
type InMemoryManager struct {
	mu    sync.RWMutex
	pipes map[string]*memPipe
	mqs   map[string]*memMQ
	shm   map[string]*memShm
	sems  map[string]*memSem
}

// NewInMemoryManager creates IPC manager.
func NewInMemoryManager() *InMemoryManager {
	return &InMemoryManager{
		pipes: make(map[string]*memPipe),
		mqs:   make(map[string]*memMQ),
		shm:   make(map[string]*memShm),
		sems:  make(map[string]*memSem),
	}
}

type memPipe struct {
	ch chan []byte
}

func (p *memPipe) Reader() <-chan []byte { return p.ch }
func (p *memPipe) Writer() chan<- []byte { return p.ch }
func (p *memPipe) Close() error {
	close(p.ch)
	return nil
}

func (m *InMemoryManager) CreatePipe(ctx context.Context, name string) (Pipe, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := name
	if key == "" {
		key = uuid.NewString()
	}
	p := &memPipe{ch: make(chan []byte, 64)}
	m.pipes[key] = p
	return p, nil
}

type memMQ struct {
	name string
	subs map[int64]MsgHandler
	next int64
	mu   sync.Mutex
}

func (q *memMQ) Publish(ctx context.Context, msg Message) error {
	q.mu.Lock()
	handlers := make([]MsgHandler, 0, len(q.subs))
	for _, h := range q.subs {
		handlers = append(handlers, h)
	}
	q.mu.Unlock()
	for _, h := range handlers {
		if err := h(ctx, msg); err != nil {
			return err
		}
	}
	return nil
}

func (q *memMQ) Subscribe(ctx context.Context, handler MsgHandler) (Subscription, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.subs == nil {
		q.subs = make(map[int64]MsgHandler)
	}
	q.next++
	id := q.next
	q.subs[id] = handler
	return &sub{q: q, id: id}, nil
}

func (q *memMQ) Close() error { return nil }

type sub struct {
	q  *memMQ
	id int64
}

func (s *sub) Unsubscribe() error {
	s.q.mu.Lock()
	defer s.q.mu.Unlock()
	delete(s.q.subs, s.id)
	return nil
}

func (m *InMemoryManager) CreateMessageQueue(ctx context.Context, name string, cfg MQConfig) (MessageQueue, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	q := &memMQ{name: name, subs: make(map[int64]MsgHandler)}
	m.mqs[name] = q
	return q, nil
}

type memShm struct {
	name string
	data []byte
}

func (s *memShm) Name() string { return s.name }
func (s *memShm) Size() int64 { return int64(len(s.data)) }

func (s *memShm) ReadAt(p []byte, off int64) (int, error) {
	if off < 0 || int(off) > len(s.data) {
		return 0, io.EOF
	}
	n := copy(p, s.data[off:])
	if n < len(p) {
		return n, io.EOF
	}
	return n, nil
}

func (s *memShm) WriteAt(p []byte, off int64) (int, error) {
	end := int(off) + len(p)
	if end > len(s.data) {
		nd := make([]byte, end)
		copy(nd, s.data)
		s.data = nd
	}
	copy(s.data[int(off):], p)
	return len(p), nil
}

func (m *InMemoryManager) CreateSharedMemory(ctx context.Context, name string, size int64) (SharedMemory, error) {
	if size <= 0 {
		return nil, fmt.Errorf("ipc: invalid shm size")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	s := &memShm{name: name, data: make([]byte, size)}
	m.shm[name] = s
	return s, nil
}

func (m *InMemoryManager) AttachShm(ctx context.Context, agentID, shmName string) (MemoryMapping, error) {
	m.mu.RLock()
	_, ok := m.shm[shmName]
	m.mu.RUnlock()
	if !ok {
		return MemoryMapping{}, fmt.Errorf("ipc: unknown shm %s", shmName)
	}
	return MemoryMapping{AgentID: agentID, ShmName: shmName}, nil
}

func (m *InMemoryManager) DetachShm(ctx context.Context, agentID, shmName string) error {
	return nil
}

type memSem struct {
	name  string
	count int
	mu    sync.Mutex
	cond  *sync.Cond
}

func newSem(name string, init int) *memSem {
	s := &memSem{name: name, count: init}
	s.cond = sync.NewCond(&s.mu)
	return s
}

func (s *memSem) Acquire(ctx context.Context, blocking bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if blocking {
		for s.count <= 0 {
			s.cond.Wait()
		}
	} else {
		if s.count <= 0 {
			return fmt.Errorf("ipc: would block")
		}
	}
	s.count--
	return nil
}

func (s *memSem) Release() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.count++
	s.cond.Signal()
	return nil
}

func (s *memSem) Name() string { return s.name }

func (m *InMemoryManager) CreateSemaphore(ctx context.Context, name string, initValue int) (Semaphore, error) {
	if initValue < 0 {
		return nil, fmt.Errorf("ipc: negative semaphore")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	s := newSem(name, initValue)
	m.sems[name] = s
	return s, nil
}

func (m *InMemoryManager) Acquire(ctx context.Context, semName string, blocking bool) error {
	m.mu.RLock()
	s, ok := m.sems[semName]
	m.mu.RUnlock()
	if !ok {
		return fmt.Errorf("ipc: unknown semaphore %s", semName)
	}
	return s.Acquire(ctx, blocking)
}

func (m *InMemoryManager) Release(ctx context.Context, semName string) error {
	m.mu.RLock()
	s, ok := m.sems[semName]
	m.mu.RUnlock()
	if !ok {
		return fmt.Errorf("ipc: unknown semaphore %s", semName)
	}
	return s.Release()
}
