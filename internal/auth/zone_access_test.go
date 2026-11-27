package auth

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/db/models"
)

type zoneAccessFixture struct {
	db       *gorm.DB
	svc      *Service
	adminID  uint
	viewerID uint
	tagID    uint
}

func newZoneAccessFixture(t *testing.T) *zoneAccessFixture {
	t.Helper()

	db := newServiceTestDB(t)
	require.NoError(t, db.AutoMigrate(
		&models.Tag{},
		&models.ZoneTag{},
		&models.UserTag{},
		&models.GroupTag{},
	))

	admin := models.Role{Name: "admin"}
	viewer := models.Role{Name: "viewer"}
	tag := models.Tag{Name: "team-a"}

	require.NoError(t, db.Create(&admin).Error)
	require.NoError(t, db.Create(&viewer).Error)
	require.NoError(t, db.Create(&tag).Error)
	require.NoError(t, db.Create(&models.ZoneTag{ZoneID: "a.example.", TagID: tag.ID}).Error)

	return &zoneAccessFixture{
		db:       db,
		svc:      NewService(db),
		adminID:  admin.ID,
		viewerID: viewer.ID,
		tagID:    tag.ID,
	}
}

func (f *zoneAccessFixture) createUser(t *testing.T, name string, roleID uint) uint64 {
	t.Helper()

	user := models.User{Username: name, Email: name + "@example.com", RoleID: roleID}
	require.NoError(t, f.db.Create(&user).Error)

	return user.ID
}

func (f *zoneAccessFixture) addToGroup(t *testing.T, userID uint64, groupName string, roleID uint) uint {
	t.Helper()

	group := models.Group{Name: groupName, ExternalID: groupName, Source: models.GroupSourceOIDC}
	require.NoError(t, f.db.Create(&group).Error)
	require.NoError(t, f.db.Create(&models.UserGroup{UserID: userID, GroupID: group.ID}).Error)
	require.NoError(t, f.db.Create(&models.GroupMapping{GroupID: group.ID, RoleID: roleID}).Error)

	return group.ID
}

func TestHasRole(t *testing.T) {
	f := newZoneAccessFixture(t)

	direct := f.createUser(t, "direct", f.adminID)
	viaGroup := f.createUser(t, "via-group", f.viewerID)
	f.addToGroup(t, viaGroup, "dns-admins", f.adminID)
	plain := f.createUser(t, "plain", f.viewerID)
	f.addToGroup(t, plain, "dns-viewers", f.viewerID)

	for _, tc := range []struct {
		name   string
		userID uint64
		want   bool
	}{
		{name: "own role", userID: direct, want: true},
		{name: "group role", userID: viaGroup, want: true},
		{name: "neither", userID: plain, want: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := f.svc.HasRole(tc.userID, "admin")
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}

	_, err := f.svc.HasRole(9999, "admin")
	require.Error(t, err)
}

func TestGetAccessibleZoneIDs(t *testing.T) {
	f := newZoneAccessFixture(t)

	directAdmin := f.createUser(t, "direct-admin", f.adminID)
	require.NoError(t, f.db.Create(&models.UserTag{UserID: directAdmin, TagID: f.tagID}).Error)

	groupAdmin := f.createUser(t, "group-admin", f.viewerID)
	adminGroup := f.addToGroup(t, groupAdmin, "dns-admins", f.adminID)
	require.NoError(t, f.db.Create(&models.GroupTag{GroupID: adminGroup, TagID: f.tagID}).Error)

	taggedViewer := f.createUser(t, "tagged-viewer", f.viewerID)
	require.NoError(t, f.db.Create(&models.UserTag{UserID: taggedViewer, TagID: f.tagID}).Error)

	untaggedViewer := f.createUser(t, "untagged-viewer", f.viewerID)

	for _, tc := range []struct {
		name   string
		userID uint64
		want   map[string]bool
	}{
		{name: "direct admin with tag is unrestricted", userID: directAdmin, want: nil},
		{name: "group admin with tag is unrestricted", userID: groupAdmin, want: nil},
		{name: "tagged viewer sees tagged zones only", userID: taggedViewer, want: map[string]bool{"a.example.": true}},
		{name: "untagged viewer is unrestricted", userID: untaggedViewer, want: nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := f.svc.GetAccessibleZoneIDs(tc.userID)
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}

	_, err := f.svc.GetAccessibleZoneIDs(9999)
	require.Error(t, err)
}
