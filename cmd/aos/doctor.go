package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/agentos/aos/internal/version"
)

// env vars that affect CLI UX (banner, colors). Documented for aos doctor.
var doctorEnvKeys = []string{
	"AOS_NO_BANNER",
	"AOS_FORCE_COLOR",
	"AOS_NO_COLOR",
	"NO_COLOR",
	"CI",
	"TERM",
}

func newDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Print environment self-check (banner, colors, version)",
		Long: `Shows version, OS/arch, config flag resolution, and values of environment
variables that affect startup banner and terminal colors.

If the welcome banner is missing on server start, check AOS_NO_BANNER:
  - unset or empty: banner shown (default)
  - 1, true, yes: banner hidden`,
		RunE: runDoctor,
	}
}

func runDoctor(cmd *cobra.Command, args []string) error {
	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "AOS doctor\n")
	fmt.Fprintf(out, "==========\n\n")

	fmt.Fprintf(out, "Binary\n")
	fmt.Fprintf(out, "  version:     %s (%s)\n", version.GetDisplayVersion(), version.GetVersion())
	fmt.Fprintf(out, "  full:        %s\n", version.GetFullVersion())
	fmt.Fprintf(out, "  go:          %s\n", runtime.Version())
	fmt.Fprintf(out, "  os/arch:     %s/%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Fprintf(out, "  executable:  %s\n", doctorExecutable())
	fmt.Fprintf(out, "\n")

	fmt.Fprintf(out, "CLI / config\n")
	fmt.Fprintf(out, "  --config:    %s\n", configFile)
	if abs, err := filepath.Abs(configFile); err == nil {
		fmt.Fprintf(out, "  config abs:  %s\n", abs)
	}
	fmt.Fprintf(out, "  --debug:     %v\n", debugMode)
	fmt.Fprintf(out, "\n")

	fmt.Fprintf(out, "Startup banner & terminal\n")
	fmt.Fprintf(out, "  banner show: %s\n", doctorBannerStatus())
	fmt.Fprintf(out, "  (hidden only when AOS_NO_BANNER is 1, true, yes, or on)\n")
	fmt.Fprintf(out, "\n")

	fmt.Fprintf(out, "Environment (relevant)\n")
	for _, k := range doctorEnvKeys {
		fmt.Fprintf(out, "  %-16s %s\n", k+":", doctorEnvValue(k))
	}
	fmt.Fprintf(out, "\n")
	fmt.Fprintf(out, "Tip: run `aos doctor` before `aos server` if the welcome banner is missing.\n")
	return nil
}

func doctorExecutable() string {
	exe, err := os.Executable()
	if err != nil {
		return "(unknown: " + err.Error() + ")"
	}
	return exe
}

func doctorEnvValue(k string) string {
	v, ok := os.LookupEnv(k)
	if !ok {
		return "(unset)"
	}
	if v == "" {
		return "(empty)"
	}
	return v
}

func doctorBannerStatus() string {
	if BannerExplicitlyDisabled() {
		return "no (AOS_NO_BANNER disables)"
	}
	return "yes"
}
