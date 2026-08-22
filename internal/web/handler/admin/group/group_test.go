package group

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
	"github.com/onsi/gomega"
	"gorm.io/gorm"

	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/config"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/db/models"
)

func uintStr(v uint64) string {
	return strconv.FormatUint(v, 10)
}

// noOpViews renders the "Error"/"error" field from fiber.Map so tests can assert
// error messages, otherwise it writes the template name.
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

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open in-memory sqlite: %v", err)
	}

	if err := db.AutoMigrate(
		&models.User{},
		&models.Role{},
		&models.Group{},
		&models.UserGroup{},
		&models.GroupMapping{},
		&models.Tag{},
		&models.GroupTag{},
	); err != nil {
		t.Fatalf("automigrate: %v", err)
	}

	return db
}

func newTestApp(t *testing.T, db *gorm.DB) *fiber.App {
	t.Helper()

	app := fiber.New(fiber.Config{Views: noOpViews{}})

	s := &Service{
		cfg:       &config.Config{},
		db:        db,
		validator: validator.New(),
	}

	app.Get(Path, s.List)
	app.Get(RouteNew, s.New)
	app.Post(Path, s.Create)
	app.Get(RouteEdit, s.Edit)
	app.Post(RouteUpdate, s.Update)
	app.Post(RouteDelete, s.Delete)

	return app
}

// captureViews records the data map passed to Render so tests can assert on the
// exact values a handler hands to the template (e.g. IsExternal, SelectedIDs).
type captureViews struct {
	mu       sync.Mutex
	lastData any
}

func (v *captureViews) Load() error { return nil }

func (v *captureViews) Render(w io.Writer, name string, data any, _ ...string) error {
	v.mu.Lock()
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

// newCaptureApp builds an app whose create/update routes use a capturing Views
// engine, so tests can inspect the render data map on validation-error rerenders.
func newCaptureApp(t *testing.T, db *gorm.DB) (*fiber.App, *captureViews) {
	t.Helper()

	views := &captureViews{}
	app := fiber.New(fiber.Config{Views: views})

	s := &Service{
		cfg:       &config.Config{},
		db:        db,
		validator: validator.New(),
	}

	app.Post(Path, s.Create)
	app.Post(RouteUpdate, s.Update)

	return app, views
}

func createRole(t *testing.T, db *gorm.DB, name string) models.Role {
	t.Helper()

	role := models.Role{Name: name}
	if err := db.Create(&role).Error; err != nil {
		t.Fatalf("create role %q: %v", name, err)
	}

	return role
}

func createUser(t *testing.T, db *gorm.DB, username string, roleID uint) models.User {
	t.Helper()

	u := models.User{
		Username:   username,
		Email:      username + "@example.com",
		AuthSource: models.AuthSourceLocal,
		Active:     true,
		RoleID:     roleID,
	}
	if err := db.Create(&u).Error; err != nil {
		t.Fatalf("create user %q: %v", username, err)
	}

	return u
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

func memberIDs(t *testing.T, db *gorm.DB, groupID uint) []uint64 {
	t.Helper()

	var ug []models.UserGroup
	if err := db.Where("group_id = ?", groupID).Find(&ug).Error; err != nil {
		t.Fatalf("load members: %v", err)
	}

	ids := make([]uint64, len(ug))
	for i := range ug {
		ids[i] = ug[i].UserID
	}

	return ids
}

func doPost(t *testing.T, app *fiber.App, path string, form url.Values) *http.Response {
	t.Helper()

	req := httptest.NewRequestWithContext(
		context.Background(), http.MethodPost, path, strings.NewReader(form.Encode()),
	)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test POST %s: %v", path, err)
	}

	return resp
}

func mappedRoleID(t *testing.T, db *gorm.DB, groupID uint) uint {
	t.Helper()

	var m models.GroupMapping
	if err := db.Where("group_id = ?", groupID).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0
		}

		t.Fatalf("load mapping: %v", err)
	}

	return m.RoleID
}

// --- Local groups: manual membership still works (regression) ---

