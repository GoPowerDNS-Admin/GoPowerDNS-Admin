package user

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/glebarez/sqlite"
	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"

	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/config"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/db/models"
	websess "github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web/session"
)

// noOpViews renders "Error" or "error" fields from fiber.Map so tests can assert
// error messages. Falls back to writing the template name on success.
type noOpViews struct{}

func (noOpViews) Load() error { return nil }

func (noOpViews) Render(w io.Writer, name string, data interface{}, _ ...string) error {
	if m, ok := data.(fiber.Map); ok {
		for _, key := range []string{"Error", "error"} {
			if v, exists := m[key]; exists && v != nil {
				_, _ = fmt.Fprint(w, v)

				return nil
			}
		}
	}

	_, _ = io.WriteString(w, name)

	return nil
}

// testStorage is a minimal in-memory session storage for tests.
type testStorage struct {
	mu   sync.RWMutex
	data map[string][]byte
}

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
	websess.Init(&testStorage{data: make(map[string][]byte)})
}

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open in-memory sqlite: %v", err)
	}

	if err := db.AutoMigrate(&models.User{}, &models.Role{}, &models.Tag{}, &models.UserTag{}); err != nil {
		t.Fatalf("automigrate: %v", err)
	}

	return db
}

func newTestConfig() *config.Config {
	return &config.Config{
		Webserver: config.Webserver{
			Session: config.Session{ExpiryTime: time.Minute},
		},
	}
}

// newTestApp builds a Fiber app with the Service routes registered directly,
// without the permission middleware, so tests don't need a valid session.
func newTestApp(t *testing.T, db *gorm.DB) *fiber.App {
	t.Helper()

	app := fiber.New(fiber.Config{Views: noOpViews{}})
	cfg := newTestConfig()

	s := &Service{
		cfg:       cfg,
		db:        db,
		validator: validator.New(),
	}

	app.Get(Path, s.List)
	app.Get(Path+"/new", s.New)
	app.Post(Path, s.Create)
	app.Get(Path+"/:id/edit", s.Edit)
	app.Post(Path+"/:id", s.Update)
	app.Post(Path+"/:id/delete", s.Delete)
	app.Post(Path+"/:id/disable-totp", s.DisableTOTP)

	return app
}

// newSessionApp builds a Service + app for tests that need a real session
// (e.g. self-deactivation and self-delete checks).
func newSessionApp(t *testing.T, db *gorm.DB) (*Service, *fiber.App) {
	t.Helper()

	app := fiber.New(fiber.Config{Views: noOpViews{}})
	cfg := newTestConfig()

	s := &Service{
		cfg:       cfg,
		db:        db,
		validator: validator.New(),
	}

	app.Post(Path+"/:id", s.Update)
	app.Post(Path+"/:id/delete", s.Delete)

	return s, app
}

func createRole(t *testing.T, db *gorm.DB, name string) models.Role {
	t.Helper()

	role := models.Role{Name: name}
	if err := db.Create(&role).Error; err != nil {
		t.Fatalf("create role %q: %v", name, err)
	}

	return role
}

func createUser(t *testing.T, db *gorm.DB, username string, roleID uint, opts ...func(*models.User)) models.User {
	t.Helper()

	u := models.User{
		Username:   username,
		Email:      username + "@example.com",
		AuthSource: models.AuthSourceLocal,
		Active:     true,
		RoleID:     roleID,
	}

	for _, opt := range opts {
		opt(&u)
	}

	if err := db.Create(&u).Error; err != nil {
		t.Fatalf("create user %q: %v", username, err)
	}

	return u
}

func writeSession(t *testing.T, cfg *config.Config, u *models.User) string {
	t.Helper()

	sid := "test-session-" + u.Username
	sessData := &websess.Data{User: *u}

	if err := sessData.Write(sid, cfg.Webserver.Session.ExpiryTime); err != nil {
		t.Fatalf("write session: %v", err)
	}

	return sid
}

