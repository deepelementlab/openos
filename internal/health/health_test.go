package health

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewChecker(t *testing.T) {
	checker := NewChecker()
	require.NotNil(t, checker)
	assert.NotNil(t, checker.components)
	assert.True(t, time.Since(checker.startTime) < time.Second)
}

func TestRegisterAndUnregister(t *testing.T) {
	checker := NewChecker()

	// Test registering a component
	checker.Register("database", "PostgreSQL database connection", func(ctx context.Context) error {
		return nil
	})

	// Verify component was registered
	component := checker.GetComponent("database")
	require.NotNil(t, component)
	assert.Equal(t, "database", component.Name)
	assert.Equal(t, "PostgreSQL database connection", component.Description)
	assert.Equal(t, StatusUnknown, component.Status)

	// Test unregistering
	checker.Unregister("database")
	component = checker.GetComponent("database")
	assert.Nil(t, component)
}

func TestCheck(t *testing.T) {
	checker := NewChecker()
	ctx := context.Background()

	// Mock healthy component
	healthyCalled := false
	checker.Register("healthy", "Healthy component", func(ctx context.Context) error {
		healthyCalled = true
		return nil
	})

	// Mock unhealthy component
	unhealthyCalled := false
	checker.Register("unhealthy", "Unhealthy component", func(ctx context.Context) error {
		unhealthyCalled = true
		return errors.New("component failure")
	})

	// Mock component with nil check function
	checker.Register("unknown", "Unknown component", nil)

	// Run checks
	results := checker.Check(ctx)

	// Verify results
	require.NotNil(t, results)
	assert.True(t, healthyCalled)
	assert.True(t, unhealthyCalled)

	healthyComp, exists := results["healthy"]
	require.True(t, exists)
	assert.Equal(t, StatusHealthy, healthyComp.Status)
	assert.Nil(t, healthyComp.Error)

	unhealthyComp, exists := results["unhealthy"]
	require.True(t, exists)
	assert.Equal(t, StatusUnhealthy, unhealthyComp.Status)
	assert.Error(t, unhealthyComp.Error)
	assert.Equal(t, "component failure", unhealthyComp.Error.Error())

	unknownComp, exists := results["unknown"]
	require.True(t, exists)
	assert.Equal(t, StatusUnknown, unknownComp.Status)
	assert.Nil(t, unknownComp.Error)
}

func TestCheckComponent(t *testing.T) {
	checker := NewChecker()
	ctx := context.Background()

	// Register a component
	checkCalls := 0
	checker.Register("test", "Test component", func(ctx context.Context) error {
		checkCalls++
		if checkCalls == 1 {
			return nil
		}
		return errors.New("failed")
	})

	// Test first check (should succeed)
	component, err := checker.CheckComponent(ctx, "test")
	require.NoError(t, err)
	require.NotNil(t, component)
	assert.Equal(t, StatusHealthy, component.Status)
	assert.Equal(t, 1, checkCalls)

	// Test second check (should fail)
	component, err = checker.CheckComponent(ctx, "test")
	require.NoError(t, err)
	require.NotNil(t, component)
	assert.Equal(t, StatusUnhealthy, component.Status)
	assert.Equal(t, "failed", component.Error.Error())
	assert.Equal(t, 2, checkCalls)

	// Test non-existent component
	component, err = checker.CheckComponent(ctx, "nonexistent")
	assert.NoError(t, err)
	assert.Nil(t, component)
}

func TestStatus(t *testing.T) {
	checker := NewChecker()
	ctx := context.Background()

	// Register components with different statuses
	checker.Register("healthy1", "Healthy component 1", func(ctx context.Context) error {
		return nil
	})
	checker.Register("healthy2", "Healthy component 2", func(ctx context.Context) error {
		return nil
	})
	checker.Register("unhealthy", "Unhealthy component", func(ctx context.Context) error {
		return errors.New("failure")
	})
	checker.Register("unknown", "Unknown component", nil)

	// Run checks to update statuses
	checker.Check(ctx)

	// Get overall status
	status := checker.Status()
	require.NotNil(t, status)

	assert.Equal(t, StatusUnhealthy, status["status"])
	assert.Equal(t, 4, status["total"])
	assert.Equal(t, 2, status["healthy"])
	assert.Equal(t, 1, status["unhealthy"])
	assert.Equal(t, 1, status["unknown"])

	// Check uptime is a string
	uptime, ok := status["uptime"].(string)
	assert.True(t, ok)
	assert.NotEmpty(t, uptime)
}

