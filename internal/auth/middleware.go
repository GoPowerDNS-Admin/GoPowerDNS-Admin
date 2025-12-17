package auth

import (
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"

	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web/session"
)

// RequirePermission creates Fiber middleware that requires a specific permission.
func RequirePermission(authService *Service, permission string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get session cookie
		sessionID := c.Cookies("session")
		if sessionID == "" {
			log.Error().Msg("No session cookie found")
			return c.Status(fiber.StatusUnauthorized).SendString("Unauthorized")
		}

		// Read session data
		sessionData := new(session.Data)
		if err := sessionData.Read(sessionID); err != nil {
			log.Error().Err(err).Msg("Failed to read session")
			return c.Status(fiber.StatusUnauthorized).SendString("Unauthorized")
		}

		// Check if the session is valid
		if sessionData.User.ID == 0 {
			log.Error().Msg("Invalid session data")
			return c.Status(fiber.StatusUnauthorized).SendString("Unauthorized")
		}

		// Check if the user has permission
		hasPermission, err := authService.HasPermission(sessionData.User.ID, permission)
		if err != nil {
			log.Error().Err(err).Uint64("user_id", sessionData.User.ID).Str("permission", permission).
				Msg("Failed to check permission")

			return c.Status(fiber.StatusInternalServerError).SendString("Internal Server Error")
		}

		if !hasPermission {
			log.Warn().Uint64("user_id", sessionData.User.ID).Str("permission", permission).
				Msg("User lacks required permission")

			return c.Status(fiber.StatusForbidden).SendString("Forbidden: You don't have permission to access this resource")
		}

		// User has permission, proceed
		return c.Next()
	}
}

// RequireAnyPermission creates Fiber middleware that requires at least one of the given permissions.
func RequireAnyPermission(authService *Service, permissions ...string) fiber.Handler { //nolint:dupl // ok for now
	return func(c *fiber.Ctx) error {
		// Get session cookie
		sessionID := c.Cookies("session")
		if sessionID == "" {
			return c.Status(fiber.StatusUnauthorized).SendString("Unauthorized")
		}

		// Read session data
		sessionData := new(session.Data)
		if err := sessionData.Read(sessionID); err != nil {
			return c.Status(fiber.StatusUnauthorized).SendString("Unauthorized")
		}

		if sessionData.User.ID == 0 {
			return c.Status(fiber.StatusUnauthorized).SendString("Unauthorized")
		}

		// Check if user has any of the permissions
		hasPermission, err := authService.HasAnyPermission(sessionData.User.ID, permissions)
		if err != nil {
			log.Error().Err(err).Uint64("user_id", sessionData.User.ID).Strs("permissions", permissions).
				Msg("Failed to check permissions")

			return c.Status(fiber.StatusInternalServerError).SendString("Internal Server Error")
		}

		if !hasPermission {
			log.Warn().Uint64("user_id", sessionData.User.ID).Strs("permissions", permissions).
				Msg("User lacks required permissions")

			return c.Status(fiber.StatusForbidden).SendString("Forbidden: You don't have permission to access this resource")
		}

		// User has at least one permission, proceed
		return c.Next()
	}
}

// RequireAllPermissions creates Fiber middleware that requires all the given permissions.
func RequireAllPermissions(authService *Service, permissions ...string) fiber.Handler { //nolint:dupl // ok for now
	return func(c *fiber.Ctx) error {
		// Get session cookie
		sessionID := c.Cookies("session")
		if sessionID == "" {
			return c.Status(fiber.StatusUnauthorized).SendString("Unauthorized")
		}

		// Read session data
		sessionData := new(session.Data)
		if err := sessionData.Read(sessionID); err != nil {
			return c.Status(fiber.StatusUnauthorized).SendString("Unauthorized")
		}

		if sessionData.User.ID == 0 {
			return c.Status(fiber.StatusUnauthorized).SendString("Unauthorized")
		}

		// Check if user has all permissions
		hasPermissions, err := authService.HasAllPermissions(sessionData.User.ID, permissions)
		if err != nil {
			log.Error().Err(err).Uint64("user_id", sessionData.User.ID).Strs("permissions", permissions).
				Msg("Failed to check permissions")

			return c.Status(fiber.StatusInternalServerError).SendString("Internal Server Error")
		}

		if !hasPermissions {
			log.Warn().Uint64("user_id", sessionData.User.ID).Strs("permissions", permissions).
				Msg("User lacks required permissions")

			return c.Status(fiber.StatusForbidden).SendString("Forbidden: You don't have permission to access this resource")
		}

		// User has all permissions, proceed
		return c.Next()
	}
}

// RequireAuthenticated ensures the user is authenticated (already handled by AuthMiddleware).
// This is a no-op middleware for explicit route protection.
func RequireAuthenticated() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Session already checked by AuthMiddleware
		return c.Next()
	}
}

// HasPermissionInContext checks if the current user in the Fiber context has a permission.
// Useful for conditional rendering in handlers.
func HasPermissionInContext(c *fiber.Ctx, authService *Service, permission string) bool {
	sessionID := c.Cookies("session")
	if sessionID == "" {
		return false
	}

	sessionData := new(session.Data)
	if err := sessionData.Read(sessionID); err != nil {
		return false
	}

	if sessionData.User.ID == 0 {
		return false
	}

	hasPermission, err := authService.HasPermission(sessionData.User.ID, permission)
	if err != nil {
		return false
	}

	return hasPermission
}

// GetUserPermissionsFromContext retrieves all permissions for the current user.
func GetUserPermissionsFromContext(c *fiber.Ctx, authService *Service) ([]string, error) {
	sessionID := c.Cookies("session")
	if sessionID == "" {
		return nil, nil
	}

	sessionData := new(session.Data)
	if err := sessionData.Read(sessionID); err != nil {
		return nil, err
	}

	if sessionData.User.ID == 0 {
		return nil, nil
	}

	return authService.GetUserPermissions(sessionData.User.ID)
}

// AddPermissionsToLocals is a Fiber middleware that adds user permissions to fiber.Locals.
// This allows templates to access permissions for conditional rendering.
func AddPermissionsToLocals(authService *Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		sessionID := c.Cookies("session")
		if sessionID == "" {
			// Not authenticated, continue without permissions
			return c.Next()
		}

		sessionData := new(session.Data)
		if err := sessionData.Read(sessionID); err != nil {
			// Invalid session, continue without permissions
			return c.Next()
		}

		if sessionData.User.ID == 0 {
			// Invalid session data, continue without permissions
			return c.Next()
		}

		permissions, err := authService.GetUserPermissions(sessionData.User.ID)
		if err != nil {
			log.Error().Err(err).Uint64("user_id", sessionData.User.ID).
				Msg("Failed to get user permissions")

			return c.Next()
		}

		// Add permissions to locals for template access
		c.Locals("permissions", permissions)
		c.Locals("hasPermission", func(perm string) bool {
			if has, errHas := authService.HasPermission(sessionData.User.ID, perm); errHas == nil {
				return has
			}

			return false
		})

		return c.Next()
	}
}
