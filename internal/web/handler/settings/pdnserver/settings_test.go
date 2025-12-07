package pdnserver

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/db/models"
)

// setupTestDB creates an in-memory SQLite database for testing.
func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err, "failed to create test database")

	// Migrate the schema
	err = db.AutoMigrate(&models.Setting{})
	require.NoError(t, err, "failed to migrate test database")

	return db
}

func TestPDNSServerSettings_Save(t *testing.T) {
	db := setupTestDB(t)

	settings := &PDNSServerSettings{
		APIServerURL: "https://pdns.example.com:8081",
		APIKey:       "secret-api-key-123",
		Version:      "4.8.0",
	}

	err := settings.Save(db)
	require.NoError(t, err)

	// Verify the setting was saved
	var savedSetting models.Setting
	err = db.Where("name = ?", SettingKeyPDNSServer).First(&savedSetting).Error
	require.NoError(t, err)
	assert.Equal(t, SettingKeyPDNSServer, savedSetting.Name)
	assert.NotEmpty(t, savedSetting.Value)
}

func TestPDNSServerSettings_Load(t *testing.T) {
	db := setupTestDB(t)

	// First save a setting
	original := &PDNSServerSettings{
		APIServerURL: "https://pdns.example.com:8081",
		APIKey:       "secret-api-key-123",
		Version:      "4.8.0",
	}

	err := original.Save(db)
	require.NoError(t, err)

	// Now load it into a new struct
	loaded := &PDNSServerSettings{}
	err = loaded.Load(db)
	require.NoError(t, err)

	// Verify all fields match
	assert.Equal(t, original.APIServerURL, loaded.APIServerURL)
	assert.Equal(t, original.APIKey, loaded.APIKey)
	assert.Equal(t, original.Version, loaded.Version)
}

func TestPDNSServerSettings_Load_NotFound(t *testing.T) {
	db := setupTestDB(t)

	settings := &PDNSServerSettings{}
	err := settings.Load(db)
	require.Error(t, err)
}

func TestPDNSServerSettings_SaveAndLoadMultipleTimes(t *testing.T) {
	db := setupTestDB(t)

	// First save
	settings1 := &PDNSServerSettings{
		APIServerURL: "https://pdns1.example.com:8081",
		APIKey:       "key1",
		Version:      "4.7.0",
	}
	err := settings1.Save(db)
	require.NoError(t, err)

	// Update and save again
	settings2 := &PDNSServerSettings{
		APIServerURL: "https://pdns2.example.com:8082",
		APIKey:       "key2",
		Version:      "4.8.0",
	}
	err = settings2.Save(db)
	require.NoError(t, err)

	// Load and verify it has the latest values
	loaded := &PDNSServerSettings{}
	err = loaded.Load(db)
	require.NoError(t, err)

	assert.Equal(t, settings2.APIServerURL, loaded.APIServerURL)
	assert.Equal(t, settings2.APIKey, loaded.APIKey)
	assert.Equal(t, settings2.Version, loaded.Version)

	// Verify only one setting exists in the database
	var count int64
	err = db.Model(&models.Setting{}).Where("name = ?", SettingKeyPDNSServer).Count(&count).Error
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
}

func TestPDNSServerSettings_EmptyValues(t *testing.T) {
	db := setupTestDB(t)

	// Save with empty values
	settings := &PDNSServerSettings{
		APIServerURL: "",
		APIKey:       "",
		Version:      "",
	}
	err := settings.Save(db)
	require.NoError(t, err)

	// Load and verify
	loaded := &PDNSServerSettings{}
	err = loaded.Load(db)
	require.NoError(t, err)

	assert.Empty(t, loaded.APIServerURL)
	assert.Empty(t, loaded.APIKey)
	assert.Empty(t, loaded.Version)
}

func TestPDNSServerSettings_NilDatabase(t *testing.T) {
	settings := &PDNSServerSettings{
		APIServerURL: "https://pdns.example.com:8081",
		APIKey:       "secret-api-key-123",
		Version:      "4.8.0",
	}

	err := settings.Save(nil)
	require.Error(t, err)

	err = settings.Load(nil)
	require.Error(t, err)
}
