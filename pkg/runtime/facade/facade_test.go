package facade

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewRuntimeFacade_SupportedBackends(t *testing.T) {
	f := NewRuntimeFacade()
	b := f.SupportedBackends()
	require.Len(t, b, 3)
}
