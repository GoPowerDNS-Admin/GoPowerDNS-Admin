package config

import (
	"time"

	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/logger"
)

// Session settings.
type Session struct {
	ExpiryTime time.Duration
}

// Config overall data structure.
type Config struct {
	DevMode   bool // enable dev mode for development
	DB        DB
	Log       logger.Log
	Title     string
	Webserver Webserver
	Record    Record // DNS record type settings
}

// Webserver implement webserver settings.
type Webserver struct {
	BrowseStatic        bool    // enable static file browsing (for development purposes only)
	CacheEnabled        bool    // true = enable cache, false = disable cache
	CleanPath           bool    // use clean path middleware to allow multi slash requests
	DisableRecover      bool    // disable recover middleware
	Domain              string  // domain name for the webserver
	Port                int     // listening port for the webserver
	ShutDownTime        int     // wait time for shutdown
	URL                 string  // base url for the webserver
	CookieEncryptionKey string  // encryption key for cookies
	Argon2Salt          string  // salt for argon2 hashing
	Session             Session // session settings
}

// RecordTypeSettings defines whether a DNS record type can be edited in forward or reverse zones.
type RecordTypeSettings struct {
	Forward bool `form:"forward" json:"forward"` // allow editing in forward zones
	Reverse bool `form:"reverse" json:"reverse"` // allow editing in reverse zones
}

// Record holds configuration for DNS record type editing permissions.
// Keys are DNS record types (A, AAAA, CNAME, etc.)
type Record map[string]RecordTypeSettings
