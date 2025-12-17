package models

import (
	"time"

	"github.com/alexedwards/argon2id"
	"github.com/rs/zerolog/log"
)

// AuthSource represents the authentication source for a user account.
// It indicates how the user authenticates (local database, LDAP, or OIDC).
type AuthSource string

const (
	// AuthSourceLocal indicates the user authenticates with a local database password.
	AuthSourceLocal AuthSource = "local"
	// AuthSourceOIDC indicates the user authenticates via OpenID Connect (OIDC).
	AuthSourceOIDC AuthSource = "oidc"
	// AuthSourceLDAP indicates the user authenticates via LDAP or Active Directory.
	AuthSourceLDAP AuthSource = "ldap"
)

// User represents a user account in the system.
// Users can authenticate via local database, LDAP, or OIDC.
// They are assigned roles and can belong to multiple groups for permission management.
type User struct {
	// ID is the unique identifier for the user.
	ID uint64 `gorm:"primaryKey"`
	// Active indicates whether the user account is active and can log in.
	Active bool
	// Username is the unique username for login.
	Username string `gorm:"unique;size:100;not null"`
	// Email is the user's email address.
	Email string `gorm:"size:255;not null"`
	// Password is the Argon2id hashed password (only used for local authentication).
	Password string `gorm:"size:255"`
	// FirstName is the user's first or given name.
	FirstName string `gorm:"size:100"`
	// LastName is the user's last or family name.
	LastName string `gorm:"size:100"`
	// RoleID is the ID of the role assigned to this user.
	RoleID uint `gorm:"column:role_id;not null"`
	// Role is the associated role (enforced with a foreign key constraint).
	Role Role `gorm:"foreignKey:RoleID;references:ID;constraint:OnDelete:RESTRICT,OnUpdate:CASCADE"`
	// AuthSource indicates how this user authenticates (local, oidc, or ldap).
	AuthSource AuthSource `gorm:"type:varchar(20);not null;default:'local'"`
	// ExternalID is the external identifier for OIDC (sub claim) or LDAP (DN) users.
	ExternalID string `gorm:"size:255"`
	// CreatedAt is the timestamp when the user was created (managed by GORM).
	CreatedAt time.Time
	// UpdatedAt is the timestamp when the user was last updated (managed by GORM).
	UpdatedAt time.Time
	// DeletedAt is the soft delete timestamp (nil if not deleted, managed by GORM).
	DeletedAt *time.Time
}

// HashPassword hashes a plaintext password using the Argon2id algorithm.
// This function should be used when creating or updating local user passwords.
// It uses the default Argon2id parameters for secure password hashing.
func HashPassword(password string) string {
	hashedPassword, err := argon2id.CreateHash(password, argon2id.DefaultParams)
	if err != nil {
		log.Fatal().Msgf("failed to hash password: %v", err)
	}

	return hashedPassword
}

// VerifyPassword verifies a plaintext password against the user's stored hashed password.
// It uses constant-time comparison to prevent timing attacks.
// Returns true if the password matches, false otherwise.
func (u *User) VerifyPassword(password string) bool {
	match, err := argon2id.ComparePasswordAndHash(password, u.Password)
	if err != nil {
		log.Error().Msgf("failed to verify password: %v", err)
		return false
	}

	return match
}
