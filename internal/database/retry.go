package database

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
)

type RetryConfig struct {
	MaxRetries      int           `json:"max_retries"`
	InitialBackoff  time.Duration `json:"initial_backoff"`
	MaxBackoff      time.Duration `json:"max_backoff"`
	BackoffMultiple float64       `json:"backoff_multiple"`
	RetryableCheck  func(error) bool
}

func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxRetries:      3,
		InitialBackoff:  100 * time.Millisecond,
		MaxBackoff:      5 * time.Second,
		BackoffMultiple: 2.0,
		RetryableCheck:  IsRetryableError,
	}
}

func (c *RetryConfig) Validate() error {
	if c.MaxRetries < 0 {
		return fmt.Errorf("max_retries must be non-negative")
	}
	if c.InitialBackoff <= 0 {
		return fmt.Errorf("initial_backoff must be positive")
	}
	if c.MaxBackoff < c.InitialBackoff {
		return fmt.Errorf("max_backoff must be >= initial_backoff")
	}
	if c.BackoffMultiple < 1.0 {
		return fmt.Errorf("backoff_multiple must be >= 1.0")
	}
	return nil
}

func RetryWithBackoff(ctx context.Context, cfg *RetryConfig, logger *zap.Logger, fn func() error) error {
	if cfg == nil {
		cfg = DefaultRetryConfig()
	}

	var lastErr error
	backoff := cfg.InitialBackoff

	for attempt := 0; attempt <= cfg.MaxRetries; attempt++ {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("context cancelled before attempt %d: %w", attempt, err)
		}

		lastErr = fn()
		if lastErr == nil {
			return nil
		}

		if cfg.RetryableCheck != nil && !cfg.RetryableCheck(lastErr) {
			return lastErr
		}

		if attempt < cfg.MaxRetries {
			logger.Warn("operation failed, retrying",
				zap.Int("attempt", attempt+1),
				zap.Int("max_retries", cfg.MaxRetries),
				zap.Duration("backoff", backoff),
				zap.Error(lastErr),
			)

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}

			backoff = time.Duration(float64(backoff) * cfg.BackoffMultiple)
			if backoff > cfg.MaxBackoff {
				backoff = cfg.MaxBackoff
			}
		}
	}

	return fmt.Errorf("operation failed after %d retries: %w", cfg.MaxRetries, lastErr)
}

func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}
	return true
}

func (db *Database) ConnectWithRetry(ctx context.Context, cfg *Config, logger *zap.Logger) error {
	retryCfg := DefaultRetryConfig()
	return RetryWithBackoff(ctx, retryCfg, logger, func() error {
		newDB, err := NewDatabase(cfg, logger)
		if err != nil {
			return err
		}
		db.DB = newDB.DB
		db.Logger = newDB.Logger
		return nil
	})
}
