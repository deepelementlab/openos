package closure

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/agentos/aos/pkg/runtime/types"
	"github.com/stretchr/testify/require"
)

func TestBinder_BindSimple(t *testing.T) {
	b := NewBinder()
	ctx := context.Background()
	require.NoError(t, b.BindSimple(ctx, "a1", "n1", &ResourceSpec{CPUMilli: 100}))
}

func TestBinder_Bind_Compensation(t *testing.T) {
	br := &okBinding{}
	store := &failStore{}
	b := &Binder{
		Reserver: NewMemoryReserver(),
		Store:    store,
		Binding:  br,
	}
	ctx := context.Background()
	err := b.Bind(ctx, &BindRequest{
		AgentID: "a1", NodeID: "n1",
		AgentSpec: &types.AgentSpec{ID: "a1", Name: "x", Image: "nginx:latest"},
	})
	require.Error(t, err)
	require.Equal(t, 1, br.deleted)
}

type failStore struct{}

func (f *failStore) SaveBinding(ctx context.Context, agentID, nodeID string) error {
	_ = ctx
	_ = agentID
	_ = nodeID
	return errors.New("boom")
}

func (f *failStore) GetNode(ctx context.Context, agentID string) (string, error) {
	return "", errors.New("none")
}

func (f *failStore) DeleteBinding(ctx context.Context, agentID string) error { return nil }

type okBinding struct {
	deleted int
}

func (o *okBinding) CreateAgent(ctx context.Context, spec *types.AgentSpec) (*types.Agent, error) {
	return &types.Agent{ID: spec.ID}, nil
}

func (o *okBinding) DeleteAgent(ctx context.Context, agentID string) error {
	o.deleted++
	return nil
}

func TestRescheduler_Heartbeat(t *testing.T) {
	r := NewRescheduler()
	r.HeartbeatTimeout = 200 * time.Millisecond
	r.RecordHeartbeat("n1", time.Now())
	require.True(t, r.IsNodeHealthy("n1"))
	r.MarkNodeFailed("n1")
	require.False(t, r.IsNodeHealthy("n1"))
}

func TestRescheduler_RescheduleFromStore(t *testing.T) {
	var gotAgent, from, reason string
	r := &Rescheduler{
		OnReschedule: func(ctx context.Context, agentID, fromNode, rsn string) error {
			gotAgent, from, reason = agentID, fromNode, rsn
			return nil
		},
	}
	st := NewMemoryBindingStore()
	_ = st.SaveBinding(context.Background(), "a1", "n9")
	require.NoError(t, r.RescheduleFromStore(context.Background(), "a1", st, "node lost"))
	require.Equal(t, "a1", gotAgent)
	require.Equal(t, "n9", from)
	require.Equal(t, "node lost", reason)
}
