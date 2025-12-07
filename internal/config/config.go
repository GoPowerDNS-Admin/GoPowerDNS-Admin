// Package config handles input from etc/*.toml files
package config

import (
	"bytes"
	"encoding/json"
	"os"

	"github.com/pkg/errors"

	"github.com/BurntSushi/toml"
)

// ReadConfig from config file.
func ReadConfig(path string) (Config, error) {
	var (
		c             Config
		JSONConfigEnv string
		err           error
	)

	// Read main configuration
	if path == "" {
		path = "./etc/"
	}

	if _, err = toml.DecodeFile(path+"main.toml", &c); err != nil {
		return Config{}, errors.Wrap(err, "failed to read main config file")
	}

	// override it from env
	JSONConfigEnv = os.Getenv("GO_POWERDNS_ADMIN_CONFIG_JSON")

	if JSONConfigEnv != "" {
		c, err = decodeAndMergeConfig(c, JSONConfigEnv)
		if err != nil {
			return c, err
		}
	}

	return c, validate(c)
}

func decodeAndMergeConfig(c Config, configAsJSON string) (Config, error) {
	err := json.Unmarshal([]byte(configAsJSON), &c)
	if err != nil {
		return Config{}, errors.Wrap(err, "failed to read main config file")
	}

	return c, nil
}

// DumpConfig config as TOML String.
func DumpConfig(c Config) (string, error) {
	var buffer bytes.Buffer
	t := toml.NewEncoder(&buffer)

	if err := t.Encode(c); err != nil {
		return "", err //nolint: wrapcheck
	}

	return buffer.String(), nil
}

// DumpConfigJSON config as JSON String.
func DumpConfigJSON(c Config) (string, error) {
	var buffer bytes.Buffer
	j := json.NewEncoder(&buffer)
	j.SetIndent("", "  ")

	if err := j.Encode(c); err != nil {
		return "", err //nolint: wrapcheck
	}

	return buffer.String(), nil
}

// validate minimal config settings for marvin.
// Validates only a very small part of the params needed
// by marvin.
func validate(c Config) error {
	// validate webserver listening port
	invalidErrMessage := "invalid config"

	if c.Webserver.Port == 0 {
		return errors.Wrap(ErrWebServerPortCanNotBeZero, invalidErrMessage)
	}

	// validate access-control-allow-origin
	if c.Webserver.URL == "" {
		return errors.Wrap(ErrEmptyURL, invalidErrMessage)
	}

	if c.Webserver.ShutDownTime == 0 {
		c.Webserver.ShutDownTime = 5 // set default of 5 seconds
	}

	return nil
}
