package zoneedit

import (
	"context"
	"errors"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	pdnsapi "github.com/joeig/go-powerdns/v3"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"

	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/auth"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/config"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/powerdns"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web/handler"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web/handler/dashboard"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web/navigation"
)

// uriRecordRe matches the RFC 7553 content format for URI records:
// priority weight "target"  OR  priority weight target
// It captures:
//
//	1: priority (digits)
//	2: weight (digits)
//	3: target if quoted (inner content without quotes)
//	4: target if unquoted (rest of the line)
var uriRecordRe = regexp.MustCompile(`^\s*(\d+)\s+(\d+)\s+(?:"((?:[^"\\]|\\.)*)"|(.*))\s*$`)

const (
	// Path is the path to the edit zone page.
	Path = handler.RootPath + "zone/edit/:name"

	// TemplateName is the name of the edit zone template.
	TemplateName = "zone/edit"

	// PageTitle is the title of the edit zone page.
	PageTitle = "Edit Zone"

	// ErrMsgZoneNameRequired is the error message when zone name is missing.
	ErrMsgZoneNameRequired = "Zone name is required"

	defaultTimeout = 30 * time.Second
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

// ZoneForm represents the form data for editing a zone.
type ZoneForm struct {
	Name       string     `form:"name"`
	Kind       string     `form:"kind"         validate:"required,oneof=Native Master Slave"`
	SOAEditAPI SOAEditAPI `form:"soa_edit_api" validate:"required,oneof=DEFAULT INCREASE EPOCH OFF"`
	Masters    string     `form:"masters"` // Comma-separated list for Slave zones
}

// RecordData represents a single DNS record for display.
type RecordData struct {
	Name        string `json:"name"`         // Full canonical name
	DisplayName string `json:"display_name"` // Shortened name for display (without zone)
	Type        string `json:"type"`
	TTL         uint32 `json:"ttl"`
	Content     string `json:"content"`
	Disabled    bool   `json:"disabled"`
	Comment     string `json:"comment"` // Record comment
}

// RecordChange represents a change to be applied to records.
type RecordChange struct {
	Existed bool     `json:"existed"` // Whether the record existed before
	Changed bool     `json:"changed"` // Whether this RRset has actually changed
	Name    string   `json:"name"`
	Type    string   `json:"type"`
	TTL     uint32   `json:"ttl"`
	Records []Record `json:"records"`
	Comment string   `json:"comment"` // Comment for the RRset
}

// Record represents a single record entry.
type Record struct {
	Content  string `json:"content"`
	Disabled bool   `json:"disabled"`
}

// RecordsUpdateRequest represents the request for updating records.
type RecordsUpdateRequest struct {
	Changes []RecordChange `json:"changes"`
}

// RecordTypeOption represents a record type option for the dropdown.
type RecordTypeOption struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	Enabled     bool   `json:"enabled"`
	Help        string `json:"help"`
}

// Service is the edit zone handler service.
type Service struct {
	handler.Service
	cfg       *config.Config
	db        *gorm.DB
	validator *validator.Validate
}

// Handler is the edit zone handler.
var Handler = Service{}

// Init initializes the edit zone handler.
func (s *Service) Init(app *fiber.App, cfg *config.Config, db *gorm.DB, authService *auth.Service) {
	if app == nil || cfg == nil || db == nil {
		log.Fatal().Msg(handler.ErrNilACDFatalLogMsg)
		return
	}

	s.db = db
	s.cfg = cfg
	s.validator = validator.New()

	// register routes with permission checks
	app.Get(Path,
		auth.RequirePermission(authService, auth.PermZoneUpdate),
		s.Get,
	)
	app.Post(Path,
		auth.RequirePermission(authService, auth.PermZoneUpdate),
		s.Post,
	)
	app.Post(Path+"/records",
		auth.RequirePermission(authService, auth.PermZoneUpdate),
		s.PostRecords,
	)
	app.Post(Path+"/delete",
		auth.RequirePermission(authService, auth.PermZoneDelete),
		s.Delete,
	)
}

