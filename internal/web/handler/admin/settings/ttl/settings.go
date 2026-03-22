// Package ttl provides TTL preset settings for DNS record editing.
package ttl

import (
	"encoding/json"
	"errors"

	"gorm.io/gorm"

	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/db/controller/setting"
)

const (
	// SettingKey is the database key for TTL preset settings.
	SettingKey = "zone_ttl_presets"

	ttlOneMinute      uint32 = 60
	ttlFiveMinutes    uint32 = 300
	ttlTenMinutes     uint32 = 600
	ttlFifteenMinutes uint32 = 900
	ttlThirtyMinutes  uint32 = 1800
	ttlOneHour        uint32 = 3600
	ttlTwoHours       uint32 = 7200
	ttlFourHours      uint32 = 14400
	ttlEightHours     uint32 = 28800
	ttlTwelveHours    uint32 = 43200
	ttlOneDay         uint32 = 86400
	ttlTwoDays        uint32 = 172800
	ttlOneWeek        uint32 = 604800
)

// Preset is a named TTL value shown in the record edit dropdown.
type Preset struct {
	Seconds uint32 `json:"seconds"`
	Label   string `json:"label"`
}

// Settings holds the list of configured TTL presets.
type Settings struct {
	Presets []Preset `json:"presets"`
}

// Load loads TTL settings from the database.
func (s *Settings) Load(db *gorm.DB) error {
	entry, err := setting.Get(db, SettingKey)
	if err != nil {
		return err
	}

	return json.Unmarshal(entry.Value, s)
}

// Save persists TTL settings to the database.
func (s *Settings) Save(db *gorm.DB) error {
	data, err := json.Marshal(s)
	if err != nil {
		return err
	}

	_, err = setting.Set(db, SettingKey, data)

	return err
}

// DefaultPresets returns the built-in TTL preset list.
func DefaultPresets() []Preset {
	return []Preset{
		{Seconds: ttlOneMinute, Label: "1 minute"},
		{Seconds: ttlFiveMinutes, Label: "5 minutes"},
		{Seconds: ttlTenMinutes, Label: "10 minutes"},
		{Seconds: ttlFifteenMinutes, Label: "15 minutes"},
		{Seconds: ttlThirtyMinutes, Label: "30 minutes"},
		{Seconds: ttlOneHour, Label: "1 hour"},
		{Seconds: ttlTwoHours, Label: "2 hours"},
		{Seconds: ttlFourHours, Label: "4 hours"},
		{Seconds: ttlEightHours, Label: "8 hours"},
		{Seconds: ttlTwelveHours, Label: "12 hours"},
		{Seconds: ttlOneDay, Label: "1 day"},
		{Seconds: ttlTwoDays, Label: "2 days"},
		{Seconds: ttlOneWeek, Label: "1 week"},
	}
}

// LoadWithDefaults with fallback to defaults when the setting does not exist yet.
func LoadWithDefaults(db *gorm.DB) []Preset {
	var s Settings
	if err := s.Load(db); err != nil {
		if errors.Is(err, setting.ErrSettingNotFound) {
			return DefaultPresets()
		}

		return DefaultPresets()
	}

	return s.Presets
}
