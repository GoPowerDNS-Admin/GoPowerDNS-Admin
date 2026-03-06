package auth

import (
	"strings"

	"github.com/gofiber/fiber/v3"

	oidchandler "github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web/handler/auth/oidc"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web/handler/login"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web/session"
)

// Middleware is a Fiber middleware that checks for user authentication.
func Middleware(c fiber.Ctx) error {
	var (
		isLoginPage   = IsLoginPage(c)
		isLogoutPage  = IsLogoutPage(c)
		sessDataValid bool
	)

	originalURL := strings.ToLower(c.OriginalURL())
	if strings.HasPrefix(originalURL, "/static") {
		return c.Next()
	}

	// Allow logout and OIDC flow pages without authentication
	if isLogoutPage || isOIDCPage(c) {
		return c.Next()
	}

	// get session cookie
	loginCookie := c.Cookies("session")

	// if no session cookie, redirect to login page
	if loginCookie == "" && !isLoginPage {
		return c.Redirect().To(login.Path)
	}

	// check session validity
	sessData := new(session.Data)
	if err := sessData.Read(loginCookie); err != nil {
		// If we're already on the login page, don't redirect (would cause loop)
		if isLoginPage {
			return c.Next()
		}

		return c.Redirect().To(login.Path)
	}

	// valid data in session
	if sessData.User.ID > 0 {
		sessDataValid = true
		// Add the current user to locals for template access
		c.Locals("CurrentUser", sessData.User)
	}

	if sessDataValid && isLoginPage {
		return c.Redirect().To("/dashboard")
	}

	return c.Next()
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
