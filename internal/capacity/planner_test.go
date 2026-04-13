package capacity

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSimpleMovingAverage(t *testing.T) {
	now := time.Now()
	s := []Point{{T: now, V: 1}, {T: now.Add(time.Minute), V: 2}, {T: now.Add(2 * time.Minute), V: 3}}
	require.InDelta(t, 2.0, SimpleMovingAverage(s, 3), 0.01)
}
