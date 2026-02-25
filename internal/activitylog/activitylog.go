// Package activitylog provides helpers for recording audit trail entries.
package activitylog

import (
	"encoding/json"

	"github.com/rs/zerolog/log"
	"gorm.io/gorm"

	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/db/models"
)

// Action constants define the supported audit event types.
const (
	ActionLogin         = "login"
	ActionLoginFailed   = "login_failed"
	ActionLogout        = "logout"
	ActionZoneCreated   = "zone_created"
	ActionZoneUpdated   = "zone_updated"
	ActionZoneDeleted   = "zone_deleted"
	ActionRecordChanged = "record_changed"
	ActionRecordUndone  = "record_undone"
)

// ResourceType constants categorize the resource affected by an action.
const (
	ResourceTypeAuth = "auth"
	ResourceTypeZone = "zone"
)

// Entry holds all fields needed to record an activity log event.
type Entry struct {
	DB           *gorm.DB
	UserID       *uint64
	Username     string
	Action       string
	ResourceType string
	ResourceName string
	Details      any
	IPAddress    string
}

// Record creates a new ActivityLog entry in the database.
func Record(e *Entry) {
	detailsJSON := ""

	if e.Details != nil {
		b, err := json.Marshal(e.Details)
		if err == nil {
			detailsJSON = string(b)
		}
	}

	entry := &models.ActivityLog{
		UserID:       e.UserID,
		Username:     e.Username,
		Action:       e.Action,
		ResourceType: e.ResourceType,
		ResourceName: e.ResourceName,
		Details:      detailsJSON,
		IPAddress:    e.IPAddress,
	}

	if err := e.DB.Create(entry).Error; err != nil {
		log.Error().Err(err).Str("action", e.Action).Str("username", e.Username).
			Msg("failed to record activity log entry")
	}
}
