package models

import (
	"time"

	"github.com/alexedwards/argon2id"
	"github.com/rs/zerolog/log"
)

// User represents a user in the system.
type User struct {
	ID        uint64 `gorm:"primaryKey"`
	Active    bool
	Username  string    `gorm:"unique;size:100;not null"`
	Email     string    `gorm:"size:255;not null"`
	Password  string    `gorm:"size:255"`
	FirstName string    `gorm:"size:100"`
	LastName  string    `gorm:"size:100"`
	RoleID    uint      `gorm:"column:role_id;not null;default:0"` // Default role ID
	CreatedAt time.Time // Automatically managed by GORM for creation time
	UpdatedAt time.Time
	DeletedAt *time.Time // Soft delete field, nil if not deleted
}

// HashPassword hashes the given password using Argon algorithm.
func HashPassword(password string) string {
	hashedPassword, err := argon2id.CreateHash(password, argon2id.DefaultParams)
	if err != nil {
		log.Fatal().Msgf("failed to hash password: %v", err)
	}

	return hashedPassword
}

// VerifyPassword verifies the given password against the stored hashed password.
func (u User) VerifyPassword(password string) bool {
	match, err := argon2id.ComparePasswordAndHash(password, u.Password)
	if err != nil {
		log.Error().Msgf("failed to verify password: %v", err)
		return false
	}

	return match
}
