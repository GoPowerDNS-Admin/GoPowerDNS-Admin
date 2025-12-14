package zoneedit

import "strings"

// getDisplayNameForZone returns a user-friendly name for a record by stripping the zone suffix.
func getDisplayNameForZone(fullName, zoneName string) string {
	// If it's the zone itself, return @
	if fullName == zoneName || fullName == strings.TrimSuffix(zoneName, ".") {
		return "@"
	}
	// Strip the zone name suffix for display
	zoneWithoutDot := strings.TrimSuffix(zoneName, ".")
	if strings.HasSuffix(fullName, "."+zoneWithoutDot+".") {
		return strings.TrimSuffix(fullName, "."+zoneWithoutDot+".")
	} else if strings.HasSuffix(fullName, "."+zoneWithoutDot) {
		return strings.TrimSuffix(fullName, "."+zoneWithoutDot)
	}

	return fullName
}
