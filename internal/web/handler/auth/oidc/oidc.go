package oidc

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"

	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/activitylog"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/auth"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/config"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/db/models"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web/handler"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web/handler/dashboard"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web/session"
)

const (
	// LoginPath is the path to initiate OIDC login.
	LoginPath = handler.RootPath + "auth/oidc/login"

	// CallbackPath is the path for OIDC callback.
	CallbackPath = handler.RootPath + "auth/oidc/callback"

	// LogoutPath is the path for OIDC logout.
	LogoutPath = handler.RootPath + "auth/oidc/logout"
)

// Service is the OIDC handler service.
type Service struct {
	handler.Service
	cfg          *config.Config
	db           *gorm.DB
	oidcProvider *auth.OIDCProvider
	authService  *auth.Service
	stateMu      sync.Mutex
	stateStore   map[string]time.Time // in-memory state store protected by stateMu
}

// Handler is the OIDC handler.
var Handler = Service{
	stateStore: make(map[string]time.Time),
}

// Init initializes the OIDC handler.
func (s *Service) Init(app *fiber.App, cfg *config.Config, db *gorm.DB) {
	if app == nil || cfg == nil || db == nil {
		log.Fatal().Msg(handler.ErrNilACDFatalLogMsg)
		return
	}

	s.db = db
	s.cfg = cfg
	s.authService = auth.NewService(db)

	// Initialize OIDC provider if enabled
	if cfg.Auth.OIDC.Enabled {
		oidcConfig := auth.OIDCConfig{
			Enabled:      cfg.Auth.OIDC.Enabled,
			ProviderURL:  cfg.Auth.OIDC.ProviderURL,
			ClientID:     cfg.Auth.OIDC.ClientID,
			ClientSecret: cfg.Auth.OIDC.ClientSecret,
			RedirectURL:  cfg.Auth.OIDC.RedirectURL,
			Scopes:       cfg.Auth.OIDC.Scopes,
			GroupsClaim:  cfg.Auth.OIDC.GroupsClaim,
		}

		ctx := context.Background()

		oidcProvider, err := auth.NewOIDCProvider(ctx, &oidcConfig, db)
		if err != nil {
			if errors.Is(err, auth.ErrOIDCDisabled) {
				log.Info().Msg("OIDC authentication is disabled by configuration")
			} else {
				log.Warn().Err(err).Msg("Failed to initialize OIDC provider - OIDC authentication will be disabled")
			}

			return // Don't fail, just disable OIDC
		}

		s.oidcProvider = oidcProvider

		log.Info().Msg("OIDC authentication provider initialized")

		// Register routes
		app.Get(LoginPath, s.Login)
		app.Get(CallbackPath, s.Callback)
		app.Get(LogoutPath, s.Logout)

		// Start state cleanup goroutine
		go s.cleanupStates()
	}
}

// Login initiates the OIDC login flow.
func (s *Service) Login(c fiber.Ctx) error {
	if s.oidcProvider == nil {
		return c.Status(fiber.StatusServiceUnavailable).SendString("OIDC authentication is not available")
	}

	// Generate state token for CSRF protection
	state, err := auth.GenerateStateToken()
	if err != nil {
		log.Error().Err(err).Msg("Failed to generate state token")
		return c.Status(fiber.StatusInternalServerError).SendString("Internal server error")
	}

	// Store state with expiration (5 minutes)
	s.stateMu.Lock()
	s.stateStore[state] = time.Now().Add(5 * time.Minute)
	s.stateMu.Unlock()

	// Get authorization URL
	authURL := s.oidcProvider.GetAuthURL(state)

	// Redirect to OIDC provider
	return c.Redirect().To(authURL)
}

