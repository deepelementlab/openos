package version

import (
	"fmt"
	"strings"
)

var (
	// Semver string without leading V; override at link time, e.g.:
	// -ldflags "-X github.com/agentos/aos/internal/version.version=0.1.2"
	version   = "0.1.2"
	gitCommit = "unknown"
	buildDate = "unknown"
)

func GetVersion() string {
	return version
}

// GetDisplayVersion returns a user-facing label such as "V0.1.2".
func GetDisplayVersion() string {
	v := strings.TrimSpace(version)
	v = strings.TrimPrefix(strings.TrimPrefix(v, "V"), "v")
	if v == "" {
		return "Vdev"
	}
	return "V" + v
}

func GetFullVersion() string {
	return fmt.Sprintf("%s (commit: %s, built: %s)", version, gitCommit, buildDate)
}
