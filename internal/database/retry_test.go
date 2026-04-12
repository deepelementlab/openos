package database

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestDefaultRetryConfig(t *testing.T) {
	cfg := DefaultRetryConfig()
	assert.Equal(t, 3, cfg.MaxRetries)
	assert.Equal(t, 100*time.Millisecond, cfg.InitialBackoff)
	assert.Equal(t, 5*time.Second, cfg.MaxBackoff)
	assert.Equal(t, 2.0, cfg.BackoffMultiple)
	assert.NotNil(t, cfg.RetryableCheck)
}

func TestRetryConfig_Validate(t *testing.T) {
	cfg := DefaultRetryConfig()
	assert.NoError(t, cfg.Validate())

	invalidConfigs := []RetryConfig{
		{MaxRetries: -1, InitialBackoff: time.Millisecond, MaxBackoff: time.Second, BackoffMultiple: 2.0},
		{MaxRetries: 3, InitialBackoff: 0, MaxBackoff: time.Second, BackoffMultiple: 2.0},
		{MaxRetries: 3, InitialBackoff: time.Second, MaxBackoff: time.Millisecond, BackoffMultiple: 2.0},
		{MaxRetries: 3, InitialBackoff: time.Millisecond, MaxBackoff: time.Second, BackoffMultiple: 0.5},
	}
	for _, ic := range invalidConfigs {
		assert.Error(t, ic.Validate())
	}
}

func TestRetryWithBackoff_SuccessFirstTry(t *testing.T) {
	cfg := &RetryConfig{MaxRetries: 3, InitialBackoff: time.Millisecond, MaxBackoff: time.Second, BackoffMultiple: 2.0}
	calls := 0
	err := RetryWithBackoff(context.Background(), cfg, zap.NewNop(), func() error {
		calls++
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t, 1, calls)
}

func TestRetryWithBackoff_SuccessAfterRetries(t *testing.T) {
	cfg := &RetryConfig{MaxRetries: 3, InitialBackoff: time.Millisecond, MaxBackoff: time.Second, BackoffMultiple: 2.0}
	calls := 0
	err := RetryWithBackoff(context.Background(), cfg, zap.NewNop(), func() error {
		calls++
		if calls < 3 {
			return errors.New("transient error")
		}
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t, 3, calls)
}

func TestRetryWithBackoff_ExhaustedRetries(t *testing.T) {
	cfg := &RetryConfig{MaxRetries: 2, InitialBackoff: time.Millisecond, MaxBackoff: time.Second, BackoffMultiple: 2.0}
	err := RetryWithBackoff(context.Background(), cfg, zap.NewNop(), func() error {
		return errors.New("permanent error")
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed after 2 retries")
}

func TestRetryWithBackoff_NonRetryableError(t *testing.T) {
	cfg := &RetryConfig{
		MaxRetries:      3,
		InitialBackoff:  time.Millisecond,
		MaxBackoff:      time.Second,
		BackoffMultiple: 2.0,
		RetryableCheck:  func(err error) bool { return false },
	}
	calls := 0
	err := RetryWithBackoff(context.Background(), cfg, zap.NewNop(), func() error {
		calls++
		return errors.New("non-retryable")
	})
	require.Error(t, err)
	assert.Equal(t, 1, calls)
}

func TestRetryWithBackoff_CancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	cfg := DefaultRetryConfig()
	err := RetryWithBackoff(ctx, cfg, zap.NewNop(), func() error {
		return errors.New("fail")
	})
	require.Error(t, err)
}

func TestRetryWithBackoff_NilConfig(t *testing.T) {
	calls := 0
	err := RetryWithBackoff(context.Background(), nil, zap.NewNop(), func() error {
		calls++
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t, 1, calls)
}

func TestIsRetryableError(t *testing.T) {
	assert.False(t, IsRetryableError(nil))
	assert.True(t, IsRetryableError(errors.New("some error")))
}
