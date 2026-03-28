package zoneedit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"

	pdnsapi "github.com/joeig/go-powerdns/v3"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"

	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/activitylog"
	settingctrl "github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/db/controller/setting"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/powerdns"
)

const (
	zoneSettingsKey = "zone_settings"
	rrTypeAAAA      = "AAAA"
	nibbleMask      = 0x0f
)

// ZoneSettings holds per-zone application settings stored in the database.
type ZoneSettings struct {
	AutoPTR bool `json:"auto_ptr"`
}

// allZoneSettings is the top-level structure stored under zoneSettingsKey.
type allZoneSettings map[string]ZoneSettings

// loadZoneSettings returns the stored settings for the given zone, or defaults.
func loadZoneSettings(db *gorm.DB, zoneName string) ZoneSettings {
	row, err := settingctrl.Get(db, zoneSettingsKey)
	if err != nil {
		return ZoneSettings{}
	}

	var all allZoneSettings

	if err := json.Unmarshal(row.Value, &all); err != nil {
		return ZoneSettings{}
	}

	return all[zoneName]
}

// saveZoneSettings persists the settings for the given zone to the database.
func saveZoneSettings(db *gorm.DB, zoneName string, settings ZoneSettings) error {
	var all allZoneSettings

	row, err := settingctrl.Get(db, zoneSettingsKey)
	if err != nil && !errors.Is(err, settingctrl.ErrSettingNotFound) {
		return fmt.Errorf("load zone settings: %w", err)
	}

	if err == nil {
		if jsonErr := json.Unmarshal(row.Value, &all); jsonErr != nil {
			all = allZoneSettings{}
		}
	} else {
		all = allZoneSettings{}
	}

	all[zoneName] = settings

	data, err := json.Marshal(all)
	if err != nil {
		return fmt.Errorf("marshal zone settings: %w", err)
	}

	_, err = settingctrl.Set(db, zoneSettingsKey, data)

	return err
}

// ipv4PTRName converts an IPv4 address string to its PTR record name.
// Example: "192.0.2.1" → "1.2.0.192.in-addr.arpa.".
func ipv4PTRName(ip string) (string, error) {
	parsed := net.ParseIP(ip).To4()
	if parsed == nil {
		return "", fmt.Errorf("invalid IPv4 address: %q", ip)
	}

	return fmt.Sprintf("%d.%d.%d.%d.in-addr.arpa.", parsed[3], parsed[2], parsed[1], parsed[0]), nil
}

// ipv6PTRName converts an IPv6 address string to its PTR record name.
// Example: "2001:db8::1" → "1.0.0.0...0.8.b.d.0.1.0.0.2.ip6.arpa.".
func ipv6PTRName(ip string) (string, error) {
	p := net.ParseIP(ip)
	if p == nil || p.To4() != nil {
		return "", fmt.Errorf("invalid IPv6 address: %q", ip)
	}

	parsed := p.To16()

	nibbles := make([]string, 32)
	for i := range 16 {
		nibbles[i*2] = strconv.FormatUint(uint64(parsed[i]>>4), 16)
		nibbles[i*2+1] = strconv.FormatUint(uint64(parsed[i]&nibbleMask), 16)
	}

	reversed := make([]string, 32)
	for i, n := range nibbles {
		reversed[31-i] = n
	}

	return strings.Join(reversed, ".") + ".ip6.arpa.", nil
}

// ptrNameForIP returns the PTR record name for an IP given its RR type ("A" or "AAAA").
func ptrNameForIP(ip, rrType string) (string, error) {
	switch rrType {
	case "A":
		return ipv4PTRName(ip)
	case rrTypeAAAA:
		return ipv6PTRName(ip)
	default:
		return "", fmt.Errorf("unsupported type for PTR: %q", rrType)
	}
}

