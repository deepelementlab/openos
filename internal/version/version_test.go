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
	// Default semver when not set via ldflags
	if v != "0.1.2" {
		t.Errorf("expected '0.1.2', got '%s'", v)
	}
}

func TestGetFullVersion(t *testing.T) {
	fv := GetFullVersion()
	if fv == "" {
		t.Error("expected non-empty full version")
	}
	if !strings.Contains(fv, "0.1.2") {
		t.Error("expected full version to contain semver")
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
	expected := "0.1.2 (commit: unknown, built: unknown)"
	if fv != expected {
		t.Errorf("expected '%s', got '%s'", expected, fv)
	}
}

func TestGetDisplayVersion(t *testing.T) {
	d := GetDisplayVersion()
	if d != "V0.1.2" {
		t.Errorf("expected V0.1.2, got %q", d)
	}
}
