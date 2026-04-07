// Package health provides health check functionality for Agent OS.
package health

import (
	"context"
	"sync"
	"time"
)

// Status represents the health status of a component.
type Status string

const (
	// StatusHealthy indicates the component is healthy.
	StatusHealthy Status = "healthy"
	// StatusUnhealthy indicates the component is unhealthy.
	StatusUnhealthy Status = "unhealthy"
	// StatusUnknown indicates the health status is unknown.
	StatusUnknown Status = "unknown"
)

// Component represents a health-checkable component.
type Component struct {
	Name        string
	Description string
	Check       func(ctx context.Context) error
	Status      Status
	LastCheck   time.Time
	Error       error
}

// Checker manages health checks for multiple components.
type Checker struct {
	mu         sync.RWMutex
	components map[string]*Component
	startTime  time.Time
}

// NewChecker creates a new health checker.
func NewChecker() *Checker {
	return &Checker{
		components: make(map[string]*Component),
		startTime:  time.Now(),
	}
}

// Register registers a new component for health checking.
func (c *Checker) Register(name, description string, checkFunc func(ctx context.Context) error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	c.components[name] = &Component{
		Name:        name,
		Description: description,
		Check:       checkFunc,
		Status:      StatusUnknown,
		LastCheck:   time.Time{},
	}
}

// Unregister removes a component from health checking.
func (c *Checker) Unregister(name string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.components, name)
}

// Check performs health checks on all registered components.
func (c *Checker) Check(ctx context.Context) map[string]*Component {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	results := make(map[string]*Component)
	
	for name, component := range c.components {
		componentCopy := *component
		componentCopy.LastCheck = time.Now()
		
		if component.Check != nil {
			if err := component.Check(ctx); err != nil {
				componentCopy.Status = StatusUnhealthy
				componentCopy.Error = err
			} else {
				componentCopy.Status = StatusHealthy
				componentCopy.Error = nil
			}
		} else {
			componentCopy.Status = StatusUnknown
			componentCopy.Error = nil
		}
		
		// Update the component status
		component.Status = componentCopy.Status
		component.LastCheck = componentCopy.LastCheck
		component.Error = componentCopy.Error
		
		results[name] = &componentCopy
	}
	
	return results
}

// CheckComponent performs health check on a single component.
func (c *Checker) CheckComponent(ctx context.Context, name string) (*Component, error) {
	c.mu.RLock()
	component, exists := c.components[name]
	c.mu.RUnlock()
	
	if !exists {
		return nil, nil
	}
	
	componentCopy := *component
	componentCopy.LastCheck = time.Now()
	
	if component.Check != nil {
		if err := component.Check(ctx); err != nil {
			componentCopy.Status = StatusUnhealthy
			componentCopy.Error = err
		} else {
			componentCopy.Status = StatusHealthy
			componentCopy.Error = nil
		}
	} else {
		componentCopy.Status = StatusUnknown
		componentCopy.Error = nil
	}
	
	// Update the original component
	c.mu.Lock()
	component.Status = componentCopy.Status
	component.LastCheck = componentCopy.LastCheck
	component.Error = componentCopy.Error
	c.mu.Unlock()
	
	return &componentCopy, nil
}

// Status returns the overall health status of the system.
func (c *Checker) Status() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	total := len(c.components)
	healthy := 0
	unhealthy := 0
	unknown := 0
	
	for _, component := range c.components {
		switch component.Status {
		case StatusHealthy:
			healthy++
		case StatusUnhealthy:
			unhealthy++
		default:
			unknown++
		}
	}
	
	overallStatus := StatusHealthy
	if unhealthy > 0 {
		overallStatus = StatusUnhealthy
	} else if unknown > 0 && healthy == 0 {
		overallStatus = StatusUnknown
	}
	
	return map[string]interface{}{
		"status":    overallStatus,
		"uptime":    time.Since(c.startTime).String(),
		"started":   c.startTime.Format(time.RFC3339),
		"total":     total,
		"healthy":   healthy,
		"unhealthy": unhealthy,
		"unknown":   unknown,
	}
}

// GetComponent returns a component by name.
func (c *Checker) GetComponent(name string) *Component {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	component, exists := c.components[name]
	if !exists {
		return nil
	}
	
	// Return a copy to avoid race conditions
	componentCopy := *component
	return &componentCopy
}

// ListComponents returns a list of all registered components.
func (c *Checker) ListComponents() []*Component {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	components := make([]*Component, 0, len(c.components))
	for _, component := range c.components {
		componentCopy := *component
		components = append(components, &componentCopy)
	}
	
	return components
}