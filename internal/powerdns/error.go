package powerdns

import (
	"errors"
	"net"
)

const (
	// ErrMsgClientNotInitialized is the error message when the PowerDNS client is not initialized.
	ErrMsgClientNotInitialized = "PowerDNS client not initialized"

	// ErrMsgClientNotInitializedDetailed is the detailed user-facing error message.
	ErrMsgClientNotInitializedDetailed = "PowerDNS client not initialized. Please configure PowerDNS server settings."

	// ErrMsgServerUnreachable is the user-facing message when the PowerDNS server cannot be reached.
	ErrMsgServerUnreachable = "PowerDNS server is unreachable. Please check that the server is running" +
		" and the configured address is correct."
)

var (
	// ErrClientNotInitialized is returned when the PowerDNS client is not initialized.
	ErrClientNotInitialized = errors.New(ErrMsgClientNotInitialized)
)

// IsServerUnreachable reports whether err indicates a network-level failure
// reaching the PowerDNS server (connection refused, timeout, DNS failure, etc.).
func IsServerUnreachable(err error) bool {
	if err == nil {
		return false
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}

	var opErr *net.OpError

	return errors.As(err, &opErr)
}
