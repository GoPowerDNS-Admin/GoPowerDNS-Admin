package auth

import "errors"

var (
	// ErrNoIDToken is returned when the OAuth2 token response doesn't contain an ID token.
	// This typically indicates a misconfigured OIDC provider or an incomplete authentication flow.
	ErrNoIDToken = errors.New("no id_token in token response")

	// ErrInvalidOldPassword is returned when the provided old password does not match the user's current password.
	ErrInvalidOldPassword = errors.New("invalid old password")

	// ErrUserNameOrEmailExists is returned when attempting to create a user with a username or email that already exists.
	ErrUserNameOrEmailExists = errors.New("user with username or email already exists")

	// ErrUserAccountDisabled is returned when attempting to authenticate a disabled user account.
	ErrUserAccountDisabled = errors.New("user account is disabled")

	// ErrInvalidPassword is returned when the provided password is incorrect during authentication.
	ErrInvalidPassword = errors.New("invalid password")

	// ErrUserNotFound is returned when a user cannot be found in the database or directory.
	ErrUserNotFound = errors.New("user not found")

	// ErrMultipleUsersFound is returned when a query expected one user but found multiple.
	// This typically indicates a misconfigured LDAP filter or duplicate entries.
	ErrMultipleUsersFound = errors.New("multiple users found")
)
