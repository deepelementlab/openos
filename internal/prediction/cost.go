package prediction

// CostBreakdown estimates spend by dimension (USD, arbitrary units).
type CostBreakdown struct {
	Compute float64
	Network float64
	Storage float64
}

// OptimizeSuggestion returns a human-readable hint (placeholder rules engine).
func OptimizeSuggestion(b CostBreakdown) string {
	if b.Network > b.Compute*0.5 {
		return "consider colocating agents to reduce cross-AZ traffic"
	}
	if b.Storage > b.Compute {
		return "review checkpoint retention and tiered storage"
	}
	return "no major anomalies detected"
}