func doGet(t *testing.T, app *fiber.App, path string) *http.Response {
	t.Helper()

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, path, http.NoBody)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test GET %s: %v", path, err)
	}

	return resp
}

func doPost(t *testing.T, app *fiber.App, path string, form url.Values, cookies ...http.Cookie) *http.Response {
	t.Helper()

	var body io.Reader
	if form != nil {
		body = strings.NewReader(form.Encode())
	}

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, path, body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	for i := range cookies {
		req.AddCookie(&cookies[i])
	}

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test POST %s: %v", path, err)
	}

	return resp
}

func roleID(r *models.Role) string {
	return strconv.FormatUint(uint64(r.ID), 10)
}

// --- List ---

func TestList_ReturnsOK(t *testing.T) {
	db := newTestDB(t)

	initSessionStore()

	app := newTestApp(t, db)

	resp := doGet(t, app, Path)

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

// --- New ---

func TestNew_ReturnsOK(t *testing.T) {
	db := newTestDB(t)

	initSessionStore()

	app := newTestApp(t, db)

	resp := doGet(t, app, Path+"/new")

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

// --- Create ---

func TestCreate_Success(t *testing.T) {
	db := newTestDB(t)

	initSessionStore()

	role := createRole(t, db, "user")
	app := newTestApp(t, db)

	form := url.Values{
		"username": {"alice"},
		"email":    {"alice@example.com"},
		"source":   {"local"},
		"password": {"secret123"},
		"active":   {"true"},
		"role_id":  {roleID(&role)},
	}

	resp := doPost(t, app, Path, form)

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("expected 303 redirect, got %d", resp.StatusCode)
	}

	var u models.User
	if err := db.Where("username = ?", "alice").First(&u).Error; err != nil {
		t.Fatalf("user alice not found in db: %v", err)
	}
}

func TestCreate_MissingRequiredFields_ReturnsBadRequest(t *testing.T) {
	db := newTestDB(t)

	initSessionStore()

	app := newTestApp(t, db)

	form := url.Values{
		"username": {""},
		"email":    {"bad@example.com"},
		"source":   {"local"},
	}

	resp := doPost(t, app, Path, form)

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestCreate_OIDCUser_Succeeds(t *testing.T) {
	db := newTestDB(t)

	initSessionStore()

	role := createRole(t, db, "user")
	app := newTestApp(t, db)

	form := url.Values{
		"username": {"oidcuser"},
		"email":    {"oidcuser@example.com"},
		"source":   {"oidc"},
		"role_id":  {roleID(&role)},
	}

	resp := doPost(t, app, Path, form)

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("expected 303 redirect, got %d", resp.StatusCode)
	}

	var u models.User
	if err := db.Where("username = ?", "oidcuser").First(&u).Error; err != nil {
		t.Fatalf("oidcuser not found in db: %v", err)
	}
}

// --- Edit ---

func TestEdit_ExistingUser_ReturnsOK(t *testing.T) {
	db := newTestDB(t)

	initSessionStore()

	role := createRole(t, db, "user")
	u := createUser(t, db, "bob", role.ID)
	app := newTestApp(t, db)

	resp := doGet(t, app, fmt.Sprintf("%s/%d/edit", Path, u.ID))

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestEdit_NonExistentUser_Redirects(t *testing.T) {
	db := newTestDB(t)

	initSessionStore()

	app := newTestApp(t, db)

	resp := doGet(t, app, Path+"/9999/edit")

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("expected 303 redirect, got %d", resp.StatusCode)
	}
}

func TestEdit_InvalidID_Redirects(t *testing.T) {
	db := newTestDB(t)

	initSessionStore()

	app := newTestApp(t, db)

	resp := doGet(t, app, Path+"/abc/edit")

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("expected 303 redirect, got %d", resp.StatusCode)
	}
}

// --- Update ---

