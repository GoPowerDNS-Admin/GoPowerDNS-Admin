package zoneedit

import "strings"

// normalizeZoneName ensures the zone name has a trailing dot.
func normalizeZoneName(name string) string {
	if !strings.HasSuffix(name, ".") {
		return name + "."
	}

	return name
}
