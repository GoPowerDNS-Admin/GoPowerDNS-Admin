package login

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"

	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/config"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/db/models"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web/handler"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web/session"
)

const (
	// Path is the path to the login page.
	Path = "/login"
)

// Service is the login handler service.
type Service struct {
	handler.Service
	cfg *config.Config
	db  *gorm.DB
}

// Handler is the login handler.
var Handler = Service{}

// Init initializes the login handler.
func (s *Service) Init(app *fiber.App, cfg *config.Config, db *gorm.DB) error {
	if app == nil || cfg == nil || db == nil {
		return errors.New("app or db is nil")
	}

	s.db = db
	s.cfg = cfg

	// register routes
	app.Route(Path, func(router fiber.Router) {
		router.Get(handler.RouterRootPath, s.Get)
		router.Post(handler.RouterRootPath, s.Post)
	})

	return nil
}

// Get handles the login page rendering.
func (s *Service) Get(c *fiber.Ctx) error {
	return c.Render("login", fiber.Map{
		"local_db_enabled": true,
		"ldap_enabled":     false,
	})
}

// Post handles the login form submission.
func (s *Service) Post(c *fiber.Ctx) error {
	user := new(models.User)

	if err := c.BodyParser(user); err != nil {
		return err
	}

	// find user in db
	var dbUser models.User
	result := s.db.Where("username = ?", user.Username).First(&dbUser)
	if result.Error != nil {
		return c.Render("login", fiber.Map{
			"local_db_enabled": true,
			"ldap_enabled":     false,
			"error":            "Invalid username or password",
		})
	}

	// check if user is active
	if !dbUser.Active {
		return c.Render("login", fiber.Map{
			"local_db_enabled": true,
			"ldap_enabled":     false,
			"error":            "Data is inactive",
		})
	}

	// check if password matches
	if !dbUser.VerifyPassword(user.Password) {
		return c.Render("login", fiber.Map{
			"local_db_enabled": true,
			"ldap_enabled":     false,
			"error":            "Invalid username or password",
		})
	}

	sessionID, err := session.GenerateSessionID()
	if err != nil {
		log.Error().Err(err).Msg("failed to generate session ID")
		return c.Render("login", fiber.Map{
			"local_db_enabled": true,
			"ldap_enabled":     false,
			"error":            "Internal server error",
		})
	}

	userSession := &session.Data{
		User: dbUser,
	}

	if err = userSession.Write(sessionID, s.cfg.Webserver.Session.ExpiryTime); err != nil {
		log.Error().Err(err).Msg("failed to write session")
		return c.Render("login", fiber.Map{
			"local_db_enabled": true,
			"ldap_enabled":     false,
			"error":            "Internal server error",
		})
	}

	// set login cookie
	cookieSettings := &fiber.Cookie{
		Name:     "session",
		Value:    sessionID,
		MaxAge:   int(s.cfg.Webserver.Session.ExpiryTime.Seconds()),
		Secure:   true,
		HTTPOnly: true,
		SameSite: "Lax", // TODO: make this configurable
	}

	if s.cfg.DevMode {
		cookieSettings.Secure = false
	}

	c.Cookie(cookieSettings)

	return c.Redirect("/dashboard")
}
