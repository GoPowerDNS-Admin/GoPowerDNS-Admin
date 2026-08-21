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

	"github.com/glebarez/sqlite"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
	"github.com/onsi/gomega"
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

	if err := db.AutoMigrate(
		&models.User{},
		&models.Role{},
		&models.Tag{},
		&models.UserTag{},
		&models.Group{},
		&models.UserGroup{},
		&models.GroupMapping{},
	); err != nil {
		t.Fatalf("automigrate: %v", err)
	}

	return db
}

func createGroup(t *testing.T, db *gorm.DB, name string, source models.GroupSource) models.Group {
	t.Helper()

	g := models.Group{Name: name, ExternalID: name, Source: source}
	if err := db.Create(&g).Error; err != nil {
		t.Fatalf("create group %q: %v", name, err)
	}

	return g
}

func addUserToGroup(t *testing.T, db *gorm.DB, userID uint64, groupID uint) {
	t.Helper()

	if err := db.Create(&models.UserGroup{UserID: userID, GroupID: groupID}).Error; err != nil {
		t.Fatalf("add user %d to group %d: %v", userID, groupID, err)
	}
}

func mapGroupToRole(t *testing.T, db *gorm.DB, groupID, roleID uint) {
	t.Helper()

	if err := db.Create(&models.GroupMapping{GroupID: groupID, RoleID: roleID}).Error; err != nil {
		t.Fatalf("map group %d to role %d: %v", groupID, roleID, err)
	}
}

func newTestConfig() *config.Config {
	return &config.Config{
		Webserver: config.Webserver{
			Session: config.Session{ExpiryTime: time.Minute},
		},
	}
}

// newTestApp builds a Fiber app with Service routes registered directly,
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

// captureViews records the data map passed to Render so tests can assert on the
// exact values a handler hands to the template (e.g. GroupRoles).
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

func (v *captureViews) data() fiber.Map {
	v.mu.Lock()
	defer v.mu.Unlock()

	m, _ := v.lastData.(fiber.Map)

	return m
}

// newCaptureApp builds an app whose List route uses a capturing Views engine, so
// tests can inspect the render data map instead of only the HTTP status.
func newCaptureApp(t *testing.T, db *gorm.DB) (*fiber.App, *captureViews) {
	t.Helper()

	views := &captureViews{}
	app := fiber.New(fiber.Config{Views: views})

	s := &Service{
		cfg:       newTestConfig(),
		db:        db,
		validator: validator.New(),
	}

	app.Get(Path, s.List)

	return app, views
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
	g := gomega.NewWithT(t)
	db := newTestDB(t)

	initSessionStore()

	app := newTestApp(t, db)

	resp := doGet(t, app, Path)

	defer func() { _ = resp.Body.Close() }()

	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusOK))
}

// --- New ---

func TestNew_ReturnsOK(t *testing.T) {
	g := gomega.NewWithT(t)
	db := newTestDB(t)

	initSessionStore()

	app := newTestApp(t, db)

	resp := doGet(t, app, Path+"/new")

	defer func() { _ = resp.Body.Close() }()

	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusOK))
}

// --- Create ---

func TestCreate_Success(t *testing.T) {
	g := gomega.NewWithT(t)
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

	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusSeeOther))

	var u models.User
	g.Expect(db.Where("username = ?", "alice").First(&u).Error).To(gomega.Succeed())
}

func TestCreate_MissingRequiredFields_ReturnsBadRequest(t *testing.T) {
	g := gomega.NewWithT(t)
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

	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusBadRequest))
}

func TestCreate_OIDCUser_Succeeds(t *testing.T) {
	g := gomega.NewWithT(t)
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

	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusSeeOther))

	var u models.User
	g.Expect(db.Where("username = ?", "oidcuser").First(&u).Error).To(gomega.Succeed())
}

// --- Edit ---

func TestEdit_ExistingUser_ReturnsOK(t *testing.T) {
	g := gomega.NewWithT(t)
	db := newTestDB(t)

	initSessionStore()

	role := createRole(t, db, "user")
	u := createUser(t, db, "bob", role.ID)
	app := newTestApp(t, db)

	resp := doGet(t, app, fmt.Sprintf("%s/%d/edit", Path, u.ID))

	defer func() { _ = resp.Body.Close() }()

	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusOK))
}

