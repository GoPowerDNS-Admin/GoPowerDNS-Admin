// Package dsn provides Data Source Name construction utilities for database connections.
package dsn

import (
	"fmt"
	"net"
	"strconv"

	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/config"
)

// Create builds the MySQL Data Source Name from the configuration.
func Create(dbCfg *config.Config) string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?%s",
		dbCfg.DB.User,
		dbCfg.DB.Password,
		dbCfg.DB.Host,
		dbCfg.DB.Port,
		dbCfg.DB.Name,
		dbCfg.DB.Extras,
	)
}

// CreatePostgres builds the PostgreSQL Data Source Name from the configuration.
// The format follows the libpq keyword/value connection string convention used by gorm's postgres driver.
func CreatePostgres(dbCfg *config.Config) string {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d",
		dbCfg.DB.Host,
		dbCfg.DB.User,
		dbCfg.DB.Password,
		dbCfg.DB.Name,
		dbCfg.DB.Port,
	)

	if dbCfg.DB.Extras != "" {
		dsn += " sslmode=" + dbCfg.DB.Extras
	} else {
		dsn += " sslmode=disable"
	}

	return dsn
}

// CreateSQLite returns the SQLite database file path from the configuration.
// Only cfg.DB.Name is used; host, port, user, and password are ignored.
func CreateSQLite(dbCfg *config.Config) string {
	return dbCfg.DB.Name
}

// CreatePostgresURL builds a PostgreSQL connection URL from the configuration.
// This format is used by gofiber session storage.
func CreatePostgresURL(dbCfg *config.Config) string {
	sslmode := "disable"
	if dbCfg.DB.Extras != "" {
		sslmode = dbCfg.DB.Extras
	}

	hostPort := net.JoinHostPort(dbCfg.DB.Host, strconv.Itoa(dbCfg.DB.Port))

	return fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=%s",
		dbCfg.DB.User,
		dbCfg.DB.Password,
		hostPort,
		dbCfg.DB.Name,
		sslmode,
	)
}
