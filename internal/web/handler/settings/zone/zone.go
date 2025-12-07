// Package zone provides handlers for DNS zone record type settings management.
package zone

import (
	"errors"
	"regexp"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"

	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/config"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/db/controller/setting"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web/handler"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web/navigation"
)

const (
	// Path is the path to the zone record settings page.
	Path = "settings/zone-records"
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
func (s *Service) Init(app *fiber.App, cfg *config.Config, db *gorm.DB) error {
	if app == nil || cfg == nil || db == nil {
		return errors.New("app, cfg, or db is nil")
	}

	s.db = db
	s.cfg = cfg
	s.validator = validator.New()

	// register routes
	app.Route("/"+Path, func(router fiber.Router) {
		router.Get(handler.RouterRootPath, s.Get)
		router.Post(handler.RouterRootPath, s.Post)
	})

	return nil
}

// Get handles the zone record settings page rendering.
func (s *Service) Get(c *fiber.Ctx) error {
	// Create navigation context
	nav := navigation.NewContext("Zone Record Settings", "settings", "zone-records").
		AddBreadcrumb("Home", "/dashboard", false).
		AddBreadcrumb("Settings", "#", false).
		AddBreadcrumb("Zone Records", "/settings/zone-records", true)

	// Load zone record settings
	settings := &RecordSettings{}
	if err := settings.Load(s.db); err != nil {
		// If settings don't exist yet, use default config values
		if errors.Is(err, setting.ErrSettingNotFound) {
			log.Debug().Msg("zone record settings not found, using config defaults")
			settings.Records = s.cfg.Record
			return c.Render(Path, fiber.Map{
				"Settings":   settings,
				"Navigation": nav,
			}, handler.BaseLayout)
		}

		// Log and return error for other failures
		log.Error().Err(err).Msg("failed to load zone record settings")
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to load settings")
	}

	// Ensure all record types from config are present in loaded settings
	if settings.Records == nil {
		settings.Records = s.cfg.Record
	} else {
		// check if loaded settings match config, if not use config defaults
		for k, v := range s.cfg.Record {
			if _, ok := settings.Records[k]; !ok {
				log.Debug().Str("record_type", k).Msg("adding missing record type from config defaults")
				settings.Records[k] = v
			}
		}

		// remove any record types not in config
		for k := range settings.Records {
			if _, ok := s.cfg.Record[k]; !ok {
				log.Debug().Str("record_type", k).Msg("removing unknown record type not in config")
				delete(settings.Records, k)
			}
		}
	}

	// Render form with loaded settings
	return c.Render(Path, fiber.Map{
		"Settings":   settings,
		"Navigation": nav,
	}, handler.BaseLayout)
}

// Post handles the zone record settings form submission.
func (s *Service) Post(c *fiber.Ctx) error {
	// Create navigation context
	nav := navigation.NewContext("Zone Record Settings", "settings", "zone-records").
		AddBreadcrumb("Home", "/dashboard", false).
		AddBreadcrumb("Settings", "#", false).
		AddBreadcrumb("Zone Records", "/settings/zone-records", true)

	// Parse form data into settings struct
	settings := &RecordSettings{Records: make(config.Record)}

	// initialize settings with keys from config to ensure all keys are present
	for k := range s.cfg.Record {
		settings.Records[k] = config.RecordTypeSettings{}
	}

	// Manually parse checkbox inputs for dynamic record types
	// Expected form keys: record[<TYPE>].forward and record[<TYPE>].reverse
	// where <TYPE> is the DNS record type (e.g., A, AAAA, CNAME)
	// Example keys: record[AAAA].forward, record[A].reverse
	// Values are "on" if checked, absent if unchecked
	// We iterate over all POST args to find these keys
	c.Request().PostArgs().VisitAll(func(key, value []byte) {
		matches := recordKeyRegex.FindStringSubmatch(string(key))
		if len(matches) != 3 { //nolint: mnd
			return // not a record setting key
		}

		recordType := matches[1] // e.g., "AAAA"
		fieldName := matches[2]  // e.g., "forward"

		// Get or create the record config
		recordConfig, exists := settings.Records[recordType]
		if !exists {
			recordConfig = config.RecordTypeSettings{}
		}

		// Parse checkbox value
		isChecked := string(value) == "true" || string(value) == "on"

		switch fieldName {
		case "forward":
			recordConfig.Forward = isChecked
		case "reverse":
			recordConfig.Reverse = isChecked
		}

		settings.Records[recordType] = recordConfig
	})

	// if err := c.BodyParser(settings); err != nil {
	//	log.Error().Err(err).Msg("failed to parse zone record settings form")
	//	return c.Status(fiber.StatusBadRequest).Render(Path, fiber.Map{
	//		"Settings":   settings,
	//		"Navigation": nav,
	//		"Error":      "Invalid form data",
	//	}, handler.BaseLayout)
	//}

	// Validate settings
	if err := s.validator.Struct(settings); err != nil {
		log.Error().Err(err).Msg("validation failed for zone record settings")
		return c.Status(fiber.StatusBadRequest).Render(Path, fiber.Map{
			"Settings":   settings,
			"Navigation": nav,
			"Error":      "Validation failed: " + err.Error(),
		}, handler.BaseLayout)
	}

	// Save settings to database
	if err := settings.Save(s.db); err != nil {
		log.Error().Err(err).Msg("failed to save zone record settings")
		return c.Status(fiber.StatusInternalServerError).Render(Path, fiber.Map{
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
	return c.Render(Path, fiber.Map{
		"Settings":   settings,
		"Navigation": nav,
		"Success":    "Settings saved successfully",
	}, handler.BaseLayout)
}