func TestEdit_NonExistentUser_Redirects(t *testing.T) {
	g := gomega.NewWithT(t)
	db := newTestDB(t)

	initSessionStore()

	app := newTestApp(t, db)

	resp := doGet(t, app, Path+"/9999/edit")

	defer func() { _ = resp.Body.Close() }()

	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusSeeOther))
}

func TestEdit_InvalidID_Redirects(t *testing.T) {
	g := gomega.NewWithT(t)
	db := newTestDB(t)

	initSessionStore()

	app := newTestApp(t, db)

	resp := doGet(t, app, Path+"/abc/edit")

	defer func() { _ = resp.Body.Close() }()

	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusSeeOther))
}

// --- Update ---

func TestUpdate_Success_StaysOnEditPage(t *testing.T) {
	g := gomega.NewWithT(t)
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

	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusSeeOther))
	g.Expect(resp.Header.Get("Location")).To(gomega.Equal(fmt.Sprintf("%s/%d/edit", Path, u.ID)))

	var updated models.User
	db.First(&updated, u.ID)

	g.Expect(updated.Username).To(gomega.Equal("carol-updated"))
}

func TestUpdate_PreventsSelfDeactivation(t *testing.T) {
	g := gomega.NewWithT(t)
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

	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusBadRequest))

	body, _ := io.ReadAll(resp.Body)
	g.Expect(string(body)).To(gomega.ContainSubstring("deactivate"))
}

func TestUpdate_PreventsLastAdminDemotion(t *testing.T) {
	g := gomega.NewWithT(t)
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

	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusBadRequest))
}

func TestUpdate_SecondAdminAllowsDemotion(t *testing.T) {
	g := gomega.NewWithT(t)
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

	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusSeeOther))
}

// --- Delete ---

func TestDelete_Success(t *testing.T) {
	g := gomega.NewWithT(t)
	db := newTestDB(t)

	initSessionStore()

	role := createRole(t, db, "user")
	u := createUser(t, db, "eve", role.ID)
	app := newTestApp(t, db)

	resp := doPost(t, app, fmt.Sprintf("%s/%d/delete", Path, u.ID), nil)

	defer func() { _ = resp.Body.Close() }()

	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusSeeOther))

	var count int64
	db.Model(&models.User{}).Where("id = ?", u.ID).Count(&count)

	g.Expect(count).To(gomega.BeZero())
}

func TestDelete_PreventsSelfDelete(t *testing.T) {
	g := gomega.NewWithT(t)
	db := newTestDB(t)

	initSessionStore()

	role := createRole(t, db, "user")
	u := createUser(t, db, "frank", role.ID)
	s, app := newSessionApp(t, db)
	sid := writeSession(t, s.cfg, &u)

	resp := doPost(t, app, fmt.Sprintf("%s/%d/delete", Path, u.ID), nil, http.Cookie{Name: "session", Value: sid})

	defer func() { _ = resp.Body.Close() }()

	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusBadRequest))
}

func TestDelete_PreventsAdminRoleDelete(t *testing.T) {
	g := gomega.NewWithT(t)
	db := newTestDB(t)

	initSessionStore()

	adminRole := createRole(t, db, "admin")
	admin := createUser(t, db, "superadmin", adminRole.ID)
	app := newTestApp(t, db)

	resp := doPost(t, app, fmt.Sprintf("%s/%d/delete", Path, admin.ID), nil)

	defer func() { _ = resp.Body.Close() }()

	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusForbidden))
}

// --- DisableTOTP ---

func TestDisableTOTP_Success(t *testing.T) {
	g := gomega.NewWithT(t)
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

	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusSeeOther))
	g.Expect(resp.Header.Get("Location")).To(gomega.Equal(fmt.Sprintf("%s/%d/edit", Path, u.ID)))

	var updated models.User
	db.First(&updated, u.ID)

	g.Expect(updated.TOTPEnabled).To(gomega.BeFalse())
	g.Expect(updated.TOTPSecret).To(gomega.BeEmpty())
}

func TestDisableTOTP_BlockedWhenRequired(t *testing.T) {
	g := gomega.NewWithT(t)
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
	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusSeeOther))

	var updated models.User
	db.First(&updated, u.ID)

	g.Expect(updated.TOTPEnabled).To(gomega.BeTrue())
	g.Expect(updated.TOTPSecret).NotTo(gomega.BeEmpty())
}

