package login

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/storage"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/auth"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/config"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/db/models"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web/handler/dashboard"
	websess "github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web/session"
)

// noOpViews is a minimal Fiber Views engine used for tests.
// It writes the "error" field from the provided fiber.Map (if any)
// so tests can assert error messages rendered by handlers.
type noOpViews struct{}

func (noOpViews) Load() error { return nil }

func (noOpViews) Render(w io.Writer, name string, data interface{}, _ ...string) error {
	if m, ok := data.(fiber.Map); ok {
		if v, exists := m["error"]; exists && v != nil {
			_, _ = io.WriteString(w, v.(string))
			return nil
		}
	}
	// write template name to have some content
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
		t.Fatalf("failed to open sqlite in-memory db: %v", err)
	}

	if err := db.AutoMigrate(&models.User{}); err != nil {
		t.Fatalf("failed to migrate user model: %v", err)
	}

	return db
}

func newTestConfig() *config.Config {
	return &config.Config{
		DevMode: false,
		Webserver: config.Webserver{
			URL:     "http://localhost",
			Port:    3000,
			Session: config.Session{ExpiryTime: time.Minute},
		},
		Auth: config.Auth{
			LocalDB: config.LocalDBAuth{Enabled: true},
			OIDC:    config.OIDCAuth{Enabled: false},
			LDAP:    config.LDAPAuth{Enabled: false},
		},
	}
}

// testStorage is a minimal in-memory implementation of storage.Storage for tests.
type testStorage struct {
	mu   sync.RWMutex
	data map[string][]byte
}

// Ensure testStorage implements the storage.Storage interface.
// This also prevents the storage import from being marked as unused in some environments.
var _ storage.Storage = (*testStorage)(nil)

func (s *testStorage) Get(key string) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	v := s.data[key]
	out := make([]byte, len(v))
	copy(out, v)

	return out, nil
}

func (s *testStorage) Set(key string, val []byte, _ time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.data == nil {
		s.data = make(map[string][]byte)
	}

	buf := make([]byte, len(val))
	copy(buf, val)
	s.data[key] = buf

	return nil
}

func (s *testStorage) Delete(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.data, key)

	return nil
}

func (s *testStorage) Reset() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.data = make(map[string][]byte)

	return nil
}

func (s *testStorage) Close() error { return nil }

func initSessionStore() {
	// Initialize a fresh in-memory session store for each test.
	websess.Init(&testStorage{data: make(map[string][]byte)})
}

func TestPickAuthType_DefaultsAndErrors(t *testing.T) {
	db := newTestDB(t)
	cfg := newTestConfig()
	app := newTestApp()

	initSessionStore()

	var s Service
	s.Init(app, cfg, db)

	// No requested type, Local enabled → choose local
	at, err := s.pickAuthType("")
	if err != nil || at != "local" {
		t.Fatalf("expected local, got at=%q err=%v", at, err)
	}

	// Disable Local, enable LDAP but ldapAuth nil → default pick returns ldap when none requested
	s.cfg.Auth.LocalDB.Enabled = false
	s.cfg.Auth.LDAP.Enabled = true
	// Default pick chooses ldap if enabled regardless of provider presence
	if at, err = s.pickAuthType(""); err != nil || at != "ldap" {
		t.Fatalf("expected default pick ldap, got at=%q err=%v", at, err)
	}
	// When explicitly asking ldap with Enabled but ldapAuth == nil → ErrLDAPAuthDisabled
	if _, err = s.pickAuthType("ldap"); err == nil || !errors.Is(err, ErrLDAPAuthDisabled) {
		t.Fatalf("expected ErrLDAPAuthDisabled, got %v", err)
	}

	// Provide a non-nil ldapAuth and keep Enabled → selecting ldap should succeed
	s.ldapAuth = &auth.LDAPProvider{}
	if at, err = s.pickAuthType("ldap"); err != nil || at != "ldap" {
		t.Fatalf("expected ldap, got at=%q err=%v", at, err)
	}

	// Invalid method
	if _, errAuthType := s.pickAuthType("unknown"); errAuthType == nil || !errors.Is(errAuthType, ErrInvalidAuthMethod) {
		t.Fatalf("expected ErrInvalidAuthMethod, got %v", errAuthType)
	}
}

