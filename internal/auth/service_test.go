package auth

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/db/models"
)

func newServiceTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&models.Role{},
		&models.User{},
		&models.Group{},
		&models.UserGroup{},
		&models.GroupMapping{},
	))

	return db
}

func TestGroupDisplayName(t *testing.T) {
	cases := []struct {
		name     string
		external string
		source   models.GroupSource
		want     string
	}{
		{
			name:     "ldap dn uses first RDN value",
			external: "cn=dns-admins,ou=groups,dc=example,dc=com",
			source:   models.GroupSourceLDAP,
			want:     "dns-admins",
		},
		{
			name:     "ldap dn with spaces",
			external: "CN=DNS Admins,OU=Groups,DC=example,DC=com",
			source:   models.GroupSourceLDAP,
			want:     "DNS Admins",
		},
		{
			name:     "ldap non-dn falls back to raw",
			external: "not-a-dn",
			source:   models.GroupSourceLDAP,
			want:     "not-a-dn",
		},
		{
			name:     "oidc uses raw value",
			external: "dns-admins",
			source:   models.GroupSourceOIDC,
			want:     "dns-admins",
		},
		{
			name:     "local uses raw value",
			external: "some-group",
			source:   models.GroupSourceLocal,
			want:     "some-group",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, groupDisplayName(tc.external, tc.source))
		})
	}
}

func TestSyncUserGroups_LDAP_StoresFriendlyNameAndFullDN(t *testing.T) {
	db := newServiceTestDB(t)
	svc := NewService(db)

	role := models.Role{Name: "viewer"}
	require.NoError(t, db.Create(&role).Error)

	user := models.User{
		Username:   "ldapuser",
		Email:      "ldapuser@example.com",
		AuthSource: models.AuthSourceLDAP,
		Active:     true,
		RoleID:     role.ID,
	}
	require.NoError(t, db.Create(&user).Error)

	dn := "cn=dns-admins,ou=groups,dc=example,dc=com"
	require.NoError(t, svc.SyncUserGroups(user.ID, []string{dn}, models.GroupSourceLDAP))

	var group models.Group
	require.NoError(t, db.Where("external_id = ? AND source = ?", dn, models.GroupSourceLDAP).First(&group).Error)

	// Friendly display name, full DN preserved as the external id.
	require.Equal(t, "dns-admins", group.Name)
	require.Equal(t, dn, group.ExternalID)

	// Membership was recorded.
	var count int64
	require.NoError(t, db.Model(&models.UserGroup{}).
		Where("user_id = ? AND group_id = ?", user.ID, group.ID).Count(&count).Error)
	require.Equal(t, int64(1), count)
}

func TestSyncUserGroups_PreservesLongDN(t *testing.T) {
	db := newServiceTestDB(t)
	svc := NewService(db)

	role := models.Role{Name: "viewer"}
	require.NoError(t, db.Create(&role).Error)

	user := models.User{
		Username:   "deepuser",
		Email:      "deepuser@example.com",
		AuthSource: models.AuthSourceLDAP,
		Active:     true,
		RoleID:     role.ID,
	}
	require.NoError(t, db.Create(&user).Error)

	// A deeply-nested DN comfortably longer than the old 255-char limit.
	dn := "cn=platform-engineering-oncall-secondary-responders-primary-escalation," +
		"ou=oncall-rotations,ou=site-reliability,ou=engineering,ou=technology," +
		"ou=departments,ou=north-america,ou=regions,ou=business-units,ou=staff," +
		"ou=directory,dc=corp,dc=subsidiary,dc=holding,dc=example,dc=com"
	require.Greater(t, len(dn), 255)

	require.NoError(t, svc.SyncUserGroups(user.ID, []string{dn}, models.GroupSourceLDAP))

	var group models.Group
	require.NoError(t, db.Where("source = ?", models.GroupSourceLDAP).First(&group).Error)
	require.Equal(t, dn, group.ExternalID)
	require.Equal(t, "platform-engineering-oncall-secondary-responders-primary-escalation", group.Name)
}

func TestEnsureGroupRole_CreatesGroupAndMapping(t *testing.T) {
	db := newServiceTestDB(t)
	require.NoError(t, db.AutoMigrate(&models.Permission{}, &models.RolePermission{}))
	svc := NewService(db)

	viewer := models.Role{Name: "viewer"}
	admin := models.Role{Name: "admin"}

	require.NoError(t, db.Create(&viewer).Error)
	require.NoError(t, db.Create(&admin).Error)

	perm := models.Permission{Name: PermAdminUsers, Resource: "admin", Action: "users"}
	require.NoError(t, db.Create(&perm).Error)
	require.NoError(t, db.Create(&models.RolePermission{RoleID: admin.ID, PermissionID: perm.ID}).Error)

	mappedRole, created, err := svc.EnsureGroupRole(models.GroupSourceOIDC, "/DNS-Admins", "admin")
	require.NoError(t, err)
	require.True(t, created)
	require.Equal(t, "admin", mappedRole)

	mappedRole, created, err = svc.EnsureGroupRole(models.GroupSourceOIDC, "/DNS-Admins", "admin")
	require.NoError(t, err)
	require.False(t, created)
	require.Equal(t, "admin", mappedRole)

	user := models.User{
		Username:   "oidcuser@example.com",
		Email:      "oidcuser@example.com",
		AuthSource: models.AuthSourceOIDC,
		Active:     true,
		RoleID:     viewer.ID,
	}
	require.NoError(t, db.Create(&user).Error)
	require.NoError(t, svc.SyncUserGroups(user.ID, []string{"/DNS-Admins"}, models.GroupSourceOIDC))

	var groupCount int64
	require.NoError(t, db.Model(&models.Group{}).Count(&groupCount).Error)
	require.Equal(t, int64(1), groupCount, "login must reuse the seeded group")

	has, err := svc.HasPermission(user.ID, PermAdminUsers)
	require.NoError(t, err)
	require.True(t, has, "a member of the seeded group must inherit the mapped role")
}

func TestEnsureGroupRole_KeepsExistingMapping(t *testing.T) {
	db := newServiceTestDB(t)
	svc := NewService(db)

	viewer := models.Role{Name: "viewer"}
	admin := models.Role{Name: "admin"}

	require.NoError(t, db.Create(&viewer).Error)
	require.NoError(t, db.Create(&admin).Error)

	_, _, err := svc.EnsureGroupRole(models.GroupSourceLDAP, "cn=dns-admins,ou=groups,dc=example,dc=com", "viewer")
	require.NoError(t, err)

	mappedRole, created, err := svc.EnsureGroupRole(
		models.GroupSourceLDAP, "cn=dns-admins,ou=groups,dc=example,dc=com", "admin")
	require.NoError(t, err)
	require.False(t, created)
	require.Equal(t, "viewer", mappedRole)

	var mapping models.GroupMapping
	require.NoError(t, db.First(&mapping).Error)
	require.Equal(t, viewer.ID, mapping.RoleID)
}

func TestEnsureGroupRole_UnknownRole(t *testing.T) {
	db := newServiceTestDB(t)
	svc := NewService(db)

	_, created, err := svc.EnsureGroupRole(models.GroupSourceOIDC, "dns-admins", "superuser")
	require.ErrorIs(t, err, ErrRoleNotFound)
	require.False(t, created)

	var groupCount int64
	require.NoError(t, db.Model(&models.Group{}).Count(&groupCount).Error)
	require.Zero(t, groupCount)
}
