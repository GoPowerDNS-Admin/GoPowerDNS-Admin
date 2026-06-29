package version

import "testing"

func TestVersion(t *testing.T) {
	orig := version

	t.Cleanup(func() { version = orig })

	// A tag injected via -ldflags is returned verbatim; Version never appends
	// the branch (that is Get's job).
	version = "v0.3.2"

	if got := Version(); got != "v0.3.2" {
		t.Errorf("Version() = %q, want %q", got, "v0.3.2")
	}

	// An unversioned build falls back to the module version (unavailable under
	// `go test`, reported as "(devel)"), so it resolves to "dev".
	version = devVersion

	if got := Version(); got != devVersion {
		t.Errorf("Version() = %q, want %q for dev build", got, devVersion)
	}
}

func TestGetIncludesBranch(t *testing.T) {
	origV, origB := version, branch

	t.Cleanup(func() { version, branch = origV, origB })

	version = "v0.3.2"
	branch = "feat/x"

	if got := Get(); got != "v0.3.2 (feat/x)" {
		t.Errorf("Get() = %q, want %q", got, "v0.3.2 (feat/x)")
	}

	branch = ""

	if got := Get(); got != "v0.3.2" {
		t.Errorf("Get() = %q, want %q", got, "v0.3.2")
	}
}
