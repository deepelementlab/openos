package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/agentos/aos/internal/builder/engine"
	"github.com/agentos/aos/internal/builder/registry"
)

func newBuildCmd() *cobra.Command {
	var file, tag, out string
	cmd := &cobra.Command{
		Use:   "build",
		Short: "Build an AOS agent package (AAP) from an Agentfile",
		RunE: func(cmd *cobra.Command, args []string) error {
			if file == "" {
				return fmt.Errorf("use -f to specify Agentfile path")
			}
			eng := engine.NewEngine()
			spec, err := eng.Parse(engine.BuildSource{Path: file})
			if err != nil {
				return err
			}
			plan, err := eng.Plan(spec)
			if err != nil {
				return err
			}
			res, err := eng.Build(cmd.Context(), plan, engine.BuildOptions{Tag: tag})
			if err != nil {
				return err
			}
			if out == "" {
				out = filepath.Join(".", "dist", spec.Metadata.Name+"-"+spec.Metadata.Version)
			}
			if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
				return err
			}
			if err := engine.WriteLocalAAP(out, res, spec.Metadata.Name, spec.Metadata.Version); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Built AAP at %s (package_id=%s)\n", out, res.PackageID)
			return nil
		},
	}
	cmd.Flags().StringVarP(&file, "file", "f", "", "Path to Agentfile (JSON or YAML)")
	cmd.Flags().StringVarP(&tag, "tag", "t", "", "Optional tag label (recorded in future registry)")
	cmd.Flags().StringVarP(&out, "output", "o", "", "Output directory for AAP bundle")
	return cmd
}

func newPushCmd() *cobra.Command {
	var regRoot, name, version, pkgDir, httpURL, token string
	cmd := &cobra.Command{
		Use:   "push",
		Short: "Push a local AAP directory to a local or HTTP registry",
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" || version == "" || pkgDir == "" {
				return fmt.Errorf("require --name, --version, and --package-dir")
			}
			if httpURL != "" {
				hr := registry.NewHTTPRegistry(httpURL)
				hr.Token = token
				return hr.Push(cmd.Context(), pkgDir, name, version)
			}
			if regRoot == "" {
				regRoot = filepath.Join(".", "var", "aos-registry")
			}
			r := registry.NewLocalRegistry(regRoot)
			return r.Push(pkgDir, name, version, registry.PushOptions{})
		},
	}
	cmd.Flags().StringVar(&regRoot, "registry", "", "Local registry root directory")
	cmd.Flags().StringVar(&httpURL, "registry-url", "", "HTTP registry base URL (mutually exclusive with local --registry for storage)")
	cmd.Flags().StringVar(&token, "registry-token", "", "Bearer token for HTTP registry")
	cmd.Flags().StringVar(&name, "name", "", "Package name")
	cmd.Flags().StringVar(&version, "version", "", "Package version")
	cmd.Flags().StringVar(&pkgDir, "package-dir", "", "Path to AAP directory")
	return cmd
}

func newPullCmd() *cobra.Command {
	var regRoot, name, version, httpURL, token, dest string
	cmd := &cobra.Command{
		Use:   "pull",
		Short: "Pull a package from local or HTTP registry (prints path)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" || version == "" {
				return fmt.Errorf("require --name and --version")
			}
			if httpURL != "" {
				if dest == "" {
					var err error
					dest, err = os.MkdirTemp("", "aos-pull-*")
					if err != nil {
						return err
					}
				} else {
					if err := os.MkdirAll(dest, 0o755); err != nil {
						return err
					}
				}
				hr := registry.NewHTTPRegistry(httpURL)
				hr.Token = token
				if err := hr.Pull(cmd.Context(), name, version, dest); err != nil {
					return err
				}
				fmt.Fprintln(cmd.OutOrStdout(), dest)
				return nil
			}
			if regRoot == "" {
				regRoot = filepath.Join(".", "var", "aos-registry")
			}
			r := registry.NewLocalRegistry(regRoot)
			p, err := r.Pull(name, version, registry.PullOptions{})
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), p)
			return nil
		},
	}
	cmd.Flags().StringVar(&regRoot, "registry", "", "Local registry root directory")
	cmd.Flags().StringVar(&httpURL, "registry-url", "", "HTTP registry base URL")
	cmd.Flags().StringVar(&token, "registry-token", "", "Bearer token for HTTP registry")
	cmd.Flags().StringVar(&dest, "dest", "", "Destination directory for HTTP pull (default: temp dir)")
	cmd.Flags().StringVar(&name, "name", "", "Package name")
	cmd.Flags().StringVar(&version, "version", "", "Package version")
	return cmd
}
