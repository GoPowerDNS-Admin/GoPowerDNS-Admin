package daemon

import (
	sessionmysql "github.com/gofiber/storage/mysql"
	"github.com/rs/zerolog/log"
	gormmysql "gorm.io/driver/mysql"
	"gorm.io/gorm"

	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/config"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/db/dsn"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/db/models"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/powerdns"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web/session"
)

// Daemon represents the main application daemon.
type Daemon struct {
	webService web.Service
}

// Start starts the Daemon's web service.
func (d *Daemon) Start() error {
	return d.webService.Start(":8080")
}

// New creates a new Daemon instance with the provided configuration.
func New(cfg *config.Config) *Daemon {
	if cfg == nil {
		log.Fatal().Msg("config is nil")
		return nil
	}

	dbDriver := gormmysql.Open(dsn.Create(cfg)) // open db with gorm mysql driver

	db, err := gorm.Open(dbDriver, &gorm.Config{})
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect database")
	}

	if err = db.AutoMigrate(
		&models.User{},
		&models.Setting{},
		&models.Role{},
		&models.Permission{},
		&models.RolePermission{},
		&models.Group{},
		&models.GroupMapping{},
		&models.UserGroup{},
	); err != nil {
		log.Fatal().Err(err).Msg("failed to migrate database")
	}

	seed(cfg, db)

	// Initialize fiber session store
	sessionStorage := sessionmysql.New(sessionmysql.Config{
		ConnectionURI: dsn.Create(cfg),
		Table:         "sessions",
	})

	session.Init(sessionStorage)

	// Initialize PowerDNS client
	if err = powerdns.Open(db); err != nil {
		log.Warn().Err(err).Msg("failed to initialize PowerDNS client - server configuration features will be unavailable")
		log.Info().Msg("PowerDNS client will be available after configuring server settings")
	} else {
		log.Info().Msg("PowerDNS client initialized successfully")

		// Test the connection
		if err = powerdns.Engine.Test(); err != nil {
			log.Warn().Err(err).Msg("PowerDNS API connection test failed - please verify server settings")
		}
	}

	return &Daemon{
		webService: *web.New(cfg, db),
	}
}
