package login

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"

	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/auth"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/config"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/db/models"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web/handler"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web/handler/dashboard"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web/session"
)

const (
	// Path is the path to the login page.
	Path = handler.RootPath + "login"

	// TemplateName is the name of the login template.
	TemplateName = "login/login"
)

// Service is the login handler service.
type Service struct {
	handler.Service
	cfg         *config.Config
	db          *gorm.DB
	localAuth   *auth.LocalProvider
	ldapAuth    *auth.LDAPProvider
	authService *auth.Service
}

// Handler is the login handler.
var Handler = Service{}

// Init initializes the login handler.
func (s *Service) Init(app *fiber.App, cfg *config.Config, db *gorm.DB) {
	if app == nil || cfg == nil || db == nil {
		log.Fatal().Msg(handler.ErrNilACDFatalLogMsg)
		return
	}

	s.db = db
	s.cfg = cfg

	// Initialize auth providers
	s.localAuth = auth.NewLocalProvider(db)
	s.authService = auth.NewService(db)

	// Initialize LDAP provider if enabled
	s.initLDAP()

	// register routes
	app.Route(Path, func(router fiber.Router) {
		router.Get(handler.RootPath, s.Get)
		router.Post(handler.RootPath, s.Post)
	})

	// logout route (outside auth middleware protection)
	app.Get(handler.RootPath+"logout", s.Logout)
	app.Post(handler.RootPath+"logout", s.Logout)
}

// initLDAP initializes the LDAP auth provider when enabled, using guard clauses to reduce nesting.
func (s *Service) initLDAP() {
	if !s.cfg.Auth.LDAP.Enabled {
		return
	}

	ldapCfg := s.cfg.Auth.LDAP
	ldapConfig := auth.LDAPConfig{
		Enabled:          ldapCfg.Enabled,
		Host:             ldapCfg.Host,
		Port:             ldapCfg.Port,
		UseSSL:           ldapCfg.UseSSL,
		UseTLS:           ldapCfg.UseTLS,
		SkipVerify:       ldapCfg.SkipVerify,
		BindDN:           ldapCfg.BindDN,
		BindPassword:     ldapCfg.BindPassword,
		BaseDN:           ldapCfg.BaseDN,
		UserFilter:       ldapCfg.UserFilter,
		GroupBaseDN:      ldapCfg.GroupBaseDN,
		GroupFilter:      ldapCfg.GroupFilter,
		GroupMemberAttr:  ldapCfg.GroupMemberAttr,
		UsernameAttr:     ldapCfg.UsernameAttr,
		EmailAttr:        ldapCfg.EmailAttr,
		FirstNameAttr:    ldapCfg.FirstNameAttr,
		LastNameAttr:     ldapCfg.LastNameAttr,
		GroupNameAttr:    ldapCfg.GroupNameAttr,
		Timeout:          ldapCfg.Timeout,
		SearchAttributes: ldapCfg.SearchAttrs,
	}

	ldapProvider, err := auth.NewLDAPProvider(&ldapConfig, s.db)
	if err != nil {
		if errors.Is(err, auth.ErrLDAPDisabled) {
			log.Info().Msg("LDAP authentication is disabled by configuration")
			return
		}

		log.Warn().Err(err).Msg("Failed to initialize LDAP provider - LDAP authentication will be disabled")

		return
	}

	s.ldapAuth = ldapProvider

	log.Info().Msg("LDAP authentication provider initialized")
}

// Get handles the login page rendering.
func (s *Service) Get(c *fiber.Ctx) error {
	return c.Render(TemplateName, fiber.Map{
		"local_db_enabled": s.cfg.Auth.LocalDB.Enabled,
		"ldap_enabled":     s.cfg.Auth.LDAP.Enabled,
		"oidc_enabled":     s.cfg.Auth.OIDC.Enabled,
	})
}

