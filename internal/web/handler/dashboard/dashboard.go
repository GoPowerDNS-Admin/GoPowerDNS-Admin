package dashboard

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/config"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web/handler"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web/navigation"
)

const (
	// Path is the path to the dashboard page.
	Path = "dashboard"
)

// Service is the dashboard handler service.
type Service struct {
	handler.Service
	cfg *config.Config
	db  *gorm.DB
}

// Handler is the dashboard handler.
var Handler = Service{}

// Init initializes the dashboard handler.
func (s *Service) Init(app *fiber.App, cfg *config.Config, db *gorm.DB) error {
	if app == nil || cfg == nil || db == nil {
		return errors.New("app or db is nil")
	}

	s.db = db
	s.cfg = cfg

	// register routes
	app.Route(handler.RootPath+Path, func(router fiber.Router) {
		router.Get(handler.RouterRootPath, s.Get)
	})

	return nil
}

// Get handles the dashboard page rendering.
func (s *Service) Get(c *fiber.Ctx) error {
	// Create navigation context
	nav := navigation.NewContext("Dashboard", "dashboard", "dashboard").
		AddBreadcrumb("Home", Path, false).
		AddBreadcrumb("Dashboard", Path, true)

	return c.Render("dashboard", fiber.Map{
		"Navigation": nav,
	}, handler.BaseLayout)
}
