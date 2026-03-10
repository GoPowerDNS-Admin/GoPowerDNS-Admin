// Package pdns provides middleware for requiring an initialized PowerDNS client.
package pdns

import (
	"github.com/gofiber/fiber/v3"

	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/powerdns"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web/handler/admin/settings/pdnsserver"
)

// RequireClient redirects to the PowerDNS server settings page when the
// PowerDNS client has not been initialized yet.
func RequireClient(c fiber.Ctx) error {
	if powerdns.Engine.Client == nil {
		return c.Redirect().To(pdnsserver.Path)
	}

	return c.Next()
}
