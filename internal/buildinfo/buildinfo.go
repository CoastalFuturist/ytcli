package buildinfo

import "fmt"

var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

func String() string {
	return fmt.Sprintf("ytcli %s (commit %s, built %s)", Version, Commit, Date)
}
