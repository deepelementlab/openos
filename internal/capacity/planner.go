// Package capacity provides trend-based capacity recommendations.
package capacity

import (
	"math"
	"time"
)

// Point is a single utilization sample.
type Point struct {
	T time.Time
	V float64
}

// SimpleMovingAverage forecasts next value as SMA of window.
func SimpleMovingAverage(series []Point, window int) float64 {
	if len(series) == 0 || window <= 0 {
		return 0
	}
	n := window
	if len(series) < window {
		n = len(series)
	}
	var sum float64
	for i := len(series) - n; i < len(series); i++ {
		sum += series[i].V
	}
	return sum / float64(n)
}

// RecommendAgents suggests additional agent capacity given growth trend.
func RecommendAgents(history []Point, horizonDays int, headroom float64) int {
	if len(history) < 2 {
		return 0
	}
	slope := (history[len(history)-1].V - history[0].V) / float64(len(history))
	projected := history[len(history)-1].V + slope*float64(horizonDays)
	need := math.Ceil(projected * (1 + headroom))
	return int(need)
}
