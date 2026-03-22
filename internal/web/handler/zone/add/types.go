package zoneadd

// ZoneKind represents the zone kind/type.
type ZoneKind string

const (
	// ZoneKindNative represents a Native zone.
	ZoneKindNative ZoneKind = "Native"

	// ZoneKindMaster represents a Primary/Master zone.
	ZoneKindMaster ZoneKind = "Master"

	// ZoneKindSlave represents a Secondary/Slave zone.
	ZoneKindSlave ZoneKind = "Slave"
)

// SOAEditAPI represents the SOA-EDIT-API setting.
type SOAEditAPI string

const (
	// SOAEditAPIDefault uses the default SOA-EDIT-API setting.
	SOAEditAPIDefault SOAEditAPI = "DEFAULT"

	// SOAEditAPIIncrease increments the serial number.
	SOAEditAPIIncrease SOAEditAPI = "INCREASE"

	// SOAEditAPIEpoch sets the serial to the current epoch timestamp.
	SOAEditAPIEpoch SOAEditAPI = "EPOCH"

	// SOAEditAPIOff disables SOA-EDIT-API.
	SOAEditAPIOff SOAEditAPI = "OFF"
)

// ZoneType distinguishes forward from reverse zone creation modes.
type ZoneType string

const (
	// ZoneTypeForward is a standard forward DNS zone.
	ZoneTypeForward ZoneType = "forward"

	// ZoneTypeReverseIPv4 is a reverse DNS zone for IPv4.
	ZoneTypeReverseIPv4 ZoneType = "reverse-ipv4"

	// ZoneTypeReverseIPv6 is a reverse DNS zone for IPv6.
	ZoneTypeReverseIPv6 ZoneType = "reverse-ipv6"
)

// ZoneForm represents the form data for creating a new zone.
type ZoneForm struct {
	ZoneType       ZoneType   `form:"zone_type"`
	Name           string     `form:"name"`
	ReverseNetwork string     `form:"reverse_network"` // CIDR for reverse zone conversion
	Kind           ZoneKind   `form:"kind"            validate:"required,oneof=Native Master Slave"`
	SOAEditAPI     SOAEditAPI `form:"soa_edit_api"    validate:"required,oneof=DEFAULT INCREASE EPOCH OFF"`
	Masters        string     `form:"masters"` // Comma-separated list for Slave zones
}
