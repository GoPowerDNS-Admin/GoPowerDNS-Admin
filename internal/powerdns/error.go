package powerdns

import (
	"errors"
)

const (
	// ErrMsgClientNotInitialized is the error message when the PowerDNS client is not initialized.
	ErrMsgClientNotInitialized = "PowerDNS client not initialized"

	// ErrMsgClientNotInitializedDetailed is the detailed user-facing error message.
	ErrMsgClientNotInitializedDetailed = "PowerDNS client not initialized. Please configure PowerDNS server settings."
)

var (
	// ErrClientNotInitialized is returned when the PowerDNS client is not initialized.
	ErrClientNotInitialized = errors.New(ErrMsgClientNotInitialized)
)
