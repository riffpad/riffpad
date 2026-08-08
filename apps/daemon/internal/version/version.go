// Package version is the single source of truth for the riffpad CLI
// version string. Release builds inject the tagged version at link time;
// local/dev builds report "dev".
package version

// Version is set by the release workflow via:
//
//	go build -ldflags "-X github.com/riffpad/riffpad/apps/daemon/internal/version.Version=0.2.1"
//
// Keep it a variable (not a const) so -X can override it.
var Version = "dev"
