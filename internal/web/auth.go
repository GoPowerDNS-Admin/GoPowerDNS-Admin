package web

import (
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web/handler/login"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web/session"
)

// AuthMiddleware is a Fiber middleware that checks for user authentication.
func AuthMiddleware(c *fiber.Ctx) error {
	var (
		isLoginPage   = IsLoginPage(c)
		sessDataValid bool
	)

	originalURL := strings.ToLower(c.OriginalURL())
	if strings.HasPrefix(originalURL, "/static") {
		return c.Next()
	}

	// get session cookie
	loginCookie := c.Cookies("session")

	// if no session cookie, redirect to login page
	if loginCookie == "" && !isLoginPage {
		return c.Redirect(login.Path)
	}

	// check session validity
	sessData := new(session.Data)
	_ = sessData.Read(loginCookie)

	// valid data in session
	if sessData.User.ID > 0 {
		sessDataValid = true
	}

	if sessDataValid && isLoginPage {
		return c.Redirect("/dashboard")
	}

	return c.Next()
}

// IsLoginPage checks if the current request is for the login page.
func IsLoginPage(c *fiber.Ctx) bool {
	originalURL := strings.ToLower(c.OriginalURL())
	return strings.HasPrefix(originalURL, login.Path)
}
