// Package deployment provides pull → parse → prepare pipeline for agent packages.
package deployment

import (
	"context"
	"io"
	"os"

	"github.com/agentos/aos/pkg/packaging"
)

// Puller fetches package manifests and layers from a registry or local path.
type Puller struct{}

// NewPuller creates a Puller.
func NewPuller() *Puller {
	return &Puller{}
}

// PullFromFile reads a manifest from a local JSON file.
func (p *Puller) PullFromFile(ctx context.Context, path string) (*packaging.Manifest, error) {
	_ = ctx
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return packaging.ParseManifest(f)
}

// PullFromReader parses manifest from an arbitrary reader (tests, HTTP body).
func (p *Puller) PullFromReader(ctx context.Context, r io.Reader) (*packaging.Manifest, error) {
	_ = ctx
	return packaging.ParseManifest(r)
}