// findBestReverseZoneFromList returns the most-specific zone from the given
// slice that owns ptrName, or "" if none match.
func findBestReverseZoneFromList(ptrName string, reverseZones []string) string {
	best := ""

	for _, zn := range reverseZones {
		if ptrName != zn && !strings.HasSuffix(ptrName, "."+zn) {
			continue
		}

		if len(zn) > len(best) {
			best = zn
		}
	}

	return best
}

// findBestReverseZone returns the name of the most-specific reverse zone in
// PowerDNS that contains ptrName, or "" if none exists.
func findBestReverseZone(ctx context.Context, ptrName string) (string, error) {
	zones, err := powerdns.Engine.Zones.List(ctx)
	if err != nil {
		return "", fmt.Errorf("list zones: %w", err)
	}

	names := make([]string, 0, len(zones))

	for i := range zones {
		if zones[i].Name != nil && zoneIsReverse(*zones[i].Name) {
			names = append(names, *zones[i].Name)
		}
	}

	return findBestReverseZoneFromList(ptrName, names), nil
}

// buildExistingPTRsMap returns a map of PTR record name → reverse zone name
// for every A/AAAA record in records that has a matching PTR in one of the
// known reverse zones. Each unique reverse zone is fetched at most once.
func buildExistingPTRsMap(ctx context.Context, records []RecordData, reverseZoneNames []string) map[string]string {
	result := make(map[string]string)

	// Determine which PTR names to look for and which reverse zones to fetch.
	type candidate struct{ ptrName, reverseZone string }

	var candidates []candidate

	rzToFetch := make(map[string]bool)

	for _, r := range records {
		if r.Type != "A" && r.Type != rrTypeAAAA {
			continue
		}

		ptrName, err := ptrNameForIP(r.Content, r.Type)
		if err != nil {
			continue
		}

		rz := findBestReverseZoneFromList(ptrName, reverseZoneNames)
		if rz == "" {
			continue
		}

		candidates = append(candidates, candidate{ptrName, rz})
		rzToFetch[rz] = true
	}

	if len(candidates) == 0 {
		return result
	}

	// Fetch each unique reverse zone once and build a set of its PTR names.
	rzPTRs := make(map[string]map[string]bool)

	for rz := range rzToFetch {
		zone, err := powerdns.Engine.Zones.Get(ctx, rz)
		if err != nil {
			log.Warn().Err(err).Str("zone", rz).Msg("buildExistingPTRsMap: failed to fetch reverse zone")

			continue
		}

		ptrs := make(map[string]bool)

		for _, rr := range zone.RRsets {
			if rr.Name != nil && rr.Type != nil && string(*rr.Type) == "PTR" {
				ptrs[strings.ToLower(*rr.Name)] = true
			}
		}

		rzPTRs[rz] = ptrs
	}

	for _, c := range candidates {
		if ptrs, ok := rzPTRs[c.reverseZone]; ok && ptrs[strings.ToLower(c.ptrName)] {
			result[c.ptrName] = c.reverseZone
		}
	}

	return result
}

// ipsFromCurrentZone returns the IP addresses stored in the current zone's RRset
// for the given fully-qualified name and type.
func ipsFromCurrentZone(currentZone *pdnsapi.Zone, fqdn, rrType string) []string {
	if currentZone == nil {
		return nil
	}

	fqdn = strings.ToLower(fqdn)
	if !strings.HasSuffix(fqdn, ".") {
		fqdn += "."
	}

	for _, rr := range currentZone.RRsets {
		if rr.Name == nil || rr.Type == nil {
			continue
		}

		if !strings.EqualFold(*rr.Name, fqdn) || string(*rr.Type) != rrType {
			continue
		}

		var ips []string

		for _, r := range rr.Records {
			if r.Content != nil {
				ips = append(ips, *r.Content)
			}
		}

		return ips
	}

	return nil
}

