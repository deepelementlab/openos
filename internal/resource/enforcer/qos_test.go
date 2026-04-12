package enforcer

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestClassify(t *testing.T) {
	require.Equal(t, QoSBestEffort, Classify(QoSSpec{}))
	require.Equal(t, QoSGuaranteed, Classify(QoSSpec{
		CPURequestNano: 1000, CPULimitNano: 1000,
		MemoryRequest: 128 * 1024 * 1024, MemoryLimit: 128 * 1024 * 1024,
	}))
	require.Equal(t, QoSBurstable, Classify(QoSSpec{
		CPURequestNano: 500, CPULimitNano: 1000,
		MemoryRequest: 64 * 1024 * 1024, MemoryLimit: 128 * 1024 * 1024,
	}))
}
