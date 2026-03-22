package config

import (
	"errors"
)

var (
	// ErrEmptyURL error if config webserver.URL is empty.
	ErrEmptyURL = errors.New("toml config webserver.url can not be empty")

	// ErrWebServerPortCanNotBeZero error if config webserver listening port is 0.
	ErrWebServerPortCanNotBeZero = errors.New("toml config webserver.port listening port can not be 0")

	// ErrPlaceholderCookieKey is returned when CookieEncryptionKey still holds
	// the default placeholder value.
	ErrPlaceholderCookieKey = errors.New("webserver.cookieencryptionkey must be changed from the default placeholder")

	// ErrCookieKeyTooShort is returned when CookieEncryptionKey is set but too short.
	ErrCookieKeyTooShort = errors.New("webserver.cookieencryptionkey must be at least 32 characters")

	// ErrPlaceholderArgon2Salt is returned when Argon2Salt still holds the
	// default placeholder value.
	ErrPlaceholderArgon2Salt = errors.New("webserver.argon2salt must be changed from the default placeholder")

	// ErrArgon2SaltTooShort is returned when Argon2Salt is set but too short.
	ErrArgon2SaltTooShort = errors.New("webserver.argon2salt must be at least 16 characters")

	// ErrNoAuthProviderEnabled is returned when no authentication provider is
	// enabled.
	ErrNoAuthProviderEnabled = errors.New(
		"at least one auth provider must be enabled (auth.localdb, auth.oidc, or auth.ldap)",
	)

	// ErrOIDCMissingProviderURL is returned when OIDC is enabled, but ProviderURL
	// is empty.
	ErrOIDCMissingProviderURL = errors.New("auth.oidc.provider_url is required when OIDC is enabled")

	// ErrOIDCMissingClientID is returned when OIDC is enabled, but ClientID is
	// empty.
	ErrOIDCMissingClientID = errors.New("auth.oidc.client_id is required when OIDC is enabled")

	// ErrOIDCMissingClientSecret is returned when OIDC is enabled, but
	// ClientSecret is empty.
	ErrOIDCMissingClientSecret = errors.New("auth.oidc.client_secret is required when OIDC is enabled")

	// ErrOIDCMissingRedirectURL is returned when OIDC is enabled, but RedirectURL
	// is empty.
	ErrOIDCMissingRedirectURL = errors.New("auth.oidc.redirect_url is required when OIDC is enabled")

	// ErrLDAPMissingHost is returned when LDAP is enabled, but Host is empty.
	ErrLDAPMissingHost = errors.New("auth.ldap.host is required when LDAP is enabled")

	// ErrLDAPMissingPort is returned when LDAP is enabled, but Port is zero.
	ErrLDAPMissingPort = errors.New("auth.ldap.port is required when LDAP is enabled")

	// ErrLDAPMissingBaseDN is returned when LDAP is enabled, but BaseDN is empty.
	ErrLDAPMissingBaseDN = errors.New("auth.ldap.base_dn is required when LDAP is enabled")

	// ErrDBMissingEngine is returned when no database engine is configured.
	ErrDBMissingEngine = errors.New("db.gormengine is required (sqlite, mysql, or postgres)")
)
