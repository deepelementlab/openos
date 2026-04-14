package runtime

import (
	"context"
	"testing"

	"github.com/agentos/aos/pkg/runtime/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestNewFactory(t *testing.T) {
	f := NewFactory()
	assert.NotNil(t, f)
}

func TestFactory_SupportedRuntimes(t *testing.T) {
	f := NewFactory()
	runtimes := f.SupportedRuntimes()
	assert.Contains(t, runtimes, "containerd")
	assert.Contains(t, runtimes, "gvisor")
	assert.Contains(t, runtimes, "kata")
}

func TestFactory_CreateRuntime_GVisor(t *testing.T) {
	f := NewFactory()
	config := &types.RuntimeConfig{Type: types.RuntimeGVisor}
	rt, err := f.CreateRuntime(context.Background(), "gvisor", config)
	require.NoError(t, err)
	require.NotNil(t, rt)

	info := rt.GetRuntimeInfo()
	assert.Equal(t, types.RuntimeGVisor, info.Type)
}

func TestFactory_CreateRuntime_Kata(t *testing.T) {
	f := NewFactory()
	rt, err := f.CreateRuntime(context.Background(), "kata", &types.RuntimeConfig{Type: types.RuntimeKata})
	require.NoError(t, err)
	require.NotNil(t, rt)
	assert.Equal(t, types.RuntimeKata, rt.GetRuntimeInfo().Type)
}

func TestFactory_CreateRuntime_Unsupported(t *testing.T) {
	f := NewFactory()
	_, err := f.CreateRuntime(context.Background(), "unsupported", nil)
	assert.EqualError(t, err, "unsupported runtime type: unsupported")
}

func TestNewManager(t *testing.T) {
	m, err := NewManager(zap.NewNop(), NewFactory(), &types.RuntimeConfig{Type: types.RuntimeGVisor})
	require.NoError(t, err)
	assert.NotNil(t, m)
}

func TestNewManager_NilConfig(t *testing.T) {
	_, err := NewManager(zap.NewNop(), NewFactory(), nil)
	assert.EqualError(t, err, "config cannot be nil")
}

func TestNewManager_NilFactory(t *testing.T) {
	_, err := NewManager(zap.NewNop(), nil, &types.RuntimeConfig{Type: types.RuntimeGVisor})
	assert.EqualError(t, err, "factory cannot be nil")
}

func TestManager_Initialize(t *testing.T) {
	m, err := NewManager(zap.NewNop(), NewFactory(), &types.RuntimeConfig{Type: types.RuntimeGVisor})
	require.NoError(t, err)

	err = m.Initialize(context.Background())
	require.NoError(t, err)

	rt := m.DefaultRuntime()
	assert.NotNil(t, rt)
}

func TestManager_GetRuntime(t *testing.T) {
	m, err := NewManager(zap.NewNop(), NewFactory(), &types.RuntimeConfig{Type: types.RuntimeGVisor})
	require.NoError(t, err)
	require.NoError(t, m.Initialize(context.Background()))

	rt, err := m.GetRuntime("default")
	require.NoError(t, err)
	assert.NotNil(t, rt)

	_, err = m.GetRuntime("nonexistent")
	assert.Error(t, err)
}

func TestManager_CreateRuntime(t *testing.T) {
	m, err := NewManager(zap.NewNop(), NewFactory(), &types.RuntimeConfig{Type: types.RuntimeGVisor})
	require.NoError(t, err)
	require.NoError(t, m.Initialize(context.Background()))

	rt, err := m.CreateRuntime(context.Background(), "custom", "gvisor", &types.RuntimeConfig{Type: types.RuntimeGVisor})
	require.NoError(t, err)
	assert.NotNil(t, rt)

	runtimes := m.ListRuntimes()
	assert.Contains(t, runtimes, "default")
	assert.Contains(t, runtimes, "custom")
}

func TestManager_CreateRuntime_Duplicate(t *testing.T) {
	m, err := NewManager(zap.NewNop(), NewFactory(), &types.RuntimeConfig{Type: types.RuntimeGVisor})
	require.NoError(t, err)
	require.NoError(t, m.Initialize(context.Background()))

	_, err = m.CreateRuntime(context.Background(), "default", "gvisor", &types.RuntimeConfig{Type: types.RuntimeGVisor})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestManager_RemoveRuntime(t *testing.T) {
	m, err := NewManager(zap.NewNop(), NewFactory(), &types.RuntimeConfig{Type: types.RuntimeGVisor})
	require.NoError(t, err)
	require.NoError(t, m.Initialize(context.Background()))

	_, err = m.CreateRuntime(context.Background(), "temp", "gvisor", &types.RuntimeConfig{Type: types.RuntimeGVisor})
	require.NoError(t, err)

	err = m.RemoveRuntime(context.Background(), "temp")
	require.NoError(t, err)

	_, err = m.GetRuntime("temp")
	assert.Error(t, err)
}

func TestManager_RemoveRuntime_Default(t *testing.T) {
	m, err := NewManager(zap.NewNop(), NewFactory(), &types.RuntimeConfig{Type: types.RuntimeGVisor})
	require.NoError(t, err)
	require.NoError(t, m.Initialize(context.Background()))

	err = m.RemoveRuntime(context.Background(), "default")
	assert.EqualError(t, err, "cannot remove default runtime")
}

func TestManager_RemoveRuntime_NotFound(t *testing.T) {
	m, err := NewManager(zap.NewNop(), NewFactory(), &types.RuntimeConfig{Type: types.RuntimeGVisor})
	require.NoError(t, err)
	require.NoError(t, m.Initialize(context.Background()))

	err = m.RemoveRuntime(context.Background(), "nonexistent")
	assert.Error(t, err)
}

func TestManager_HealthCheck(t *testing.T) {
	m, err := NewManager(zap.NewNop(), NewFactory(), &types.RuntimeConfig{Type: types.RuntimeGVisor})
	require.NoError(t, err)
	require.NoError(t, m.Initialize(context.Background()))

	results := m.HealthCheck(context.Background())
	assert.Empty(t, results, "gVisor runtime should pass health check")
}

func TestManager_Shutdown(t *testing.T) {
	m, err := NewManager(zap.NewNop(), NewFactory(), &types.RuntimeConfig{Type: types.RuntimeGVisor})
	require.NoError(t, err)
	require.NoError(t, m.Initialize(context.Background()))

	err = m.Shutdown(context.Background())
	require.NoError(t, err)

	assert.Nil(t, m.DefaultRuntime())
}

func TestManager_SupportedRuntimes(t *testing.T) {
	m, err := NewManager(zap.NewNop(), NewFactory(), &types.RuntimeConfig{Type: types.RuntimeGVisor})
	require.NoError(t, err)

	runtimes := m.SupportedRuntimes()
	assert.Contains(t, runtimes, "gvisor")
}

func TestManager_ListRuntimes(t *testing.T) {
	m, err := NewManager(zap.NewNop(), NewFactory(), &types.RuntimeConfig{Type: types.RuntimeGVisor})
	require.NoError(t, err)
	require.NoError(t, m.Initialize(context.Background()))

	runtimes := m.ListRuntimes()
	assert.Contains(t, runtimes, "default")
}
