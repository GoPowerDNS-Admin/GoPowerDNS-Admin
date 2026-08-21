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

	// Attacker/admin tries to swap membership to bob via a crafted POST.
	form := url.Values{
		"name":     {"cn=dns-admins,dc=example,dc=com"},
		"source":   {"ldap"},
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

	form := url.Values{
		"name":    {"DNS Admins"},
		"source":  {"ldap"},
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
