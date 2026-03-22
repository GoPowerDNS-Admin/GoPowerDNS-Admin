package zonetag

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/config"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/db/models"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/powerdns"
)

// captureViews is a minimal Fiber Views engine that captures the data passed to
// Render so tests can assert on handler output without a real template engine.
type captureViews struct {
	mu       sync.Mutex
	lastName string
	lastData any
}

func (v *captureViews) Load() error { return nil }

func (v *captureViews) Render(w io.Writer, name string, data any, _ ...string) error {
	v.mu.Lock()
	v.lastName = name
	v.lastData = data
	v.mu.Unlock()

	_, _ = io.WriteString(w, name)

	return nil
}

func newTestApp(t *testing.T) (*fiber.App, *captureViews) {
	t.Helper()

	views := &captureViews{}

	app := fiber.New(fiber.Config{
		Views:             views,
		PassLocalsToViews: true,
	})

	return app, views
}

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open in-memory sqlite: %v", err)
	}

	if err := db.AutoMigrate(&models.Tag{}, &models.ZoneTag{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	return db
}

func newTestService(t *testing.T, app *fiber.App) *Service {
	t.Helper()

	db := newTestDB(t)
	cfg := &config.Config{
		Webserver: config.Webserver{URL: "http://localhost", Port: 3000},
	}

	svc := &Service{cfg: cfg, db: db}

	// Register routes directly — bypasses permission middleware for unit tests.
	app.Get(PathList, svc.List)
	app.Get(PathEdit, svc.Edit)
	app.Post(PathEdit, svc.Update)

	return svc
}

func doGet(t *testing.T, app *fiber.App, path string) *http.Response {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, path, http.NoBody)

	resp, err := app.Test(req, fiber.TestConfig{Timeout: 10 * time.Second})
	if err != nil {
		t.Fatalf("app.Test GET %s: %v", path, err)
	}

	return resp
}

// TestList_PDNSClientNil checks that the list handler returns 500 when the
// PowerDNS client is not initialized.
func TestList_PDNSClientNil(t *testing.T) {
	powerdns.Engine.Client = nil

	app, _ := newTestApp(t)
	newTestService(t, app)

	resp := doGet(t, app, PathList)

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected 500 when PDNS client is nil, got %d", resp.StatusCode)
	}
}

// TestEdit_RendersForm checks that the edit handler returns 200 and passes all
// available tags to the template as AllTags.
func TestEdit_RendersForm(t *testing.T) {
	app, views := newTestApp(t)
	svc := newTestService(t, app)

	svc.db.Create(&models.Tag{Name: "production"})
	svc.db.Create(&models.Tag{Name: "staging"})

	resp := doGet(t, app, "/admin/zone-tag/example.com./edit")

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	views.mu.Lock()
	data := views.lastData
	views.mu.Unlock()

	m, ok := data.(fiber.Map)
	if !ok {
		t.Fatal("expected fiber.Map data passed to template")
	}

	tags, ok := m["AllTags"].([]models.Tag)
	if !ok {
		t.Fatal("expected AllTags to be []models.Tag")
	}

	tagNames := make(map[string]bool, len(tags))
	for _, tag := range tags {
		tagNames[tag.Name] = true
	}

	for _, want := range []string{"production", "staging"} {
		if !tagNames[want] {
			t.Errorf("expected tag %q in AllTags, got %v", want, tags)
		}
	}

	if _, ok := m["AssignedSet"]; !ok {
		t.Error("expected AssignedSet to be present in template data")
	}
}

// TestEdit_ZoneNameInData checks that the zone name from the URL is passed to
// the template.
func TestEdit_ZoneNameInData(t *testing.T) {
	app, views := newTestApp(t)
	newTestService(t, app)

	resp := doGet(t, app, "/admin/zone-tag/example.com./edit")

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	views.mu.Lock()
	data := views.lastData
	views.mu.Unlock()

	m, ok := data.(fiber.Map)
	if !ok {
		t.Fatal("expected fiber.Map data")
	}

	if m["ZoneName"] != "example.com." {
		t.Errorf("expected ZoneName %q, got %v", "example.com.", m["ZoneName"])
	}
}

// TestParseTagIDs_Valid checks that valid numeric IDs are parsed correctly.
func TestParseTagIDs_Valid(t *testing.T) {
	app := fiber.New()

	var got []uint

	app.Post("/test", func(c fiber.Ctx) error {
		got = parseTagIDs(c)
		return c.SendStatus(fiber.StatusOK)
	})

	body := "tag_ids=1&tag_ids=2&tag_ids=42"
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := app.Test(req, fiber.TestConfig{Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}

	defer func() { _ = resp.Body.Close() }()

	want := []uint{1, 2, 42}
	if len(got) != len(want) {
		t.Fatalf("expected %v, got %v", want, got)
	}

	for i := range want {
		if got[i] != want[i] {
			t.Errorf("index %d: expected %d, got %d", i, want[i], got[i])
		}
	}
}

// TestParseTagIDs_SkipsInvalidAndZero checks that non-numeric and zero values
// are silently dropped.
func TestParseTagIDs_SkipsInvalidAndZero(t *testing.T) {
	app := fiber.New()

	var got []uint

	app.Post("/test", func(c fiber.Ctx) error {
		got = parseTagIDs(c)
		return c.SendStatus(fiber.StatusOK)
	})

	body := "tag_ids=0&tag_ids=abc&tag_ids=5&tag_ids=-1"
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := app.Test(req, fiber.TestConfig{Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}

	defer func() { _ = resp.Body.Close() }()

	if len(got) != 1 || got[0] != 5 {
		t.Errorf("expected [5], got %v", got)
	}
}

// TestParseTagIDs_Empty checks that an empty submission returns an empty slice.
func TestParseTagIDs_Empty(t *testing.T) {
	app := fiber.New()

	var got []uint

	app.Post("/test", func(c fiber.Ctx) error {
		got = parseTagIDs(c)
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(""))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := app.Test(req, fiber.TestConfig{Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}

	defer func() { _ = resp.Body.Close() }()

	if len(got) != 0 {
		t.Errorf("expected empty slice, got %v", got)
	}
}
