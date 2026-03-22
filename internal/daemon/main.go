package daemon

import (
	"fmt"

	sessionmysql "github.com/gofiber/storage/mysql/v2"
	sessionpostgres "github.com/gofiber/storage/postgres/v3"
	sessionsqlite "github.com/gofiber/storage/sqlite3"
	"github.com/rs/zerolog/log"
	gormmysql "gorm.io/driver/mysql"
	gormpostgres "gorm.io/driver/postgres"
	gormsqlite "gorm.io/driver/sqlite"
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
	cfg        *config.Config
	webService web.Service
}

// Start starts the Daemon's web service on the configured port.
func (d *Daemon) Start() error {
	addr := fmt.Sprintf(":%d", d.cfg.Webserver.Port)

	return d.webService.Start(addr)
}

// New creates a new Daemon instance with the provided configuration.
func New(cfg *config.Config) *Daemon {
	if cfg == nil {
		log.Fatal().Msg("config is nil")
		return nil
	}

	db, sessionStorage := openDB(cfg)

	if err := db.AutoMigrate(
		&models.User{},
		&models.Setting{},
		&models.Role{},
		&models.Permission{},
		&models.RolePermission{},
		&models.Group{},
		&models.GroupMapping{},
		&models.UserGroup{},
		&models.ActivityLog{},
		&models.Tag{},
		&models.ZoneTag{},
		&models.UserTag{},
		&models.GroupTag{},
	); err != nil {
		log.Fatal().Err(err).Msg("failed to migrate database")
	}

	seed(cfg, db)

	session.Init(sessionStorage)

	// Initialize PowerDNS client
	if err := powerdns.Open(db); err != nil {
		log.Warn().Err(err).Msg("failed to initialize PowerDNS client - server configuration features will be unavailable")
		log.Info().Msg("PowerDNS client will be available after configuring server settings")
	} else {
		log.Info().Msg("PowerDNS client initialized successfully")

		if err = powerdns.Engine.Test(); err != nil {
			log.Warn().Err(err).Msg("PowerDNS API connection test failed - please verify server settings")
		}
	}

	return &Daemon{
		cfg:        cfg,
		webService: *web.New(cfg, db),
	}
}

// openDB opens the GORM database and session storage based on cfg.DB.GormEngine.
// Supported values: "mysql" (default), "postgres".
func openDB(cfg *config.Config) (*gorm.DB, session.StorageBackend) {
	driver := cfg.DB.GormEngine
	if driver == "" {
		driver = "mysql"
	}

	var (
		db             *gorm.DB
		err            error
		sessionStorage session.StorageBackend
	)

	switch driver {
	case "sqlite":
		log.Info().Msg("using SQLite database driver")

		db, err = gorm.Open(gormsqlite.Open(dsn.CreateSQLite(cfg)), &gorm.Config{})
		if err != nil {
			log.Fatal().Err(err).Msg("failed to connect database")
		}

		sessionStorage = sessionsqlite.New(sessionsqlite.Config{
			Database: dsn.CreateSQLite(cfg) + "-sessions.db",
			Table:    "sessions",
		})

	case "postgres":
		log.Info().Msg("using PostgreSQL database driver")

		db, err = gorm.Open(gormpostgres.Open(dsn.CreatePostgres(cfg)), &gorm.Config{})
		if err != nil {
			log.Fatal().Err(err).Msg("failed to connect database")
		}

		sessionStorage = sessionpostgres.New(sessionpostgres.Config{
			ConnectionURI: dsn.CreatePostgresURL(cfg),
			Table:         "sessions",
		})

	default:
		if driver != "mysql" {
			log.Warn().Str("driver", driver).Msg("unknown database driver, falling back to mysql")
		} else {
			log.Info().Msg("using MySQL database driver")
		}

		db, err = gorm.Open(gormmysql.Open(dsn.Create(cfg)), &gorm.Config{})
		if err != nil {
			log.Fatal().Err(err).Msg("failed to connect database")
		}

		sessionStorage = sessionmysql.New(sessionmysql.Config{
			ConnectionURI: dsn.Create(cfg),
			Table:         "sessions",
		})
	}

	return db, sessionStorage
}
