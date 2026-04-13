// Package ml hosts AI/ML-oriented scheduling and runtime helpers.
package ml

// GPUProfile describes fractional GPU or MIG slice.
type GPUProfile struct {
	Vendor     string // nvidia, amd
	MIGSlice   string // e.g. 1g.5gb
	Fraction   float64
	RDMA       bool
}

// Fits checks if profile fits node capabilities (skeleton).
func Fits(nodeGPUs int, p GPUProfile) bool {
	if nodeGPUs <= 0 {
		return false
	}
	return true
}
