// Package zone provides handlers for DNS zone record type settings management.
package zone

import (
	"errors"
	"regexp"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"

	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/auth"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/config"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/db/controller/setting"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web/handler"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web/handler/dashboard"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web/navigation"
)

const (
	// Path is the path to the zone record settings page.
	Path = handler.RootPath + "admin/settings/zone-records"

	// TemplateName is the name of the zone record settings template.
	TemplateName = "admin/settings/zone-records"
)

// Service is the zone record settings handler service.
type Service struct {
	handler.Service
	cfg       *config.Config
	db        *gorm.DB
	validator *validator.Validate
}

var (
	// Handler is the zone record settings handler.
	Handler = Service{}

	recordKeyRegex = regexp.MustCompile(`^records\[([A-Z0-9]+)]\.(\w+)$`)
)

// Init initializes the zone record settings handler.
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
		auth.RequirePermission(authService, auth.PermAdminZoneRecords),
		s.Get,
	)
	app.Post(Path,
		auth.RequirePermission(authService, auth.PermAdminZoneRecords),
		s.Post,
	)
}

// Get handles the zone record settings page rendering.
func (s *Service) Get(c *fiber.Ctx) error {
	// Create navigation context
	nav := navigation.NewContext("Zone Record Settings", "settings", "zone-records").
		AddBreadcrumb("Home", dashboard.Path, false).
		AddBreadcrumb("Settings", "#", false).
		AddBreadcrumb("Zone Records", Path, true)

	// Load zone record settings
	settings := &RecordSettings{}
	if err := settings.Load(s.db); err != nil {
		// If settings don't exist yet, use default config values
		if errors.Is(err, setting.ErrSettingNotFound) {
			log.Debug().Msg("zone record settings not found, using config defaults")

			settings.Records = s.cfg.Record

			return c.Render(TemplateName, fiber.Map{
				"Settings":   settings,
				"Navigation": nav,
			}, handler.BaseLayout)
		}

		// Log and return error for other failures
		log.Error().Err(err).Msg("failed to load zone record settings")

		return c.Status(fiber.StatusInternalServerError).SendString("Failed to load settings")
	}

	// Normalize settings.Records based on config: ensure presence, merge fields, and drop unknowns
	normalized := make(config.Record, len(s.cfg.Record))
	for k, v := range s.cfg.Record {
		merged := v
		if existing, ok := settings.Records[k]; ok {
			// Preserve boolean flags from existing settings
			merged.Forward = existing.Forward
			merged.Reverse = existing.Reverse
			// Prefer existing text when provided; otherwise keep defaults
			if existing.Description != "" {
				merged.Description = existing.Description
			}

			if existing.Help != "" {
				merged.Help = existing.Help
			}
		}

		normalized[k] = merged
	}

	settings.Records = normalized

	// Render form with loaded settings
	return c.Render(TemplateName, fiber.Map{
		"Settings":   settings,
		"Navigation": nav,
	}, handler.BaseLayout)
}

// Post handles the zone record settings form submission.
func (s *Service) Post(c *fiber.Ctx) error {
	// Create navigation context
	nav := navigation.NewContext("Zone Record Settings", "settings", "zone-records").
		AddBreadcrumb("Home", dashboard.Path, false).
		AddBreadcrumb("Settings", "#", false).
		AddBreadcrumb("Zone Records", Path, true)

	// Parse form data into settings struct
	settings := &RecordSettings{Records: make(config.Record)}

	// initialize settings with keys from config to ensure all keys are present
	// and preserve descriptions/help from config
	for k, v := range s.cfg.Record {
		settings.Records[k] = config.RecordTypeSettings{
			Description: v.Description, // Preserve description from config
			Help:        v.Help,        // Preserve help from config
		}
	}

	// Manually parse checkbox inputs for dynamic record types
	// Expected form keys: record[<TYPE>].forward and record[<TYPE>].reverse
	// where <TYPE> is the DNS record type (e.g., A, AAAA, CNAME)
	// Example keys: record[AAAA].forward, record[A].reverse
	// Values are "on" if checked, absent if unchecked
	// We iterate over all POST args to find these keys
	c.Request().PostArgs().VisitAll(func(key, value []byte) {
		matches := recordKeyRegex.FindStringSubmatch(string(key))
		if len(matches) != 3 {
			return // not a record setting key
		}

		recordType := matches[1] // e.g., "AAAA"
		fieldName := matches[2]  // e.g., "forward"

		// Get or create the record config
		recordConfig, exists := settings.Records[recordType]
		if !exists {
			recordConfig = config.RecordTypeSettings{}
		}

		switch fieldName {
		case "forward":
			// Parse checkbox value
			isChecked := string(value) == "true" || string(value) == "on"
			recordConfig.Forward = isChecked
		case "reverse":
			// Parse checkbox value
			isChecked := string(value) == "true" || string(value) == "on"
			recordConfig.Reverse = isChecked
		case "description":
			// Parse text value
			recordConfig.Description = string(value)
		case "help":
			// Parse text value for help
			recordConfig.Help = string(value)
		}

		settings.Records[recordType] = recordConfig
	})

	// Validate settings
	if err := s.validator.Struct(settings); err != nil {
		log.Error().Err(err).Msg("validation failed for zone record settings")

		return c.Status(fiber.StatusBadRequest).Render(TemplateName, fiber.Map{
			"Settings":   settings,
			"Navigation": nav,
			"Error":      "Validation failed: " + err.Error(),
		}, handler.BaseLayout)
	}

	// Save settings to database
	if err := settings.Save(s.db); err != nil {
		log.Error().Err(err).Msg("failed to save zone record settings")

		return c.Status(fiber.StatusInternalServerError).Render(TemplateName, fiber.Map{
			"Settings":   settings,
			"Navigation": nav,
			"Error":      "Failed to save settings",
		}, handler.BaseLayout)
	}

	// Log success
	log.Info().
		Int("record_types_configured", len(settings.Records)).
		Msg("zone record settings saved successfully")

	// Redirect to the same page with success message
	return c.Render(TemplateName, fiber.Map{
		"Settings":   settings,
		"Navigation": nav,
		"Success":    "Settings saved successfully",
	}, handler.BaseLayout)
}
