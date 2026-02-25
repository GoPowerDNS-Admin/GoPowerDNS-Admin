package models

import "time"

// ActivityLog records administrative and user actions for auditing purposes.
// It tracks authentication events (login, logout, failures) and resource changes
// (zone create/update/delete, record changes) along with who performed the action.
type ActivityLog struct {
	// ID is the unique identifier for the log entry.
	ID uint64 `gorm:"primaryKey;autoIncrement"`
	// UserID is the ID of the user who performed the action. Nil for failed login attempts
	// where the user could not be identified.
	UserID *uint64 `gorm:"index"`
	// Username is always stored for the audit trail, even if the user is later deleted.
	Username string `gorm:"size:100;not null"`
	// Action describes what was performed (e.g. login, logout, zone_created).
	Action string `gorm:"size:50;not null;index"`
	// ResourceType is the category of resource affected (e.g. auth, zone).
	ResourceType string `gorm:"size:50"`
	// ResourceName is the specific resource identifier (e.g. zone name).
	ResourceName string `gorm:"size:255"`
	// Details holds optional JSON-encoded extra context for the entry.
	Details string `gorm:"type:text"`
	// IPAddress is the client IP address at the time of the action.
	IPAddress string `gorm:"size:45"`
	// CreatedAt is the timestamp when the event occurred.
	CreatedAt time.Time `gorm:"index"`
}