func TestStatus_AllHealthy(t *testing.T) {
	checker := NewChecker()
	ctx := context.Background()

	// Register only healthy components
	checker.Register("healthy1", "Healthy component 1", func(ctx context.Context) error {
		return nil
	})
	checker.Register("healthy2", "Healthy component 2", func(ctx context.Context) error {
		return nil
	})

	checker.Check(ctx)
	status := checker.Status()

	assert.Equal(t, StatusHealthy, status["status"])
	assert.Equal(t, 2, status["total"])
	assert.Equal(t, 2, status["healthy"])
	assert.Equal(t, 0, status["unhealthy"])
	assert.Equal(t, 0, status["unknown"])
}

func TestStatus_AllUnknown(t *testing.T) {
	checker := NewChecker()

	// Register only unknown components (no check functions)
	checker.Register("unknown1", "Unknown component 1", nil)
	checker.Register("unknown2", "Unknown component 2", nil)

	status := checker.Status()

	assert.Equal(t, StatusUnknown, status["status"])
	assert.Equal(t, 2, status["total"])
	assert.Equal(t, 0, status["healthy"])
	assert.Equal(t, 0, status["unhealthy"])
	assert.Equal(t, 2, status["unknown"])
}

func TestListComponents(t *testing.T) {
	checker := NewChecker()

	checker.Register("db", "Database", nil)
	checker.Register("cache", "Redis cache", nil)
	checker.Register("api", "API service", nil)

	components := checker.ListComponents()
	assert.Len(t, components, 3)

	// Check that we get copies, not original references
	for _, component := range components {
		assert.NotNil(t, component)
		assert.Contains(t, []string{"db", "cache", "api"}, component.Name)
	}
}

func TestConcurrentAccess(t *testing.T) {
	checker := NewChecker()
	ctx := context.Background()
	done := make(chan bool)

	// Start multiple goroutines accessing the checker
	for i := 0; i < 10; i++ {
		go func(id int) {
			componentName := "component"
			checker.Register(componentName, "Test component", func(ctx context.Context) error {
				if id%2 == 0 {
					return nil
				}
				return errors.New("error")
			})
			
			checker.CheckComponent(ctx, componentName)
			checker.ListComponents()
			checker.Status()
			
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify no panic occurred
	status := checker.Status()
	assert.NotNil(t, status)
}

func TestLastCheckTime(t *testing.T) {
	checker := NewChecker()
	ctx := context.Background()

	checker.Register("test", "Test component", func(ctx context.Context) error {
		time.Sleep(10 * time.Millisecond)
		return nil
	})

	// Get initial status
	initialComponent := checker.GetComponent("test")
	require.NotNil(t, initialComponent)
	assert.True(t, initialComponent.LastCheck.IsZero())

	// Run check
	component, err := checker.CheckComponent(ctx, "test")
	require.NoError(t, err)

	// Verify last check time was updated
	assert.False(t, component.LastCheck.IsZero())
	assert.True(t, time.Since(component.LastCheck) < time.Second)

	// Verify original component was also updated
	updatedComponent := checker.GetComponent("test")
	assert.False(t, updatedComponent.LastCheck.IsZero())
}

func TestErrorPropagation(t *testing.T) {
	checker := NewChecker()
	ctx := context.Background()

	expectedError := errors.New("connection timeout")
	checker.Register("failing", "Failing component", func(ctx context.Context) error {
		return expectedError
	})

	results := checker.Check(ctx)
	component, exists := results["failing"]
	require.True(t, exists)

	assert.Equal(t, StatusUnhealthy, component.Status)
	assert.Equal(t, expectedError, component.Error)
	assert.Equal(t, "connection timeout", component.Error.Error())
}