package logout

import (
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"

	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/config"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web/handler"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web/handler/login"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web/session"
)

// Service is the logout handler service.
type Service struct {
	handler.Service
	cfg *config.Config
}

// Handler is the logout handler.
var Handler = Service{}

// Init initializes the logout handler.
func (s *Service) Init(app *fiber.App, cfg *config.Config) {
	if app == nil || cfg == nil {
		log.Fatal().Msg(handler.ErrNilACDFatalLogMsg)
		return
	}

	s.cfg = cfg

	// logout route (outside auth middleware protection)
	app.Get(handler.RootPath+"logout", s.Logout)
	app.Post(handler.RootPath+"logout", s.Logout)
}

// Logout handles user logout by clearing the session.
func (s *Service) Logout(c *fiber.Ctx) error {
	// Get session cookie
	sessionID := c.Cookies("session")
	if sessionID != "" {
		// Delete session from store
		if err := session.Store.Storage.Delete(sessionID); err != nil {
			log.Error().Err(err).Msg("failed to delete session")
		}
	}

	// Clear the session cookie
	c.Cookie(&fiber.Cookie{
		Name:     "session",
		Value:    "",
		MaxAge:   -1,
		Secure:   true,
		HTTPOnly: true,
		SameSite: "Lax",
	})

	return c.Redirect(login.Path)
}
