// Package ipc provides Agent Kernel IPC primitives (pipe, MQ, shm, semaphore).
package ipc

import "context"

// Message is a generic IPC payload.
type Message struct {
	Topic   string
	Payload []byte
}

// MsgHandler receives messages.
type MsgHandler func(ctx context.Context, m Message) error

// Subscription can be cancelled.
type Subscription interface {
	Unsubscribe() error
}

// Pipe is a simple in-process byte stream (pipe analogue).
type Pipe interface {
	Reader() <-chan []byte
	Writer() chan<- []byte
	Close() error
}

// MessageQueue is a named queue abstraction (NATS-backed in production).
type MessageQueue interface {
	Publish(ctx context.Context, msg Message) error
	Subscribe(ctx context.Context, handler MsgHandler) (Subscription, error)
	Close() error
}

// SharedMemory is a named blob (distributed via object store in production).
type SharedMemory interface {
	Name() string
	Size() int64
	ReadAt(p []byte, off int64) (n int, err error)
	WriteAt(p []byte, off int64) (n int, err error)
}

// MemoryMapping ties an agent to a shm segment.
type MemoryMapping struct {
	AgentID string
	ShmName string
}

// Semaphore is a counting semaphore (stub: in-process).
type Semaphore interface {
	Acquire(ctx context.Context, blocking bool) error
	Release() error
	Name() string
}
