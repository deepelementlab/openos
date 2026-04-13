package process

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInMemoryManager_GroupSessionNamespace(t *testing.T) {
	ctx := context.Background()
	m := NewInMemoryManager()

	g, err := m.CreateGroup(ctx, "leader-1")
	require.NoError(t, err)
	require.NotEmpty(t, g.GroupID)
	require.Contains(t, g.Members, "leader-1")

	require.NoError(t, m.AddToGroup(ctx, g.GroupID, "agent-2"))

	s, err := m.CreateSession(ctx, "leader-1")
	require.NoError(t, err)
	require.NoError(t, m.AttachGroupToSession(ctx, s.SessionID, g.GroupID))

	ns, err := m.CreateNamespace(ctx)
	require.NoError(t, err)
	require.NoError(t, m.EnterNamespace(ctx, "agent-2", ns))

	require.NoError(t, m.SignalGroup(ctx, g.GroupID, SignalTerminate))
	g2, err := m.GetGroup(ctx, g.GroupID)
	require.NoError(t, err)
	require.Equal(t, GroupStateStopping, g2.State)
}
