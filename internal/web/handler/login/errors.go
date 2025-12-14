// Package login provides HTTP handlers and helpers for user authentication.
//
// This file defines exported error values used throughout the login flow.
package login

import "errors"

var (
	// ErrInvalidFormData is returned when the submitted login form cannot be parsed
	// or fails validation.
	ErrInvalidFormData = errors.New("invalid form data")

	// ErrNoAuthMethod is returned when no authentication method is configured or
	// none of the available methods are enabled.
	ErrNoAuthMethod = errors.New("no authentication method available")

	// ErrLocalAuthDisabled is returned when local (username/password) authentication
	// is disabled by configuration.
	ErrLocalAuthDisabled = errors.New("local authentication is disabled")

	// ErrLDAPAuthDisabled is returned when LDAP authentication is disabled by
	// configuration.
	ErrLDAPAuthDisabled = errors.New("ldap authentication is disabled")

	// ErrInvalidAuthMethod is returned when a requested authentication method is
	// unknown or not permitted.
	ErrInvalidAuthMethod = errors.New("invalid authentication method")

	// ErrInvalidCredentials is returned when the provided username and/or password
	// are not valid for the selected authentication method.
	ErrInvalidCredentials = errors.New("invalid username or password")

	// ErrInternalServerError is returned for unexpected failures during the login
	// process.
	ErrInternalServerError = errors.New("internal server error")
)
