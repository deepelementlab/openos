package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newAgentCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agent",
		Short: "Manage agents (create, list, delete — stubs for AOS roadmap)",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "create",
		Short: "Create an agent from a package manifest",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, _ = cmd, args
			fmt.Println("agent create: not yet wired to control plane API")
			return nil
		},
	})
	return cmd
}
