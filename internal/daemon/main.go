package daemon

import (
	sessionmysql "github.com/gofiber/storage/mysql"
	"github.com/rs/zerolog/log"
	gormmysql "gorm.io/driver/mysql"
	"gorm.io/gorm"

	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/config"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/db/dsn"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/db/models"
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
		panic("failed to connect database")
	}

	if err = db.AutoMigrate(
		&models.User{},
		&models.Setting{},
	); err != nil {
		panic("failed to migrate database")
	}

	seed(cfg, db)

	// Initialize fiber session store
	sessionStorage := sessionmysql.New(sessionmysql.Config{
		ConnectionURI: dsn.Create(cfg),
		Table:         "sessions",
	})

	session.Init(sessionStorage)

	return &Daemon{
		webService: *web.New(cfg, db),
	}
}
