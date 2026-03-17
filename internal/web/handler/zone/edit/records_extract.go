package zoneedit

import (
	"strings"

	pdnsapi "github.com/joeig/go-powerdns/v3"
)

// dnssecManagedTypes contains record types that are automatically managed by
// the DNSSEC system and must not be edited or deleted by users.
var dnssecManagedTypes = map[string]bool{
	"RRSIG":      true,
	"NSEC":       true,
	"NSEC3":      true,
	"NSEC3PARAM": true,
}

// isDNSSECManaged reports whether the given RR type is auto-managed by DNSSEC
// and should therefore be displayed as read-only. This includes the well-known
// DNSSEC types as well as PowerDNS-internal unknown types (TYPE<n>) such as
// TYPE65534 which PowerDNS uses for pre-published NSEC3 parameters.
func isDNSSECManaged(rrType string) bool {
	if dnssecManagedTypes[rrType] {
		return true
	}
	// PowerDNS represents unknown/internal record types as "TYPE<n>".
	// These are always system-managed and must not be edited manually.
	return strings.HasPrefix(rrType, "TYPE")
}

// extractRecordsFromRRSets extracts record data from PowerDNS RRsets.
func extractRecordsFromRRSets(
	rrSets []pdnsapi.RRset,
	zoneName string,
	getDisplayName func(string, string) string,
) []RecordData {
	var records []RecordData

	for _, rrSet := range rrSets {
		// Skip RRsets with a missing name or type
		if rrSet.Name == nil || rrSet.Type == nil {
			continue
		}

		rrType := string(*rrSet.Type)

		// Skip DNSSEC-managed and PowerDNS-internal types (e.g. TYPE65534).
		// These records have no meaningful user-facing representation and
		// cannot be edited manually.
		if isDNSSECManaged(rrType) {
			continue
		}

		// Get comment from RRset (if any)
		comment := extractCommentFromRRSet(&rrSet)

		// Process each record in the RRset
		for _, rec := range rrSet.Records {
			recordData := RecordData{
				Name:        *rrSet.Name,
				DisplayName: getDisplayName(*rrSet.Name, zoneName),
				Type:        rrType,
				Content:     "",
				Disabled:    false,
				Comment:     comment,
			}

			if rrSet.TTL != nil {
				recordData.TTL = *rrSet.TTL
			}

			if rec.Content != nil {
				recordData.Content = *rec.Content
			}

			if rec.Disabled != nil {
				recordData.Disabled = *rec.Disabled
			}

			records = append(records, recordData)
		}
	}

	return records
}

// extractCommentFromRRSet extracts the comment from an RRset.
func extractCommentFromRRSet(rrSet *pdnsapi.RRset) string {
	if rrSet != nil && len(rrSet.Comments) > 0 && rrSet.Comments[0].Content != nil {
		return *rrSet.Comments[0].Content
	}

	return ""
}
