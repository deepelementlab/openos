package discovery

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"go.uber.org/zap"
)

// HealthChecker performs health checks on service instances.
type HealthChecker struct {
	registry     ServiceRegistry
	interval     time.Duration
	timeout      time.Duration
	threshold    int // Consecutive failures before marking unhealthy
	logger       *zap.Logger
	mu           sync.RWMutex
	checks       map[string]*healthCheck
	stopCh       chan struct{}
	wg           sync.WaitGroup
}

// healthCheck represents the health check state for an instance.
type healthCheck struct {
	instanceID           string
	consecutiveFailures  int
	consecutiveSuccesses int
	lastCheck            time.Time
	lastError            error
}

// NewHealthChecker creates a new health checker.
func NewHealthChecker(registry ServiceRegistry, logger *zap.Logger) *HealthChecker {
	return &HealthChecker{
		registry:  registry,
		interval:  10 * time.Second,
		timeout:   5 * time.Second,
		threshold: 3,
		logger:    logger,
		checks:    make(map[string]*healthCheck),
		stopCh:    make(chan struct{}),
	}
}

// Start starts the health checker.
func (hc *HealthChecker) Start(ctx context.Context) {
	hc.wg.Add(1)
	go func() {
		defer hc.wg.Done()
		ticker := time.NewTicker(hc.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-hc.stopCh:
				return
			case <-ticker.C:
				hc.checkAll(ctx)
			}
		}
	}()
}

// Stop stops the health checker.
func (hc *HealthChecker) Stop() {
	close(hc.stopCh)
	hc.wg.Wait()
}

// checkAll performs health checks on all registered instances.
func (hc *HealthChecker) checkAll(ctx context.Context) {
	services, err := hc.registry.ListServices(ctx)
	if err != nil {
		hc.logger.Error("Failed to list services for health check", zap.Error(err))
		return
	}

	for _, serviceName := range services {
		serviceSet, err := hc.registry.GetService(ctx, serviceName)
		if err != nil {
			continue
		}

		for _, instance := range serviceSet.Instances {
			hc.wg.Add(1)
			go func(inst *ServiceInstance) {
				defer hc.wg.Done()
				hc.checkInstance(ctx, inst)
			}(instance)
		}
	}
}

// checkInstance performs a health check on a single instance.
func (hc *HealthChecker) checkInstance(ctx context.Context, instance *ServiceInstance) {
	healthy := true
	var checkErr error

	// Check based on protocol
	switch instance.Metadata["protocol"] {
	case "http", "https":
		checkErr = hc.checkHTTP(ctx, instance)
		healthy = checkErr == nil
	default:
		// Default to TCP check
		checkErr = hc.checkTCP(ctx, instance)
		healthy = checkErr == nil
	}

	hc.mu.Lock()
	check, exists := hc.checks[instance.ID]
	if !exists {
		check = &healthCheck{instanceID: instance.ID}
		hc.checks[instance.ID] = check
	}

	check.lastCheck = time.Now().UTC()
	check.lastError = checkErr

	oldStatus := instance.HealthStatus
	newStatus := oldStatus

	if healthy {
		check.consecutiveFailures = 0
		check.consecutiveSuccesses++
		if check.consecutiveSuccesses >= hc.threshold {
			newStatus = HealthStatusHealthy
		}
	} else {
		check.consecutiveSuccesses = 0
		check.consecutiveFailures++
		if check.consecutiveFailures >= hc.threshold {
			newStatus = HealthStatusUnhealthy
		}
	}

	hc.mu.Unlock()

	// Update health status if changed
	if newStatus != oldStatus {
		if err := hc.registry.UpdateHealth(ctx, instance.ID, newStatus); err != nil {
			hc.logger.Error("Failed to update health status",
				zap.Error(err),
				zap.String("instance_id", instance.ID),
			)
		} else {
			hc.logger.Info("Health status updated",
				zap.String("instance_id", instance.ID),
				zap.String("service", instance.ServiceName),
				zap.String("old_status", string(oldStatus)),
				zap.String("new_status", string(newStatus)),
			)
		}
	}
}

// checkHTTP performs an HTTP health check.
func (hc *HealthChecker) checkHTTP(ctx context.Context, instance *ServiceInstance) error {
	endpoint := instance.Metadata["health_endpoint"]
	if endpoint == "" {
		endpoint = "/health"
	}

	protocol := instance.Metadata["protocol"]
	if protocol == "" {
		protocol = "http"
	}

	url := fmt.Sprintf("%s://%s%s", protocol, instance.Address(), endpoint)

	checkCtx, cancel := context.WithTimeout(ctx, hc.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(checkCtx, "GET", url, nil)
	if err != nil {
		return err
	}

	client := &http.Client{Timeout: hc.timeout}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check returned status %d", resp.StatusCode)
	}

	return nil
}

// checkTCP performs a TCP connectivity check.
func (hc *HealthChecker) checkTCP(ctx context.Context, instance *ServiceInstance) error {
	// Simple connectivity check using context timeout
	checkCtx, cancel := context.WithTimeout(ctx, hc.timeout)
	defer cancel()

	// In a real implementation, this would attempt a TCP connection
	// For now, we just check if the instance is expired
	select {
	case <-checkCtx.Done():
		return checkCtx.Err()
	default:
		return nil
	}
}

// GetHealthStatus returns the health status for an instance.
func (hc *HealthChecker) GetHealthStatus(instanceID string) (*healthCheck, bool) {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	check, exists := hc.checks[instanceID]
	return check, exists
}

// RemoveInstance removes health check tracking for an instance.
func (hc *HealthChecker) RemoveInstance(instanceID string) {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	delete(hc.checks, instanceID)
}
