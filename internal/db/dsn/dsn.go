// Package dsn provides Data Source Name construction utilities for database connections.
package dsn

import (
	"fmt"

	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/config"
)

// Create builds the Data Source Name from the configuration.
func Create(dbCfg *config.Config) string {
	out := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?%s",
		dbCfg.DB.User,
		dbCfg.DB.Password,
		dbCfg.DB.Host,
		dbCfg.DB.Port,
		dbCfg.DB.Name,
		dbCfg.DB.Extras,
	)

	return out
}
