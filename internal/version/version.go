package version

import "fmt"

var (
	version   = "dev"
	gitCommit = "unknown"
	buildDate = "unknown"
)

func GetVersion() string {
	return version
}

func GetFullVersion() string {
	return fmt.Sprintf("%s (commit: %s, built: %s)", version, gitCommit, buildDate)
}
