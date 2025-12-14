// Package oidc provides handlers for OpenID Connect (OIDC) authentication flow.
//
// This package implements the OAuth2/OIDC authentication callback handler,
// supporting login, logout, and user provisioning from external identity providers
// such as Google, Okta, Keycloak, Azure AD, and other OIDC-compliant providers.
//
// The OIDC flow includes:
//   - Login initiation with CSRF protection via state tokens
//   - Authorization callback handling with ID token verification
//   - Automatic user creation/update from OIDC claims
//   - Group synchronization from OIDC group claims
//   - Session creation and cookie management
//   - Logout with provider end session support
//
// Example usage:
//
//	// Initialize OIDC handler
//	_ = oidc.Handler.Init(app, cfg, db)
//
//	// Users can then access:
//	// GET  /auth/oidc/login    - Initiate OIDC login flow
//	// GET  /auth/oidc/callback - Handle provider callback
//	// GET  /auth/oidc/logout   - Logout and optionally end provider session
package oidc
