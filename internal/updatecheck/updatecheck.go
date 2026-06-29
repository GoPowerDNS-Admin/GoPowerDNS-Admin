// Package updatecheck periodically checks GitHub for newer GoPowerDNS-Admin
// releases and exposes a thread-safe snapshot used to render a "new version
// available" hint in the UI. It fails soft: network or parse errors are logged
// and leave the previous result intact, never affecting request handling.
package updatecheck

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"golang.org/x/mod/semver"

	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/config"
)

const (
	defaultInterval   = 24 * time.Hour
	minInterval       = 1 * time.Hour
	defaultRepository = "GoPowerDNS-Admin/GoPowerDNS-Admin"
	requestTimeout    = 15 * time.Second
	githubAPIBase     = "https://api.github.com"
)

// Info is an immutable snapshot of the latest check result, safe to copy.
type Info struct {
	CurrentVersion  string
	LatestVersion   string
	UpdateAvailable bool
	ReleaseURL      string
}

// Checker periodically queries the GitHub releases API and caches the result.
type Checker struct {
	enabled    bool
	interval   time.Duration
	repository string
	current    string
	apiBase    string
	client     *http.Client

	mu   sync.RWMutex
	info Info
}

// New builds a Checker from config and the current build version. Interval and
// repository fall back to sane defaults; an interval below one hour is raised
// to avoid hammering the GitHub API (unauthenticated: 60 requests/hour/IP).
func New(cfg config.Update, current string) *Checker {
	interval := cfg.Interval
	if interval < minInterval {
		interval = defaultInterval
	}

	repository := cfg.Repository
	if repository == "" {
		repository = defaultRepository
	}

	return &Checker{
		enabled:    cfg.Enabled,
		interval:   interval,
		repository: repository,
		current:    current,
		apiBase:    githubAPIBase,
		client:     &http.Client{Timeout: requestTimeout},
		info:       Info{CurrentVersion: current},
	}
}

// Info returns the latest snapshot. Safe for concurrent use (per request).
func (c *Checker) Info() Info {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.info
}

// Run performs an initial check and then re-checks at the configured interval
// until ctx is canceled. It returns immediately when checking is disabled or
// the current build is not a released version (e.g. a dev or commit build),
// since there is nothing meaningful to compare against.
func (c *Checker) Run(ctx context.Context) {
	if !c.enabled {
		log.Debug().Msg("updatecheck: disabled by config")
		return
	}

	if !semver.IsValid(c.current) {
		log.Debug().Str("version", c.current).Msg("updatecheck: current build is not a released version; skipping")
		return
	}

	c.checkOnce(ctx)

	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.checkOnce(ctx)
		}
	}
}

// githubRelease is the subset of the GitHub release payload we consume.
type githubRelease struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
}

// checkOnce queries the latest release and updates the cached Info. Errors are
// logged and the previous snapshot is kept.
func (c *Checker) checkOnce(ctx context.Context) {
	latest, err := c.fetchLatest(ctx)
	if err != nil {
		log.Debug().Err(err).Str("repository", c.repository).Msg("updatecheck: failed to fetch latest release")
		return
	}

	available := semver.IsValid(c.current) && semver.IsValid(latest.TagName) &&
		semver.Compare(latest.TagName, c.current) > 0

	c.mu.Lock()
	c.info = Info{
		CurrentVersion:  c.current,
		LatestVersion:   latest.TagName,
		UpdateAvailable: available,
		ReleaseURL:      latest.HTMLURL,
	}
	c.mu.Unlock()

	if available {
		log.Info().Str("current", c.current).Str("latest", latest.TagName).
			Msg("updatecheck: a newer version is available")
	}
}

func (c *Checker) fetchLatest(ctx context.Context) (githubRelease, error) {
	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()

	url := c.apiBase + "/repos/" + c.repository + "/releases/latest"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return githubRelease{}, err
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-Github-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent", "GoPowerDNS-Admin/"+c.current)

	resp, err := c.client.Do(req)
	if err != nil {
		return githubRelease{}, err
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return githubRelease{}, &httpStatusError{status: resp.StatusCode}
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return githubRelease{}, err
	}

	return release, nil
}

type httpStatusError struct {
	status int
}

func (e *httpStatusError) Error() string {
	return "unexpected GitHub API status: " + http.StatusText(e.status)
}
