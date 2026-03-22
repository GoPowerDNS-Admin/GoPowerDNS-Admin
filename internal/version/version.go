// Package version provides version information for the application.
package version

import (
	"runtime/debug"
	"strings"
)

// version and branch are set at build time via -ldflags:
//
//	-X github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/version.version=v1.2.3
//	-X github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/version.branch=main
const devVersion = "dev"

var version = devVersion //nolint:gochecknoglobals // must be a var so -ldflags -X can override it at link time
var branch = ""          //nolint:gochecknoglobals // must be a var so -ldflags -X can override it at link time

// Get returns the application version, optionally suffixed with the branch
// name when available (e.g. "abc1234-dirty (feat/my-feature)").
// When not set via -ldflags, the commit hash is read from the VCS info
// embedded by the Go toolchain (Go 1.18+).
func Get() string {
	v := version

	if v == devVersion {
		v = commitFromBuildInfo()
	}

	if b := strings.TrimSpace(branch); b != "" && b != "HEAD" {
		return v + " (" + b + ")"
	}

	return v
}

func commitFromBuildInfo() string {
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