func TestDisableTOTP_NoopForNonLocalUser(t *testing.T) {
	g := gomega.NewWithT(t)
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

	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusSeeOther))
}

func TestDisableTOTP_NoopWhenNotEnabled(t *testing.T) {
	g := gomega.NewWithT(t)
	db := newTestDB(t)

	initSessionStore()

	role := createRole(t, db, "user")
	u := createUser(t, db, "julia", role.ID)
	app := newTestApp(t, db)

	resp := doPost(t, app, fmt.Sprintf("%s/%d/disable-totp", Path, u.ID), nil)

	defer func() { _ = resp.Body.Close() }()

	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusSeeOther))
}

// --- loadGroupRoles ---

func TestLoadGroupRoles_EmptyUsers_ReturnsNil(t *testing.T) {
	db := newTestDB(t)

	result := loadGroupRoles(db, nil)

	if result != nil {
		t.Fatalf("expected nil for empty user slice, got %v", result)
	}
}

func TestLoadGroupRoles_UserWithNoGroups_ReturnsEmptyMap(t *testing.T) {
	db := newTestDB(t)

	viewerRole := createRole(t, db, "viewer")
	u := createUser(t, db, "nogroups", viewerRole.ID)

	// Preload the role so loadGroupRoles has directRoleByUserID populated correctly.
	if err := db.Preload("Role").First(&u, u.ID).Error; err != nil {
		t.Fatalf("preload: %v", err)
	}

	result := loadGroupRoles(db, []models.User{u})

	if len(result) != 0 {
		t.Fatalf("expected no group roles, got %v", result)
	}
}

func TestLoadGroupRoles_UserWithGroupRole_DifferentFromDirect(t *testing.T) {
	g := gomega.NewWithT(t)
	db := newTestDB(t)

	viewerRole := createRole(t, db, "viewer")
	adminRole := createRole(t, db, "admin")

	u := createUser(t, db, "ldapuser", viewerRole.ID, func(u *models.User) {
		u.AuthSource = models.AuthSourceLDAP
	})

	grp := createGroup(t, db, "cn=dns-admins,ou=groups,dc=example,dc=com", models.GroupSourceLDAP)
	addUserToGroup(t, db, u.ID, grp.ID)
	mapGroupToRole(t, db, grp.ID, adminRole.ID)

	if err := db.Preload("Role").First(&u, u.ID).Error; err != nil {
		t.Fatalf("preload: %v", err)
	}

	result := loadGroupRoles(db, []models.User{u})

	g.Expect(result[u.ID]).To(gomega.ConsistOf("admin"))
}

func TestLoadGroupRoles_UserWithGroupRoleSameAsDirect_NotDuplicated(t *testing.T) {
	db := newTestDB(t)

	adminRole := createRole(t, db, "admin")

	u := createUser(t, db, "localadmin", adminRole.ID)

	grp := createGroup(t, db, "admins", models.GroupSourceLocal)
	addUserToGroup(t, db, u.ID, grp.ID)
	mapGroupToRole(t, db, grp.ID, adminRole.ID)

	if err := db.Preload("Role").First(&u, u.ID).Error; err != nil {
		t.Fatalf("preload: %v", err)
	}

	result := loadGroupRoles(db, []models.User{u})

	// The group-inherited role is identical to the direct role — no badge needed.
	if len(result[u.ID]) != 0 {
		t.Fatalf("expected no extra group roles when group role matches direct role, got %v", result[u.ID])
	}
}

func TestLoadGroupRoles_MultipleUsers_IndependentResults(t *testing.T) {
	g := gomega.NewWithT(t)
	db := newTestDB(t)

	viewerRole := createRole(t, db, "viewer")
	adminRole := createRole(t, db, "admin")

	ldapUser := createUser(t, db, "ldapuser2", viewerRole.ID, func(u *models.User) {
		u.AuthSource = models.AuthSourceLDAP
	})
	localUser := createUser(t, db, "localuser", viewerRole.ID)

	grp := createGroup(t, db, "cn=dns-admins2,ou=groups,dc=example,dc=com", models.GroupSourceLDAP)
	addUserToGroup(t, db, ldapUser.ID, grp.ID)
	mapGroupToRole(t, db, grp.ID, adminRole.ID)

	var users []models.User
	if err := db.Preload("Role").Find(&users).Error; err != nil {
		t.Fatalf("preload: %v", err)
	}

	result := loadGroupRoles(db, users)

	g.Expect(result[ldapUser.ID]).To(gomega.ConsistOf("admin"))
	g.Expect(result[localUser.ID]).To(gomega.BeEmpty())
}