func TestUpdate_Success_StaysOnEditPage(t *testing.T) {
	db := newTestDB(t)

	initSessionStore()

	role := createRole(t, db, "user")
	u := createUser(t, db, "carol", role.ID)
	app := newTestApp(t, db)

	form := url.Values{
		"username":    {"carol-updated"},
		"email":       {"carol@example.com"},
		"source":      {"local"},
		"active":      {"true"},
		"role_id":     {roleID(&role)},
		"displayname": {"Carol"},
	}

	resp := doPost(t, app, fmt.Sprintf("%s/%d", Path, u.ID), form)

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("expected 303 redirect, got %d", resp.StatusCode)
	}

	loc := resp.Header.Get("Location")
	expected := fmt.Sprintf("%s/%d/edit", Path, u.ID)

	if loc != expected {
		t.Fatalf("expected redirect to %s, got %s", expected, loc)
	}

	var updated models.User
	db.First(&updated, u.ID)

	if updated.Username != "carol-updated" {
		t.Fatalf("expected username carol-updated, got %s", updated.Username)
	}
}

func TestUpdate_PreventsSelfDeactivation(t *testing.T) {
	db := newTestDB(t)

	initSessionStore()

	role := createRole(t, db, "user")
	u := createUser(t, db, "dave", role.ID)
	s, app := newSessionApp(t, db)
	sid := writeSession(t, s.cfg, &u)

	form := url.Values{
		"username": {"dave"},
		"email":    {"dave@example.com"},
		"source":   {"local"},
		"active":   {"false"},
		"role_id":  {roleID(&role)},
	}

	resp := doPost(t, app, fmt.Sprintf("%s/%d", Path, u.ID), form, http.Cookie{Name: "session", Value: sid})

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "deactivate") {
		t.Fatalf("expected deactivation error in body, got %q", string(body))
	}
}

func TestUpdate_PreventsLastAdminDemotion(t *testing.T) {
	db := newTestDB(t)

	initSessionStore()

	adminRole := createRole(t, db, "admin")
	userRole := createRole(t, db, "user")
	admin := createUser(t, db, "onlyadmin", adminRole.ID)
	app := newTestApp(t, db)

	form := url.Values{
		"username": {"onlyadmin"},
		"email":    {"onlyadmin@example.com"},
		"source":   {"local"},
		"active":   {"true"},
		"role_id":  {roleID(&userRole)},
	}

	resp := doPost(t, app, fmt.Sprintf("%s/%d", Path, admin.ID), form)

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestUpdate_SecondAdminAllowsDemotion(t *testing.T) {
	db := newTestDB(t)

	initSessionStore()

	adminRole := createRole(t, db, "admin")
	userRole := createRole(t, db, "user")
	admin1 := createUser(t, db, "admin1", adminRole.ID)
	createUser(t, db, "admin2", adminRole.ID)
	app := newTestApp(t, db)

	form := url.Values{
		"username": {"admin1"},
		"email":    {"admin1@example.com"},
		"source":   {"local"},
		"active":   {"true"},
		"role_id":  {roleID(&userRole)},
	}

	resp := doPost(t, app, fmt.Sprintf("%s/%d", Path, admin1.ID), form)

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("expected 303 (demotion allowed with 2 admins), got %d", resp.StatusCode)
	}
}

// --- Delete ---

func TestDelete_Success(t *testing.T) {
	db := newTestDB(t)

	initSessionStore()

	role := createRole(t, db, "user")
	u := createUser(t, db, "eve", role.ID)
	app := newTestApp(t, db)

	resp := doPost(t, app, fmt.Sprintf("%s/%d/delete", Path, u.ID), nil)

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("expected 303 redirect, got %d", resp.StatusCode)
	}

	var count int64
	db.Model(&models.User{}).Where("id = ?", u.ID).Count(&count)

	if count != 0 {
		t.Fatal("expected user to be gone from normal queries after delete")
	}
}

