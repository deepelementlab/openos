package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newPackageCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "package",
		Short: "Agent package helpers (use `aos build`, `aos push`, `aos pull` for AAP workflow)",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, _ = cmd, args
			fmt.Println("Use: aos build -f Agentfile.json | aos push --name X --version Y --package-dir ./dist/... | aos pull --name X --version Y")
			fmt.Println("Spec: internal/builder/spec (aos.io/v1 AgentPackage)")
			return nil
		},
	}
}
