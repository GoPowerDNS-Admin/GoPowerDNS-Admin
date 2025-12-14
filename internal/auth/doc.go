// Package auth provides authentication and authorization functionality for the application.
//
// This package implements a comprehensive Role-Based Access Control (RBAC) system
// with support for multiple authentication sources:
//   - Local database authentication with Argon2id password hashing
//   - LDAP/Active Directory authentication with group synchronization
//   - OpenID Connect (OIDC) authentication with external identity providers
//
// # Authentication Providers
//
// LocalProvider handles traditional username/password authentication against
// the local database with secure Argon2id password hashing.
//
// LDAPProvider connects to LDAP or Active Directory servers, authenticates users,
// and synchronizes their group memberships for permission mapping.
//
// OIDCProvider implements OAuth2/OIDC flows for authentication with external
// identity providers like Google, Okta, Keycloak, and Azure AD.
//
// # Authorization System
//
// The authorization system uses a flexible permission model:
//   - Users can have a direct role assignment
//   - Users can belong to multiple groups (local, LDAP, or OIDC)
//   - Groups are mapped to roles
//   - Roles contain a set of permissions
//   - Permissions are checked for resource access
//
// # Permission Checking
//
// The Service type provides methods for checking user permissions:
//   - HasPermission: Check if user has a specific permission
//   - HasAnyPermission: Check if user has at least one permission from a list
//   - HasAllPermissions: Check if user has all permissions from a list
//   - GetUserPermissions: Retrieve all permissions for a user
//
// # Middleware
//
// Fiber middleware functions are provided for route protection:
//   - RequirePermission: Protect routes requiring a specific permission
//   - RequireAnyPermission: Protect routes requiring any of several permissions
//   - RequireAllPermissions: Protect routes requiring all of several permissions
//   - AddPermissionsToLocals: Add user permissions to template context
//
// # Group Synchronization
//
// For LDAP and OIDC authentication, user groups are automatically synchronized:
//   - External groups are created or retrieved in the local database
//   - User group memberships are updated to match external groups
//   - Group-to-role mappings determine effective permissions
//   - Old group memberships are removed on each sync
//
// Example usage:
//
//	// Initialize auth service
//	authService := auth.NewService(db)
//
//	// Check permission in handler
//	hasPermission, err := authService.HasPermission(userID, auth.PermZoneCreate)
//
//	// Protect route with middleware
//	app.Get("/admin/users",
//	    auth.RequirePermission(authService, auth.PermAdminUsers),
//	    handler,
//	)
//
//	// LDAP authentication
//	ldapProvider, err := auth.NewLDAPProvider(ldapConfig, db)
//	user, groups, err := ldapProvider.Authenticate(username, password)
//	err = authService.SyncUserGroups(user.ID, groups, models.GroupSourceLDAP)
package auth
