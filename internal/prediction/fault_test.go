package prediction

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestFaultRisk(t *testing.T) {
	s := NodeSignal{CPUUtil: 0.95, MemUtil: 0.96, ErrorRate: 0.1}
	require.True(t, ShouldEvacuate(s, 0.5))
}

func TestMaintenanceWindow(t *testing.T) {
	m := MaintenanceWindow{Start: time.Now().Add(-time.Hour), End: time.Now().Add(time.Hour)}
	require.True(t, m.Active(time.Now()))
}