// applyAutoPTR creates, updates, or deletes PTR records in the appropriate
// reverse zones for every A/AAAA change in the given list. It returns the IPs
// for which no reverse zone could be found (PTR creation skipped).
func (s *Service) applyAutoPTR(
	ctx context.Context,
	currentZone *pdnsapi.Zone,
	changes []RecordChange,
	userID *uint64,
	username, ipAddress string,
) []string {
	// Cache fetched reverse zones to avoid redundant API calls when multiple
	// IPs share the same reverse zone.
	rzCache := make(map[string]*pdnsapi.Zone)

	fetchReverseZone := func(zoneName string) *pdnsapi.Zone {
		if z, ok := rzCache[zoneName]; ok {
			return z
		}

		z, err := powerdns.Engine.Zones.Get(ctx, zoneName)
		if err != nil {
			log.Warn().Err(err).Str("zone", zoneName).Msg("auto-PTR: failed to fetch reverse zone")
			rzCache[zoneName] = nil

			return nil
		}

		rzCache[zoneName] = z

		return z
	}

	var noReverseZoneIPs []string

	for _, change := range changes {
		if change.Type != "A" && change.Type != rrTypeAAAA {
			continue
		}

		isDeletion := change.Existed && len(change.Records) == 0
		if !change.Changed && !isDeletion {
			continue
		}

		fqdn := change.Name
		if !strings.HasSuffix(fqdn, ".") {
			fqdn += "."
		}

		oldIPs := ipsFromCurrentZone(currentZone, fqdn, change.Type)

		oldIPSet := make(map[string]bool, len(oldIPs))

		for _, ip := range oldIPs {
			oldIPSet[ip] = true
		}

		newIPSet := make(map[string]bool, len(change.Records))

		for _, r := range change.Records {
			newIPSet[r.Content] = true
		}

		// Delete PTR records for IPs that are being removed.
		for ip := range oldIPSet {
			if !newIPSet[ip] {
				s.deleteAutoPTR(ctx, fetchReverseZone, fqdn, ip, change.Type, userID, username, ipAddress)
			}
		}

		// Create/replace PTR records for all IPs in the new state.
		for ip := range newIPSet {
			if !s.createAutoPTR(ctx, fqdn, ip, change.Type, change.TTL, userID, username, ipAddress) {
				noReverseZoneIPs = append(noReverseZoneIPs, ip)
			}
		}
	}

	return noReverseZoneIPs
}

// deleteAutoPTR removes the PTR record for ip if it still points to fqdn.
// It is a no-op if no reverse zone is found, the PTR does not exist, or the
// PTR was manually changed to a different target.
func (s *Service) deleteAutoPTR(
	ctx context.Context,
	fetchReverseZone func(string) *pdnsapi.Zone,
	fqdn, ip, rrType string,
	userID *uint64,
	username, ipAddress string,
) {
	ptrName, err := ptrNameForIP(ip, rrType)
	if err != nil {
		log.Warn().Err(err).Str("ip", ip).Msg("auto-PTR: skipping invalid IP on delete")

		return
	}

	reverseZone, err := findBestReverseZone(ctx, ptrName)
	if err != nil {
		log.Warn().Err(err).Str("ptr_name", ptrName).Msg("auto-PTR: error finding reverse zone")

		return
	}

	if reverseZone == "" {
		log.Debug().Str("ptr_name", ptrName).Msg("auto-PTR: no reverse zone found for deletion")

		return
	}

	// Fetch the current PTR value. If it no longer points to our fqdn
	// (i.e. it was manually changed), leave it alone.
	rz := fetchReverseZone(reverseZone)

	currentPTR := ptrContentFromZone(rz, ptrName)
	if currentPTR == "" {
		return
	}

	if currentPTR != fqdn {
		log.Info().
			Str("ptr_name", ptrName).
			Str("current_target", currentPTR).
			Str("expected_target", fqdn).
			Msg("auto-PTR: skipping delete — PTR was manually changed")

		return
	}

	s.patchPTR(ctx, reverseZone, ptrName, currentPTR, 0, true, userID, username, ipAddress)
}

