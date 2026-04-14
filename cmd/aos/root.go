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
		Short: "OpenOS (AOS) - Agent Operating System CLI",
		Long: `OpenOS (AOS) - Agent Operating System: control plane and runtime for long-running AI agents.

The welcome banner is shown by default on startup; set AOS_NO_BANNER to 1, true, yes, or on to disable it.
Run "aos doctor" for an environment self-check. The API server runs with "aos", "aos server", or "aos --config <file>" (default action is the control-plane server).
Other subcommands include agent, deploy, package, and debug (some are roadmap stubs).`,
		Version: version.GetDisplayVersion(),
	}
	root.PersistentFlags().StringVarP(&configFile, "config", "c", "config.yaml", "Path to config file")
	root.PersistentFlags().BoolVarP(&debugMode, "debug", "d", false, "Enable debug mode")

	// Default when no subcommand: run the API server (so "aos --config path" starts the server, not help-only).
	root.RunE = runServer

	root.AddCommand(newDoctorCmd())
	root.AddCommand(newServerCmd())
	root.AddCommand(newBuildCmd())
	root.AddCommand(newPushCmd())
	root.AddCommand(newPullCmd())
	root.AddCommand(newRunCmd())
	root.AddCommand(newAgentCmd())
	root.AddCommand(newDeployCmd())
	root.AddCommand(newPackageCmd())
	root.AddCommand(newDebugCmd())
	return root
}
