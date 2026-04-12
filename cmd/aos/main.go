package main

import (
	"fmt"
	"os"
)

func main() {
	// Backward compatible: bare `aos` runs the control plane server.
	if len(os.Args) == 1 {
		os.Args = append(os.Args, "server")
	}
	if err := newRootCmd().Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
