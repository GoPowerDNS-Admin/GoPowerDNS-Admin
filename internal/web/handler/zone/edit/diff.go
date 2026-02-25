package zoneedit

import (
	"strings"

	pdnsapi "github.com/joeig/go-powerdns/v3"

	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/activitylog"
)

// buildZoneSettingsDiff returns a ZoneSettingsDiff between the current zone state
// and the values submitted in the form. Only fields that actually differ are
// included. When the currentZone is nil (e.g., the pre-fetch failed), every field is
// included with an empty Old value, so the new values are at least recorded.
func buildZoneSettingsDiff(currentZone *pdnsapi.Zone, form *ZoneForm) *activitylog.ZoneSettingsDiff {
	diff := &activitylog.ZoneSettingsDiff{}

	if currentZone == nil {
		diff.Fields = []activitylog.FieldDiff{
			{Field: "kind", Old: "", New: form.Kind},
			{Field: "soa_edit_api", Old: "", New: string(form.SOAEditAPI)},
		}

		return diff
	}

	if currentZone.Kind != nil {
		oldKind := string(*currentZone.Kind)
		if oldKind != form.Kind {
			diff.Fields = append(diff.Fields, activitylog.FieldDiff{
				Field: "kind", Old: oldKind, New: form.Kind,
			})
		}
	}

	oldSOA := string(getSOAEditAPIFromZone(currentZone))
	if oldSOA != string(form.SOAEditAPI) {
		diff.Fields = append(diff.Fields, activitylog.FieldDiff{
			Field: "soa_edit_api", Old: oldSOA, New: string(form.SOAEditAPI),
		})
	}

	oldMasters := strings.Join(currentZone.Masters, ", ")
	if oldMasters != form.Masters {
		diff.Fields = append(diff.Fields, activitylog.FieldDiff{
			Field: "masters", Old: oldMasters, New: form.Masters,
		})
	}

	return diff
}

// buildRecordsDiff compares the current zone RRsets against the incoming changes
// and returns a RecordsDiff describing what was added, modified, or deleted.
// When the currentZone is nil, the old state is treated as unknown (Old is empty).
func buildRecordsDiff(currentZone *pdnsapi.Zone, changes []RecordChange) *activitylog.RecordsDiff {
	diff := &activitylog.RecordsDiff{}
	oldRRSets := buildOldRRSetsMap(currentZone)

	for _, change := range changes {
		if !change.Changed {
			continue
		}

		name := change.Name
		if !strings.HasSuffix(name, ".") {
			name += "."
		}

		key := rrKey{name: strings.ToLower(name), rtype: change.Type}
		old, hasOld := oldRRSets[key]

		oldContents, oldTTL := extractOldRRSetData(&old, hasOld)

		var newContents []string
		for _, r := range change.Records {
			newContents = append(newContents, r.Content)
		}

		action := determineRecordChangeAction(&change, hasOld)

		diff.Records = append(diff.Records, activitylog.RecordEntryDiff{
			Name:   name,
			Type:   change.Type,
			Action: action,
			OldTTL: oldTTL,
			NewTTL: change.TTL,
			Old:    oldContents,
			New:    newContents,
		})
	}

	return diff
}

type rrKey struct{ name, rtype string }

func buildOldRRSetsMap(currentZone *pdnsapi.Zone) map[rrKey]pdnsapi.RRset {
	oldRRSets := make(map[rrKey]pdnsapi.RRset)
	if currentZone == nil {
		return oldRRSets
	}

	for _, rr := range currentZone.RRsets {
		if rr.Name == nil || rr.Type == nil {
			continue
		}

		name := strings.ToLower(*rr.Name)
		if !strings.HasSuffix(name, ".") {
			name += "."
		}

		oldRRSets[rrKey{name: name, rtype: string(*rr.Type)}] = rr
	}

	return oldRRSets
}

func extractOldRRSetData(oldRRSet *pdnsapi.RRset, exists bool) ([]string, uint32) {
	if !exists {
		return nil, 0
	}

	var oldContents []string

	for _, r := range oldRRSet.Records {
		if r.Content != nil {
			oldContents = append(oldContents, *r.Content)
		}
	}

	var oldTTL uint32
	if oldRRSet.TTL != nil {
		oldTTL = *oldRRSet.TTL
	}

	return oldContents, oldTTL
}

func determineRecordChangeAction(change *RecordChange, hasOld bool) string {
	switch {
	case !change.Existed || !hasOld:
		return "added"
	case len(change.Records) == 0:
		return "deleted"
	default:
		return "modified"
	}
}
