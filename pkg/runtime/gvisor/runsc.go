package gvisor

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/agentos/aos/pkg/runtime/types"
)

func resolveRunsc(cfg *types.RuntimeConfig) string {
	if cfg != nil && cfg.Options != nil {
		if p, ok := cfg.Options["runsc_path"].(string); ok && p != "" {
			return p
		}
	}
	if runtime.GOOS != "linux" {
		return ""
	}
	p, err := exec.LookPath("runsc")
	if err != nil {
		return ""
	}
	return p
}

func (r *GVisorRuntime) runscEnabled() bool {
	return os.Getenv("AOS_RUNSC") == "1" && r.runsc != ""
}

// ociConfigMinimal builds a minimal OCI config (caller must provide usable rootfs for real runsc).
func ociConfigMinimal(spec *types.AgentSpec) map[string]interface{} {
	args := []string{"/bin/sh"}
	if spec != nil && len(spec.Command) > 0 {
		args = append(spec.Command, spec.Args...)
	}
	return map[string]interface{}{
		"ociVersion": "1.0.2",
		"process": map[string]interface{}{
			"terminal": false,
			"user":     map[string]interface{}{"uid": 0, "gid": 0},
			"args":     args,
			"cwd":      "/",
		},
		"root": map[string]interface{}{
			"path":     "rootfs",
			"readonly": false,
		},
		"hostname": "aos-runsc",
		"linux": map[string]interface{}{
			"namespaces": []map[string]string{
				{"type": "mount"},
			},
		},
	}
}

func (r *GVisorRuntime) runscCreate(ctx context.Context, spec *types.AgentSpec) error {
	bundle := filepath.Join(r.root, "bundles", spec.ID)
	if err := os.RemoveAll(bundle); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(bundle, "rootfs"), 0o755); err != nil {
		return err
	}
	// Placeholder file so rootfs is non-empty (runsc still needs real binaries for exec).
	if err := os.WriteFile(filepath.Join(bundle, "rootfs", ".aos-placeholder"), []byte("aos\n"), 0o644); err != nil {
		return err
	}
	cfg := ociConfigMinimal(spec)
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(bundle, "config.json"), data, 0o644); err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, r.runsc, "create", "--bundle", bundle, spec.ID)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("runsc create: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (r *GVisorRuntime) runscStart(ctx context.Context, agentID string) error {
	cmd := exec.CommandContext(ctx, r.runsc, "start", agentID)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("runsc start: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (r *GVisorRuntime) runscDelete(ctx context.Context, agentID string) error {
	cmd := exec.CommandContext(ctx, r.runsc, "delete", agentID)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("runsc delete: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}
