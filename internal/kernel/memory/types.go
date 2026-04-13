// Package memory provides Agent Kernel virtual memory and checkpoint abstractions.
package memory

// ProtectionFlags are analogous to mmap PROT_* (subset).
type ProtectionFlags uint8

const (
	ProtRead ProtectionFlags = 1 << iota
	ProtWrite
	ProtExec
)

// BackingType describes what backs a region.
type BackingType string

const (
	BackingVolume BackingType = "volume"
	BackingMemory BackingType = "memory"
	BackingFile   BackingType = "file"
)

// MemoryRegionSpec defines a mappable region.
type MemoryRegionSpec struct {
	Size        int64
	Protection  ProtectionFlags
	BackingType BackingType
	BackingRef  string // volume id, path, or opaque ref
}

// MemoryRegion is a mapped region for an agent.
type MemoryRegion struct {
	RegionID    string
	AgentID     string
	Start       uintptr // logical; stub uses 0
	Size        int64
	Protection  ProtectionFlags
	BackingType BackingType
	BackingRef  string
	Dirty       bool
}

// WorkingSet is a hint list of backing refs to prioritize.
type WorkingSet struct {
	Refs []string
}

// Checkpoint is an opaque snapshot handle (CRIU integration point later).
type Checkpoint struct {
	ID      string
	AgentID string
	// Payload stores JSON or path to checkpoint bundle
	Payload []byte
}
