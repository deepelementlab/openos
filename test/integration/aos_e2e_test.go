package integration_test

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/agentos/aos/internal/deployment"
	"github.com/agentos/aos/internal/network/dns"
	"github.com/agentos/aos/internal/network/policy"
	"github.com/agentos/aos/internal/scheduler/closure"
	"github.com/agentos/aos/internal/slo"
	"github.com/agentos/aos/pkg/packaging"
	"github.com/agentos/aos/pkg/runtime/facade"
)

// TestAOSPipeline_Minimal chains packaging → deployment → scheduling stubs (no real containerd).
func TestAOSPipeline_Minimal(t *testing.T) {
	f := facade.NewRuntimeFacade()
	require.Len(t, f.SupportedBackends(), 3)

	j := `{
	  "apiVersion": "openos.agent/v0alpha1",
	  "kind": "AgentPackage",
	  "metadata": {"name": "e2e", "version": "0.0.1"},
	  "spec": {"image": "docker.io/library/alpine:latest"}
	}`
	m, err := packaging.ParseManifest(strings.NewReader(j))
	require.NoError(t, err)
	p := deployment.NewPipeline()
	ctx := context.Background()
	res, err := p.PrepareFromManifest(ctx, m)
	require.NoError(t, err)
	require.Equal(t, "e2e", res.Spec.Name)

	b := closure.NewBinder()
	require.NoError(t, b.Bind(ctx, &closure.BindRequest{AgentID: "a1", NodeID: "n1"}))
	v := closure.NewVerifier()
	require.NoError(t, v.VerifyReady(ctx, "a1"))

	fqdn := dns.FQDN("e2e", "t1")
	ips, err := dns.Resolve(fqdn)
	require.NoError(t, err)
	require.NotEmpty(t, ips)

	pol := policy.NewEnforcer()
	require.True(t, pol.Allowed(ctx, "t1", "t1"))
	require.False(t, pol.Allowed(ctx, "t1", "t2"))

	col := slo.NewCollector()
	col.RecordAgentStart(slo.AgentStartSample{Success: true, LatencyMS: 10})
	g := slo.NewGate()
	g.MinStartSuccessRate = 0.5
	require.NoError(t, g.Evaluate(col))
}
