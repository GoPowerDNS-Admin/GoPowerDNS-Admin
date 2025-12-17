package zoneedit

import (
	"strings"
)

// zoneIsReverse checks if the given zone name is a reverse DNS zone.
func zoneIsReverse(zoneName string) (reverse bool) {
	switch {
	case strings.HasSuffix(zoneName, "ip6.arpa."):
		reverse = true

	case strings.HasSuffix(zoneName, "in-addr.arpa."):
		reverse = true
	}

	return
}