func TestUpdate_LocalGroup_UpdatesMembership(t *testing.T) {
	g := gomega.NewWithT(t)
	db := newTestDB(t)

	role := createRole(t, db, "viewer")
	u1 := createUser(t, db, "alice", role.ID)
	u2 := createUser(t, db, "bob", role.ID)

	grp := createGroup(t, db, "local-team", models.GroupSourceLocal)
	addUserToGroup(t, db, u1.ID, grp.ID)

	app := newTestApp(t, db)

	form := url.Values{
		"name":     {"local-team"},
		"source":   {"local"},
		"user_ids": {uintStr(u2.ID)}, // replace alice with bob
	}

	resp := doPost(t, app, fmt.Sprintf("%s/%d", Path, grp.ID), form)

	defer func() { _ = resp.Body.Close() }()

	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusSeeOther))
	g.Expect(memberIDs(t, db, grp.ID)).To(gomega.ConsistOf(u2.ID))
}

// --- External groups: membership is preserved, not editable ---

func TestUpdate_ExternalGroup_PreservesMembership(t *testing.T) {
	g := gomega.NewWithT(t)
	db := newTestDB(t)

	role := createRole(t, db, "viewer")
	u1 := createUser(t, db, "ldapalice", role.ID)
	u2 := createUser(t, db, "ldapbob", role.ID)

	grp := createGroup(t, db, "cn=dns-admins,dc=example,dc=com", models.GroupSourceLDAP)
	addUserToGroup(t, db, u1.ID, grp.ID) // synced member

	app := newTestApp(t, db)

	// Attacker/admin tries to swap membership to bob via a crafted POST. The real UI
	// disables the source field for external groups, so it is not submitted here.
	form := url.Values{
		"name":     {"cn=dns-admins,dc=example,dc=com"},
		"user_ids": {uintStr(u2.ID)},
	}

	resp := doPost(t, app, fmt.Sprintf("%s/%d", Path, grp.ID), form)

	defer func() { _ = resp.Body.Close() }()

	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusSeeOther))
	// Synced membership must be untouched: still only alice, not bob.
	g.Expect(memberIDs(t, db, grp.ID)).To(gomega.ConsistOf(u1.ID))
}

func TestUpdate_ExternalGroup_LocksSourceAndExternalID(t *testing.T) {
	g := gomega.NewWithT(t)
	db := newTestDB(t)

	grp := createGroup(t, db, "cn=team,dc=example,dc=com", models.GroupSourceLDAP)

	app := newTestApp(t, db)

	// Try to change source to local and rewrite the external id.
	form := url.Values{
		"name":        {"cn=team,dc=example,dc=com"},
		"source":      {"local"},
		"external_id": {"tampered-id"},
	}

	resp := doPost(t, app, fmt.Sprintf("%s/%d", Path, grp.ID), form)

	defer func() { _ = resp.Body.Close() }()

	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusSeeOther))

	var reloaded models.Group
	g.Expect(db.First(&reloaded, grp.ID).Error).To(gomega.Succeed())
	g.Expect(reloaded.Source).To(gomega.Equal(models.GroupSourceLDAP))
	g.Expect(reloaded.ExternalID).To(gomega.Equal("cn=team,dc=example,dc=com"))
}

func TestUpdate_ExternalGroup_AllowsRoleMappingAndName(t *testing.T) {
	g := gomega.NewWithT(t)
	db := newTestDB(t)

	adminRole := createRole(t, db, "admin")
	grp := createGroup(t, db, "cn=dns-admins,dc=example,dc=com", models.GroupSourceLDAP)

	app := newTestApp(t, db)

	// The source field is disabled in the UI for external groups, so it is omitted.
	form := url.Values{
		"name":    {"DNS Admins"},
		"role_id": {uintStr(uint64(adminRole.ID))},
	}

	resp := doPost(t, app, fmt.Sprintf("%s/%d", Path, grp.ID), form)

	defer func() { _ = resp.Body.Close() }()

	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusSeeOther))

	var reloaded models.Group
	g.Expect(db.First(&reloaded, grp.ID).Error).To(gomega.Succeed())
	g.Expect(reloaded.Name).To(gomega.Equal("DNS Admins"))
	g.Expect(mappedRoleID(t, db, grp.ID)).To(gomega.Equal(adminRole.ID))
}

// --- Local -> external conversion ---

