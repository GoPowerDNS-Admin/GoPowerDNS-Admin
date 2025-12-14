package zoneedit

import pdnsapi "github.com/joeig/go-powerdns/v3"

// extractRecordsFromRRSets extracts record data from PowerDNS RRsets.
func extractRecordsFromRRSets(
	rrSets []pdnsapi.RRset,
	zoneName string,
	getDisplayName func(string, string) string,
) []RecordData {
	var records []RecordData

	for _, rrSet := range rrSets {
		// Skip RRsets with missing name or type
		if rrSet.Name == nil || rrSet.Type == nil {
			continue
		}

		// Get comment from RRset (if any)
		comment := extractCommentFromRRSet(&rrSet)

		// Process each record in the RRset
		for _, rec := range rrSet.Records {
			recordData := RecordData{
				Name:        *rrSet.Name,
				DisplayName: getDisplayName(*rrSet.Name, zoneName),
				Type:        string(*rrSet.Type),
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
