// Package version provides version information for the application.
package version

import "runtime/debug"

// version is set at build time via -ldflags:
//
//	-X github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/version.version=v1.2.3
const devVersion = "dev"

var version = devVersion //nolint:gochecknoglobals // must be a var so -ldflags -X can override it at link time

// Get returns the application version. When not set via -ldflags (e.g. go build
// on a branch), it falls back to the VCS commit hash embedded by the Go toolchain
// (Go 1.18+), optionally suffixed with "-dirty" if there are uncommitted changes.
func Get() string {
	if version != devVersion {
		return version
	}

	info, ok := debug.ReadBuildInfo()
	if !ok {
		return devVersion
	}

	var commit, dirty string

	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			if len(s.Value) > 7 {
				commit = s.Value[:7]
			} else {
				commit = s.Value
			}
		case "vcs.modified":
			if s.Value == "true" {
				dirty = "-dirty"
			}
		}
	}

	if commit == "" {
		return devVersion
	}

	return commit + dirty
}
