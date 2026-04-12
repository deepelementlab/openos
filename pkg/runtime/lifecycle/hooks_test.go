package lifecycle

import (
	"context"
	"testing"
	"time"

	"github.com/agentos/aos/pkg/runtime/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func newTestHookExecutor() *DefaultHookExecutor {
	return NewDefaultHookExecutor(zap.NewNop())
}

func TestHookPhase_Constants(t *testing.T) {
	assert.Equal(t, HookPhase("pre_create"), PhasePreCreate)
	assert.Equal(t, HookPhase("post_create"), PhasePostCreate)
	assert.Equal(t, HookPhase("pre_start"), PhasePreStart)
	assert.Equal(t, HookPhase("post_start"), PhasePostStart)
	assert.Equal(t, HookPhase("pre_stop"), PhasePreStop)
	assert.Equal(t, HookPhase("post_stop"), PhasePostStop)
	assert.Equal(t, HookPhase("pre_delete"), PhasePreDelete)
	assert.Equal(t, HookPhase("post_delete"), PhasePostDelete)
}

func TestDefaultHookExecutor_Execute_NilHook(t *testing.T) {
	e := newTestHookExecutor()
	result, err := e.Execute(context.Background(), PhasePreCreate, nil)
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, PhasePreCreate, result.Phase)
}

func TestDefaultHookExecutor_Execute_WithCommand(t *testing.T) {
	e := newTestHookExecutor()
	hook := &types.LifecycleHook{
		Command: []string{"/bin/echo", "hello"},
	}
	result, err := e.Execute(context.Background(), PhasePreStart, hook)
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, PhasePreStart, result.Phase)
	assert.True(t, result.Duration >= 0)
}

func TestDefaultHookExecutor_Execute_CustomTimeout(t *testing.T) {
	e := newTestHookExecutor()
	hook := &types.LifecycleHook{
		Command: []string{"/bin/true"},
		Timeout: 10 * time.Second,
	}
	result, err := e.Execute(context.Background(), PhasePostStart, hook)
	require.NoError(t, err)
	assert.True(t, result.Success)
}

func TestDefaultHookExecutor_ExecuteAll_NilHooks(t *testing.T) {
	e := newTestHookExecutor()
	result, err := e.ExecuteAll(context.Background(), nil, PhasePreCreate)
	require.NoError(t, err)
	assert.True(t, result.Success)
}

func TestDefaultHookExecutor_ExecuteAll_WithHooks(t *testing.T) {
	e := newTestHookExecutor()
	hooks := &types.LifecycleHooks{
		PreCreate: &types.LifecycleHook{
			Command: []string{"/bin/echo", "pre-create"},
		},
	}
	result, err := e.ExecuteAll(context.Background(), hooks, PhasePreCreate)
	require.NoError(t, err)
	assert.True(t, result.Success)
}

func TestDefaultHookExecutor_ExecuteAll_MissingPhase(t *testing.T) {
	e := newTestHookExecutor()
	hooks := &types.LifecycleHooks{}
	result, err := e.ExecuteAll(context.Background(), hooks, PhasePreStart)
	require.NoError(t, err)
	assert.True(t, result.Success)
}

func TestDefaultHookExecutor_ExecuteAll_AllPhases(t *testing.T) {
	e := newTestHookExecutor()
	hooks := &types.LifecycleHooks{
		PreCreate:  &types.LifecycleHook{Command: []string{"a"}},
		PostCreate: &types.LifecycleHook{Command: []string{"b"}},
		PreStart:   &types.LifecycleHook{Command: []string{"c"}},
		PostStart:  &types.LifecycleHook{Command: []string{"d"}},
		PreStop:    &types.LifecycleHook{Command: []string{"e"}},
		PostStop:   &types.LifecycleHook{Command: []string{"f"}},
		PreDelete:  &types.LifecycleHook{Command: []string{"g"}},
		PostDelete: &types.LifecycleHook{Command: []string{"h"}},
	}

	for _, phase := range []HookPhase{PhasePreCreate, PhasePostCreate, PhasePreStart, PhasePostStart, PhasePreStop, PhasePostStop, PhasePreDelete, PhasePostDelete} {
		result, err := e.ExecuteAll(context.Background(), hooks, phase)
		require.NoError(t, err)
		assert.True(t, result.Success, "phase %s should succeed", phase)
	}
}

func TestDefaultHookExecutor_ExecuteAll_UnknownPhase(t *testing.T) {
	e := newTestHookExecutor()
	hooks := &types.LifecycleHooks{
		PreCreate: &types.LifecycleHook{Command: []string{"a"}},
	}
	result, err := e.ExecuteAll(context.Background(), hooks, HookPhase("unknown"))
	require.NoError(t, err)
	assert.True(t, result.Success)
}

func TestHookResult_Fields(t *testing.T) {
	r := HookResult{
		Phase:    PhasePreStart,
		Success:  true,
		Duration: 100 * time.Millisecond,
	}
	assert.Equal(t, PhasePreStart, r.Phase)
	assert.True(t, r.Success)
	assert.Equal(t, 100*time.Millisecond, r.Duration)
}
