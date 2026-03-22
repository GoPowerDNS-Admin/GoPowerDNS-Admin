// Package config handles input from etc/*.toml files
package config

import (
	"bytes"
	"encoding/json"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

// EnvPrefix is the prefix for environment variable overrides.
// e.g. GPDNS_WEBSERVER_PORT overrides webserver.port.
const EnvPrefix = "GPDNS"

// ReadConfig reads configuration from a directory path (e.g. "./etc/") or a
// specific overlay file (e.g. "etc/local/dev.toml").
//
// When a .toml file path is given, the main.toml is loaded first from the parent
// directory and the given file is merged on top, overriding only the keys it
// defines. Environment variables prefixed with GPDNS_ take the highest priority.
func ReadConfig(path string) (Config, error) {
	if path == "" {
		path = "./etc/"
	}

	v := viper.New()
	v.SetConfigType("toml")

	// Env var support: GPDNS_WEBSERVER_PORT → webserver.port
	v.SetEnvPrefix(EnvPrefix)
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	var mainConfig, overlayFile string

	if strings.HasSuffix(path, ".toml") {
		// A specific overlay file was given; derive main.toml from its grandparent dir.
		// e.g. "etc/local/dev.toml" → main = "etc/main.toml", overlay = "etc/local/dev.toml"
		overlayFile = path
		parts := strings.Split(strings.TrimSuffix(path, ".toml"), "/")

		baseDir := strings.Join(parts[:len(parts)-2], "/")
		if baseDir == "" {
			baseDir = "."
		}

		mainConfig = baseDir + "/main.toml"
	} else {
		dir := strings.TrimRight(path, "/")
		mainConfig = dir + "/main.toml"
	}

	v.SetConfigFile(mainConfig)

	if err := v.ReadInConfig(); err != nil {
		return Config{}, errors.Wrap(err, "failed to read main config file")
	}

	if overlayFile != "" {
		v.SetConfigFile(overlayFile)

		if err := v.MergeInConfig(); err != nil {
			return Config{}, errors.Wrap(err, "failed to merge overlay config file")
		}
	}

	var c Config
	if err := v.Unmarshal(&c); err != nil {
		return Config{}, errors.Wrap(err, "failed to unmarshal config")
	}

	// Bridge log.level → Log.LogLevel (logger.Log uses a different field name).
	if c.Log.LogLevel == "" {
		c.Log.LogLevel = v.GetString("log.level")
	}

	// Viper lowercases all keys; DNS record types must be uppercase (A, AAAA, …).
	if len(c.Record) > 0 {
		upper := make(Record, len(c.Record))
		for k, settings := range c.Record {
			upper[strings.ToUpper(k)] = settings
		}

		c.Record = upper
	}

	if errValidate := validate(&c); errValidate != nil {
		return c, errValidate
	}

	return c, nil
}

// DumpConfigJSON serializes the config as an indented JSON string.
func DumpConfigJSON(c *Config) (string, error) {
	var buffer bytes.Buffer

	j := json.NewEncoder(&buffer)
	j.SetIndent("", "  ")

	if err := j.Encode(c); err != nil {
		return "", err
	}

	return buffer.String(), nil
}

const (
	placeholderSecret  = "change_this_to_a_random_string"
	minCookieKeyLength = 32
	minArgon2SaltLen   = 16
)

// validate checks the minimal required config fields.
func validate(c *Config) error {
	const invalidErrMessage = "invalid config"

	if c.Webserver.Port == 0 {
		return errors.Wrap(ErrWebServerPortCanNotBeZero, invalidErrMessage)
	}

	if c.Webserver.URL == "" {
		return errors.Wrap(ErrEmptyURL, invalidErrMessage)
	}

	if c.Webserver.ShutDownTime == 0 {
		c.Webserver.ShutDownTime = 5
	}

	if err := validateSecrets(c); err != nil {
		return errors.Wrap(err, invalidErrMessage)
	}

	if err := validateDB(c); err != nil {
		return errors.Wrap(err, invalidErrMessage)
	}

	if err := validateAuth(c); err != nil {
		return errors.Wrap(err, invalidErrMessage)
	}

	certSet := c.Webserver.TLSCertFile != ""
	keySet := c.Webserver.TLSKeyFile != ""

	if certSet != keySet {
		return errors.Wrap(ErrTLSPartialConfig, invalidErrMessage)
	}

	if err := validateACME(c); err != nil {
		return errors.Wrap(err, invalidErrMessage)
	}

	return nil
}

func validateACME(c *Config) error {
	if !c.Webserver.ACMEEnabled {
		return nil
	}

	if c.Webserver.TLSCertFile != "" || c.Webserver.TLSKeyFile != "" {
		return ErrACMEConflict
	}

	if c.Webserver.ACMEDomain == "" {
		return ErrACMEMissingDomain
	}

	if c.Webserver.ACMEEmail == "" {
		return ErrACMEMissingEmail
	}

	if c.Webserver.ACMECacheDir == "" {
		return ErrACMEMissingCacheDir
	}

	return nil
}

func validateSecrets(c *Config) error {
	key := c.Webserver.CookieEncryptionKey
	if key == placeholderSecret {
		return ErrPlaceholderCookieKey
	}

	if key != "" && len(key) < minCookieKeyLength {
		return ErrCookieKeyTooShort
	}

	salt := c.Webserver.Argon2Salt
	if salt == placeholderSecret {
		return ErrPlaceholderArgon2Salt
	}

	if salt != "" && len(salt) < minArgon2SaltLen {
		return ErrArgon2SaltTooShort
	}

	return nil
}

func validateDB(c *Config) error {
	if c.DB.GormEngine == "" {
		return ErrDBMissingEngine
	}

	return nil
}

func validateAuth(c *Config) error {
	if !c.Auth.LocalDB.Enabled && !c.Auth.OIDC.Enabled && !c.Auth.LDAP.Enabled {
		return ErrNoAuthProviderEnabled
	}

	if c.Auth.OIDC.Enabled {
		if c.Auth.OIDC.ProviderURL == "" {
			return ErrOIDCMissingProviderURL
		}

		if c.Auth.OIDC.ClientID == "" {
			return ErrOIDCMissingClientID
		}

		if c.Auth.OIDC.ClientSecret == "" {
			return ErrOIDCMissingClientSecret
		}

		if c.Auth.OIDC.RedirectURL == "" {
			return ErrOIDCMissingRedirectURL
		}
	}

	if c.Auth.LDAP.Enabled {
		if c.Auth.LDAP.Host == "" {
			return ErrLDAPMissingHost
		}

		if c.Auth.LDAP.Port == 0 {
			return ErrLDAPMissingPort
		}

		if c.Auth.LDAP.BaseDN == "" {
			return ErrLDAPMissingBaseDN
		}
	}

	return nil
}

// TLSEnabled reports whether TLS is configured (both cert and key are set).
func (w *Webserver) TLSEnabled() bool {
	return w.TLSCertFile != "" && w.TLSKeyFile != ""
}