// Get handles the edit zone page rendering.
func (s *Service) Get(c *fiber.Ctx) error {
	zoneName := c.Params("name")
	if zoneName == "" {
		return c.Status(fiber.StatusBadRequest).SendString(ErrMsgZoneNameRequired)
	}

	zoneName = normalizeZoneName(zoneName)

	// Create navigation context
	nav := navigation.NewContext(PageTitle, "zones", "edit").
		AddBreadcrumb("Dashboard", dashboard.Path, false).
		AddBreadcrumb(PageTitle, "", true)

	// Ensure PDNS client is available and fetch zone
	zone, err := s.getZoneOrRender(c, nav, zoneName)
	if err != nil {
		return err
	}
	// If an error response was already rendered inside getZoneOrRender,
	// it returns a nil zone with no error. Stop processing in that case.
	if zone == nil {
		return nil
	}

	// Extract SOA-EDIT-API and masters
	soaEditAPI := getSOAEditAPIFromZone(zone)
	masters := strings.Join(zone.Masters, ", ")

	// Populate form with zone data
	form := &ZoneForm{
		Name:       *zone.Name,
		Kind:       string(*zone.Kind),
		SOAEditAPI: soaEditAPI,
		Masters:    masters,
	}

	// Extract records from RRsets
	records := extractRecordsFromRRSets(zone.RRsets, zoneName, getDisplayNameForZone)

	// Check DNSSEC status
	dnssecEnabled := zone.DNSsec != nil && *zone.DNSsec

	// Load allowed record types from settings
	allowedRecordTypes := s.loadAllowedRecordTypes()

	// Sort record types alphabetically by type
	sort.Slice(allowedRecordTypes, func(i, j int) bool {
		return allowedRecordTypes[i].Type < allowedRecordTypes[j].Type
	})

	// Render form with existing zone data
	return c.Render(TemplateName, fiber.Map{
		"Navigation":         nav,
		"Form":               form,
		"Zone":               zone,
		"Records":            records,
		"DNSSECEnabled":      dnssecEnabled,
		"AllowedRecordTypes": allowedRecordTypes,
	}, handler.BaseLayout)
}

