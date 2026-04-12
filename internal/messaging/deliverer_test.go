package messaging

import (
	"context"
	"testing"

	"github.com/agentos/aos/internal/messaging/nats"
	"go.uber.org/zap"
)

func newTestDeliverer() *Deliverer {
	return NewDeliverer(&nats.Client{}, zap.NewNop())
}

func TestDeliverer_Deliver_Unhealthy(t *testing.T) {
	d := newTestDeliverer()
	defer d.Close()

	err := d.Deliver(context.Background(), "event-1", "agent.created", []byte(`{}`), map[string]string{})
	if err == nil {
		t.Error("expected error when NATS connection is not healthy")
	}
}

func TestDeliverer_IsHealthy(t *testing.T) {
	d := newTestDeliverer()
	defer d.Close()

	if d.IsHealthy() {
		t.Error("expected unhealthy with zero-value client")
	}
}

func TestDeliverer_Stats(t *testing.T) {
	d := newTestDeliverer()
	defer d.Close()

	stats := d.Stats()
	if stats["healthy"] != false {
		t.Error("expected healthy=false")
	}
	if stats["publisher_stats"] == nil {
		t.Error("expected publisher_stats")
	}
	if stats["nats_client_stats"] == nil {
		t.Error("expected nats_client_stats")
	}
}

func TestDeliverer_Close(t *testing.T) {
	d := newTestDeliverer()
	d.Close()
}

func TestDeliverer_Close_Nil(t *testing.T) {
	d := &Deliverer{client: &nats.Client{}, logger: zap.NewNop()}
	d.Close()
}

func TestDeliverer_buildTopic(t *testing.T) {
	d := newTestDeliverer()
	topic := d.buildTopic("agent.created")
	if topic != "aos.agent.created.v1" {
		t.Errorf("expected aos.agent.created.v1, got %s", topic)
	}
	topic2 := d.buildTopic("workflow.step.completed")
	if topic2 != "aos.workflow.step.completed.v1" {
		t.Errorf("expected aos.workflow.step.completed.v1, got %s", topic2)
	}
}

func TestDeliverer_ToDeliverFunc(t *testing.T) {
	d := newTestDeliverer()
	defer d.Close()

	fn := d.ToDeliverFunc()
	if fn == nil {
		t.Error("expected non-nil DeliverFunc")
	}

	err := fn(context.Background(), "evt-1", "test.type", []byte(`{}`), nil)
	if err == nil {
		t.Error("expected error from Deliver with unhealthy client")
	}
}

func TestDeliverFunc_Type(t *testing.T) {
	var fn DeliverFunc = func(ctx context.Context, eventID string, eventType string, payload []byte, metadata map[string]string) error {
		return nil
	}
	if fn == nil {
		t.Error("expected non-nil function")
	}
}
