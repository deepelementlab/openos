package policy

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEnforcer_Render(t *testing.T) {
	e := NewEnforcer()
	e.AllowTenantPair("a", "b")
	require.True(t, e.Allowed(context.Background(), "a", "b"))
	lines := e.RenderIPTables("AOS-FWD")
	require.NotEmpty(t, lines)
	nft := e.RenderNftables("inet", "forward")
	require.Contains(t, nft[0], "inet")
}
