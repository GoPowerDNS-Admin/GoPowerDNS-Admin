package ttl

import (
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"

	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/auth"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/config"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web/handler"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web/handler/dashboard"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web/navigation"
)

const (
	// Path is the URL path for the TTL presets settings page.
	Path = handler.RootPath + "admin/settings/ttl-presets"

	// TemplateName is the template used for this page.
	TemplateName = "admin/settings/ttl-presets"
)

// Service is the TTL presets settings handler.
type Service struct {
	handler.Service
	db *gorm.DB
}

// Handler is the singleton handler instance.
var Handler = Service{}

// Init registers the routes.
func (s *Service) Init(app *fiber.App, _ *config.Config, db *gorm.DB, authService *auth.Service) {
	if app == nil || db == nil {
		log.Fatal().Msg(handler.ErrNilACDFatalLogMsg)
		return
	}

	s.db = db

	app.Get(Path,
		auth.RequirePermission(authService, auth.PermAdminTTLPresets),
		s.Get,
	)
	app.Post(Path,
		auth.RequirePermission(authService, auth.PermAdminTTLPresets),
		s.Post,
	)
}

func newNav() *navigation.Context {
	return navigation.NewContext("TTL Presets", "settings", "ttl-presets").
		AddBreadcrumb("Home", dashboard.Path, false).
		AddBreadcrumb("Settings", "#", false).
		AddBreadcrumb("TTL Presets", Path, true)
}

// Get renders the TTL presets settings page.
func (s *Service) Get(c fiber.Ctx) error {
	presets := LoadWithDefaults(s.db)

	return c.Render(TemplateName, fiber.Map{
		"Navigation": newNav(),
		"Presets":    presets,
	}, handler.BaseLayout)
}

// Post handles add and delete actions.
func (s *Service) Post(c fiber.Ctx) error {
	action := c.FormValue("action")
	nav := newNav()

	settings := &Settings{}
	if err := settings.Load(s.db); err != nil {
		settings.Presets = DefaultPresets()
	}

	switch action {
	case "add":
		secondsStr := strings.TrimSpace(c.FormValue("seconds"))
		label := strings.TrimSpace(c.FormValue("label"))

		if secondsStr == "" || label == "" {
			return c.Render(TemplateName, fiber.Map{
				"Navigation": nav,
				"Presets":    settings.Presets,
				"Error":      "Both seconds and label are required.",
			}, handler.BaseLayout)
		}

		sec, err := strconv.ParseUint(secondsStr, 10, 32)
		if err != nil || sec == 0 {
			return c.Render(TemplateName, fiber.Map{
				"Navigation": nav,
				"Presets":    settings.Presets,
				"Error":      "Seconds must be a positive integer.",
			}, handler.BaseLayout)
		}

		// Reject duplicate seconds values.
		for _, p := range settings.Presets {
			if uint64(p.Seconds) == sec {
				return c.Render(TemplateName, fiber.Map{
					"Navigation": nav,
					"Presets":    settings.Presets,
					"Error":      "A preset with that TTL value already exists.",
				}, handler.BaseLayout)
			}
		}

		settings.Presets = append(settings.Presets, Preset{Seconds: uint32(sec), Label: label})

	case "delete":
		secondsStr := c.FormValue("seconds")

		sec, err := strconv.ParseUint(secondsStr, 10, 32)
		if err != nil {
			return c.Redirect().To(Path)
		}

		filtered := settings.Presets[:0]
		for _, p := range settings.Presets {
			if uint64(p.Seconds) != sec {
				filtered = append(filtered, p)
			}
		}

		settings.Presets = filtered

	default:
		return c.Redirect().To(Path)
	}

	if err := settings.Save(s.db); err != nil {
		log.Error().Err(err).Msg("failed to save TTL presets")

		return c.Render(TemplateName, fiber.Map{
			"Navigation": nav,
			"Presets":    settings.Presets,
			"Error":      "Failed to save settings.",
		}, handler.BaseLayout)
	}

	return c.Render(TemplateName, fiber.Map{
		"Navigation": nav,
		"Presets":    settings.Presets,
		"Success":    "TTL presets saved.",
	}, handler.BaseLayout)
}
