package main

import (
	"github.com/spf13/cobra"

	"github.com/agentos/aos/internal/version"
)

var (
	configFile string
	debugMode  bool
)

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "aos",
		Short: "OpenOS (AOS) — Agent Operating System CLI",
		Long: `OpenOS provides a control plane and runtime for long-running AI agents.
Use "aos server" to run the API server, or subcommands for agent/package/deploy operations.`,
		Version: version.GetVersion(),
	}
	root.PersistentFlags().StringVarP(&configFile, "config", "c", "config.yaml", "Path to config file")
	root.PersistentFlags().BoolVarP(&debugMode, "debug", "d", false, "Enable debug mode")

	root.AddCommand(newServerCmd())
	root.AddCommand(newAgentCmd())
	root.AddCommand(newDeployCmd())
	root.AddCommand(newPackageCmd())
	root.AddCommand(newDebugCmd())
	return root
}