func TestUpdate_LocalToExternal_DoesNotWriteSubmittedMembers(t *testing.T) {
	g := gomega.NewWithT(t)
	db := newTestDB(t)

	role := createRole(t, db, "viewer")
	u := createUser(t, db, "frank", role.ID)

	grp := createGroup(t, db, "team", models.GroupSourceLocal)

	app := newTestApp(t, db)

	// Convert the local group to LDAP while also submitting members. The submitted
	// members must NOT be written, otherwise they become phantom external memberships.
	form := url.Values{
		"name":        {"team"},
		"source":      {"ldap"},
		"external_id": {"cn=team,dc=example,dc=com"},
		"user_ids":    {uintStr(u.ID)},
	}

	resp := doPost(t, app, fmt.Sprintf("%s/%d", Path, grp.ID), form)

	defer func() { _ = resp.Body.Close() }()

	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusSeeOther))

	var reloaded models.Group
	g.Expect(db.First(&reloaded, grp.ID).Error).To(gomega.Succeed())
	g.Expect(reloaded.Source).To(gomega.Equal(models.GroupSourceLDAP))
	g.Expect(memberIDs(t, db, grp.ID)).To(gomega.BeEmpty())
}

func TestUpdate_LocalToExternal_ClearsExistingMembers(t *testing.T) {
	g := gomega.NewWithT(t)
	db := newTestDB(t)

	role := createRole(t, db, "viewer")
	u := createUser(t, db, "grace", role.ID)

	grp := createGroup(t, db, "team2", models.GroupSourceLocal)
	addUserToGroup(t, db, u.ID, grp.ID) // pre-existing local member

	app := newTestApp(t, db)

	form := url.Values{
		"name":        {"team2"},
		"source":      {"oidc"},
		"external_id": {"team2-claim"},
	}

	resp := doPost(t, app, fmt.Sprintf("%s/%d", Path, grp.ID), form)

	defer func() { _ = resp.Body.Close() }()

	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusSeeOther))

	var reloaded models.Group
	g.Expect(db.First(&reloaded, grp.ID).Error).To(gomega.Succeed())
	g.Expect(reloaded.Source).To(gomega.Equal(models.GroupSourceOIDC))
	// Manual local members are dropped so the directory becomes the source of truth.
	g.Expect(memberIDs(t, db, grp.ID)).To(gomega.BeEmpty())
}

// --- Create ---

func TestCreate_LocalGroup_AddsMembership(t *testing.T) {
	g := gomega.NewWithT(t)
	db := newTestDB(t)

	role := createRole(t, db, "viewer")
	u := createUser(t, db, "carol", role.ID)

	app := newTestApp(t, db)

	form := url.Values{
		"name":     {"local-group"},
		"source":   {"local"},
		"user_ids": {uintStr(u.ID)},
	}

	resp := doPost(t, app, Path, form)

	defer func() { _ = resp.Body.Close() }()

	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusSeeOther))

	var grp models.Group
	g.Expect(db.Where("name = ?", "local-group").First(&grp).Error).To(gomega.Succeed())
	g.Expect(memberIDs(t, db, grp.ID)).To(gomega.ConsistOf(u.ID))
}

func TestCreate_ExternalGroup_SkipsMembership(t *testing.T) {
	g := gomega.NewWithT(t)
	db := newTestDB(t)

	role := createRole(t, db, "viewer")
	u := createUser(t, db, "dave", role.ID)

	app := newTestApp(t, db)

	form := url.Values{
		"name":        {"cn=new-ldap,dc=example,dc=com"},
		"source":      {"ldap"},
		"external_id": {"cn=new-ldap,dc=example,dc=com"},
		"user_ids":    {uintStr(u.ID)},
	}

	resp := doPost(t, app, Path, form)

	defer func() { _ = resp.Body.Close() }()

	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusSeeOther))

	var grp models.Group
	g.Expect(db.Where("name = ?", "cn=new-ldap,dc=example,dc=com").First(&grp).Error).To(gomega.Succeed())
	// Manual member seeding must be ignored for external groups.
	g.Expect(memberIDs(t, db, grp.ID)).To(gomega.BeEmpty())
}

// --- External ID required for external sources (fix B) ---

