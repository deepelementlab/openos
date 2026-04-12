package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newPackageCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "package",
		Short: "Build and push agent packages (stub)",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, _ = cmd, args
			fmt.Println("package: stub — see pkg/packaging manifest format")
			return nil
		},
	}
}
