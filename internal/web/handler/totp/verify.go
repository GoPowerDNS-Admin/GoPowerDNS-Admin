// Package totp handles TOTP (Time-based One-Time Password) verification.
package totp

import (
	"github.com/gofiber/fiber/v3"
	"github.com/pquerna/otp/totp"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"

	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/config"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web/handler"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web/handler/dashboard"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web/session"
)

// Route and template paths for TOTP verification.
const (
	VerifyPath     = "/auth/totp/verify"
	VerifyTemplate = "totp/verify"
)

// Service handles TOTP verification.
type Service struct {
	handler.Service
	cfg *config.Config
	db  *gorm.DB
}

// Handler is the exported instance.
var Handler = Service{}

// Init registers routes.
func (s *Service) Init(app *fiber.App, cfg *config.Config, db *gorm.DB) {
	s.cfg = cfg
	s.db = db
	app.Get(VerifyPath, s.Get)
	app.Post(VerifyPath, s.Post)
}

// Get renders the TOTP verification page.
func (s *Service) Get(c fiber.Ctx) error {
	return c.Render(VerifyTemplate, fiber.Map{})
}

// Post handles TOTP code submission.
func (s *Service) Post(c fiber.Ctx) error {
	sessionID := c.Cookies("session")

	sessData := new(session.Data)
	if err := sessData.Read(sessionID); err != nil || !sessData.TOTPPending {
		return c.Redirect().To("/login")
	}

	var form struct {
		Code string `form:"code"`
	}
	if err := c.Bind().Body(&form); err != nil || form.Code == "" {
		return c.Status(fiber.StatusBadRequest).Render(VerifyTemplate, fiber.Map{
			"Error": "Please enter your 6-digit code.",
		})
	}

	if !totp.Validate(form.Code, sessData.User.TOTPSecret) {
		log.Warn().Uint64("user_id", sessData.User.ID).Msg("invalid TOTP code")

		return c.Status(fiber.StatusUnauthorized).Render(VerifyTemplate, fiber.Map{
			"Error": "Invalid code. Please try again.",
		})
	}

	// Upgrade session: clear pending flag
	sessData.TOTPPending = false
	if err := sessData.Write(sessionID, s.cfg.Webserver.Session.ExpiryTime); err != nil {
		log.Error().Err(err).Msg("failed to upgrade session after TOTP")
		return c.Redirect().To("/login")
	}

	return c.Redirect().To(dashboard.Path)
}
