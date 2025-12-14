// Package zoneadd provides the handler for adding new DNS zones.
package zoneadd

import (
	"context"
	"errors"
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

const (
	// Path is the path to the add zone page.
	Path = handler.RootPath + "zone/add"

	// TemplateName is the name of the add zone template.
	TemplateName = "zone/add"

	// PageTitle is the title of the add zone page.
	PageTitle = "Add Zone"

	defaultTimeout = 30 * time.Second
)

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

// ZoneForm represents the form data for creating a new zone.
type ZoneForm struct {
	Name       string     `form:"name"         validate:"required,fqdn"`
	Kind       ZoneKind   `form:"kind"         validate:"required,oneof=Native Master Slave"`
	SOAEditAPI SOAEditAPI `form:"soa_edit_api" validate:"required,oneof=DEFAULT INCREASE EPOCH OFF"`
	Masters    string     `form:"masters"` // Comma-separated list for Slave zones
}

// Service is the add zone handler service.
type Service struct {
	handler.Service
	cfg       *config.Config
	db        *gorm.DB
	validator *validator.Validate
}

// Handler is the add zone handler.
var Handler = Service{}

// Init initializes the add zone handler.
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
		auth.RequirePermission(authService, auth.PermZoneCreate),
		s.Get,
	)
	app.Post(Path,
		auth.RequirePermission(authService, auth.PermZoneCreate),
		s.Post,
	)
}

// Get handles the add zone page rendering.
func (s *Service) Get(c *fiber.Ctx) error {
	// Create navigation context
	nav := navigation.NewContext(PageTitle, "zones", "add").
		AddBreadcrumb("Home", dashboard.Path, false).
		AddBreadcrumb("Dashboard", dashboard.Path, false).
		AddBreadcrumb(PageTitle, Path, true)

	// Render empty form
	return c.Render(TemplateName, fiber.Map{
		"Navigation": nav,
		"Form":       &ZoneForm{},
	}, handler.BaseLayout)
}

// Post handles the add zone form submission.
func (s *Service) Post(c *fiber.Ctx) error {
	// Create navigation context
	nav := navigation.NewContext(PageTitle, "zones", "add").
		AddBreadcrumb("Home", dashboard.Path, false).
		AddBreadcrumb("Dashboard", dashboard.Path, false).
		AddBreadcrumb(PageTitle, Path, true)

	// Parse form data
	form := &ZoneForm{}
	if err := c.BodyParser(form); err != nil {
		log.Error().Err(err).Msg("failed to parse add zone form")

		return c.Status(fiber.StatusBadRequest).Render(TemplateName, fiber.Map{
			"Navigation": nav,
			"Form":       form,
			"Error":      "Invalid form data",
		}, handler.BaseLayout)
	}

	// Ensure zone name ends with a dot
	if !strings.HasSuffix(form.Name, ".") {
		form.Name += "."
	}

	// Validate form
	if err := s.validator.Struct(form); err != nil {
		var validationErrors validator.ValidationErrors
		errors.As(err, &validationErrors)

		errorMessages := make([]string, len(validationErrors))
		for i, ve := range validationErrors {
			errorMessages[i] = "Field '" + ve.Field() + "' failed validation tag '" + ve.Tag() + "'"
		}

		log.Error().Err(err).Msg("validation failed for add zone")

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

	// Create zone via PowerDNS API
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	// Prepare common parameters
	var (
		nameservers   []string
		soaEditAPIStr = string(form.SOAEditAPI)
		createdZone   *pdnsapi.Zone
		err           error
	)

	switch form.Kind { // Create zone based on kind
	case ZoneKindNative:
		createdZone, err = powerdns.Engine.Zones.AddNative(
			ctx,
			form.Name,
			false, // dnssec
			"",    // nsec3Param
			false, // nsec3Narrow
			"",    // soaEdit
			soaEditAPIStr,
			false, // apiRectify
			nameservers,
		)
	case ZoneKindMaster:
		createdZone, err = powerdns.Engine.Zones.AddMaster(
			ctx,
			form.Name,
			false, // dnssec
			"",    // nsec3Param
			false, // nsec3Narrow
			"",    // soaEdit
			soaEditAPIStr,
			false, // apiRectify
			nameservers,
		)
	case ZoneKindSlave:
		var masters []string

		if form.Masters != "" {
			for _, master := range strings.Split(form.Masters, ",") {
				masters = append(masters, strings.TrimSpace(master))
			}
		}

		createdZone, err = powerdns.Engine.Zones.AddSlave(
			ctx,
			form.Name,
			masters,
		)
	}

	if err != nil {
		log.Error().
			Err(err).
			Str("zone_name", form.Name).
			Str("zone_kind", string(form.Kind)).
			Msg("failed to create zone")

		return c.Status(fiber.StatusInternalServerError).Render(TemplateName, fiber.Map{
			"Navigation": nav,
			"Form":       form,
			"Error":      "Failed to create zone: " + err.Error(),
		}, handler.BaseLayout)
	}

	_ = createdZone // Zone was created successfully

	log.Info().
		Str("zone_name", form.Name).
		Str("zone_kind", string(form.Kind)).
		Str("soa_edit_api", string(form.SOAEditAPI)).
		Msg("Zone created successfully")

	// Redirect to dashboard with success message
	return c.Redirect(dashboard.Path + "?success=Zone created successfully")
}
