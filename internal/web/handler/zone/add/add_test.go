package zoneadd

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/config"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/powerdns"
)

// noOpViews is a minimal Fiber Views engine that writes the template name or
// the "Error" field from the fiber.Map so tests can assert on rendered errors.
type noOpViews struct{}

func (noOpViews) Load() error { return nil }

func (noOpViews) Render(w io.Writer, name string, data interface{}, _ ...string) error {
	if m, ok := data.(fiber.Map); ok {
		if v, exists := m["Error"]; exists && v != nil {
			switch e := v.(type) {
			case string:
				_, _ = io.WriteString(w, e)
			case []string:
				_, _ = io.WriteString(w, strings.Join(e, "\n"))
			}

			return nil
		}
	}

	_, _ = io.WriteString(w, name)

	return nil
}

func newTestApp() *fiber.App {
	return fiber.New(fiber.Config{Views: noOpViews{}})
}

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open in-memory sqlite: %v", err)
	}

	return db
}

func newTestConfig() *config.Config {
	return &config.Config{
		Webserver: config.Webserver{
			URL:     "http://localhost",
			Port:    3000,
			Session: config.Session{ExpiryTime: time.Minute},
		},
	}
}

func newTestService(t *testing.T, app *fiber.App) {
	t.Helper()

	svc := &Service{
		cfg:       newTestConfig(),
		db:        newTestDB(t),
		validator: validator.New(),
	}

	// Register routes directly — bypasses permission middleware for unit tests.
	app.Get(Path, svc.Get)
	app.Post(Path, svc.Post)
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

func doPost(t *testing.T, app *fiber.App, form url.Values) *http.Response {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, Path, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := app.Test(req, fiber.TestConfig{Timeout: 10 * time.Second})
	if err != nil {
		t.Fatalf("app.Test POST %s: %v", Path, err)
	}

	return resp
}

// TestGet_Returns200 checks that the GET handler returns a 200 response.
func TestGet_Returns200(t *testing.T) {
	app := newTestApp()
	newTestService(t, app)

	resp := doGet(t, app, Path)

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

// TestPost_PDNSNotInitialized checks that the POST handler returns 500 when
// the PowerDNS client is not initialized.
func TestPost_PDNSNotInitialized(t *testing.T) {
	powerdns.Engine.Client = nil

	app := newTestApp()
	newTestService(t, app)

	form := url.Values{
		"zone_type":    {"forward"},
		"name":         {"example.com"},
		"kind":         {"Native"},
		"soa_edit_api": {"DEFAULT"},
	}

	resp := doPost(t, app, form)

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected 500 when PDNS client is nil, got %d", resp.StatusCode)
	}
}

// TestPost_ValidationError_InvalidKind checks that an invalid zone kind returns
// 400 and includes the validation error in the response body.
func TestPost_ValidationError_InvalidKind(t *testing.T) {
	app := newTestApp()
	newTestService(t, app)

	form := url.Values{
		"zone_type":    {"forward"},
		"name":         {"example.com"},
		"kind":         {"InvalidKind"},
		"soa_edit_api": {"DEFAULT"},
	}

	resp := doPost(t, app, form)

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid kind, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()

	if !strings.Contains(string(body), "Kind") {
		t.Errorf("expected validation error mentioning 'Kind', got %q", string(body))
	}
}

// TestPost_InvalidIPv4CIDR checks that an invalid reverse IPv4 CIDR returns 400.
func TestPost_InvalidIPv4CIDR(t *testing.T) {
	app := newTestApp()
	newTestService(t, app)

	form := url.Values{
		"zone_type":       {"reverse-ipv4"},
		"reverse_network": {"not-a-cidr"},
		"kind":            {"Native"},
		"soa_edit_api":    {"DEFAULT"},
	}

	resp := doPost(t, app, form)

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid CIDR, got %d", resp.StatusCode)
	}
}

// TestPost_InvalidIPv6CIDR checks that an invalid reverse IPv6 CIDR returns 400.
func TestPost_InvalidIPv6CIDR(t *testing.T) {
	app := newTestApp()
	newTestService(t, app)

	form := url.Values{
		"zone_type":       {"reverse-ipv6"},
		"reverse_network": {"not-a-cidr"},
		"kind":            {"Native"},
		"soa_edit_api":    {"DEFAULT"},
	}

	resp := doPost(t, app, form)

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid IPv6 CIDR, got %d", resp.StatusCode)
	}
}
