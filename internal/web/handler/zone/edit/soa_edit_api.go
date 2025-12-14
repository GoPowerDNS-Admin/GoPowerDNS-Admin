package zoneedit

import pdnsapi "github.com/joeig/go-powerdns/v3"

// getSOAEditAPIFromZone extracts SOA-EDIT-API value from zone with a default.
func getSOAEditAPIFromZone(zone *pdnsapi.Zone) SOAEditAPI {
	if zone.SOAEditAPI != nil && *zone.SOAEditAPI != "" {
		return SOAEditAPI(*zone.SOAEditAPI)
	}

	return SOAEditAPIDefault
}
