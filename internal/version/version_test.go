package version

import (
	"strings"
	"testing"
)

func TestGetVersion(t *testing.T) {
	v := GetVersion()
	if v == "" {
		t.Error("expected non-empty version")
	}
	// Default is "dev" when not set via ldflags
	if v != "dev" {
		t.Errorf("expected 'dev', got '%s'", v)
	}
}

func TestGetFullVersion(t *testing.T) {
	fv := GetFullVersion()
	if fv == "" {
		t.Error("expected non-empty full version")
	}
	if !strings.Contains(fv, "dev") {
		t.Error("expected full version to contain 'dev'")
	}
	if !strings.Contains(fv, "commit:") {
		t.Error("expected full version to contain 'commit:'")
	}
	if !strings.Contains(fv, "built:") {
		t.Error("expected full version to contain 'built:'")
	}
}

func TestGetFullVersionFormat(t *testing.T) {
	fv := GetFullVersion()
	expected := "dev (commit: unknown, built: unknown)"
	if fv != expected {
		t.Errorf("expected '%s', got '%s'", expected, fv)
	}
}