func TestDelete_PreventsSelfDelete(t *testing.T) {
	db := newTestDB(t)

	initSessionStore()

	role := createRole(t, db, "user")
	u := createUser(t, db, "frank", role.ID)
	s, app := newSessionApp(t, db)
	sid := writeSession(t, s.cfg, &u)

	resp := doPost(t, app, fmt.Sprintf("%s/%d/delete", Path, u.ID), nil, http.Cookie{Name: "session", Value: sid})

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestDelete_PreventsAdminRoleDelete(t *testing.T) {
	db := newTestDB(t)

	initSessionStore()

	adminRole := createRole(t, db, "admin")
	admin := createUser(t, db, "superadmin", adminRole.ID)
	app := newTestApp(t, db)

	resp := doPost(t, app, fmt.Sprintf("%s/%d/delete", Path, admin.ID), nil)

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", resp.StatusCode)
	}
}

// --- DisableTOTP ---

func TestDisableTOTP_Success(t *testing.T) {
	db := newTestDB(t)

	initSessionStore()

	role := createRole(t, db, "user")
	u := createUser(t, db, "grace", role.ID, func(u *models.User) {
		u.TOTPEnabled = true
		u.TOTPSecret = "SOMESECRET"
	})
	app := newTestApp(t, db)

	resp := doPost(t, app, fmt.Sprintf("%s/%d/disable-totp", Path, u.ID), nil)

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("expected 303 redirect, got %d", resp.StatusCode)
	}

	loc := resp.Header.Get("Location")
	expected := fmt.Sprintf("%s/%d/edit", Path, u.ID)

	if loc != expected {
		t.Fatalf("expected redirect to edit page %s, got %s", expected, loc)
	}

	var updated models.User
	db.First(&updated, u.ID)

	if updated.TOTPEnabled {
		t.Fatal("expected TOTPEnabled=false after disable")
	}

	if updated.TOTPSecret != "" {
		t.Fatal("expected TOTPSecret cleared after disable")
	}
}

func TestDisableTOTP_BlockedWhenRequired(t *testing.T) {
	db := newTestDB(t)

	initSessionStore()

	role := createRole(t, db, "user")
	u := createUser(t, db, "henry", role.ID, func(u *models.User) {
		u.TOTPEnabled = true
		u.TOTPSecret = "SOMESECRET"
		u.TOTPRequired = true
	})
	app := newTestApp(t, db)

	resp := doPost(t, app, fmt.Sprintf("%s/%d/disable-totp", Path, u.ID), nil)

	defer func() { _ = resp.Body.Close() }()

	// Handler redirects back to edit page — TOTP must not have been cleared.
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("expected 303 redirect, got %d", resp.StatusCode)
	}

	var updated models.User
	db.First(&updated, u.ID)

	if !updated.TOTPEnabled {
		t.Fatal("TOTPEnabled must remain true when TOTP is required")
	}

	if updated.TOTPSecret == "" {
		t.Fatal("TOTPSecret must not be cleared when TOTP is required")
	}
}

func TestDisableTOTP_NoopForNonLocalUser(t *testing.T) {
	db := newTestDB(t)

	initSessionStore()

	role := createRole(t, db, "user")
	u := createUser(t, db, "ivan", role.ID, func(u *models.User) {
		u.AuthSource = models.AuthSourceOIDC
		u.TOTPEnabled = true
	})
	app := newTestApp(t, db)

	resp := doPost(t, app, fmt.Sprintf("%s/%d/disable-totp", Path, u.ID), nil)

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("expected 303 redirect, got %d", resp.StatusCode)
	}
}

func TestDisableTOTP_NoopWhenNotEnabled(t *testing.T) {
	db := newTestDB(t)

	initSessionStore()

	role := createRole(t, db, "user")
	u := createUser(t, db, "julia", role.ID)
	app := newTestApp(t, db)

	resp := doPost(t, app, fmt.Sprintf("%s/%d/disable-totp", Path, u.ID), nil)

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("expected 303 redirect, got %d", resp.StatusCode)
	}
}
