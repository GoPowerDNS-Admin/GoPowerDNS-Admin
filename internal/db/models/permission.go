package models

import "time"

// Permission represents a specific permission in the authorization system.
// Permissions define granular access rights to resources and actions.
// They are assigned to roles, which are then assigned to users or mapped from groups.
type Permission struct {
	// ID is the unique identifier for the permission.
	ID uint `gorm:"primaryKey"`
	// Name is the unique permission identifier in resource.action format (e.g., "zone.create").
	Name string `gorm:"unique;size:100;not null"`
	// Resource is the resource this permission applies to (e.g., "zone", "admin", "server").
	Resource string `gorm:"size:100;not null"`
	// Action is the action allowed on the resource (e.g., "create", "read", "update", "delete").
	Action string `gorm:"size:50;not null"`
	// Description provides a human-readable explanation of what this permission grants.
	Description string `gorm:"size:255"`
	// CreatedAt is the timestamp when the permission was created (managed by GORM).
	CreatedAt time.Time
	// UpdatedAt is the timestamp when the permission was last updated (managed by GORM).
	UpdatedAt time.Time
}

// TableName specifies the database table name for the Permission model.
// This overrides GORM's default pluralized table naming.
func (Permission) TableName() string {
	return "permissions"
}
