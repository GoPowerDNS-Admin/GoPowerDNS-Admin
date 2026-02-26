package activity

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	pdnsapi "github.com/joeig/go-powerdns/v3"
	"github.com/rs/zerolog/log"

	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/activitylog"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/db/models"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/powerdns"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web/session"
)

const undoTimeout = 30 * time.Second

// PostUndo reverses a record_changed activity log entry by applying the inverse
// of its recorded diff back to PowerDNS.
func (s *Service) PostUndo(c *fiber.Ctx) error {
	redirectBase := Path + "?" + buildQueryString(c)

	id, err := c.ParamsInt("id")
	if err != nil || id <= 0 {
		return c.Redirect(redirectBase + "&error=Invalid+activity+log+ID")
	}

	// Load the activity log entry.
	var entry models.ActivityLog
	if err := s.db.First(&entry, uint64(id)).Error; err != nil {
		log.Error().Err(err).Int("id", id).Msg("undo: activity log entry not found")
		return c.Redirect(redirectBase + "&error=Activity+log+entry+not+found")
	}

	switch entry.Action {
	case activitylog.ActionRecordChanged:
		return s.undoRecordChanged(c, id, &entry, redirectBase)
	case activitylog.ActionZoneDeleted:
		return s.undoZoneDeleted(c, id, &entry, redirectBase)
	default:
		return c.Redirect(redirectBase + "&error=Undo+is+only+available+for+record_changed+and+zone_deleted+entries")
	}
}

