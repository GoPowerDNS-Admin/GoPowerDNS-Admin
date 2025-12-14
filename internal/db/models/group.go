package models

import "time"

// GroupSource represents the origin or source system of a user group.
// It indicates whether the group is managed locally or synchronized from external systems.
type GroupSource string

const (
	// GroupSourceLocal indicates the group is locally managed within the application.
	GroupSourceLocal GroupSource = "local"
	// GroupSourceOIDC indicates the group is synchronized from an OIDC identity provider.
	GroupSourceOIDC GroupSource = "oidc"
	// GroupSourceLDAP indicates the group is synchronized from an LDAP or Active Directory server.
	GroupSourceLDAP GroupSource = "ldap"
)

// Group represents a user group for organizing users and mapping to roles.
// Groups can be locally managed or automatically synchronized from external sources (OIDC or LDAP).
// External groups are mapped to internal roles to determine user permissions.
type Group struct {
	// ID is the unique identifier for the group.
	ID uint `gorm:"primaryKey"`
	// Name is the display name of the group as it appears in the system.
	Name string `gorm:"size:100;not null"`
	// ExternalID is the external identifier for the group (DN for LDAP, claim value for OIDC).
	// Combined with Source, this forms a unique constraint.
	ExternalID string `gorm:"size:255;uniqueIndex:idx_source_external"`
	// Source indicates where the group originates from (local, oidc, or ldap).
	Source GroupSource `gorm:"type:varchar(20);not null;uniqueIndex:idx_source_external"`
	// Description provides a human-readable explanation of the group's purpose.
	Description string `gorm:"size:255"`
	// CreatedAt is the timestamp when the group was created (managed by GORM).
	CreatedAt time.Time
	// UpdatedAt is the timestamp when the group was last updated (managed by GORM).
	UpdatedAt time.Time
}

// TableName specifies the database table name for the Group model.
// This overrides GORM's default pluralized table naming.
func (Group) TableName() string {
	return "groups"
}
