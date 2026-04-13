package edge

// LiteRuntimeSpec configures reduced footprint agents on edge.
type LiteRuntimeSpec struct {
	MaxMemoryMB int
	DisableGPU  bool
	Sandbox     string // e.g. runsc-lite (conceptual)
}

// DefaultLite returns conservative defaults for far-edge devices.
func DefaultLite() LiteRuntimeSpec {
	return LiteRuntimeSpec{
		MaxMemoryMB: 512,
		DisableGPU:  true,
		Sandbox:     "runsc",
	}
}
