package handler

import (
	"github.com/gofiber/fiber/v3"

	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web/navigation"
)

const errorTemplateName = "errors/error"

// ErrorAction is an optional call-to-action button shown on the error page.
type ErrorAction struct {
	Label string
	URL   string
	Icon  string // Bootstrap icon class, e.g. "bi-gear"
}

// ErrorData holds the content for the error page.
type ErrorData struct {
	Title   string
	Message string
	Action  *ErrorAction // optional; nil = show Dashboard button only
}

// RenderError renders the shared AdminLTE error page with the given HTTP status,
// title, and message. Pass a non-nil action to show a contextual CTA button.
func RenderError(c fiber.Ctx, status int, title, message string, action *ErrorAction) error {
	nav := navigation.NewContext("Error", "", "").
		AddBreadcrumb("Home", DashboardPath, false).
		AddBreadcrumb("Error", "", true)

	return c.Status(status).Render(errorTemplateName, fiber.Map{
		"Navigation":    nav,
		"DashboardPath": DashboardPath,
		"Error": ErrorData{
			Title:   title,
			Message: message,
			Action:  action,
		},
	}, BaseLayout)
}

// PDNSServerSettingsAction is the standard action pointing to PowerDNS server settings.
var PDNSServerSettingsAction = &ErrorAction{
	Label: "Server Settings",
	URL:   PDNSServerSettingsPath,
	Icon:  "bi-gear",
}
