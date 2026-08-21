package auth

import (
	"errors"
	"strings"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"

	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/db/models"
	oidchandler "github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web/handler/auth/oidc"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web/handler/login"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web/session"
)

// New returns a Fiber middleware that enforces authentication.
//
// In addition to validating the session cookie, it revalidates the session's
// user against the database on every request so that an account which was
// deleted or deactivated while logged in is force-logged-out (session cleared
// and redirected to login) instead of hitting a confusing permission error or
// retaining access until the session expires.
func New(db *gorm.DB) fiber.Handler {
	return func(c fiber.Ctx) error {
		originalURL := strings.ToLower(c.OriginalURL())

		// Public paths and the auth flow itself never require a session.
		if isAssetPath(originalURL) || IsLogoutPage(c) || isOIDCPage(c) {
			return c.Next()
		}

		isLoginPage := IsLoginPage(c)
		loginCookie := c.Cookies("session")

		// No session cookie: allow the login page, redirect everything else.
		if loginCookie == "" {
			return loginOrRedirect(c, isLoginPage)
		}

		// Unreadable/expired session: same handling as no session.
		sessData := new(session.Data)
		if err := sessData.Read(loginCookie); err != nil {
			return loginOrRedirect(c, isLoginPage)
		}

		if sessData.User.ID > 0 {
			// Revalidate against the database: the account may have been deleted
			// or deactivated after the session was issued.
			if !userIsActive(db, sessData.User.ID) {
				invalidateSession(c, loginCookie)

				return loginOrRedirect(c, isLoginPage)
			}

			// Add the current user to locals for template access.
			c.Locals("CurrentUser", sessData.User)

			if isLoginPage {
				return c.Redirect().To("/dashboard")
			}
		}

		// If TOTP is pending, restrict to TOTP-related pages only.
		if sessData.TOTPPending && !isTOTPAllowedPage(c) {
			return totpRedirect(c, sessData)
		}

		return c.Next()
	}
}

// loginOrRedirect returns Next on the login page (to avoid a redirect loop) or a
// redirect to the login page otherwise.
func loginOrRedirect(c fiber.Ctx, isLoginPage bool) error {
	if isLoginPage {
		return c.Next()
	}

	return c.Redirect().To(login.Path)
}

// totpRedirect sends the user to the correct TOTP page for a pending challenge.
func totpRedirect(c fiber.Ctx, sessData *session.Data) error {
	if sessData.User.TOTPEnabled {
		return c.Redirect().To("/auth/totp/verify")
	}

	return c.Redirect().To("/profile/totp/setup")
}

// isAssetPath reports whether the URL targets a public, unauthenticated asset.
func isAssetPath(url string) bool {
	return strings.HasPrefix(url, "/static") ||
		strings.HasPrefix(url, "/branding") ||
		strings.HasPrefix(url, "/health")
}

// userIsActive reports whether a user with the given id exists and is active.
// A query error (including "record not found" for a deleted user) yields false
// so the caller fails closed and forces re-authentication.
func userIsActive(db *gorm.DB, userID uint64) bool {
	var user models.User

	err := db.Select("id", "active").First(&user, userID).Error
	if err != nil {
		return false
	}

	return user.Active
}

// invalidateSession removes the session from the store and expires the cookie.
func invalidateSession(c fiber.Ctx, sessionID string) {
	if sessionID != "" {
		if err := session.DeleteSession(sessionID); err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			// Best effort: the cookie is expired below regardless.
			_ = err
		}
	}

	c.Cookie(&fiber.Cookie{
		Name:     "session",
		Value:    "",
		MaxAge:   -1,
		Secure:   true,
		HTTPOnly: true,
		SameSite: "Lax",
	})
}

// isTOTPAllowedPage returns true if the request path is accessible during a pending TOTP challenge.
func isTOTPAllowedPage(c fiber.Ctx) bool {
	url := strings.ToLower(c.OriginalURL())

	return strings.HasPrefix(url, "/auth/totp") ||
		strings.HasPrefix(url, "/profile/totp/setup") ||
		strings.HasPrefix(url, "/logout") ||
		strings.HasPrefix(url, "/static") ||
		strings.HasPrefix(url, "/branding")
}

// IsLoginPage checks if the current request is for the login page.
func IsLoginPage(c fiber.Ctx) bool {
	originalURL := strings.ToLower(c.OriginalURL())
	return strings.HasPrefix(originalURL, login.Path)
}

// IsLogoutPage checks if the current request is for the logout page.
func IsLogoutPage(c fiber.Ctx) bool {
	originalURL := strings.ToLower(c.OriginalURL())
	return strings.HasPrefix(originalURL, "/logout")
}

// isOIDCPage checks if the current request is part of the OIDC authentication flow.
func isOIDCPage(c fiber.Ctx) bool {
	originalURL := strings.ToLower(c.OriginalURL())

	return strings.HasPrefix(originalURL, oidchandler.LoginPath) ||
		strings.HasPrefix(originalURL, oidchandler.CallbackPath) ||
		strings.HasPrefix(originalURL, oidchandler.LogoutPath)
}