// Post handles the edit zone form submission.
func (s *Service) Post(c *fiber.Ctx) error {
	zoneName := c.Params("name")
	if zoneName == "" {
		return c.Status(fiber.StatusBadRequest).SendString(ErrMsgZoneNameRequired)
	}

	// Ensure zone name ends with a dot
	if !strings.HasSuffix(zoneName, ".") {
		zoneName += "."
	}

	// Create navigation context
	nav := navigation.NewContext(PageTitle, "zones", "edit").
		AddBreadcrumb("Dashboard", dashboard.Path, false).
		AddBreadcrumb(PageTitle, "", true)

	// Parse form data
	form := &ZoneForm{}
	if err := c.BodyParser(form); err != nil {
		log.Error().Err(err).Msg("failed to parse edit zone form")

		return c.Status(fiber.StatusBadRequest).Render(TemplateName, fiber.Map{
			"Navigation": nav,
			"Form":       form,
			"Error":      "Invalid form data",
		}, handler.BaseLayout)
	}

	// Validate form
	if err := s.validator.Struct(form); err != nil {
		var validationErrors validator.ValidationErrors
		errors.As(err, &validationErrors)

		errorMessages := make([]string, len(validationErrors))
		for i, ve := range validationErrors {
			errorMessages[i] = "Field '" + ve.Field() + "' failed validation tag '" + ve.Tag() + "'"
		}

		log.Error().Err(err).Msg("validation failed for edit zone")

		return c.Status(fiber.StatusBadRequest).Render(TemplateName, fiber.Map{
			"Navigation": nav,
			"Form":       form,
			"Error":      errorMessages,
		}, handler.BaseLayout)
	}

	// Check if PowerDNS client is initialized
	if powerdns.Engine.Client == nil {
		log.Error().Msg(powerdns.ErrMsgClientNotInitialized)

		return c.Status(fiber.StatusInternalServerError).Render(TemplateName, fiber.Map{
			"Navigation": nav,
			"Form":       form,
			"Error":      powerdns.ErrMsgClientNotInitializedDetailed,
		}, handler.BaseLayout)
	}

	// Update zone via PowerDNS API
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	// Prepare zone update
	soaEditAPIStr := string(form.SOAEditAPI)
	kind := pdnsapi.ZoneKind(form.Kind)
	zoneUpdate := pdnsapi.Zone{
		SOAEditAPI: &soaEditAPIStr,
		Kind:       &kind,
	}

	// Add masters if zone type is Slave
	if form.Kind == "Slave" {
		if form.Masters == "" {
			return c.Status(fiber.StatusBadRequest).Render(TemplateName, fiber.Map{
				"Navigation": nav,
				"Form":       form,
				"Error":      "Master servers are required for Slave zones",
			}, handler.BaseLayout)
		}

		var masters []string

		for _, master := range strings.Split(form.Masters, ",") {
			trimmed := strings.TrimSpace(master)
			if trimmed != "" {
				masters = append(masters, trimmed)
			}
		}

		if len(masters) == 0 {
			return c.Status(fiber.StatusBadRequest).Render(TemplateName, fiber.Map{
				"Navigation": nav,
				"Form":       form,
				"Error":      "At least one master server is required for Slave zones",
			}, handler.BaseLayout)
		}

		zoneUpdate.Masters = masters
	}

	err := powerdns.Engine.Zones.Change(ctx, zoneName, &zoneUpdate)
	if err != nil {
		log.Error().
			Err(err).
			Str("zone_name", zoneName).
			Msg("failed to update zone")

		return c.Status(fiber.StatusInternalServerError).Render(TemplateName, fiber.Map{
			"Navigation": nav,
			"Form":       form,
			"Error":      "Failed to update zone: " + err.Error(),
		}, handler.BaseLayout)
	}

	log.Info().
		Str("zone_name", zoneName).
		Str("zone_kind", form.Kind).
		Str("soa_edit_api", string(form.SOAEditAPI)).
		Msg("Zone updated successfully")

	// Redirect to dashboard with success message
	return c.Redirect(dashboard.Path + "?success=Zone updated successfully")
}

