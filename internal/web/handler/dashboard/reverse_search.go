package dashboard

import (
	"context"
	"net"
	"strconv"
	"strings"

	pdnsapi "github.com/joeig/go-powerdns/v3"
	"github.com/rs/zerolog/log"

	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/powerdns"
)

// maxSearchResults caps how many record matches we ask PowerDNS to return for a
// single reverse-zone search. The dashboard only needs the set of owning zones,
// so a generous cap keeps one zone from starving others while bounding the work.
const maxSearchResults = 1000

// reverse-zone name suffixes (with the leading dot) used to strip the zone name
// down to its address octets/nibbles and to constrain record-data hits.
const (
	suffixReverseV4 = ".in-addr.arpa."
	suffixReverseV6 = ".ip6.arpa."
)

// lowNibbleMask isolates the low four bits of a byte when splitting it into
// hex nibbles for IPv6 reverse-name expansion.
const lowNibbleMask = 0x0f

// filterReverseZones filters reverse zones by name, IP address, and record data
// (PTR hostnames). It augments the plain zone-name substring match used on the
// forward tab so that, on a reverse tab, a user can find the zone owning an IP by
// typing it in natural order (e.g. "192.168.1.50") or by typing the hostname a
// PTR points to. categorySuffix is suffixReverseV4 or suffixReverseV6 and scopes
// the record-data search to the active tab's category.
func (s *Service) filterReverseZones(
	ctx context.Context,
	zones []Zone,
	query, filterKind, categorySuffix string,
) []Zone {
	q := strings.ToLower(strings.TrimSpace(query))
	frag, isIP := ipQueryToReverseFragment(query)
	recordZones := s.searchRecordZones(ctx, query, categorySuffix)

	matched := make([]Zone, 0, len(zones))

	for _, zone := range zones {
		name := strings.ToLower(zone.Name)

		match := q != "" && strings.Contains(name, q)

		if !match && isIP {
			octets := strings.TrimSuffix(name, categorySuffix)
			match = dotAlignedSuffix(octets, frag) || dotAlignedSuffix(frag, octets)
		}

		if !match && recordZones[name] {
			match = true
		}

		if match {
			matched = append(matched, zone)
		}
	}

	if filterKind != "" {
		kept := make([]Zone, 0, len(matched))

		for _, zone := range matched {
			if zone.Kind == filterKind {
				kept = append(kept, zone)
			}
		}

		matched = kept
	}

	return matched
}

// searchRecordZones queries the PowerDNS server-side search for records whose
// name or content matches the query, returning the set of owning zone names
// (lower-cased) that belong to categorySuffix. On error it logs and returns an
// empty set so the caller falls back to name/IP matching rather than failing.
func (s *Service) searchRecordZones(ctx context.Context, query, categorySuffix string) map[string]bool {
	out := make(map[string]bool)

	query = strings.TrimSpace(query)
	if query == "" {
		return out
	}

	results, err := powerdns.Engine.Search.Data(ctx, "*"+query+"*", maxSearchResults, pdnsapi.SearchObjectTypeRecord)
	if err != nil {
		log.Debug().Err(err).Str("query", query).Msg("dashboard: reverse record search failed; falling back to name/IP match")
		return out
	}

	if len(results) >= maxSearchResults {
		log.Warn().Int("max", maxSearchResults).Str("query", query).
			Msg("dashboard: reverse record search hit result cap; some matches may be omitted")
	}

	for i := range results {
		if results[i].Zone == nil {
			continue
		}

		name := strings.ToLower(*results[i].Zone)
		if strings.HasSuffix(name, categorySuffix) {
			out[name] = true
		}
	}

	return out
}

// ipQueryToReverseFragment converts a forward IP address or partial prefix into
// the reversed octet/nibble fragment used in reverse zone names, so it can be
// matched against them. It returns false when the query is not IP-shaped (e.g. a
// hostname), in which case the caller relies on name and record-data matching.
func ipQueryToReverseFragment(query string) (string, bool) {
	query = strings.TrimSpace(query)
	if query == "" {
		return "", false
	}

	if strings.Contains(query, ":") {
		return ipv6ToReverseFragment(query)
	}

	return ipv4ToReverseFragment(query)
}

// ipv4ToReverseFragment reverses the octets of a full or partial IPv4 address.
// "192.168.1.50" -> "50.1.168.192"; "192.168" -> "168.192".
func ipv4ToReverseFragment(query string) (string, bool) {
	octets := make([]string, 0, 4)

	for _, part := range strings.Split(query, ".") {
		if part == "" {
			continue
		}

		n, err := strconv.Atoi(part)
		if err != nil || n < 0 || n > 255 {
			return "", false
		}

		octets = append(octets, strconv.Itoa(n))
	}

	if len(octets) == 0 {
		return "", false
	}

	reverseInPlace(octets)

	return strings.Join(octets, "."), true
}

// ipv6ToReverseFragment reverses the nibbles of a full or partial IPv6 address.
// A full address is expanded via net.ParseIP; a partial prefix is taken up to any
// "::" and each colon-separated group is left-padded to four nibbles, so
// "2001:db8" -> "8.b.d.0.1.0.0.2".
func ipv6ToReverseFragment(query string) (string, bool) {
	query = strings.ToLower(strings.TrimSpace(query))

	if ip := net.ParseIP(query); ip != nil {
		if ip.To4() != nil {
			return "", false
		}

		v6 := ip.To16()
		if v6 == nil {
			return "", false
		}

		nibbles := make([]string, 0, 32)

		for _, b := range v6 {
			nibbles = append(nibbles, hexNibble(b>>4), hexNibble(b&lowNibbleMask))
		}

		reverseInPlace(nibbles)

		return strings.Join(nibbles, "."), true
	}

	if !strings.Contains(query, ":") {
		return "", false
	}

	head := query
	if i := strings.Index(query, "::"); i >= 0 {
		head = query[:i]
	}

	head = strings.Trim(head, ":")
	if head == "" {
		return "", false
	}

	nibbles := make([]string, 0, 32)

	for _, group := range strings.Split(head, ":") {
		if group == "" || len(group) > 4 || !isHex(group) {
			return "", false
		}

		group = strings.Repeat("0", 4-len(group)) + group
		for _, c := range group {
			nibbles = append(nibbles, string(c))
		}
	}

	if len(nibbles) == 0 {
		return "", false
	}

	reverseInPlace(nibbles)

	return strings.Join(nibbles, "."), true
}

// dotAlignedSuffix reports whether suffix is a suffix of s at a dot boundary, so
// "1.168.192" matches "50.1.168.192" but "1.168.192" does not match "11.168.192".
func dotAlignedSuffix(s, suffix string) bool {
	if suffix == "" {
		return false
	}

	if s == suffix {
		return true
	}

	if !strings.HasSuffix(s, suffix) {
		return false
	}

	return s[len(s)-len(suffix)-1] == '.'
}

func reverseInPlace(s []string) {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
}

func hexNibble(b byte) string {
	return string("0123456789abcdef"[b])
}

func isHex(s string) bool {
	for _, c := range s {
		switch {
		case c >= '0' && c <= '9':
		case c >= 'a' && c <= 'f':
		default:
			return false
		}
	}

	return true
}
