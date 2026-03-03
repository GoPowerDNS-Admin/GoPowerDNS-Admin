package logout

import (
	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"

	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/activitylog"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/config"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web/handler"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web/handler/login"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web/session"
)

// Service is the logout handler service.
type Service struct {
	handler.Service
	cfg *config.Config
	db  *gorm.DB
}

// Handler is the logout handler.
var Handler = Service{}

// Init initializes the logout handler.
func (s *Service) Init(app *fiber.App, cfg *config.Config, db *gorm.DB) {
	if app == nil || cfg == nil || db == nil {
		log.Fatal().Msg(handler.ErrNilACDFatalLogMsg)
		return
	}

	s.cfg = cfg
	s.db = db

	// logout route (outside auth middleware protection)
	app.Get(handler.RootPath+"logout", s.Logout)
	app.Post(handler.RootPath+"logout", s.Logout)
}

// Logout handles user logout by clearing the session.
func (s *Service) Logout(c fiber.Ctx) error {
	// Get session cookie
	sessionID := c.Cookies("session")
	if sessionID != "" {
		// Read the session before deleting so we can record who logged out
		sessData := new(session.Data)
		if err := sessData.Read(sessionID); err == nil && sessData.User.ID > 0 {
			userID := sessData.User.ID
			activitylog.Record(
				&activitylog.Entry{
					DB:           s.db,
					UserID:       &userID,
					Username:     sessData.User.Username,
					Action:       activitylog.ActionLogout,
					ResourceType: activitylog.ResourceTypeAuth,
					IPAddress:    c.IP(),
				},
			)
		}

		// Delete session from the store
		if err := session.DeleteSession(sessionID); err != nil {
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

	return c.Redirect().To(login.Path)
}
