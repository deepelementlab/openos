package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/agentos/aos/internal/builder/registry"
	"github.com/agentos/aos/internal/kernel"
	"github.com/agentos/aos/pkg/runtime/facade"
	"github.com/agentos/aos/pkg/runtime/types"
)

func parsePackageRef(ref string) (name, version string, err error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return "", "", fmt.Errorf("empty package ref")
	}
	i := strings.LastIndex(ref, ":")
	if i <= 0 || i == len(ref)-1 {
		return "", "", fmt.Errorf("expected NAME:VERSION (e.g. myorg/myapp:v1.0)")
	}
	return ref[:i], ref[i+1:], nil
}

func newRunCmd() *cobra.Command {
	var regRoot, httpRegistry, token string
	var replicas int
	var dryRun bool
	var env []string

	cmd := &cobra.Command{
		Use:   "run REF",
		Short: "Pull an AAP and start agent(s) via RuntimeFacade with Kernel integration",
		Long: `REF must be NAME:VERSION (e.g. demo:0.1.0 or myorg_demo:v1).

Without --registry-url, pulls from the local registry (--registry).
Use --dry-run to validate the package without connecting to containerd/gVisor/Kata.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name, version, err := parsePackageRef(args[0])
			if err != nil {
				return err
			}
			ctx := cmd.Context()
			var pkgDir string
			var cleanup func()

			if httpRegistry != "" {
				tmp, err := os.MkdirTemp("", "aos-pull-*")
				if err != nil {
					return err
				}
				cleanup = func() { _ = os.RemoveAll(tmp) }
				defer cleanup()
				hr := registry.NewHTTPRegistry(httpRegistry)
				hr.Token = token
				if err := hr.Pull(ctx, name, version, tmp); err != nil {
					return err
				}
				pkgDir = tmp
			} else {
				if regRoot == "" {
					regRoot = filepath.Join(".", "var", "aos-registry")
				}
				r := registry.NewLocalRegistry(regRoot)
				pkgDir, err = r.Pull(name, version, registry.PullOptions{})
				if err != nil {
					return err
				}
			}

			pkg, err := registry.LoadAgentPackage(pkgDir)
			if err != nil {
				return err
			}
			sigPath := filepath.Join(pkgDir, registry.PackageSignatureFile)
			if st, statErr := os.Stat(sigPath); statErr == nil && !st.IsDir() {
				if err := registry.VerifyPackageDir(pkgDir); err != nil {
					return fmt.Errorf("package signature: %w", err)
				}
			}

			agentSpec, _, err := facade.AgentSpecFromPackage(pkg)
			if err != nil {
				return err
			}
			for _, e := range env {
				agentSpec.Env = append(agentSpec.Env, e)
			}

			if dryRun {
				fmt.Fprintf(cmd.OutOrStdout(), "dry-run: would start %d replica(s) for %s:%s (image=%q command=%v)\n",
					replicas, name, version, agentSpec.Image, agentSpec.Command)
				return nil
			}

			kf := kernel.NewDefaultFacade()
			rf := facade.NewRuntimeFacade(facade.WithKernel(kf))
			cfg := &types.RuntimeConfig{Type: types.RuntimeContainerd}
			if err := rf.Connect(ctx, facade.SelectBackendForSpec(agentSpec), cfg); err != nil {
				return fmt.Errorf("runtime connect failed (use --dry-run to skip): %w", err)
			}

			for i := 0; i < replicas; i++ {
				spec := *agentSpec
				if replicas > 1 {
					spec.ID = uuid.NewString()
					spec.Name = fmt.Sprintf("%s-%d", agentSpec.Name, i)
				}
				ag, err := rf.CreateAgent(ctx, &spec)
				if err != nil {
					return err
				}
				id := ag.ID
				if id == "" {
					id = spec.ID
				}
				if err := rf.StartAgent(ctx, id); err != nil {
					return err
				}
				fmt.Fprintf(cmd.OutOrStdout(), "started agent %s\n", id)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&regRoot, "registry", "", "Local registry root directory")
	cmd.Flags().StringVar(&httpRegistry, "registry-url", "", "HTTP registry base URL")
	cmd.Flags().StringVar(&token, "registry-token", "", "Bearer token for HTTP registry")
	cmd.Flags().IntVar(&replicas, "replicas", 1, "Number of agent instances")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Load and validate package only; do not connect to the container runtime")
	cmd.Flags().StringSliceVar(&env, "env", nil, "Extra environment KEY=VAL (repeatable)")
	return cmd
}
