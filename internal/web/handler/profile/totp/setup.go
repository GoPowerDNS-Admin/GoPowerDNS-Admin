// Package profiletotp handles TOTP setup and management from the user profile.
package profiletotp

import (
	"bytes"
	"encoding/base32"
	"encoding/base64"
	"html/template"
	"image/png"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/pquerna/otp/totp"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"

	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/auth"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/config"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/db/models"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web/handler"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web/handler/dashboard"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web/session"
)

// Route and template paths for TOTP setup.
const (
	SetupPath     = "/profile/totp/setup"
	DisablePath   = "/profile/totp/disable"
	SetupTemplate = "profile/totp/setup"

	qrImageSize = 200 // pixels
)

// Service handles TOTP setup and disable.
type Service struct {
	handler.Service
	cfg *config.Config
	db  *gorm.DB
}

// Handler is the exported instance.
var Handler = Service{}

// Init registers routes.
func (s *Service) Init(app *fiber.App, cfg *config.Config, db *gorm.DB, _ *auth.Service) {
	s.cfg = cfg
	s.db = db
	app.Get(SetupPath, s.SetupGet)
	app.Post(SetupPath, s.SetupPost)
	app.Post(DisablePath, s.Disable)
}

// SetupGet renders the TOTP setup page with a QR code.
func (s *Service) SetupGet(c fiber.Ctx) error {
	sessionID := c.Cookies("session")

	sessData := new(session.Data)
	if err := sessData.Read(sessionID); err != nil || sessData.User.ID == 0 {
		return c.Redirect().To("/login")
	}

	if sessData.User.AuthSource != models.AuthSourceLocal {
		return c.Redirect().To("/profile")
	}

	// Generate a fresh temp secret if not already in the session
	tempSecret := sessData.TOTPTempSecret
	if tempSecret == "" {
		key, err := totp.Generate(totp.GenerateOpts{
			Issuer:      "GoPowerDNS-Admin",
			AccountName: sessData.User.Username,
		})
		if err != nil {
			log.Error().Err(err).Msg("failed to generate TOTP key")
			return c.Redirect().To("/profile")
		}

		tempSecret = key.Secret()

		sessData.TOTPTempSecret = tempSecret
		if err := sessData.Write(sessionID, s.cfg.Webserver.Session.ExpiryTime); err != nil {
			log.Error().Err(err).Msg("failed to write temp TOTP secret to session")
			return c.Redirect().To("/profile")
		}
	}

	qrDataURL, otpauthURL := generateQRDataURL(sessData.User.Username, tempSecret)

	return c.Render(SetupTemplate, fiber.Map{
		"QRDataURL":   qrDataURL,
		"OtpauthURL":  otpauthURL,
		"TempSecret":  tempSecret,
		"TOTPPending": sessData.TOTPPending,
	}, handler.BaseLayout)
}

// SetupPost verifies the submitted code and activates TOTP.
func (s *Service) SetupPost(c fiber.Ctx) error {
	sessionID := c.Cookies("session")

	sessData := new(session.Data)
	if err := sessData.Read(sessionID); err != nil || sessData.User.ID == 0 {
		return c.Redirect().To("/login")
	}

	if sessData.User.AuthSource != models.AuthSourceLocal {
		return c.Redirect().To("/profile")
	}

	if sessData.TOTPTempSecret == "" {
		return c.Redirect().To(SetupPath)
	}

	var form struct {
		Code string `form:"code"`
	}
	if err := c.Bind().Body(&form); err != nil || form.Code == "" {
		qrDataURL, otpauthURL := generateQRDataURL(sessData.User.Username, sessData.TOTPTempSecret)

		return c.Status(fiber.StatusBadRequest).Render(SetupTemplate, fiber.Map{
			"QRDataURL":   qrDataURL,
			"OtpauthURL":  otpauthURL,
			"TempSecret":  sessData.TOTPTempSecret,
			"TOTPPending": sessData.TOTPPending,
			"Error":       "Please enter the 6-digit code from your authenticator app.",
		}, handler.BaseLayout)
	}

	if !totp.Validate(form.Code, sessData.TOTPTempSecret) {
		qrDataURL, otpauthURL := generateQRDataURL(sessData.User.Username, sessData.TOTPTempSecret)

		return c.Status(fiber.StatusUnauthorized).Render(SetupTemplate, fiber.Map{
			"QRDataURL":   qrDataURL,
			"OtpauthURL":  otpauthURL,
			"TempSecret":  sessData.TOTPTempSecret,
			"TOTPPending": sessData.TOTPPending,
			"Error":       "Code is invalid. Please try again.",
		}, handler.BaseLayout)
	}

	// Save secret and enable TOTP on user
	if err := s.db.Model(&sessData.User).Updates(map[string]any{
		"totp_secret":  sessData.TOTPTempSecret,
		"totp_enabled": true,
	}).Error; err != nil {
		log.Error().Err(err).Msg("failed to save TOTP secret")
		return c.Redirect().To(SetupPath)
	}

	// Upgrade session: clear pending and temp secret, mark TOTP enabled
	confirmedSecret := sessData.TOTPTempSecret
	sessData.TOTPPending = false
	sessData.TOTPTempSecret = ""
	sessData.User.TOTPEnabled = true

	sessData.User.TOTPSecret = confirmedSecret
	if err := sessData.Write(sessionID, s.cfg.Webserver.Session.ExpiryTime); err != nil {
		log.Error().Err(err).Msg("failed to update session after TOTP setup")
	}

	return c.Redirect().To(dashboard.Path)
}

// Disable removes TOTP from the user's account.
func (s *Service) Disable(c fiber.Ctx) error {
	sessionID := c.Cookies("session")

	sessData := new(session.Data)
	if err := sessData.Read(sessionID); err != nil || sessData.User.ID == 0 {
		return c.Redirect().To("/login")
	}

	if sessData.User.AuthSource != models.AuthSourceLocal {
		return c.Redirect().To("/profile")
	}

	// Disallow if admin requires TOTP
	if sessData.User.TOTPRequired {
		return c.Redirect().To("/profile")
	}

	if err := s.db.Model(&models.User{}).Where("id = ?", sessData.User.ID).Updates(map[string]any{
		"totp_secret":  "",
		"totp_enabled": false,
	}).Error; err != nil {
		log.Error().Err(err).Msg("failed to disable TOTP")
	}

	sessData.User.TOTPEnabled = false

	sessData.User.TOTPSecret = ""
	if err := sessData.Write(sessionID, s.cfg.Webserver.Session.ExpiryTime); err != nil {
		log.Error().Err(err).Msg("failed to update session after TOTP disable")
	}

	return c.Redirect().To("/profile")
}

// generateQRDataURL generates a QR code PNG as a data URL and returns the otpauth URL.
// secret must be a base32-encoded string (as returned by key.Secret()).
func generateQRDataURL(username, secret string) (template.URL, string) {
	// Decode the base32 secret back to raw bytes so Generate doesn't double-encode it.
	secretBytes, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(secret))
	if err != nil {
		secretBytes = []byte(secret)
	}

	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "GoPowerDNS-Admin",
		AccountName: username,
		Secret:      secretBytes,
	})
	if err != nil {
		return "", ""
	}

	img, err := key.Image(qrImageSize, qrImageSize)
	if err != nil {
		return "", key.URL()
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return "", key.URL()
	}

	//nolint:gosec // data URL is server-generated PNG, not user input
	return template.URL("data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())), key.URL()
}
