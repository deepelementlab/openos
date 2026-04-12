package supplychain

import "context"

// ScanResult is a vulnerability scan summary (stub).
type ScanResult struct {
	ImageRef     string
	CriticalCVEs int
	HighCVEs     int
}

// Scanner runs image scans (stub: integrate Trivy/Grype).
type Scanner struct{}

// NewScanner creates a scanner.
func NewScanner() *Scanner {
	return &Scanner{}
}

// Scan returns zero CVEs in stub mode.
func (s *Scanner) Scan(ctx context.Context, imageRef string) (*ScanResult, error) {
	_ = ctx
	return &ScanResult{ImageRef: imageRef}, nil
}
