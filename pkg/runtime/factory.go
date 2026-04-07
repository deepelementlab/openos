// Package runtime provides runtime factory implementation
package runtime

import (
	"context"
	"fmt"

	"github.com/agentos/aos/pkg/runtime/containerd"
	"github.com/agentos/aos/pkg/runtime/gvisor"
	"github.com/agentos/aos/pkg/runtime/interfaces"
	"github.com/agentos/aos/pkg/runtime/kata"
	"github.com/agentos/aos/pkg/runtime/types"
)

// Factory implements RuntimeFactory interface
type Factory struct{}

// NewFactory creates a new runtime factory
func NewFactory() *Factory {
	return &Factory{}
}

// CreateRuntime creates a runtime instance for the specified type
func (f *Factory) CreateRuntime(ctx context.Context, runtimeType string, config *types.RuntimeConfig) (interfaces.Runtime, error) {
	switch runtimeType {
	case string(types.RuntimeContainerd):
		return containerd.NewRuntime(ctx, config)
	case string(types.RuntimeGVisor):
		return gvisor.NewRuntime(ctx, config)
	case string(types.RuntimeKata):
		return kata.NewRuntime(ctx, config)
	default:
		return nil, fmt.Errorf("unsupported runtime type: %s", runtimeType)
	}
}

// SupportedRuntimes returns list of supported runtime types
func (f *Factory) SupportedRuntimes() []string {
	return []string{
		string(types.RuntimeContainerd),
		string(types.RuntimeGVisor),
		string(types.RuntimeKata),
	}
}