// PostRecords handles the record updates for a zone.
func (s *Service) PostRecords(c *fiber.Ctx) error {
	zoneName := c.Params("name")
	if zoneName == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": ErrMsgZoneNameRequired,
		})
	}

	// Ensure zone name ends with a dot
	if !strings.HasSuffix(zoneName, ".") {
		zoneName += "."
	}

	// Parse JSON request
	var request RecordsUpdateRequest
	if err := c.BodyParser(&request); err != nil {
		log.Error().Err(err).Msg("failed to parse records update request")

		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid request data",
		})
	}

	// Check if PowerDNS client is initialized
	if powerdns.Engine.Client == nil {
		log.Error().Msg(powerdns.ErrMsgClientNotInitialized)

		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": powerdns.ErrMsgClientNotInitialized,
		})
	}

	// Build RRsets for PowerDNS API
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	// Build list only with actual changes
	// Pre-allocate capacity based on incoming changes to reduce reallocations
	var rrSets = make([]pdnsapi.RRset, 0, len(request.Changes))
	for _, change := range request.Changes {
		// Skip entries that aren't marked as changed (defensive; frontend should only submit changed ones)
		if !change.Changed {
			continue
		}

		var records []pdnsapi.Record

		for _, rec := range change.Records {
			content := ensureQuotedContent(change.Type, rec.Content)
			disabled := rec.Disabled
			records = append(records, pdnsapi.Record{
				Content:  &content,
				Disabled: &disabled,
			})
		}

		// Ensure name is in canonical format (ends with a dot)
		name := change.Name
		if !strings.HasSuffix(name, ".") {
			name += "."
		}

		rrType := pdnsapi.RRType(change.Type)
		ttl := change.TTL
		// Decide change type: delete if RRset existed and now has no records
		var changeType pdnsapi.ChangeType
		if change.Existed && len(records) == 0 {
			changeType = pdnsapi.ChangeTypeDelete
		} else {
			changeType = pdnsapi.ChangeTypeReplace
		}

		// Prepare comments (always include to allow clearing)
		var comments []pdnsapi.Comment

		commentContent := change.Comment // include even if empty to allow clearing
		commentAccount := ""             // Empty account field is required by PowerDNS
		comments = []pdnsapi.Comment{
			{
				Content: &commentContent,
				Account: &commentAccount,
			},
		}

		rrSet := pdnsapi.RRset{
			Name:       &name,
			Type:       &rrType,
			TTL:        &ttl,
			ChangeType: &changeType,
			Records:    records,
			Comments:   comments,
		}

		rrSets = append(rrSets, rrSet)
	}

	// Update records via PowerDNS API
	err := powerdns.Engine.Records.Patch(ctx, zoneName, &pdnsapi.RRsets{
		Sets: rrSets,
	})
	if err != nil {
		log.Error().
			Err(err).
			Str("zone_name", zoneName).
			Msg("failed to update zone records")

		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to update records: " + err.Error(),
		})
	}

	log.Info().
		Str("zone_name", zoneName).
		Int("changes_count", len(request.Changes)).
		Msg("Zone records updated successfully")

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Records updated successfully",
	})
}

// Delete handles the zone deletion.
func (s *Service) Delete(c *fiber.Ctx) error {
	zoneName := c.Params("name")
	if zoneName == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": ErrMsgZoneNameRequired,
		})
	}

	// Ensure zone name ends with a dot
	if !strings.HasSuffix(zoneName, ".") {
		zoneName += "."
	}

	// Check if PowerDNS client is initialized
	if powerdns.Engine.Client == nil {
		log.Error().Msg(powerdns.ErrMsgClientNotInitialized)

		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": powerdns.ErrMsgClientNotInitialized,
		})
	}

	// Delete zone via PowerDNS API
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	err := powerdns.Engine.Zones.Delete(ctx, zoneName)
	if err != nil {
		log.Error().
			Err(err).
			Str("zone_name", zoneName).
			Msg("failed to delete zone")

		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to delete zone: " + err.Error(),
		})
	}

	log.Info().
		Str("zone_name", zoneName).
		Msg("Zone deleted successfully")

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Zone deleted successfully",
	})
}

// getZoneOrRender validates PDNS client availability and fetches the zone; renders errors when needed.
func (s *Service) getZoneOrRender(c *fiber.Ctx, nav *navigation.Context, zoneName string) (*pdnsapi.Zone, error) {
	if powerdns.Engine.Client == nil {
		log.Error().Msg(powerdns.ErrMsgClientNotInitialized)

		return nil, c.Status(fiber.StatusInternalServerError).Render(TemplateName, fiber.Map{
			"Navigation": nav,
			"Error":      powerdns.ErrMsgClientNotInitializedDetailed,
		}, handler.BaseLayout)
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	zone, err := powerdns.Engine.Zones.Get(ctx, zoneName)
	if err != nil {
		log.Error().Err(err).Str("zone_name", zoneName).Msg("failed to fetch zone")

		return nil, c.Status(fiber.StatusNotFound).Render(TemplateName, fiber.Map{
			"Navigation": nav,
			"Error":      "Zone not found: " + zoneName,
		}, handler.BaseLayout)
	}

	return zone, nil
}