func TestLoadGroupRoles_DeduplicatesRolesFromMultipleGroups(t *testing.T) {
	g := gomega.NewWithT(t)
	db := newTestDB(t)

	viewerRole := createRole(t, db, "viewer")
	adminRole := createRole(t, db, "admin")

	u := createUser(t, db, "multigroup", viewerRole.ID, func(u *models.User) {
		u.AuthSource = models.AuthSourceLDAP
	})

	// Two separate LDAP groups both mapping to the same admin role.
	grp1 := createGroup(t, db, "cn=admins1,dc=example,dc=com", models.GroupSourceLDAP)
	grp2 := createGroup(t, db, "cn=admins2,dc=example,dc=com", models.GroupSourceLDAP)
	addUserToGroup(t, db, u.ID, grp1.ID)
	addUserToGroup(t, db, u.ID, grp2.ID)
	mapGroupToRole(t, db, grp1.ID, adminRole.ID)
	mapGroupToRole(t, db, grp2.ID, adminRole.ID)

	if err := db.Preload("Role").First(&u, u.ID).Error; err != nil {
		t.Fatalf("preload: %v", err)
	}

	result := loadGroupRoles(db, []models.User{u})

	// Should appear exactly once despite two groups granting it.
	g.Expect(result[u.ID]).To(gomega.ConsistOf("admin"))
}

// --- List with GroupRoles ---

// TestList_GroupRoles_PopulatedForLDAPUser verifies the List handler passes a
// GroupRoles map to the template containing the LDAP user's inherited admin role.
func TestList_GroupRoles_PopulatedForLDAPUser(t *testing.T) {
	g := gomega.NewWithT(t)
	db := newTestDB(t)

	initSessionStore()

	viewerRole := createRole(t, db, "viewer")
	adminRole := createRole(t, db, "admin")

	ldapUser := createUser(t, db, "testuser", viewerRole.ID, func(u *models.User) {
		u.AuthSource = models.AuthSourceLDAP
		u.ExternalID = "uid=testuser,ou=users,dc=example,dc=com"
	})

	grp := createGroup(t, db, "cn=dns-admins,ou=groups,dc=example,dc=com", models.GroupSourceLDAP)
	addUserToGroup(t, db, ldapUser.ID, grp.ID)
	mapGroupToRole(t, db, grp.ID, adminRole.ID)

	app, views := newCaptureApp(t, db)

	resp := doGet(t, app, Path)

	defer func() { _ = resp.Body.Close() }()

	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusOK))

	data := views.data()
	g.Expect(data).NotTo(gomega.BeNil())
	g.Expect(data).To(gomega.HaveKey("GroupRoles"))

	groupRoles, ok := data["GroupRoles"].(map[uint64][]string)
	g.Expect(ok).To(gomega.BeTrue(), "GroupRoles should be map[uint64][]string")
	g.Expect(groupRoles[ldapUser.ID]).To(gomega.ConsistOf("admin"))
}

func TestList_GroupRoles_NotShownWhenGroupRoleMatchesDirect(t *testing.T) {
	g := gomega.NewWithT(t)
	db := newTestDB(t)

	initSessionStore()

	adminRole := createRole(t, db, "admin")

	localAdmin := createUser(t, db, "localadmin2", adminRole.ID)

	grp := createGroup(t, db, "admins-local", models.GroupSourceLocal)
	addUserToGroup(t, db, localAdmin.ID, grp.ID)
	mapGroupToRole(t, db, grp.ID, adminRole.ID)

	app, views := newCaptureApp(t, db)

	resp := doGet(t, app, Path)

	defer func() { _ = resp.Body.Close() }()

	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusOK))

	data := views.data()
	g.Expect(data).NotTo(gomega.BeNil())
	g.Expect(data).To(gomega.HaveKey("GroupRoles"))

	groupRoles, ok := data["GroupRoles"].(map[uint64][]string)
	g.Expect(ok).To(gomega.BeTrue(), "GroupRoles should be map[uint64][]string")
	// The group role equals the direct role, so nothing extra should be advertised.
	g.Expect(groupRoles[localAdmin.ID]).To(gomega.BeEmpty())
}

