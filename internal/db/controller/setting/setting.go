// Package setting provides CRUD operations for managing application settings.
package setting

import (
	"errors"

	"gorm.io/gorm"

	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/db/models"
)

const (
	nameQueryPattern = "name = ?"
)

var (
	// ErrSettingNotFound is returned when a setting is not found.
	ErrSettingNotFound = errors.New("setting not found")
	// ErrSettingNameEmpty is returned when attempting to create/update a setting with an empty name.
	ErrSettingNameEmpty = errors.New("setting name cannot be empty")
	// ErrSettingAlreadyExists is returned when attempting to create a setting that already exists.
	ErrSettingAlreadyExists = errors.New("setting already exists")
	// ErrDBNil is returned when the database connection is nil.
	ErrDBNil = errors.New("database connection is nil")
)

// Get retrieves a setting by its name.
func Get(db *gorm.DB, name string) (*models.Setting, error) {
	if db == nil {
		return nil, ErrDBNil
	}
	if name == "" {
		return nil, ErrSettingNameEmpty
	}

	var setting models.Setting
	result := db.Where(nameQueryPattern, name).First(&setting)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, ErrSettingNotFound
		}
		return nil, result.Error
	}

	return &setting, nil
}

// GetByID retrieves a setting by its ID.
func GetByID(db *gorm.DB, id uint64) (*models.Setting, error) {
	if db == nil {
		return nil, ErrDBNil
	}

	var setting models.Setting
	result := db.First(&setting, id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, ErrSettingNotFound
		}
		return nil, result.Error
	}

	return &setting, nil
}

// GetAll retrieves all settings from the database.
func GetAll(db *gorm.DB) ([]models.Setting, error) {
	if db == nil {
		return nil, ErrDBNil
	}

	var settings []models.Setting
	result := db.Find(&settings)
	if result.Error != nil {
		return nil, result.Error
	}

	return settings, nil
}

// Create creates a new setting in the database.
func Create(db *gorm.DB, name string, value []byte) (*models.Setting, error) {
	if db == nil {
		return nil, ErrDBNil
	}
	if name == "" {
		return nil, ErrSettingNameEmpty
	}

	// Check if setting already exists
	var existing models.Setting
	result := db.Where(nameQueryPattern, name).First(&existing)
	if result.Error == nil {
		return nil, ErrSettingAlreadyExists
	}
	if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, result.Error
	}

	setting := &models.Setting{
		Name:  name,
		Value: value,
	}

	result = db.Create(setting)
	if result.Error != nil {
		return nil, result.Error
	}

	return setting, nil
}

// Set creates or updates a setting by name (upsert operation).
func Set(db *gorm.DB, name string, value []byte) (*models.Setting, error) {
	if db == nil {
		return nil, ErrDBNil
	}
	if name == "" {
		return nil, ErrSettingNameEmpty
	}

	var setting models.Setting
	result := db.Where(nameQueryPattern, name).First(&setting)

	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		// Setting doesn't exist, create it
		return Create(db, name, value)
	}
	if result.Error != nil {
		return nil, result.Error
	}

	// Setting exists, update it
	setting.Value = value
	result = db.Save(&setting)
	if result.Error != nil {
		return nil, result.Error
	}

	return &setting, nil
}

// Update updates an existing setting by ID.
func Update(db *gorm.DB, id uint64, value []byte) (*models.Setting, error) {
	if db == nil {
		return nil, ErrDBNil
	}

	var setting models.Setting
	result := db.First(&setting, id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, ErrSettingNotFound
		}
		return nil, result.Error
	}

	setting.Value = value
	result = db.Save(&setting)
	if result.Error != nil {
		return nil, result.Error
	}

	return &setting, nil
}

// UpdateByName updates an existing setting by name.
func UpdateByName(db *gorm.DB, name string, value []byte) (*models.Setting, error) {
	if db == nil {
		return nil, ErrDBNil
	}
	if name == "" {
		return nil, ErrSettingNameEmpty
	}

	var setting models.Setting
	result := db.Where(nameQueryPattern, name).First(&setting)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, ErrSettingNotFound
		}
		return nil, result.Error
	}

	setting.Value = value
	result = db.Save(&setting)
	if result.Error != nil {
		return nil, result.Error
	}

	return &setting, nil
}

// Delete deletes a setting by ID.
func Delete(db *gorm.DB, id uint64) error {
	if db == nil {
		return ErrDBNil
	}

	result := db.Delete(&models.Setting{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrSettingNotFound
	}

	return nil
}

// DeleteByName deletes a setting by name.
func DeleteByName(db *gorm.DB, name string) error {
	if db == nil {
		return ErrDBNil
	}
	if name == "" {
		return ErrSettingNameEmpty
	}

	result := db.Where(nameQueryPattern, name).Delete(&models.Setting{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrSettingNotFound
	}

	return nil
}
