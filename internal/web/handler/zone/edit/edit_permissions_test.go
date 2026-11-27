package zoneedit

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	pdnsapi "github.com/joeig/go-powerdns/v3"
	"gorm.io/gorm"

	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/auth"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/config"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/db/models"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/powerdns"
	websess "github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web/session"
)

const permTestZonePath = "/zone/edit/example.com"

type captureViews struct {
	mu   sync.Mutex
	data fiber.Map
}

func (v *captureViews) Load() error { return nil }

func (v *captureViews) Render(w io.Writer, name string, data any, _ ...string) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	v.data, _ = data.(fiber.Map)
	_, _ = io.WriteString(w, name)

	return nil
}

func (v *captureViews) lastData() fiber.Map {
	v.mu.Lock()
	defer v.mu.Unlock()

	return v.data
}

type memorySessionStorage struct {
	mu   sync.Mutex
	data map[string][]byte
}

func (s *memorySessionStorage) Get(key string) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.data[key], nil
}

func (s *memorySessionStorage) Set(key string, val []byte, _ time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.data[key] = append([]byte(nil), val...)

	return nil
}

func (s *memorySessionStorage) Delete(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.data, key)

	return nil
}

func newFakePowerDNS(t *testing.T) {
	t.Helper()

	const zoneJSON = `{
		"id": "example.com.", "name": "example.com.", "kind": "Native", "serial": 1, "dnssec": false,
		"rrsets": [{
			"name": "example.com.", "type": "SOA", "ttl": 3600,
			"records": [{"content": "ns1.example.com. hostmaster.example.com. 1 10800 3600 604800 3600", "disabled": false}]
		}]
	}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/servers/localhost/zones/example.com.":
			_, _ = io.WriteString(w, zoneJSON)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/servers/localhost/zones":
			_, _ = io.WriteString(w, "[]")
		default:
			w.WriteHeader(http.StatusNotFound)
			_, _ = io.WriteString(w, `{"error": "not found"}`)
		}
	}))

	powerdns.Engine.Client = pdnsapi.New(srv.URL, "localhost")

	t.Cleanup(func() {
		powerdns.Engine.Client = nil

		srv.Close()
	})
}

func seedRoleWithPermissions(t *testing.T, db *gorm.DB, roleName string, permissionNames ...string) models.Role {
	t.Helper()

	role := models.Role{Name: roleName}
	if err := db.Create(&role).Error; err != nil {
		t.Fatalf("create role %q: %v", roleName, err)
	}

	for _, name := range permissionNames {
		perm := models.Permission{Name: name, Resource: "zone", Action: name}
		if err := db.Where(models.Permission{Name: name}).FirstOrCreate(&perm).Error; err != nil {
			t.Fatalf("create permission %q: %v", name, err)
		}

		if err := db.Create(&models.RolePermission{RoleID: role.ID, PermissionID: perm.ID}).Error; err != nil {
			t.Fatalf("assign permission %q to role %q: %v", name, roleName, err)
		}
	}

	return role
}

func createSessionUser(t *testing.T, db *gorm.DB, username string, roleID uint) *http.Cookie {
	t.Helper()

	user := models.User{
		Username:   username,
		Email:      username + "@example.com",
		AuthSource: models.AuthSourceLocal,
		Active:     true,
		RoleID:     roleID,
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user %q: %v", username, err)
	}

	sessionID := "session-" + username
	if err := (&websess.Data{User: user}).Write(sessionID, time.Minute); err != nil {
		t.Fatalf("write session: %v", err)
	}

	return &http.Cookie{Name: "session", Value: sessionID}
}

// newPermissionTestApp registers the zone edit routes through Init, so the real
// permission middleware runs. It returns session cookies for a viewer, who has
// only zone.read, and an editor, who also has zone.update and zone.delete.
func newPermissionTestApp(t *testing.T) (app *fiber.App, views *captureViews, viewer, editor *http.Cookie) {
	t.Helper()

	db := newTestDB(t)
	if err := db.AutoMigrate(
		&models.User{}, &models.Role{}, &models.Permission{}, &models.RolePermission{},
		&models.Group{}, &models.UserGroup{}, &models.GroupMapping{},
		&models.Tag{}, &models.ZoneTag{}, &models.UserTag{}, &models.GroupTag{},
	); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}

	viewerRole := seedRoleWithPermissions(t, db, "viewer", auth.PermZoneRead)
	editorRole := seedRoleWithPermissions(t, db, "user", auth.PermZoneRead, auth.PermZoneUpdate, auth.PermZoneDelete)

	websess.Init(&memorySessionStorage{data: make(map[string][]byte)})

	viewer = createSessionUser(t, db, "viewer", viewerRole.ID)
	editor = createSessionUser(t, db, "editor", editorRole.ID)

	views = &captureViews{}
	app = fiber.New(fiber.Config{Views: views})

	app.Use(func(c fiber.Ctx) error {
		sess := new(websess.Data)
		if err := sess.Read(c.Cookies("session")); err == nil {
			c.Locals("CurrentUser", sess.User)
		}

		return c.Next()
	})

	svc := &Service{}
	svc.Init(app, &config.Config{}, db, auth.NewService(db))

	return app, views, viewer, editor
}

func doRequest(t *testing.T, app *fiber.App, method, path string, cookie *http.Cookie) int {
	t.Helper()

	req := httptest.NewRequestWithContext(context.Background(), method, path, strings.NewReader("{}"))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)

	resp, err := app.Test(req, fiber.TestConfig{Timeout: 30 * time.Second})
	if err != nil {
		t.Fatalf("app.Test %s %s: %v", method, path, err)
	}

	_ = resp.Body.Close()

	return resp.StatusCode
}

func TestGet_ViewerSeesZoneReadOnly(t *testing.T) {
	newFakePowerDNS(t)

	app, views, viewer, _ := newPermissionTestApp(t)

	if status := doRequest(t, app, fiber.MethodGet, permTestZonePath, viewer); status != fiber.StatusOK {
		t.Fatalf("expected viewer GET to return 200, got %d", status)
	}

	data := views.lastData()
	if data["CanUpdate"] != false || data["CanDelete"] != false {
		t.Fatalf("expected CanUpdate=false and CanDelete=false for viewer, got %v and %v",
			data["CanUpdate"], data["CanDelete"])
	}
}

func TestGet_EditorSeesEditControls(t *testing.T) {
	newFakePowerDNS(t)

	app, views, _, editor := newPermissionTestApp(t)

	if status := doRequest(t, app, fiber.MethodGet, permTestZonePath, editor); status != fiber.StatusOK {
		t.Fatalf("expected editor GET to return 200, got %d", status)
	}

	data := views.lastData()
	if data["CanUpdate"] != true || data["CanDelete"] != true {
		t.Fatalf("expected CanUpdate=true and CanDelete=true for editor, got %v and %v",
			data["CanUpdate"], data["CanDelete"])
	}
}

func TestPost_ViewerIsForbidden(t *testing.T) {
	newFakePowerDNS(t)

	app, _, viewer, _ := newPermissionTestApp(t)

	for _, path := range []string{
		permTestZonePath,
		permTestZonePath + "/records",
		permTestZonePath + "/delete",
	} {
		if status := doRequest(t, app, fiber.MethodPost, path, viewer); status != fiber.StatusForbidden {
			t.Errorf("expected viewer POST %s to return 403, got %d", path, status)
		}
	}
}