func TestAuthenticate_Local(t *testing.T) {
	db := newTestDB(t)
	cfg := newTestConfig()
	app := newTestApp()

	initSessionStore()

	var s Service
	s.Init(app, cfg, db)

	// Create a local user
	lp := auth.NewLocalProvider(db)

	user, err := lp.CreateUser("alice", "alice@example.com", "secret", "Alice", "Doe", 0)
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	if !user.Active {
		t.Fatalf("new user must be active by default")
	}

	// Success
	got, err := s.authenticate("local", "alice", "secret")
	if err != nil || got == nil || got.Username != "alice" {
		t.Fatalf("expected successful auth for alice, got user=%v err=%v", got, err)
	}

	// Wrong password
	got, err = s.authenticate("local", "alice", "wrong")
	if err == nil || !errors.Is(err, ErrInvalidCredentials) || got != nil {
		t.Fatalf("expected ErrInvalidCredentials, got user=%v err=%v", got, err)
	}

	// Invalid auth type
	if u, err := s.authenticate("bogus", "alice", "secret"); err == nil || !errors.Is(err, ErrInvalidAuthMethod) || u != nil {
		t.Fatalf("expected ErrInvalidAuthMethod, got user=%v err=%v", u, err)
	}
}

func performPost(t *testing.T, app *fiber.App, target string, form url.Values) *http.Response {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, target, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}

	return resp
}

func TestPost_Local_Success_SetsCookieAndRedirects(t *testing.T) {
	db := newTestDB(t)
	cfg := newTestConfig()
	cfg.DevMode = false // Secure cookie expected

	app := newTestApp()

	initSessionStore()

	var s Service
	s.Init(app, cfg, db)

	// Create user for local auth
	lp := auth.NewLocalProvider(db)
	if _, err := lp.CreateUser("bob", "bob@example.com", "s3cr3t", "Bob", "Doe", 0); err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	// Perform POST
	form := url.Values{
		"username":  {"bob"},
		"password":  {"s3cr3t"},
		"auth_type": {"local"},
	}
	resp := performPost(t, app, Path+"/", form)

	if resp.StatusCode != http.StatusFound {
		t.Fatalf("expected 302 Found, got %d", resp.StatusCode)
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	// Check redirect location
	if loc := resp.Header.Get("Location"); loc != dashboard.Path {
		t.Fatalf("expected redirect to %s, got %s", dashboard.Path, loc)
	}

	// Check cookie is set and Secure flag present
	setCookie := resp.Header.Get("Set-Cookie")
	if !strings.Contains(setCookie, "session=") {
		t.Fatalf("expected session cookie, got %q", setCookie)
	}

	if !strings.Contains(strings.ToLower(setCookie), "secure") {
		t.Fatalf("expected Secure flag on cookie when DevMode=false, got %q", setCookie)
	}
}

func TestPost_Local_Success_DevModeDisablesSecure(t *testing.T) {
	db := newTestDB(t)
	cfg := newTestConfig()
	cfg.DevMode = true // Secure=false expected

	app := newTestApp()

	initSessionStore()

	var s Service
	s.Init(app, cfg, db)

	lp := auth.NewLocalProvider(db)
	if _, err := lp.CreateUser("carol", "carol@example.com", "pass", "Carol", "Doe", 0); err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	form := url.Values{
		"username":  {"carol"},
		"password":  {"pass"},
		"auth_type": {"local"},
	}
	resp := performPost(t, app, Path+"/", form)

	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusFound {
		t.Fatalf("expected 302 Found, got %d", resp.StatusCode)
	}

	setCookie := resp.Header.Get("Set-Cookie")
	if strings.Contains(strings.ToLower(setCookie), "secure") {
		t.Fatalf("did not expect Secure flag when DevMode=true, got %q", setCookie)
	}
}

func TestPost_InvalidForm_RendersError(t *testing.T) {
	db := newTestDB(t)
	cfg := newTestConfig()
	app := newTestApp()

	initSessionStore()

	var s Service
	s.Init(app, cfg, db)

	// Malformed JSON to force BodyParser error
	req := httptest.NewRequest(http.MethodPost, Path+"/", strings.NewReader("{"))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK on render error page, got %d", resp.StatusCode)
	}
	// Body should contain the error message (noOpViews writes it)
	bodyBytes, _ := io.ReadAll(resp.Body)

	if !strings.Contains(string(bodyBytes), ErrInvalidFormData.Error()) {
		t.Fatalf("expected error message in body, got %q", string(bodyBytes))
	}
}

func TestPost_LocalDisabled_RendersError(t *testing.T) {
	db := newTestDB(t)
	cfg := newTestConfig()
	cfg.Auth.LocalDB.Enabled = false

	app := newTestApp()

	initSessionStore()

	var s Service
	s.Init(app, cfg, db)

	form := url.Values{
		"username":  {"dave"},
		"password":  {"whatever"},
		"auth_type": {"local"},
	}
	resp := performPost(t, app, Path+"/", form)

	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK on render error page, got %d", resp.StatusCode)
	}

	bodyBytes, _ := io.ReadAll(resp.Body)

	if !strings.Contains(string(bodyBytes), ErrLocalAuthDisabled.Error()) {
		t.Fatalf("expected local disabled error, got %q", string(bodyBytes))
	}
}