// createAutoPTR creates or replaces the PTR record for ip pointing to fqdn.
// It returns true if a suitable reverse zone was found (PTR was attempted), or
// false if no reverse zone exists for the IP.
func (s *Service) createAutoPTR(
	ctx context.Context,
	fqdn, ip, rrType string,
	ttl uint32,
	userID *uint64,
	username, ipAddress string,
) bool {
	ptrName, err := ptrNameForIP(ip, rrType)
	if err != nil {
		log.Warn().Err(err).Str("ip", ip).Msg("auto-PTR: skipping invalid IP on create")

		return false
	}

	reverseZone, err := findBestReverseZone(ctx, ptrName)
	if err != nil {
		log.Warn().Err(err).Str("ptr_name", ptrName).Msg("auto-PTR: error finding reverse zone")

		return false
	}

	if reverseZone == "" {
		log.Debug().Str("ptr_name", ptrName).Msg("auto-PTR: no reverse zone found, skipping")

		return false
	}

	s.patchPTR(ctx, reverseZone, ptrName, fqdn, ttl, false, userID, username, ipAddress)

	return true
}

// ptrContentFromZone returns the first PTR record content for ptrName in zone,
// or "" if the record does not exist.
func ptrContentFromZone(zone *pdnsapi.Zone, ptrName string) string {
	if zone == nil {
		return ""
	}

	ptrName = strings.ToLower(ptrName)

	for _, rr := range zone.RRsets {
		if rr.Name == nil || rr.Type == nil {
			continue
		}

		if !strings.EqualFold(*rr.Name, ptrName) || string(*rr.Type) != "PTR" {
			continue
		}

		for _, r := range rr.Records {
			if r.Content != nil && *r.Content != "" {
				return *r.Content
			}
		}
	}

	return ""
}

// patchPTR sends a single PTR RRset patch to the given reverse zone and logs
// the change to the activity log.
// When del is true the RRset is deleted; otherwise it is replaced with fqdn.
func (s *Service) patchPTR(
	ctx context.Context,
	reverseZone, ptrName, fqdn string,
	ttl uint32,
	del bool,
	userID *uint64,
	username, ipAddress string,
) {
	rrType := pdnsapi.RRType("PTR")

	var (
		changeType pdnsapi.ChangeType
		records    []pdnsapi.Record
	)

	if del {
		changeType = pdnsapi.ChangeTypeDelete
	} else {
		changeType = pdnsapi.ChangeTypeReplace
		records = []pdnsapi.Record{{Content: &fqdn}}
	}

	rrSets := []pdnsapi.RRset{{
		Name:       &ptrName,
		Type:       &rrType,
		TTL:        &ttl,
		ChangeType: &changeType,
		Records:    records,
	}}

	if err := powerdns.Engine.Records.Patch(ctx, reverseZone, &pdnsapi.RRsets{Sets: rrSets}); err != nil {
		log.Warn().
			Err(err).
			Str("ptr_name", ptrName).
			Str("reverse_zone", reverseZone).
			Bool("delete", del).
			Msg("auto-PTR: failed to patch PTR record")

		return
	}

	log.Info().
		Str("ptr_name", ptrName).
		Str("reverse_zone", reverseZone).
		Str("target", fqdn).
		Bool("delete", del).
		Msg("auto-PTR: PTR record patched")

	// Build a minimal diff describing the auto-PTR change.
	action := "added"

	var oldContents, newContents []string

	if del {
		action = "deleted"
		oldContents = []string{fqdn}
	} else {
		newContents = []string{fqdn}
	}

	diff := &activitylog.RecordsDiff{
		Records: []activitylog.RecordEntryDiff{
			{
				Name:   ptrName,
				Type:   "PTR",
				Action: action,
				NewTTL: ttl,
				Old:    oldContents,
				New:    newContents,
			},
		},
	}

	activitylog.Record(&activitylog.Entry{
		DB:           s.db,
		UserID:       userID,
		Username:     username,
		Action:       activitylog.ActionRecordChanged,
		ResourceType: activitylog.ResourceTypeZone,
		ResourceName: reverseZone,
		Details:      diff,
		IPAddress:    ipAddress,
	})
}