// --- Delete guard for group-inherited admin ---

func TestDelete_PreventsGroupAdminDelete(t *testing.T) {
	g := gomega.NewWithT(t)
	db := newTestDB(t)

	initSessionStore()

	viewerRole := createRole(t, db, "viewer")
	adminRole := createRole(t, db, "admin")

	// LDAP user whose direct role is viewer but inherits admin via a group mapping.
	ldapUser := createUser(t, db, "ldapadmin", viewerRole.ID, func(u *models.User) {
		u.AuthSource = models.AuthSourceLDAP
		u.ExternalID = "uid=ldapadmin,ou=users,dc=example,dc=com"
	})

	grp := createGroup(t, db, "cn=dns-admins,ou=groups,dc=example,dc=com", models.GroupSourceLDAP)
	addUserToGroup(t, db, ldapUser.ID, grp.ID)
	mapGroupToRole(t, db, grp.ID, adminRole.ID)

	app := newTestApp(t, db)

	resp := doPost(t, app, fmt.Sprintf("%s/%d/delete", Path, ldapUser.ID), nil)

	defer func() { _ = resp.Body.Close() }()

	// A user who inherits admin via a group must be protected server-side, not just
	// by the disabled button in the UI.
	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusForbidden))

	// The user must still exist.
	var count int64
	db.Model(&models.User{}).Where("id = ?", ldapUser.ID).Count(&count)
	g.Expect(count).To(gomega.Equal(int64(1)))
}

// TestDelete_FailsClosedWhenGroupRoleCheckErrors ensures that if the inherited-admin
// lookup fails (e.g. a transient DB/query error), deletion is aborted rather than
// permitted, so the admin-protection guard cannot be bypassed.
func TestDelete_FailsClosedWhenGroupRoleCheckErrors(t *testing.T) {
	g := gomega.NewWithT(t)
	db := newTestDB(t)

	initSessionStore()

	viewerRole := createRole(t, db, "viewer")
	u := createUser(t, db, "unverifiable", viewerRole.ID)

	// Break the group-role lookup by removing a table the join depends on.
	if err := db.Migrator().DropTable("group_mappings"); err != nil {
		t.Fatalf("drop table: %v", err)
	}

	app := newTestApp(t, db)

	resp := doPost(t, app, fmt.Sprintf("%s/%d/delete", Path, u.ID), nil)

	defer func() { _ = resp.Body.Close() }()

	// The guard cannot verify inherited admin rights, so it must fail closed.
	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusInternalServerError))

	var count int64
	db.Model(&models.User{}).Where("id = ?", u.ID).Count(&count)
	g.Expect(count).To(gomega.Equal(int64(1)))
}

// TestDelete_AllowsNonAdminGroupRoleDelete ensures the group-role guard only blocks
// admin inheritance, not any group role — a viewer-via-group user is still deletable.
func TestDelete_AllowsNonAdminGroupRoleDelete(t *testing.T) {
	g := gomega.NewWithT(t)
	db := newTestDB(t)

	initSessionStore()

	viewerRole := createRole(t, db, "viewer")
	editorRole := createRole(t, db, "editor")

	ldapUser := createUser(t, db, "ldapeditor", viewerRole.ID, func(u *models.User) {
		u.AuthSource = models.AuthSourceLDAP
	})

	grp := createGroup(t, db, "cn=dns-editors,ou=groups,dc=example,dc=com", models.GroupSourceLDAP)
	addUserToGroup(t, db, ldapUser.ID, grp.ID)
	mapGroupToRole(t, db, grp.ID, editorRole.ID)

	app := newTestApp(t, db)

	resp := doPost(t, app, fmt.Sprintf("%s/%d/delete", Path, ldapUser.ID), nil)

	defer func() { _ = resp.Body.Close() }()

	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusSeeOther))

	var count int64
	db.Model(&models.User{}).Where("id = ?", ldapUser.ID).Count(&count)
	g.Expect(count).To(gomega.BeZero())
}
