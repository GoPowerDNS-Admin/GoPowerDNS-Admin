package auth

import (
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/db/models"
)

// LocalProvider handles local database authentication.
type LocalProvider struct {
	db *gorm.DB
}

const (
	whereIDAndAuthSource = "id = ? AND auth_source = ?"

	whereID = "id = ?"
)

// NewLocalProvider creates a new local authentication provider.
func NewLocalProvider(db *gorm.DB) *LocalProvider {
	return &LocalProvider{
		db: db,
	}
}

// Authenticate authenticates a user against the local database.
func (p *LocalProvider) Authenticate(username, password string) (*models.User, error) {
	var user models.User

	// Find user by username
	err := p.db.Where("username = ? AND auth_source = ?", username, models.AuthSourceLocal).
		First(&user).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrUserNotFound
	}

	if err != nil {
		return nil, fmt.Errorf("failed to query user: %w", err)
	}

	// Check if user is active
	if !user.Active {
		return nil, ErrUserAccountDisabled
	}

	// Verify password
	if !user.VerifyPassword(password) {
		return nil, ErrInvalidPassword
	}

	// Update last login time (optional - would need to add field to User model)
	user.UpdatedAt = time.Now()
	p.db.Save(&user)

	return &user, nil
}

// CreateUser creates a new local user.
func (p *LocalProvider) CreateUser(
	username, email, password, firstName, lastName string,
	roleID uint,
) (*models.User, error) {
	// Check if user already exists
	var existingUser models.User

	err := p.db.Where("username = ? OR email = ?", username, email).First(&existingUser).Error
	if err == nil {
		return nil, ErrUserNameOrEmailExists
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("failed to check existing user: %w", err)
	}

	// Hash password
	hashedPassword := models.HashPassword(password)

	// Create user
	user := models.User{
		Active:     true,
		Username:   username,
		Email:      email,
		Password:   hashedPassword,
		FirstName:  firstName,
		LastName:   lastName,
		RoleID:     roleID,
		AuthSource: models.AuthSourceLocal,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if err := p.db.Create(&user).Error; err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return &user, nil
}

// UpdateUser updates an existing local user.
func (p *LocalProvider) UpdateUser(userID uint64, email, firstName, lastName string, roleID uint) error {
	updates := map[string]interface{}{
		"email":      email,
		"first_name": firstName,
		"last_name":  lastName,
		"role_id":    roleID,
		"updated_at": time.Now(),
	}

	return p.db.Model(&models.User{}).
		Where(whereIDAndAuthSource, userID, models.AuthSourceLocal).
		Updates(updates).Error
}

// ChangePassword changes a user's password.
func (p *LocalProvider) ChangePassword(userID uint64, oldPassword, newPassword string) error {
	var user models.User
	if err := p.db.Where(whereIDAndAuthSource, userID, models.AuthSourceLocal).
		First(&user).Error; err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	// Verify old password
	if !user.VerifyPassword(oldPassword) {
		return ErrInvalidOldPassword
	}

	// Hash new password
	hashedPassword := models.HashPassword(newPassword)

	// Update password
	return p.db.Model(&models.User{}).
		Where(whereID, userID).
		Update("password", hashedPassword).Error
}

// ResetPassword resets a user's password (admin function).
func (p *LocalProvider) ResetPassword(userID uint64, newPassword string) error {
	hashedPassword := models.HashPassword(newPassword)

	return p.db.Model(&models.User{}).
		Where(whereIDAndAuthSource, userID, models.AuthSourceLocal).
		Update("password", hashedPassword).Error
}

// ActivateUser activates a user account.
func (p *LocalProvider) ActivateUser(userID uint64) error {
	return p.db.Model(&models.User{}).
		Where(whereID, userID).
		Update("active", true).Error
}

// DeactivateUser deactivates a user account.
func (p *LocalProvider) DeactivateUser(userID uint64) error {
	return p.db.Model(&models.User{}).
		Where(whereID, userID).
		Update("active", false).Error
}

// DeleteUser soft deletes a user.
func (p *LocalProvider) DeleteUser(userID uint64) error {
	return p.db.Delete(&models.User{}, userID).Error
}

// GetUserByID retrieves a user by ID.
func (p *LocalProvider) GetUserByID(userID uint64) (*models.User, error) {
	var user models.User
	if err := p.db.First(&user, userID).Error; err != nil {
		return nil, err
	}

	return &user, nil
}

// GetUserByUsername retrieves a user by username.
func (p *LocalProvider) GetUserByUsername(username string) (*models.User, error) {
	var user models.User
	if err := p.db.Where("username = ?", username).First(&user).Error; err != nil {
		return nil, err
	}

	return &user, nil
}

// ListUsers lists all users with optional filters.
func (p *LocalProvider) ListUsers(
	authSource models.AuthSource,
	active *bool,
	limit, offset int,
) ([]models.User, int64, error) {
	var users []models.User

	var total int64

	query := p.db.Model(&models.User{})

	if authSource != "" {
		query = query.Where("auth_source = ?", authSource)
	}

	if active != nil {
		query = query.Where("active = ?", *active)
	}

	// Get total count
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get paginated results
	if err := query.Limit(limit).Offset(offset).Find(&users).Error; err != nil {
		return nil, 0, err
	}

	return users, total, nil
}
