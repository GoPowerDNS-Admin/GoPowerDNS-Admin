package activitylog

// FieldDiff represents a single changed setting with its old and new value.
type FieldDiff struct {
	Field string `json:"field"`
	Old   string `json:"old"`
	New   string `json:"new"`
}

// ZoneSettingsDiff holds the before/after diff of zone-level settings
// (kind, SOA-EDIT-API, masters). Only fields that actually changed are included
// unless the previous state was unavailable, in which case Old is empty.
type ZoneSettingsDiff struct {
	Fields []FieldDiff `json:"fields"`
}

// RecordEntryDiff represents a change to a single DNS RRset.
type RecordEntryDiff struct {
	// Name is the canonical record name (trailing dot).
	Name string `json:"name"`
	// Type is the DNS record type (A, AAAA, MX, …).
	Type string `json:"type"`
	// Action is one of "added", "modified", or "deleted".
	Action string `json:"action"`
	// OldTTL / NewTTL are the TTL values before and after the change.
	OldTTL uint32 `json:"old_ttl,omitempty"`
	NewTTL uint32 `json:"new_ttl,omitempty"`
	// Old / New are the record content strings before and after the change.
	Old []string `json:"old,omitempty"`
	New []string `json:"new,omitempty"`
}

// RecordsDiff holds all RRset changes for a single zone patch operation.
type RecordsDiff struct {
	Records []RecordEntryDiff `json:"records"`
}

// RecordUndoneDetails is stored with record_undone activity entries.
type RecordUndoneDetails struct {
	// OriginalID is the ID of the record_changed entry that was reversed.
	OriginalID uint64 `json:"original_id"`
	// OriginalUsername is the user who made the original change.
	OriginalUsername string `json:"original_username,omitempty"`
}
