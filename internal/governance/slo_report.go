package governance

import (
	"time"
)

// SLOReport aggregates availability/error budget for a service.
type SLOReport struct {
	Service      string
	Window       time.Duration
	Availability float64
	ErrorBudget  float64
}

// ComputeAvailability from good/total probes.
func ComputeAvailability(good, total int64) float64 {
	if total == 0 {
		return 1
	}
	return float64(good) / float64(total)
}
