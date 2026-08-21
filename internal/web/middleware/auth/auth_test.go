package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"

	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/db/models"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web/handler/login"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web/session"
)

// testStorage is a minimal in-memory session storage.
type testStorage struct {
	mu   sync.RWMutex
	data map[string][]byte
}

func (s *testStorage) Get(key string) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	v, ok := s.data[key]
	if !ok {
		return nil, nil
	}

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

func initStore() {
	session.Init(&testStorage{data: make(map[string][]byte)})
}

func newDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}

	if err := db.AutoMigrate(&models.Role{}, &models.User{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	return db
}

func createUser(t *testing.T, db *gorm.DB, username string, active bool) models.User {
	t.Helper()

	role := models.Role{Name: "viewer-" + username}
	if err := db.Create(&role).Error; err != nil {
		t.Fatalf("create role: %v", err)
	}

	u := models.User{
		Username:   username,
		Email:      username + "@example.com",
		AuthSource: models.AuthSourceLocal,
		Active:     active,
		RoleID:     role.ID,
	}
	if err := db.Create(&u).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	return u
}

func writeSession(t *testing.T, user *models.User) string {
	t.Helper()

	sid, err := session.GenerateSessionID()
	if err != nil {
		t.Fatalf("gen session id: %v", err)
	}

	data := &session.Data{User: *user}
	if err := data.Write(sid, time.Hour); err != nil {
		t.Fatalf("write session: %v", err)
	}

	return sid
}

func newApp(db *gorm.DB) *fiber.App {
	app := fiber.New()
	app.Use(New(db))
	app.Get("/dashboard", func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	return app
}

func doGet(t *testing.T, app *fiber.App, path, sessionID string) *http.Response {
	t.Helper()

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, path, http.NoBody)
	if sessionID != "" {
		req.AddCookie(&http.Cookie{Name: "session", Value: sessionID})
	}

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test %s: %v", path, err)
	}

	return resp
}

func clearsSessionCookie(resp *http.Response) bool {
	for _, ck := range resp.Cookies() {
		if ck.Name == "session" && ck.MaxAge < 0 {
			return true
		}
	}

	return false
}

func TestMiddleware_ActiveUser_Allowed(t *testing.T) {
	initStore()

	db := newDB(t)
	user := createUser(t, db, "alice", true)
	sid := writeSession(t, &user)

	resp := doGet(t, newApp(db), "/dashboard", sid)

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for active user, got %d", resp.StatusCode)
	}
}

func TestMiddleware_DeletedUser_ForcedToLogin(t *testing.T) {
	initStore()

	db := newDB(t)

	// A session referencing a user id that does not exist in the DB.
	ghost := models.User{ID: 9999, Username: "ghost", Active: true}
	sid := writeSession(t, &ghost)

	resp := doGet(t, newApp(db), "/dashboard", sid)

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("expected redirect (303) for deleted user, got %d", resp.StatusCode)
	}

	if loc := resp.Header.Get("Location"); loc != login.Path {
		t.Fatalf("expected redirect to %s, got %q", login.Path, loc)
	}

	if !clearsSessionCookie(resp) {
		t.Fatalf("expected the session cookie to be expired")
	}
}

func TestMiddleware_InactiveUser_ForcedToLogin(t *testing.T) {
	initStore()

	db := newDB(t)
	user := createUser(t, db, "bob", false) // deactivated
	sid := writeSession(t, &user)

	resp := doGet(t, newApp(db), "/dashboard", sid)

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("expected redirect (303) for inactive user, got %d", resp.StatusCode)
	}

	if loc := resp.Header.Get("Location"); loc != login.Path {
		t.Fatalf("expected redirect to %s, got %q", login.Path, loc)
	}

	if !clearsSessionCookie(resp) {
		t.Fatalf("expected the session cookie to be expired")
	}
}

func TestMiddleware_NoSession_RedirectsToLogin(t *testing.T) {
	initStore()

	db := newDB(t)

	resp := doGet(t, newApp(db), "/dashboard", "")

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("expected redirect (303) with no session, got %d", resp.StatusCode)
	}

	if loc := resp.Header.Get("Location"); loc != login.Path {
		t.Fatalf("expected redirect to %s, got %q", login.Path, loc)
	}
}

func TestMiddleware_DeletedUserOnLoginPage_NoRedirectLoop(t *testing.T) {
	initStore()

	db := newDB(t)

	ghost := models.User{ID: 9999, Username: "ghost", Active: true}
	sid := writeSession(t, &ghost)

	// On the login page itself, a stale session must not cause a redirect loop.
	resp := doGet(t, newApp(db), login.Path, sid)

	defer func() { _ = resp.Body.Close() }()

	// The login route is not registered on this test app, so reaching it yields 404
	// (i.e. the middleware called Next rather than redirecting).
	if resp.StatusCode == http.StatusSeeOther {
		t.Fatalf("did not expect a redirect on the login page, got %d", resp.StatusCode)
	}

	if !clearsSessionCookie(resp) {
		t.Fatalf("expected the stale session cookie to be expired on the login page")
	}
}
