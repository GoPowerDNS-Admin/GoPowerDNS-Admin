package config

import (
	"errors"
)

var (
	// ErrEmptyURL error if config webserver.URL is empty.
	ErrEmptyURL = errors.New("toml config webserver.url can not be empty")

	// ErrWebServerPortCanNotBeZero error if config webserver listening port is 0.
	ErrWebServerPortCanNotBeZero = errors.New("toml config webserver.port listening port can not be 0")
)
