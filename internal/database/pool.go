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

// DynamicTuneConfig drives automatic max_open / max_idle adjustment from sql.DB stats.
type DynamicTuneConfig struct {
	MinOpen     int     // floor for MaxOpenConns
	MaxOpen     int     // ceiling for MaxOpenConns
	MinIdle     int
	MaxIdle     int     // usually <= MaxOpen
	TargetUtil  float64 // desired in_use/open ratio (e.g. 0.65)
	WaitPressure int64  // if WaitCount delta exceeds this per tick, scale up
}

// DefaultDynamicTuneConfig returns conservative production defaults.
func DefaultDynamicTuneConfig() DynamicTuneConfig {
	return DynamicTuneConfig{
		MinOpen:      10,
		MaxOpen:      200,
		MinIdle:      2,
		MaxIdle:      50,
		TargetUtil:   0.65,
		WaitPressure: 10,
	}
}

// TuneBasedOnLoad adjusts pool sizes using current stats and base config.
// Pass previous WaitCount to compute delta waits since last tick.
func (pm *PoolManager) TuneBasedOnLoad(cfg *Config, tune DynamicTuneConfig, prevWaitCount int64) (*Config, error) {
	if cfg == nil {
		return nil, fmt.Errorf("nil config")
	}
	stats, err := pm.Stats()
	if err != nil {
		return nil, err
	}
	next := *cfg
	open := stats.OpenConnections
	if open == 0 {
		open = 1
	}
	util := float64(stats.InUse) / float64(open)
	waitDelta := stats.WaitCount - prevWaitCount
	changed := false

	if waitDelta >= tune.WaitPressure || util > tune.TargetUtil+0.15 {
		next.MaxOpenConns = minInt(tune.MaxOpen, cfg.MaxOpenConns+maxInt(1, cfg.MaxOpenConns/10))
		next.MaxIdleConns = minInt(tune.MaxIdle, maxInt(tune.MinIdle, next.MaxOpenConns/4))
		changed = true
	} else if util < tune.TargetUtil-0.2 && cfg.MaxOpenConns > tune.MinOpen {
		next.MaxOpenConns = maxInt(tune.MinOpen, cfg.MaxOpenConns-maxInt(1, cfg.MaxOpenConns/20))
		next.MaxIdleConns = maxInt(tune.MinIdle, minInt(tune.MaxIdle, next.MaxOpenConns/4))
		changed = true
	}

	if next.MaxIdleConns > next.MaxOpenConns {
		next.MaxIdleConns = next.MaxOpenConns
	}
	if !changed {
		return &next, nil
	}
	if err := pm.Reconfigure(&next); err != nil {
		return nil, err
	}
	pm.logger.Info("pool dynamic tune",
		zap.Int("max_open", next.MaxOpenConns),
		zap.Int("max_idle", next.MaxIdleConns),
		zap.Float64("utilization", util),
		zap.Int64("wait_delta", waitDelta),
	)
	return &next, nil
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
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
