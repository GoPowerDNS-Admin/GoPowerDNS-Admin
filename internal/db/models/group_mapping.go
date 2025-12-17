package models

import "time"

// GroupMapping maps external groups to internal roles for authorization.
// When a user logs in via OIDC or LDAP, their external groups are synchronized,
// and these mappings determine which roles (and therefore permissions) they receive.
// Each mapping defines a one-to-one relationship between a group and a role.
type GroupMapping struct {
	// ID is the unique identifier for the group mapping.
	ID uint `gorm:"primaryKey"`
	// GroupID is the ID of the group being mapped.
	// Enforced unique to ensure a group maps to exactly one role.
	GroupID uint `gorm:"not null;uniqueIndex"`
	// RoleID is the ID of the role that group members will receive.
	RoleID uint `gorm:"not null"`
	// Group is the associated group (loaded via foreign key).
	// When a group is deleted, all its mappings are automatically removed (CASCADE).
	Group Group `gorm:"foreignKey:GroupID;constraint:OnDelete:CASCADE"`
	// Role is the associated role (loaded via foreign key).
	// When a role is deleted, all its group mappings are automatically removed (CASCADE).
	Role Role `gorm:"foreignKey:RoleID;constraint:OnDelete:CASCADE"`
	// CreatedAt is the timestamp when the mapping was created (managed by GORM).
	CreatedAt time.Time
	// UpdatedAt is the timestamp when the mapping was last updated (managed by GORM).
	UpdatedAt time.Time
}

// TableName specifies the database table name for the GroupMapping model.
// This overrides GORM's default pluralized table naming.
func (GroupMapping) TableName() string {
	return "group_mappings"
}
