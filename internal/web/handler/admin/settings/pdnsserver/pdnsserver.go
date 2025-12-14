package pdnsserver

import (
	"errors"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"

	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/auth"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/config"
	controller "github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/db/controller/pdnsserver"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/db/controller/setting"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/powerdns"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web/handler"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web/handler/dashboard"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web/navigation"
)

const (
	// Path is the path to the pdns-server settings page.
	Path = handler.RootPath + "admin/settings/pdns-server"

	// TemplateName is the name of the powerdns setting template.
	TemplateName = "admin/settings/pdns-server"
)

// Service is the pdns-server settings handler service.
type Service struct {
	handler.Service
	cfg       *config.Config
	db        *gorm.DB
	validator *validator.Validate
}

// Handler is the pdns-server settings handler.
var Handler = Service{}

// Init initializes the pdns-server settings handler.
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
		auth.RequirePermission(authService, auth.PermAdminPDNSServer),
		s.Get,
	)
	app.Post(Path,
		auth.RequirePermission(authService, auth.PermAdminPDNSServer),
		s.Post,
	)
}

// Get handles the pdns-server settings page rendering.
func (s *Service) Get(c *fiber.Ctx) error {
	// Create navigation context
	nav := navigation.NewContext("PowerDNS Server Settings", "settings", "pdns-server").
		AddBreadcrumb("Home", dashboard.Path, false).
		AddBreadcrumb("Settings", "", false).
		AddBreadcrumb("PowerDNS Server", Path, true)

	// Load PDNS server settings
	settings := &controller.Settings{}
	if err := settings.Load(s.db); err != nil {
		// If settings don't exist yet, render form with empty values
		if errors.Is(err, setting.ErrSettingNotFound) {
			log.Debug().Msg("PDNS server settings not found, rendering empty form")

			return c.Render(TemplateName, fiber.Map{
				"Settings":   settings,
				"Navigation": nav,
			}, handler.BaseLayout)
		}

		// Log and return error for other failures
		log.Error().Err(err).Msg("failed to load PDNS server settings")

		return c.Status(fiber.StatusInternalServerError).SendString("Failed to load settings")
	}

	// Render form with loaded settings
	return c.Render(
		TemplateName,
		fiber.Map{
			"Settings":   settings,
			"Navigation": nav,
		}, handler.BaseLayout)
}

// Post handles the pdns-server settings form submission.
func (s *Service) Post(c *fiber.Ctx) error {
	// Create navigation context
	nav := navigation.NewContext("PowerDNS Server Settings", "settings", "pdns-server").
		AddBreadcrumb("Home", "/dashboard", false).
		AddBreadcrumb("Settings", "#", false).
		AddBreadcrumb("PowerDNS Server", "/settings/pdns-server", true)

	// Parse form data into settings struct
	settings := &controller.Settings{}
	if err := c.BodyParser(settings); err != nil {
		log.Error().Err(err).Msg("failed to parse PDNS server settings form")

		return c.Status(fiber.StatusBadRequest).Render(
			TemplateName, fiber.Map{
				"Settings":   settings,
				"Navigation": nav,
				"Error":      "Invalid form data",
			}, handler.BaseLayout)
	}

	// Validate settings
	if err := s.validator.Struct(settings); err != nil {
		var validationErrors validator.ValidationErrors
		errors.As(err, &validationErrors)

		errorMessages := make([]string, len(validationErrors))
		for i, ve := range validationErrors {
			errorMessages[i] = "Field '" + ve.Field() + "' failed validation tag '" + ve.Tag() + "'"
		}

		log.Error().Err(err).Msg("validation failed for PDNS server settings")

		return c.Status(fiber.StatusBadRequest).Render(
			TemplateName, fiber.Map{
				"Settings":   settings,
				"Navigation": nav,
				"Error":      errorMessages,
			}, handler.BaseLayout)
	}

	// Save settings to database
	if err := settings.Save(s.db); err != nil {
		log.Error().Err(err).Msg("failed to save PDNS server settings")

		return c.Status(fiber.StatusInternalServerError).Render(
			TemplateName, fiber.Map{
				"Settings":   settings,
				"Navigation": nav,
				"Error":      "Failed to save settings",
			}, handler.BaseLayout)
	}

	// Log success
	log.Info().
		Str("api_server_url", settings.APIServerURL).
		Str("version", settings.VHost).
		Msg("PDNS server settings saved successfully")

	// Re-initialize PowerDNS engine with new settings asynchronously to avoid blocking the request
	go func(db *gorm.DB) {
		if err := powerdns.Open(db); err != nil {
			log.Error().Err(err).Msg("failed to initialize PowerDNS engine after settings update")
			return
		}

		// Test PowerDNS API connection with new settings (non-blocking, log-only)
		if err := powerdns.Engine.Test(); err != nil {
			log.Error().Err(err).Msg("failed to connect to PowerDNS API with new settings")
		}
	}(s.db)

	// Redirect to the same page with success message
	return c.Render(
		TemplateName, fiber.Map{
			"Settings":   settings,
			"Navigation": nav,
			"Success":    "Settings saved successfully",
		}, handler.BaseLayout)
}