func TestCreate_ExternalGroup_RequiresExternalID(t *testing.T) {
	g := gomega.NewWithT(t)
	db := newTestDB(t)

	app := newTestApp(t, db)

	// Selecting an external source without an external id must be rejected.
	form := url.Values{
		"name":   {"cn=blank,dc=example,dc=com"},
		"source": {"ldap"},
	}

	resp := doPost(t, app, Path, form)

	defer func() { _ = resp.Body.Close() }()

	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusBadRequest))

	var count int64
	db.Model(&models.Group{}).Where("name = ?", "cn=blank,dc=example,dc=com").Count(&count)
	g.Expect(count).To(gomega.BeZero())
}

func TestUpdate_LocalToExternal_RequiresExternalID(t *testing.T) {
	g := gomega.NewWithT(t)
	db := newTestDB(t)

	role := createRole(t, db, "viewer")
	u := createUser(t, db, "hank", role.ID)

	grp := createGroup(t, db, "team", models.GroupSourceLocal)
	addUserToGroup(t, db, u.ID, grp.ID)

	app := newTestApp(t, db)

	// Convert to LDAP but leave the external id blank — must be rejected, and the
	// group must remain a valid local group with its membership intact.
	form := url.Values{
		"name":   {"team"},
		"source": {"ldap"},
	}

	resp := doPost(t, app, fmt.Sprintf("%s/%d", Path, grp.ID), form)

	defer func() { _ = resp.Body.Close() }()

	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusBadRequest))

	var reloaded models.Group
	g.Expect(db.First(&reloaded, grp.ID).Error).To(gomega.Succeed())
	g.Expect(reloaded.Source).To(gomega.Equal(models.GroupSourceLocal))
	g.Expect(memberIDs(t, db, grp.ID)).To(gomega.ConsistOf(u.ID))
}

// --- Validation-error rerender preserves external state (fix A) ---

func TestUpdate_ExternalGroup_ValidationError_PreservesStateAndMembers(t *testing.T) {
	g := gomega.NewWithT(t)
	db := newTestDB(t)

	role := createRole(t, db, "viewer")
	u := createUser(t, db, "ivy", role.ID)

	grp := createGroup(t, db, "cn=team,dc=example,dc=com", models.GroupSourceLDAP)
	addUserToGroup(t, db, u.ID, grp.ID)

	app, views := newCaptureApp(t, db)

	// Blank name fails validation. The source field is disabled in the UI, so it is
	// not submitted (defaults to local) — the rerender must still treat the group as
	// external and repopulate its synced members.
	form := url.Values{
		"name": {""},
	}

	resp := doPost(t, app, fmt.Sprintf("%s/%d", Path, grp.ID), form)

	defer func() { _ = resp.Body.Close() }()

	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusBadRequest))

	data := views.data()
	g.Expect(data).NotTo(gomega.BeNil())
	g.Expect(data["IsExternal"]).To(gomega.Equal(true))

	grpData, ok := data["Group"].(models.Group)
	g.Expect(ok).To(gomega.BeTrue())
	g.Expect(grpData.Source).To(gomega.Equal(models.GroupSourceLDAP))
	g.Expect(grpData.ExternalID).To(gomega.Equal("cn=team,dc=example,dc=com"))

	selected, ok := data["SelectedIDs"].([]uint64)
	g.Expect(ok).To(gomega.BeTrue())
	g.Expect(selected).To(gomega.ConsistOf(u.ID))
}

func TestUpdate_LocalToExternal_ValidationError_StaysEditable(t *testing.T) {
	g := gomega.NewWithT(t)
	db := newTestDB(t)

	grp := createGroup(t, db, "team", models.GroupSourceLocal)

	app, views := newCaptureApp(t, db)

	// Converting local -> external with a blank external id fails validation. Because
	// the stored group is still local, the rerender must remain editable so the admin
	// can supply the required external id.
	form := url.Values{
		"name":   {"team"},
		"source": {"ldap"},
	}

	resp := doPost(t, app, fmt.Sprintf("%s/%d", Path, grp.ID), form)

	defer func() { _ = resp.Body.Close() }()

	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusBadRequest))

	data := views.data()
	g.Expect(data).NotTo(gomega.BeNil())
	g.Expect(data["IsExternal"]).To(gomega.Equal(false))

	grpData, ok := data["Group"].(models.Group)
	g.Expect(ok).To(gomega.BeTrue())
	g.Expect(grpData.Source).To(gomega.Equal(models.GroupSourceLDAP))
}
