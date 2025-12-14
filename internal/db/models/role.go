package models

import "time"

// Role represents a role in the role-based access control (RBAC) system.
// Roles are collections of permissions that can be assigned to users or mapped from groups.
// Examples include "admin", "user", and "viewer" roles.
type Role struct {
	// ID is the unique identifier for the role.
	ID uint `gorm:"primaryKey"`
	// Name is the unique name of the role (e.g., "admin", "user").
	Name string `gorm:"unique;size:100;not null"`
	// Description provides a human-readable description of the role's purpose.
	Description string `gorm:"size:255"`
	// IsSystem indicates if this is a system role that cannot be deleted.
	IsSystem bool `gorm:"default:false"`
	// CreatedAt is the timestamp when the role was created (managed by GORM).
	CreatedAt time.Time
	// UpdatedAt is the timestamp when the role was last updated (managed by GORM).
	UpdatedAt time.Time
}

// TableName specifies the database table name for the Role model.
// This overrides GORM's default pluralized table naming.
func (Role) TableName() string {
	return "roles"
}
