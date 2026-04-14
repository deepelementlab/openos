package main

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDoctorCmd_Run(t *testing.T) {
	var buf bytes.Buffer
	cmd := newDoctorCmd()
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	require.NoError(t, cmd.ExecuteContext(context.Background()))
	out := buf.String()
	require.Contains(t, out, "AOS doctor")
	require.Contains(t, out, "AOS_NO_BANNER")
	require.Contains(t, out, "banner show:")
}

func TestBannerExplicitlyDisabled(t *testing.T) {
	t.Setenv("AOS_NO_BANNER", "")
	require.False(t, BannerExplicitlyDisabled())
	t.Setenv("AOS_NO_BANNER", "0")
	require.False(t, BannerExplicitlyDisabled())
	t.Setenv("AOS_NO_BANNER", "garbage")
	require.False(t, BannerExplicitlyDisabled())
	t.Setenv("AOS_NO_BANNER", "1")
	require.True(t, BannerExplicitlyDisabled())
	t.Setenv("AOS_NO_BANNER", "TRUE")
	require.True(t, BannerExplicitlyDisabled())
	t.Setenv("AOS_NO_BANNER", "on")
	require.True(t, BannerExplicitlyDisabled())
}
