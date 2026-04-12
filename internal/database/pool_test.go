package database

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestNewPoolManager(t *testing.T) {
	pm := NewPoolManager(nil, zap.NewNop())
	assert.NotNil(t, pm)
}

func TestPoolManager_Stats_NilDB(t *testing.T) {
	pm := NewPoolManager(&Database{DB: nil, Logger: zap.NewNop()}, zap.NewNop())
	_, err := pm.Stats()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database is nil")
}

func TestPoolManager_Reconfigure_NilDB(t *testing.T) {
	pm := NewPoolManager(&Database{DB: nil, Logger: zap.NewNop()}, zap.NewNop())
	err := pm.Reconfigure(DefaultConfig())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database is nil")
}

func TestPoolManager_Reconfigure_Closed(t *testing.T) {
	pm := NewPoolManager(nil, zap.NewNop())
	pm.Close()
	err := pm.Reconfigure(DefaultConfig())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "closed")
}

func TestPoolManager_Close(t *testing.T) {
	pm := NewPoolManager(nil, zap.NewNop())
	pm.Close()
	assert.True(t, pm.closed)
}

func TestPoolManager_StartMonitoring_Cancellation(t *testing.T) {
	pm := NewPoolManager(nil, zap.NewNop())
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	done := make(chan struct{})
	go func() {
		pm.StartMonitoring(ctx, 10*time.Millisecond)
		close(done)
	}()

	<-done
}

func TestPoolStats_Fields(t *testing.T) {
	stats := PoolStats{
		MaxOpenConnections: 25,
		OpenConnections:    10,
		InUse:              5,
		Idle:               5,
		WaitCount:          100,
		WaitDuration:       time.Second,
		MaxIdleClosed:      3,
		MaxLifetimeClosed:  7,
	}
	assert.Equal(t, 25, stats.MaxOpenConnections)
	assert.Equal(t, 10, stats.OpenConnections)
	assert.Equal(t, 5, stats.InUse)
	assert.Equal(t, 5, stats.Idle)
	assert.Equal(t, int64(100), stats.WaitCount)
	assert.Equal(t, time.Second, stats.WaitDuration)
	assert.Equal(t, int64(3), stats.MaxIdleClosed)
	assert.Equal(t, int64(7), stats.MaxLifetimeClosed)
}