// undoRecordChanged reverses a record_changed activity log entry.
func (s *Service) undoRecordChanged(c *fiber.Ctx, id int, entry *models.ActivityLog, redirectBase string) error {
	// Parse the stored diff.
	var diff activitylog.RecordsDiff
	if err := json.Unmarshal([]byte(entry.Details), &diff); err != nil {
		log.Error().Err(err).Int("id", id).Msg("undo: failed to parse records diff")
		return c.Redirect(redirectBase + "&error=Failed+to+parse+activity+log+diff")
	}

	if len(diff.Records) == 0 {
		return c.Redirect(redirectBase + "&error=No+record+changes+to+undo")
	}

	// Check that the PowerDNS client is available.
	if powerdns.Engine.Client == nil {
		log.Error().Msg("undo: PowerDNS client not initialized")
		return c.Redirect(redirectBase + "&error=PowerDNS+client+not+initialized")
	}

	// Build the reverse RRsets.
	var rrSets []pdnsapi.RRset //nolint:prealloc // prealloc not possible due to dynamic length

	for _, rec := range diff.Records {
		rrSet := buildReverseRRSet(&rec)
		if rrSet == nil {
			continue
		}

		rrSets = append(rrSets, *rrSet)
	}

	if len(rrSets) == 0 {
		return c.Redirect(redirectBase + "&error=Nothing+to+undo+(no+restorable+changes)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), undoTimeout)
	defer cancel()

	if err := powerdns.Engine.Records.Patch(ctx, entry.ResourceName, &pdnsapi.RRsets{
		Sets: rrSets,
	}); err != nil {
		log.Error().Err(err).Int("id", id).Str("zone", entry.ResourceName).Msg("undo: failed to patch records")
		return c.Redirect(redirectBase + "&error=Failed+to+apply+undo+to+PowerDNS:+" + url.QueryEscape(err.Error()))
	}

	// Record a new activity log entry for the undo operation.
	userID, username := currentUserFromSession(c)
	activitylog.Record(&activitylog.Entry{
		DB:           s.db,
		UserID:       userID,
		Username:     username,
		Action:       activitylog.ActionRecordUndone,
		ResourceType: activitylog.ResourceTypeZone,
		ResourceName: entry.ResourceName,
		Details: activitylog.RecordUndoneDetails{
			OriginalID:       entry.ID,
			OriginalUsername: entry.Username,
		},
		IPAddress: c.IP(),
	})

	log.Info().Int("original_id", id).Str("zone", entry.ResourceName).Str("user", username).
		Msg("record changes undone successfully")

	return c.Redirect(redirectBase +
		"&success=Record+changes+from+entry+%23" +
		strconv.Itoa(id) +
		"+have+been+undone")
}

// undoZoneDeleted restores a deleted zone from the snapshot stored in the activity log.
func (s *Service) undoZoneDeleted(c *fiber.Ctx, id int, entry *models.ActivityLog, redirectBase string) error {
	if entry.Details == "" {
		return c.Redirect(redirectBase + "&error=No+zone+snapshot+available+to+restore")
	}

	var snap activitylog.ZoneSnapshot
	if err := json.Unmarshal([]byte(entry.Details), &snap); err != nil {
		log.Error().Err(err).Int("id", id).Msg("undo: failed to parse zone snapshot")
		return c.Redirect(redirectBase + "&error=Failed+to+parse+zone+snapshot")
	}

	if snap.Kind == "" {
		return c.Redirect(redirectBase + "&error=Zone+snapshot+is+incomplete+(missing+kind)")
	}

	if powerdns.Engine.Client == nil {
		log.Error().Msg("undo: PowerDNS client not initialized")
		return c.Redirect(redirectBase + "&error=PowerDNS+client+not+initialized")
	}

	zoneName := entry.ResourceName

	// Build the zone object for recreation. RRsets are included in the zone
	// creation payload so that records are restored atomically in a single API
	// call, avoiding a race where a subsequent PATCH would return 404 on a
	// zone that was not yet fully committed by PowerDNS.
	zoneKind := pdnsapi.ZoneKind(snap.Kind)
	soaEditAPI := snap.SOAEditAPI

	zone := &pdnsapi.Zone{
		Name:       &zoneName,
		Kind:       &zoneKind,
		SOAEditAPI: &soaEditAPI,
		Masters:    snap.Masters,
	}

	for _, rr := range snap.RRsets {
		if len(rr.Records) == 0 {
			continue
		}

		name := rr.Name
		if !strings.HasSuffix(name, ".") {
			name += "."
		}

		rrType := pdnsapi.RRType(rr.Type)
		ttl := rr.TTL

		var records []pdnsapi.Record

		for _, content := range rr.Records {
			disabled := false
			records = append(records, pdnsapi.Record{
				Content:  &content,
				Disabled: &disabled,
			})
		}

		// No ChangeType: zone creation does not use changetype in rrsets.
		zone.RRsets = append(zone.RRsets, pdnsapi.RRset{
			Name:    &name,
			Type:    &rrType,
			TTL:     &ttl,
			Records: records,
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), undoTimeout)
	defer cancel()

	if _, err := powerdns.Engine.Zones.Add(ctx, zone); err != nil {
		log.Error().Err(err).Int("id", id).Str("zone", zoneName).Msg("undo: failed to recreate zone")
		return c.Redirect(redirectBase + "&error=Failed+to+recreate+zone:+" + url.QueryEscape(err.Error()))
	}

	// Record the undo action.
	userID, username := currentUserFromSession(c)
	activitylog.Record(&activitylog.Entry{
		DB:           s.db,
		UserID:       userID,
		Username:     username,
		Action:       activitylog.ActionZoneDeletedUndone,
		ResourceType: activitylog.ResourceTypeZone,
		ResourceName: zoneName,
		Details: activitylog.ZoneDeletedUndoneDetails{
			OriginalID:       entry.ID,
			OriginalUsername: entry.Username,
		},
		IPAddress: c.IP(),
	})

	log.Info().Int("original_id", id).Str("zone", zoneName).Str("user", username).
		Msg("zone deletion undone successfully")

	return c.Redirect(redirectBase +
		"&success=Zone+" + url.QueryEscape(zoneName) +
		"+has+been+restored+from+entry+%23" +
		strconv.Itoa(id))
}

// buildReverseRRSet constructs the inverse RRset operation for a single diff entry:
//   - "added"   → delete the RRset that was added
//   - "deleted" or "modified" → replace with the old content/TTL
//
// Returns nil if there is nothing to restore.
func buildReverseRRSet(rec *activitylog.RecordEntryDiff) *pdnsapi.RRset {
	name := rec.Name
	if !strings.HasSuffix(name, ".") {
		name += "."
	}

	rrType := pdnsapi.RRType(rec.Type)

	switch rec.Action {
	case "added":
		// The record was added; to undo we delete it.
		changeType := pdnsapi.ChangeTypeDelete

		return &pdnsapi.RRset{
			Name:       &name,
			Type:       &rrType,
			ChangeType: &changeType,
		}

	case "deleted", "modified":
		// The record was deleted or modified; restore old content.
		if len(rec.Old) == 0 {
			return nil
		}

		ttl := rec.OldTTL
		if ttl == 0 {
			ttl = 300 // sensible fallback
		}

		var records []pdnsapi.Record

		for _, content := range rec.Old {
			c := content
			disabled := false
			records = append(records, pdnsapi.Record{
				Content:  &c,
				Disabled: &disabled,
			})
		}

		changeType := pdnsapi.ChangeTypeReplace

		return &pdnsapi.RRset{
			Name:       &name,
			Type:       &rrType,
			TTL:        &ttl,
			ChangeType: &changeType,
			Records:    records,
		}
	}

	return nil
}

// currentUserFromSession extracts the current user's ID and username from the
// session cookie. Returns nil userID and empty username when no valid session exists.
func currentUserFromSession(c *fiber.Ctx) (*uint64, string) {
	sid := c.Cookies("session")
	if sid == "" {
		return nil, ""
	}

	sd := new(session.Data)
	if err := sd.Read(sid); err != nil || sd.User.ID == 0 {
		return nil, ""
	}

	id := sd.User.ID

	return &id, sd.User.Username
}

// buildQueryString preserves the existing filter/page query params so the
// redirect lands back on the same filtered view.
func buildQueryString(c *fiber.Ctx) string {
	var params []string

	for _, key := range []string{"page", "pageSize", "user", "action", "from", "to"} {
		if v := c.Query(key); v != "" {
			params = append(params, key+"="+v)
		}
	}

	return strings.Join(params, "&")
}
