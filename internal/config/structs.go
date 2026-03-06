package config

import (
	"time"

	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/logger"
)

// Session settings.
type Session struct {
	ExpiryTime time.Duration `mapstructure:"expirytime"`
}

// Config overall data structure.
type Config struct {
	DevMode   bool       `mapstructure:"devmode"`
	DB        DB         `mapstructure:"db"`
	Log       logger.Log `mapstructure:"log"`
	Title     string     `mapstructure:"title"`
	Webserver Webserver  `mapstructure:"webserver"`
	Record    Record     `mapstructure:"record"`
	Auth      Auth       `mapstructure:"auth"`
}

// Webserver implement webserver settings.
type Webserver struct {
	BrowseStatic        bool    `mapstructure:"browsestatic"`
	CacheEnabled        bool    `mapstructure:"cacheenabled"`
	CleanPath           bool    `mapstructure:"cleanpath"`
	DisableRecover      bool    `mapstructure:"disablerecover"`
	Domain              string  `mapstructure:"domain"`
	Port                int     `mapstructure:"port"`
	ShutDownTime        int     `mapstructure:"shutdowntime"`
	URL                 string  `mapstructure:"url"`
	CookieEncryptionKey string  `mapstructure:"cookieencryptionkey"`
	Argon2Salt          string  `mapstructure:"argon2salt"`
	Session             Session `mapstructure:"session"`
}

// RecordTypeSettings defines whether a DNS record type can be edited in forward or reverse zones.
type RecordTypeSettings struct {
	Description string `form:"description" json:"description" mapstructure:"description"`
	Forward     bool   `form:"forward"     json:"forward"     mapstructure:"forward"`
	Reverse     bool   `form:"reverse"     json:"reverse"     mapstructure:"reverse"`
	Help        string `form:"help"        json:"help"        mapstructure:"help"`
}

// Record holds configuration for DNS record type editing permissions.
// Keys are DNS record types (A, AAAA, CNAME, etc.)
type Record map[string]RecordTypeSettings

// Auth holds authentication configuration.
type Auth struct {
	LocalDB LocalDBAuth `mapstructure:"localdb"`
	OIDC    OIDCAuth    `mapstructure:"oidc"`
	LDAP    LDAPAuth    `mapstructure:"ldap"`
}

// LocalDBAuth holds local database authentication settings.
type LocalDBAuth struct {
	Enabled bool `mapstructure:"enabled"`
}

// OIDCAuth holds OIDC authentication settings.
type OIDCAuth struct {
	Enabled      bool     `mapstructure:"enabled"`
	ProviderURL  string   `mapstructure:"provider_url"`
	ClientID     string   `mapstructure:"client_id"`
	ClientSecret string   `mapstructure:"client_secret"`
	RedirectURL  string   `mapstructure:"redirect_url"`
	Scopes       []string `mapstructure:"scopes"`
	GroupsClaim  string   `mapstructure:"groups_claim"`
}

// LDAPAuth holds LDAP authentication settings.
type LDAPAuth struct {
	Enabled         bool     `mapstructure:"enabled"`
	Host            string   `mapstructure:"host"`
	Port            int      `mapstructure:"port"`
	UseSSL          bool     `mapstructure:"use_ssl"`
	UseTLS          bool     `mapstructure:"use_tls"`
	SkipVerify      bool     `mapstructure:"skip_verify"`
	BindDN          string   `mapstructure:"bind_dn"`
	BindPassword    string   `mapstructure:"bind_password"`
	BaseDN          string   `mapstructure:"base_dn"`
	UserFilter      string   `mapstructure:"user_filter"`
	GroupBaseDN     string   `mapstructure:"group_base_dn"`
	GroupFilter     string   `mapstructure:"group_filter"`
	GroupMemberAttr string   `mapstructure:"group_member_attr"`
	UsernameAttr    string   `mapstructure:"username_attr"`
	EmailAttr       string   `mapstructure:"email_attr"`
	FirstNameAttr   string   `mapstructure:"first_name_attr"`
	LastNameAttr    string   `mapstructure:"last_name_attr"`
	GroupNameAttr   string   `mapstructure:"group_name_attr"`
	Timeout         int      `mapstructure:"timeout"`
	SearchAttrs     []string `mapstructure:"search_attrs"`
}
