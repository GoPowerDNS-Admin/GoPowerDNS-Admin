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
	Record    Record `toml:"record"` // DNS record type settings
	Auth      Auth   // Authentication settings
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
	Description string `form:"description" json:"description" toml:"description"`
	Forward     bool   `form:"forward"     json:"forward"     toml:"forward"`
	Reverse     bool   `form:"reverse"     json:"reverse"     toml:"reverse"`
	Help        string `form:"help"        json:"help"        toml:"help"`
}

// Record holds configuration for DNS record type editing permissions.
// Keys are DNS record types (A, AAAA, CNAME, etc.)
type Record map[string]RecordTypeSettings

// Auth holds authentication configuration.
type Auth struct {
	LocalDB LocalDBAuth // Local database authentication
	OIDC    OIDCAuth    // OpenID Connect authentication
	LDAP    LDAPAuth    // LDAP authentication
}

// LocalDBAuth holds local database authentication settings.
type LocalDBAuth struct {
	Enabled bool `toml:"enabled"` // Enable local database authentication
}

// OIDCAuth holds OIDC authentication settings.
type OIDCAuth struct {
	Enabled      bool     `toml:"enabled"`       // Enable OIDC authentication
	ProviderURL  string   `toml:"provider_url"`  // OIDC provider URL (e.g., https://accounts.google.com)
	ClientID     string   `toml:"client_id"`     // OAuth2 client ID
	ClientSecret string   `toml:"client_secret"` // OAuth2 client secret
	RedirectURL  string   `toml:"redirect_url"`  // OAuth2 redirect URL
	Scopes       []string `toml:"scopes"`        // OAuth2 scopes (default: openid, profile, email)
	GroupsClaim  string   `toml:"groups_claim"`  // Claim name for groups (default: "groups")
}

// LDAPAuth holds LDAP authentication settings.
type LDAPAuth struct {
	Enabled         bool     `toml:"enabled"`           // Enable LDAP authentication
	Host            string   `toml:"host"`              // LDAP server hostname
	Port            int      `toml:"port"`              // LDAP server port (389 for LDAP, 636 for LDAPS)
	UseSSL          bool     `toml:"use_ssl"`           // Use LDAPS (SSL/TLS)
	UseTLS          bool     `toml:"use_tls"`           // Use StartTLS
	SkipVerify      bool     `toml:"skip_verify"`       // Skip TLS certificate verification
	BindDN          string   `toml:"bind_dn"`           // DN to bind with for searches
	BindPassword    string   `toml:"bind_password"`     // Password for bind DN
	BaseDN          string   `toml:"base_dn"`           // Base DN for user searches
	UserFilter      string   `toml:"user_filter"`       // LDAP filter for users (e.g., "(uid={username})")
	GroupBaseDN     string   `toml:"group_base_dn"`     // Base DN for group searches
	GroupFilter     string   `toml:"group_filter"`      // LDAP filter for groups (e.g., "(member={userdn})")
	GroupMemberAttr string   `toml:"group_member_attr"` // Attribute for group membership (default: "member")
	UsernameAttr    string   `toml:"username_attr"`     // Attribute for username (default: "uid")
	EmailAttr       string   `toml:"email_attr"`        // Attribute for email (default: "mail")
	FirstNameAttr   string   `toml:"first_name_attr"`   // Attribute for first name (default: "givenName")
	LastNameAttr    string   `toml:"last_name_attr"`    // Attribute for last name (default: "sn")
	GroupNameAttr   string   `toml:"group_name_attr"`   // Attribute for group name (default: "cn")
	Timeout         int      `toml:"timeout"`           // Connection timeout in seconds
	SearchAttrs     []string `toml:"search_attrs"`      // Additional attributes to retrieve
}
