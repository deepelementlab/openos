package lifecycle

import (
	"context"
	"fmt"
	"time"

	"github.com/agentos/aos/pkg/runtime/types"
	"go.uber.org/zap"
)

// HookPhase identifies the lifecycle phase.
type HookPhase string

const (
	PhasePreCreate  HookPhase = "pre_create"
	PhasePostCreate HookPhase = "post_create"
	PhasePreStart   HookPhase = "pre_start"
	PhasePostStart  HookPhase = "post_start"
	PhasePreStop    HookPhase = "pre_stop"
	PhasePostStop   HookPhase = "post_stop"
	PhasePreDelete  HookPhase = "pre_delete"
	PhasePostDelete HookPhase = "post_delete"
)

// HookResult holds the outcome of executing a single hook.
type HookResult struct {
	Phase    HookPhase     `json:"phase"`
	Success  bool          `json:"success"`
	Error    string        `json:"error,omitempty"`
	Duration time.Duration `json:"duration"`
}

// HookExecutor executes lifecycle hooks.
type HookExecutor interface {
	Execute(ctx context.Context, phase HookPhase, hook *types.LifecycleHook) (*HookResult, error)
	ExecuteAll(ctx context.Context, hooks *types.LifecycleHooks, phase HookPhase) (*HookResult, error)
}

// DefaultHookExecutor logs and simulates hook execution (MVP).
type DefaultHookExecutor struct {
	logger *zap.Logger
}

func NewDefaultHookExecutor(logger *zap.Logger) *DefaultHookExecutor {
	return &DefaultHookExecutor{logger: logger}
}

func (e *DefaultHookExecutor) Execute(ctx context.Context, phase HookPhase, hook *types.LifecycleHook) (*HookResult, error) {
	if hook == nil {
		return &HookResult{Phase: phase, Success: true}, nil
	}

	start := time.Now()
	e.logger.Info("executing lifecycle hook",
		zap.String("phase", string(phase)),
		zap.Strings("command", hook.Command),
	)

	timeout := hook.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// MVP: simulate command execution
	select {
	case <-execCtx.Done():
		result := &HookResult{
			Phase:    phase,
			Success:  false,
			Error:    "hook execution timed out",
			Duration: time.Since(start),
		}
		if hook.IgnoreFailure {
			result.Success = true
			return result, nil
		}
		return result, fmt.Errorf("hook %s timed out", phase)
	default:
		result := &HookResult{
			Phase:    phase,
			Success:  true,
			Duration: time.Since(start),
		}
		e.logger.Info("lifecycle hook completed",
			zap.String("phase", string(phase)),
			zap.Duration("duration", result.Duration),
		)
		return result, nil
	}
}

func (e *DefaultHookExecutor) ExecuteAll(ctx context.Context, hooks *types.LifecycleHooks, phase HookPhase) (*HookResult, error) {
	if hooks == nil {
		return &HookResult{Phase: phase, Success: true}, nil
	}

	hook := e.resolveHook(hooks, phase)
	if hook == nil {
		return &HookResult{Phase: phase, Success: true}, nil
	}
	return e.Execute(ctx, phase, hook)
}

func (e *DefaultHookExecutor) resolveHook(hooks *types.LifecycleHooks, phase HookPhase) *types.LifecycleHook {
	switch phase {
	case PhasePreCreate:
		return hooks.PreCreate
	case PhasePostCreate:
		return hooks.PostCreate
	case PhasePreStart:
		return hooks.PreStart
	case PhasePostStart:
		return hooks.PostStart
	case PhasePreStop:
		return hooks.PreStop
	case PhasePostStop:
		return hooks.PostStop
	case PhasePreDelete:
		return hooks.PreDelete
	case PhasePostDelete:
		return hooks.PostDelete
	default:
		return nil
	}
}

var _ HookExecutor = (*DefaultHookExecutor)(nil)
