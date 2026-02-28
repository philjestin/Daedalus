package version

import "fmt"

// Build-time variables injected via ldflags.
var (
	Version = "dev"
	Commit  = "unknown"
	Date    = "unknown"
	BuiltBy = "unknown"
)

// String returns a formatted version string.
func String() string {
	if Version == "dev" {
		return "dev"
	}
	return fmt.Sprintf("%s (commit: %s, built: %s)", Version, Commit, Date)
}
