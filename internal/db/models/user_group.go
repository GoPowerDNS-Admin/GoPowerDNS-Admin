package models

import "time"

// UserGroup represents the many-to-many relationship between users and groups.
// This junction table allows users to belong to multiple groups, and groups to contain multiple users.
// Group memberships are used to determine which roles (and permissions) a user receives.
// For external auth sources (OIDC/LDAP), these memberships are automatically synchronized on login.
type UserGroup struct {
	// UserID is the ID of the user in this membership.
	UserID uint64 `gorm:"primaryKey;column:user_id"`
	// GroupID is the ID of the group in this membership.
	GroupID uint `gorm:"primaryKey;column:group_id"`
	// User is the associated user (loaded via foreign key).
	// When a user is deleted, all their group memberships are automatically removed (CASCADE).
	User User `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	// Group is the associated group (loaded via foreign key).
	// When a group is deleted, all user memberships in that group are automatically removed (CASCADE).
	Group Group `gorm:"foreignKey:GroupID;constraint:OnDelete:CASCADE"`
	// CreatedAt is the timestamp when the user was added to the group (managed by GORM).
	CreatedAt time.Time
}

// TableName specifies the database table name for the UserGroup model.
// This overrides GORM's default pluralized table naming.
func (UserGroup) TableName() string {
	return "user_groups"
}
