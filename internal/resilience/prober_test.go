package resilience

import (
	"context"
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestProber_HTTP(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	p := NewProber()
	p.Timeout = 2 * time.Second
	err := p.Run(context.Background(), ProbeReadiness, srv.URL)
	require.NoError(t, err)
}

func TestProber_TCP(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) }))
	defer srv.Close()
	host := srv.Listener.Addr().String()

	p := NewProber()
	p.Timeout = 2 * time.Second
	err := p.Run(context.Background(), ProbeLiveness, "tcp:"+host)
	require.NoError(t, err)
}

func TestProber_ExecTrue(t *testing.T) {
	p := NewProber()
	p.Timeout = 5 * time.Second
	var target string
	if runtime.GOOS == "windows" {
		target = "exec:cmd /c exit 0"
	} else {
		target = "exec:/bin/sh -c true"
	}
	err := p.Run(context.Background(), ProbeStartup, target)
	require.NoError(t, err)
}
