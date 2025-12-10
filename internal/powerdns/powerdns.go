package powerdns

import (
	"context"
	"time"

	"github.com/joeig/go-powerdns/v3"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"

	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/db/controller/pdnsserver"
)

const (
	defaultTimeout = 30 * time.Second
)

type engine struct {
	*powerdns.Client
}

// Engine represents the PowerDNS client engine.
var Engine engine

func (e engine) Test() error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	// Test PowerDNS API connection
	if e.Client == nil {
		return ErrClientNotInitialized
	}

	zones, err := e.Zones.List(ctx)
	if err != nil {
		return err
	}

	log.Info().Int("zone_count", len(zones)).Msg("PowerDNS API connection test successful")

	return nil
}

// Open initializes the PowerDNS client using settings from the database.
func Open(db *gorm.DB) error {
	// Initialize PowerDNS client
	// get settings
	settings := &pdnsserver.Settings{}
	if err := settings.Load(db); err != nil {
		return err
	}

	// create new PowerDNS client
	Engine.Client = powerdns.New(settings.APIServerURL, settings.VHost, powerdns.WithAPIKey(settings.APIKey))

	return nil
}
