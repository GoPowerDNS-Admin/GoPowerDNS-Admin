package zone

import (
	"encoding/json"

	"gorm.io/gorm"

	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/config"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/db/controller/setting"
)

const (
	// SettingKeyZoneRecords is the key used to store zone record settings in the database.
	SettingKeyZoneRecords = "zone_records"
)

// RecordSettings represents DNS record type configuration for zones.
type RecordSettings struct {
	Records config.Record `form:"records" json:"records"`
}

// Load loads the zone record settings from the database.
func (r *RecordSettings) Load(db *gorm.DB) error {
	// Retrieve the setting from the database
	s, err := setting.Get(db, SettingKeyZoneRecords)
	if err != nil {
		return err
	}

	// Unmarshal the JSON blob into the struct
	if errJSON := json.Unmarshal(s.Value, r); errJSON != nil {
		return errJSON
	}

	return nil
}

// Save saves the zone record settings to the database.
func (r *RecordSettings) Save(db *gorm.DB) error {
	// Marshal the struct to JSON
	data, err := json.Marshal(r)
	if err != nil {
		return err
	}

	// Save or update the setting in the database
	_, err = setting.Set(db, SettingKeyZoneRecords, data)
	return err
}
