package health

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open in-memory sqlite: %v", err)
	}

	return db
}

func newTestApp(db *gorm.DB, alive *atomic.Bool) *fiber.App {
	app := fiber.New()
	New(db, alive).Register(app)

	return app
}

func doGet(t *testing.T, app *fiber.App) *http.Response {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, Path, http.NoBody)

	resp, err := app.Test(req, fiber.TestConfig{Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}

	return resp
}

func decodeStatus(t *testing.T, resp *http.Response) status {
	t.Helper()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}

	var s status
	if err := json.Unmarshal(body, &s); err != nil {
		t.Fatalf("decode JSON: %v\nbody: %s", err, body)
	}

	return s
}

// TestCheck_Healthy verifies that the endpoint returns 200 when alive and DB is up.
func TestCheck_Healthy(t *testing.T) {
	var alive atomic.Bool
	alive.Store(true)

	app := newTestApp(newTestDB(t), &alive)
	resp := doGet(t, app)

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	s := decodeStatus(t, resp)

	if s.Status != "ok" {
		t.Errorf("expected status ok, got %q", s.Status)
	}

	if s.Checks["alive"] != "ok" {
		t.Errorf("expected alive=ok, got %q", s.Checks["alive"])
	}

	if s.Checks["database"] != "ok" {
		t.Errorf("expected database=ok, got %q", s.Checks["database"])
	}
}

// TestCheck_ShuttingDown verifies that the endpoint returns 503 when alive is false.
func TestCheck_ShuttingDown(t *testing.T) {
	var alive atomic.Bool
	alive.Store(false)

	app := newTestApp(newTestDB(t), &alive)
	resp := doGet(t, app)

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", resp.StatusCode)
	}

	s := decodeStatus(t, resp)

	if s.Status != "degraded" {
		t.Errorf("expected status degraded, got %q", s.Status)
	}

	if s.Checks["alive"] != "shutting_down" {
		t.Errorf("expected alive=shutting_down, got %q", s.Checks["alive"])
	}
}

// TestCheck_PDNSNotConfigured verifies that missing PDNS client is reported but
// does not cause a non-200 status on its own.
func TestCheck_PDNSNotConfigured(t *testing.T) {
	var alive atomic.Bool
	alive.Store(true)

	app := newTestApp(newTestDB(t), &alive)
	resp := doGet(t, app)

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 when PDNS not configured, got %d", resp.StatusCode)
	}

	s := decodeStatus(t, resp)

	if s.Checks["powerdns"] != "not_configured" {
		t.Errorf("expected powerdns=not_configured, got %q", s.Checks["powerdns"])
	}
}
