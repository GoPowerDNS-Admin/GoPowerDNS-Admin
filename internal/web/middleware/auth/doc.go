// Package auth provides authentication middleware for the web application.
//
// The middleware handles session validation, user authentication checks,
// and automatic redirection for unauthenticated requests. It also adds
// the current user to the request context for use in handlers and templates.
//
// The middleware performs the following tasks:
//   - Validates session cookies and redirects to login if invalid
//   - Adds current user information to fiber.Locals for template access
//   - Allows public access to login and logout pages
//   - Prevents redirect loops on authentication pages
//
// Usage:
//
//	app.Use(authmiddleware.Middleware)
//
// The middleware expects sessions to be managed by the session package
// and will redirect unauthenticated users to the login handler path.
package auth
