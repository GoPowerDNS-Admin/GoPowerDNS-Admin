package powerdns

import (
	"errors"
)

var (
	// ErrClientNotInitialized is returned when the PowerDNS client is not initialized.
	ErrClientNotInitialized = errors.New("PowerDNS client not initialized")
)
