package pdnsserver

import (
	"encoding/json"

	"gorm.io/gorm"

	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/db/controller/setting"
)

const (
	// SettingKeyPDNSServer is the key used to store PDNS server settings in the database.
	SettingKeyPDNSServer = "pdns_server"
)

type (
	// Settings represents PowerDNS server configuration.
	Settings struct {
		APIServerURL string `form:"api_server_url" json:"apiServerUrl" validate:"required,url"`
		APIKey       string `form:"api_key"        json:"apiKey"       validate:"required,min=8"`
		VHost        string `form:"version"        json:"vhost"        validate:"required"`
	}
)

// Load loads the PDNS server settings from the database.
func (p *Settings) Load(db *gorm.DB) error {
	// Retrieve the setting from the database
	s, err := setting.Get(db, SettingKeyPDNSServer)
	if err != nil {
		return err
	}

	// Unmarshal the JSON blob into the struct
	return json.Unmarshal(s.Value, p)
}

// Save saves the PDNS server settings to the database.
func (p *Settings) Save(db *gorm.DB) error {
	// Marshal the struct to JSON
	data, err := json.Marshal(p)
	if err != nil {
		return err
	}

	// Save or update the setting in the database
	_, err = setting.Set(db, SettingKeyPDNSServer, data)

	return err
}
