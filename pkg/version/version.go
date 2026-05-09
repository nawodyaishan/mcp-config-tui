package version

import "fmt"

var (
	Version   = "dev"
	Commit    = "none"
	Date      = "unknown"
	GoVersion = "unknown"
)

func String() string {
	return fmt.Sprintf("%s (commit %s, built %s, %s)", Version, Commit, Date, GoVersion)
}
