package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newDeployCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "deploy",
		Short: "Apply declarative deployments (stub)",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, _ = cmd, args
			fmt.Println("deploy: stub - integrate with internal/deployment pipeline")
			return nil
		},
	}
}
