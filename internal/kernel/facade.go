// Package kernel aggregates Agent Kernel subsystems for the control plane and builder hooks.
package kernel

import (
	"github.com/agentos/aos/internal/kernel/ipc"
	"github.com/agentos/aos/internal/kernel/memory"
	"github.com/agentos/aos/internal/kernel/process"
	"github.com/agentos/aos/internal/kernel/vfs"
)

// Facade wires process, memory, vfs, and ipc managers (reserved for Builder Layer integration).
type Facade struct {
	Process process.Manager
	Memory  memory.Manager
	VFS     vfs.Manager
	IPC     ipc.Manager
}

// NewDefaultFacade returns in-memory implementations suitable for dev/tests.
func NewDefaultFacade() *Facade {
	return &Facade{
		Process: process.NewInMemoryManager(),
		Memory:  memory.NewInMemoryManager(),
		VFS:     vfs.NewInMemoryManager(),
		IPC:     ipc.NewInMemoryManager(),
	}
}
