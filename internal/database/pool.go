package database

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

type PoolManager struct {
	db     *Database
	logger *zap.Logger
	mu     sync.RWMutex
	closed bool
}

func NewPoolManager(db *Database, logger *zap.Logger) *PoolManager {
	return &PoolManager{db: db, logger: logger}
}

func (pm *PoolManager) Reconfigure(cfg *Config) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if pm.closed {
		return fmt.Errorf("pool manager is closed")
	}
	if pm.db == nil || pm.db.DB == nil {
		return fmt.Errorf("database is nil")
	}

	pm.db.DB.SetMaxOpenConns(cfg.MaxOpenConns)
	pm.db.DB.SetMaxIdleConns(cfg.MaxIdleConns)
	pm.db.DB.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	pm.logger.Info("connection pool reconfigured",
		zap.Int("max_open", cfg.MaxOpenConns),
		zap.Int("max_idle", cfg.MaxIdleConns),
	)
	return nil
}

func (pm *PoolManager) Stats() (PoolStats, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	if pm.db == nil || pm.db.DB == nil {
		return PoolStats{}, fmt.Errorf("database is nil")
	}

	s := pm.db.DB.Stats()
	return PoolStats{
		MaxOpenConnections: s.MaxOpenConnections,
		OpenConnections:    s.OpenConnections,
		InUse:              s.InUse,
		Idle:               s.Idle,
		WaitCount:          s.WaitCount,
		WaitDuration:       s.WaitDuration,
		MaxIdleClosed:      s.MaxIdleClosed,
		MaxLifetimeClosed:  s.MaxLifetimeClosed,
	}, nil
}

func (pm *PoolManager) StartMonitoring(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			stats, err := pm.Stats()
			if err != nil {
				pm.logger.Warn("failed to get pool stats", zap.Error(err))
				continue
			}
			pm.logger.Debug("pool stats",
				zap.Int("open", stats.OpenConnections),
				zap.Int("in_use", stats.InUse),
				zap.Int("idle", stats.Idle),
				zap.Int64("waits", stats.WaitCount),
			)
			if stats.OpenConnections > 0 && float64(stats.InUse)/float64(stats.OpenConnections) > 0.9 {
				pm.logger.Warn("connection pool under pressure",
					zap.Int("open", stats.OpenConnections),
					zap.Int("in_use", stats.InUse),
				)
			}
		}
	}
}

func (pm *PoolManager) Close() {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.closed = true
}

type PoolStats struct {
	MaxOpenConnections int           `json:"max_open_connections"`
	OpenConnections    int           `json:"open_connections"`
	InUse              int           `json:"in_use"`
	Idle               int           `json:"idle"`
	WaitCount          int64         `json:"wait_count"`
	WaitDuration       time.Duration `json:"wait_duration"`
	MaxIdleClosed      int64         `json:"max_idle_closed"`
	MaxLifetimeClosed  int64         `json:"max_lifetime_closed"`
}