// Post handles the login form submission.
func (s *Service) Post(c *fiber.Ctx) error {
	type LoginForm struct {
		Username string `form:"username"`
		Password string `form:"password"`
		AuthType string `form:"auth_type"` // "local", "ldap"
	}

	form := new(LoginForm)
	if err := c.BodyParser(form); err != nil {
		return s.renderError(c, ErrInvalidFormData.Error())
	}

	// Resolve and validate authentication type
	authType, err := s.pickAuthType(form.AuthType)
	if err != nil {
		return s.renderError(c, err.Error())
	}

	// Authenticate user according to the selected auth type
	authenticatedUser, err := s.authenticate(authType, form.Username, form.Password)
	if err != nil {
		return s.renderError(c, err.Error())
	}

	// Create session and set cookie
	if err := s.createSessionAndSetCookie(c, authenticatedUser); err != nil {
		return s.renderError(c, ErrInternalServerError.Error())
	}

	log.Info().Str("username", authenticatedUser.Username).Str("auth_type", authType).
		Msg("User logged in successfully")

	return c.Redirect(dashboard.Path)
}

// renderError renders the login page with an error message.
func (s *Service) renderError(c *fiber.Ctx, errorMsg string) error {
	return c.Render(TemplateName, fiber.Map{
		"local_db_enabled": s.cfg.Auth.LocalDB.Enabled,
		"ldap_enabled":     s.cfg.Auth.LDAP.Enabled,
		"oidc_enabled":     s.cfg.Auth.OIDC.Enabled,
		"error":            errorMsg,
	})
}

// pickAuthType determines which authentication method to use based on the request
// and the configuration. Returns an error when no suitable method is available
// or when an unsupported method is requested.
func (s *Service) pickAuthType(requested string) (string, error) {
	if requested == "" {
		if s.cfg.Auth.LocalDB.Enabled {
			return "local", nil
		}

		if s.cfg.Auth.LDAP.Enabled {
			return "ldap", nil
		}

		return "", ErrNoAuthMethod
	}

	switch requested {
	case "local":
		if !s.cfg.Auth.LocalDB.Enabled {
			return "", ErrLocalAuthDisabled
		}

		return "local", nil
	case "ldap":
		if !s.cfg.Auth.LDAP.Enabled || s.ldapAuth == nil {
			return "", ErrLDAPAuthDisabled
		}

		return "ldap", nil
	default:
		return "", ErrInvalidAuthMethod
	}
}

// authenticate performs the actual authentication using the selected method.
// It also takes care of LDAP group synchronization when applicable.
func (s *Service) authenticate(authType, username, password string) (*models.User, error) {
	switch authType {
	case "local":
		user, err := s.localAuth.Authenticate(username, password)
		if err != nil {
			log.Error().Err(err).Str("username", username).Msg("Local authentication failed")
			return nil, ErrInvalidCredentials
		}

		return user, nil
	case "ldap":
		user, groups, err := s.ldapAuth.Authenticate(username, password)
		if err != nil {
			log.Error().Err(err).Str("username", username).Msg("LDAP authentication failed")
			return nil, ErrInvalidCredentials
		}

		if err = s.authService.SyncUserGroups(user.ID, groups, models.GroupSourceLDAP); err != nil {
			log.Error().Err(err).Uint64("user_id", user.ID).Msg("Failed to sync LDAP groups")
		}

		return user, nil
	default:
		return nil, ErrInvalidAuthMethod
	}
}

// createSessionAndSetCookie creates a user session, writes it to the store,
// and sets the corresponding session cookie on the response.
func (s *Service) createSessionAndSetCookie(c *fiber.Ctx, user *models.User) error {
	sessionID, err := session.GenerateSessionID()
	if err != nil {
		log.Error().Err(err).Msg("failed to generate session ID")
		return err
	}

	userSession := &session.Data{User: *user}
	if err := userSession.Write(sessionID, s.cfg.Webserver.Session.ExpiryTime); err != nil {
		log.Error().Err(err).Msg("failed to write session")
		return err
	}

	cookieSettings := &fiber.Cookie{
		Name:     "session",
		Value:    sessionID,
		MaxAge:   int(s.cfg.Webserver.Session.ExpiryTime.Seconds()),
		Secure:   true,
		HTTPOnly: true,
		SameSite: "Lax",
	}
	if s.cfg.DevMode {
		cookieSettings.Secure = false
	}

	c.Cookie(cookieSettings)

	return nil
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

	return c.Redirect(Path)
}
