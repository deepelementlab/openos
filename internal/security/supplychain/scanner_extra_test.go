package supplychain

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAdmissionOK(t *testing.T) {
	require.True(t, AdmissionOK(&ScanResult{CriticalCVEs: 0, HighCVEs: 1}, 0, 1))
	require.False(t, AdmissionOK(&ScanResult{CriticalCVEs: 1, HighCVEs: 0}, 0, 0))
}

func TestScanner_Scan_EmptyRef(t *testing.T) {
	s := NewScanner()
	_, err := s.Scan(context.Background(), "")
	require.Error(t, err)
}
