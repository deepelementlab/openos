// Package vfs provides Agent Kernel virtual filesystem abstractions.
package vfs

// FileSystemType identifies backend implementation.
type FileSystemType string

const (
	FSTypeLocal  FileSystemType = "local"
	FSTypeS3     FileSystemType = "s3"
	FSTypeNFS    FileSystemType = "nfs"
	FSTypeMemory FileSystemType = "memory"
)

// MountFlags are coarse mount options.
type MountFlags uint8

const (
	MountReadOnly MountFlags = 1 << iota
)

// MountSpec describes a mount point.
type MountSpec struct {
	Source   string
	Target   string
	FSType   FileSystemType
	Flags    MountFlags
	Options  map[string]string
}

// MountPoint is a registered mount.
type MountPoint struct {
	ID     string
	Spec   MountSpec
	Active bool
}

// OpenFlags for Open().
type OpenFlags uint8

const (
	OpenRead OpenFlags = 1 << iota
	OpenWrite
	OpenCreate
)

// FileHandle is an opaque file descriptor.
type FileHandle string

// Snapshot is metadata for a path snapshot (integration with storage later).
type Snapshot struct {
	ID   string
	Path string
	Data []byte // stub inline payload
}

// DiffEntry describes a file change between snapshots.
type DiffEntry struct {
	Path   string
	Change string // add/modify/delete
}
