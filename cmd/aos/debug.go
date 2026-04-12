package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newDebugCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "debug",
		Short: "Debug helpers (events, metrics — stub)",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "events",
		Short: "Stream control-plane events (stub)",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, _ = cmd, args
			fmt.Println("debug events: stub")
			return nil
		},
	})
	return cmd
}
