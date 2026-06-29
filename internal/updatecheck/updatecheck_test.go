package updatecheck

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/config"
)

// newTestChecker builds a Checker pointed at a stub server, bypassing New so
// tests can set an arbitrary current version and API base.
func newTestChecker(current, apiBase string) *Checker {
	return &Checker{
		enabled:    true,
		interval:   defaultInterval,
		repository: defaultRepository,
		current:    current,
		apiBase:    apiBase,
		client:     &http.Client{Timeout: requestTimeout},
		info:       Info{CurrentVersion: current},
	}
}

func stubServer(t *testing.T, status int, body string) *httptest.Server {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)

	return srv
}

func TestCheckOnce(t *testing.T) {
	const release = `{"tag_name":"v0.9.0","html_url":"https://example.com/releases/v0.9.0"}`

	tests := []struct {
		name          string
		current       string
		status        int
		body          string
		wantAvailable bool
		wantLatest    string
	}{
		{"newer available", "v0.3.2", http.StatusOK, release, true, "v0.9.0"},
		{"up to date", "v0.9.0", http.StatusOK, release, false, "v0.9.0"},
		{"current is newer", "v1.0.0", http.StatusOK, release, false, "v0.9.0"},
		{"dev build never updates", "dev", http.StatusOK, release, false, "v0.9.0"},
		{"invalid latest tag", "v0.3.2", http.StatusOK, `{"tag_name":"garbage"}`, false, "garbage"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := stubServer(t, tt.status, tt.body)
			c := newTestChecker(tt.current, srv.URL)

			c.checkOnce(context.Background())
			got := c.Info()

			if got.UpdateAvailable != tt.wantAvailable {
				t.Errorf("UpdateAvailable = %v, want %v", got.UpdateAvailable, tt.wantAvailable)
			}

			if got.LatestVersion != tt.wantLatest {
				t.Errorf("LatestVersion = %q, want %q", got.LatestVersion, tt.wantLatest)
			}

			if got.CurrentVersion != tt.current {
				t.Errorf("CurrentVersion = %q, want %q", got.CurrentVersion, tt.current)
			}
		})
	}
}

func TestCheckOnceKeepsPreviousOnError(t *testing.T) {
	srv := stubServer(t, http.StatusInternalServerError, "boom")
	c := newTestChecker("v0.3.2", srv.URL)

	// Seed a prior good result.
	c.info = Info{CurrentVersion: "v0.3.2", LatestVersion: "v0.9.0", UpdateAvailable: true, ReleaseURL: "x"}

	c.checkOnce(context.Background())

	if got := c.Info(); !got.UpdateAvailable || got.LatestVersion != "v0.9.0" {
		t.Errorf("error response clobbered prior snapshot: %+v", got)
	}
}

func TestNewClampsIntervalAndDefaults(t *testing.T) {
	c := New(config.Update{Enabled: true, Interval: time.Second, Repository: ""}, "v0.3.2")

	if c.interval != defaultInterval {
		t.Errorf("interval = %v, want clamped to %v", c.interval, defaultInterval)
	}

	if c.repository != defaultRepository {
		t.Errorf("repository = %q, want default %q", c.repository, defaultRepository)
	}
}

func TestRunDisabledReturnsImmediately(t *testing.T) {
	c := New(config.Update{Enabled: false}, "v0.3.2")

	done := make(chan struct{})

	go func() { c.Run(context.Background()); close(done) }()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not return promptly when disabled")
	}
}

func TestRunDevBuildReturnsImmediately(t *testing.T) {
	c := New(config.Update{Enabled: true}, "dev")

	done := make(chan struct{})

	go func() { c.Run(context.Background()); close(done) }()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not return promptly on a dev build")
	}
}