// Callback handles the OIDC callback.
func (s *Service) Callback(c fiber.Ctx) error {
	if s.oidcProvider == nil {
		return c.Status(fiber.StatusServiceUnavailable).SendString("OIDC authentication is not available")
	}

	// Get code and state from query parameters
	code := c.Query("code")
	state := c.Query("state")

	if code == "" || state == "" {
		log.Error().Msg("Missing code or state in OIDC callback")
		return c.Status(fiber.StatusBadRequest).SendString("Invalid callback parameters")
	}

	// Verify state
	s.stateMu.Lock()
	expiration, exists := s.stateStore[state]
	delete(s.stateStore, state)
	s.stateMu.Unlock()

	if !exists {
		log.Error().Str("state", state).Msg("Invalid state token")
		return c.Status(fiber.StatusBadRequest).SendString("Invalid state token")
	}

	if time.Now().After(expiration) {
		log.Error().Str("state", state).Msg("Expired state token")
		return c.Status(fiber.StatusBadRequest).SendString("Expired state token")
	}

	// Handle callback
	ctx := context.Background()

	authenticatedUser, groups, err := s.oidcProvider.HandleCallback(ctx, code)
	if err != nil {
		log.Error().Err(err).Msg("OIDC authentication failed")
		activitylog.Record(&activitylog.Entry{
			DB:           s.db,
			Action:       activitylog.ActionLoginFailed,
			ResourceType: activitylog.ResourceTypeAuth,
			Details:      map[string]any{"auth_type": "oidc", "reason": err.Error()},
			IPAddress:    c.IP(),
		})

		return c.Status(fiber.StatusUnauthorized).SendString("Authentication failed")
	}

	// Sync OIDC groups
	if len(groups) > 0 {
		if err = s.authService.SyncUserGroups(authenticatedUser.ID, groups, models.GroupSourceOIDC); err != nil {
			log.Error().Err(err).Uint64("user_id", authenticatedUser.ID).Msg("Failed to sync OIDC groups")
		}
	}

	// Create session
	sessionID, errSession := session.GenerateSessionID()
	if errSession != nil {
		log.Error().Err(errSession).Msg("Failed to generate session ID")
		return c.Status(fiber.StatusInternalServerError).SendString("Internal server error")
	}

	userSession := &session.Data{
		User: *authenticatedUser,
	}

	if err = userSession.Write(sessionID, s.cfg.Webserver.Session.ExpiryTime); err != nil {
		log.Error().Err(err).Msg("Failed to write session")
		return c.Status(fiber.StatusInternalServerError).SendString("Internal server error")
	}

	// Set login cookie
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

	log.Info().Str("username", authenticatedUser.Username).Msg("User logged in successfully via OIDC")

	userID := authenticatedUser.ID
	activitylog.Record(&activitylog.Entry{
		DB:           s.db,
		UserID:       &userID,
		Username:     authenticatedUser.Username,
		Action:       activitylog.ActionLogin,
		ResourceType: activitylog.ResourceTypeAuth,
		Details:      map[string]any{"auth_type": "oidc"},
		IPAddress:    c.IP(),
	})

	return c.Redirect().To(dashboard.Path)
}

// Logout handles OIDC logout.
func (s *Service) Logout(c fiber.Ctx) error {
	sessionID := c.Cookies("session")
	if sessionID != "" {
		sessData := new(session.Data)
		if err := sessData.Read(sessionID); err == nil && sessData.User.ID > 0 {
			userID := sessData.User.ID
			activitylog.Record(&activitylog.Entry{
				DB:           s.db,
				UserID:       &userID,
				Username:     sessData.User.Username,
				Action:       activitylog.ActionLogout,
				ResourceType: activitylog.ResourceTypeAuth,
				IPAddress:    c.IP(),
			})
		}

		if err := session.DeleteSession(sessionID); err != nil {
			log.Error().Err(err).Msg("Failed to delete session")
		}
	}

	// Expire the session cookie
	c.Cookie(&fiber.Cookie{
		Name:     "session",
		Value:    "",
		MaxAge:   -1,
		Secure:   true,
		HTTPOnly: true,
		SameSite: "Lax",
	})

	if s.oidcProvider != nil {
		postLogoutRedirectURI := s.cfg.Webserver.URL
		logoutURL := s.oidcProvider.GetLogoutURL("", postLogoutRedirectURI)

		if logoutURL != "" {
			return c.Redirect().To(logoutURL)
		}
	}

	return c.Redirect().To("/login")
}

// cleanupStates periodically removes expired state tokens.
func (s *Service) cleanupStates() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()

		s.stateMu.Lock()

		for state, expiration := range s.stateStore {
			if now.After(expiration) {
				delete(s.stateStore, state)
			}
		}

		s.stateMu.Unlock()
	}
}
