// Package zoneadd provides the handler for adding new DNS zones.
package zoneadd

import (
	"context"
	"errors"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
	pdnsapi "github.com/joeig/go-powerdns/v3"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"

	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/activitylog"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/auth"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/config"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web/handler"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web/handler/dashboard"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web/navigation"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web/session"
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
func (s *Service) Get(c fiber.Ctx) error {
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
func (s *Service) Post(c fiber.Ctx) error {
	// Create navigation context
	nav := navigation.NewContext(PageTitle, "zones", "add").
		AddBreadcrumb("Home", dashboard.Path, false).
		AddBreadcrumb("Dashboard", dashboard.Path, false).
		AddBreadcrumb(PageTitle, Path, true)

	// Parse form data
	form := &ZoneForm{}
	if err := c.Bind().Body(form); err != nil {
		log.Error().Err(err).Msg("failed to parse add zone form")

		return c.Status(fiber.StatusBadRequest).Render(TemplateName, fiber.Map{
			"Navigation": nav,
			"Form":       form,
			"Error":      "Invalid form data",
		}, handler.BaseLayout)
	}

	// Default zone type to forward if not set
	if form.ZoneType == "" {
		form.ZoneType = ZoneTypeForward
	}

	// Compute zone name for reverse zones, or normalise forward zone name
	if err := resolveZoneName(form); err != nil {
		return c.Status(fiber.StatusBadRequest).Render(TemplateName, fiber.Map{
			"Navigation": nav,
			"Form":       form,
			"Error":      err.Error(),
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

		log.Error().Err(err).Msg("validation failed for add zone")

		return c.Status(fiber.StatusBadRequest).Render(TemplateName, fiber.Map{
			"Navigation": nav,
			"Form":       form,
			"Error":      errorMessages,
		}, handler.BaseLayout)
	}

	// Create zone via PowerDNS API
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	createdZone, err := createZone(ctx, form)
	if err != nil {
		var pdnsErr *pdnsapi.Error

		isConflict := (errors.As(err, &pdnsErr) && pdnsErr.StatusCode == fiber.StatusConflict) ||
			err.Error() == "Conflict"

		if isConflict {
			return c.Status(fiber.StatusConflict).Render(TemplateName, fiber.Map{
				"Navigation":   nav,
				"Form":         form,
				"ConflictZone": form.Name,
			}, handler.BaseLayout)
		}

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

	// Record activity: zone created
	var (
		userID   *uint64
		username string
	)

	if sid := c.Cookies("session"); sid != "" {
		sd := new(session.Data)
		if err := sd.Read(sid); err == nil && sd.User.ID > 0 {
			id := sd.User.ID
			userID = &id
			username = sd.User.Username
		}
	}

	activitylog.Record(
		&activitylog.Entry{
			DB: s.db, UserID: userID,
			Username:     username,
			Action:       activitylog.ActionZoneCreated,
			ResourceType: activitylog.ResourceTypeZone,
			ResourceName: form.Name,
			Details:      map[string]any{"kind": string(form.Kind), "soa_edit_api": string(form.SOAEditAPI)},
			IPAddress:    c.IP(),
		},
	)

	// Redirect to dashboard with success message
	return c.Redirect().To(dashboard.Path + "?success=Zone created successfully")
}
