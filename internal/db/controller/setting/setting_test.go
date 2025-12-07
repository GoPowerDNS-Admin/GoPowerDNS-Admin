package setting

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

// seedSettings inserts test data into the database.
func seedSettings(t *testing.T, db *gorm.DB, settings []models.Setting) {
	t.Helper()
	for _, setting := range settings {
		err := db.Create(&setting).Error
		require.NoError(t, err, "failed to seed test data")
	}
}

func TestGet(t *testing.T) {
	db := setupTestDB(t)

	testCases := []struct {
		name          string
		dbParam       *gorm.DB
		settingName   string
		seedData      []models.Setting
		expectedError error
		expectedValue []byte
	}{
		{
			name:          "nil database",
			dbParam:       nil,
			settingName:   "test",
			expectedError: ErrDBNil,
		},
		{
			name:          "empty name",
			dbParam:       db,
			settingName:   "",
			expectedError: ErrSettingNameEmpty,
		},
		{
			name:          "setting not found",
			dbParam:       db,
			settingName:   "nonexistent",
			expectedError: ErrSettingNotFound,
		},
		{
			name:        "successful get",
			dbParam:     db,
			settingName: "site_name",
			seedData: []models.Setting{
				{Name: "site_name", Value: []byte("My Site")},
			},
			expectedValue: []byte("My Site"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Clean database for each test
			if tc.dbParam != nil {
				tc.dbParam.Exec("DELETE FROM settings")
			}

			if tc.seedData != nil {
				seedSettings(t, tc.dbParam, tc.seedData)
			}

			setting, err := Get(tc.dbParam, tc.settingName)

			if tc.expectedError != nil {
				require.Error(t, err)
				require.ErrorIs(t, err, tc.expectedError)
				assert.Nil(t, setting)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, setting)
				assert.Equal(t, tc.settingName, setting.Name)
				assert.Equal(t, tc.expectedValue, setting.Value)
			}
		})
	}
}

func TestGetByID(t *testing.T) {
	db := setupTestDB(t)

	testCases := []struct {
		name          string
		dbParam       *gorm.DB
		settingID     uint64
		seedData      []models.Setting
		expectedError error
		expectedName  string
		expectedValue []byte
	}{
		{
			name:          "nil database",
			dbParam:       nil,
			settingID:     1,
			expectedError: ErrDBNil,
		},
		{
			name:          "setting not found",
			dbParam:       db,
			settingID:     999,
			expectedError: ErrSettingNotFound,
		},
		{
			name:      "successful get by id",
			dbParam:   db,
			settingID: 1,
			seedData: []models.Setting{
				{Name: "site_name", Value: []byte("My Site")},
			},
			expectedName:  "site_name",
			expectedValue: []byte("My Site"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.dbParam != nil {
				tc.dbParam.Exec("DELETE FROM settings")
			}

			if tc.seedData != nil {
				seedSettings(t, tc.dbParam, tc.seedData)
			}

			setting, err := GetByID(tc.dbParam, tc.settingID)

			if tc.expectedError != nil {
				require.Error(t, err)
				require.ErrorIs(t, err, tc.expectedError)
				assert.Nil(t, setting)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, setting)
				assert.Equal(t, tc.expectedName, setting.Name)
				assert.Equal(t, tc.expectedValue, setting.Value)
			}
		})
	}
}

func TestGetAll(t *testing.T) {
	db := setupTestDB(t)

	testCases := []struct {
		name          string
		dbParam       *gorm.DB
		seedData      []models.Setting
		expectedError error
		expectedCount int
	}{
		{
			name:          "nil database",
			dbParam:       nil,
			expectedError: ErrDBNil,
		},
		{
			name:          "empty database",
			dbParam:       db,
			expectedCount: 0,
		},
		{
			name:    "multiple settings",
			dbParam: db,
			seedData: []models.Setting{
				{Name: "site_name", Value: []byte("My Site")},
				{Name: "admin_email", Value: []byte("admin@example.com")},
				{Name: "max_users", Value: []byte("100")},
			},
			expectedCount: 3,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.dbParam != nil {
				tc.dbParam.Exec("DELETE FROM settings")
			}

			if tc.seedData != nil {
				seedSettings(t, tc.dbParam, tc.seedData)
			}

			settings, err := GetAll(tc.dbParam)

			if tc.expectedError != nil {
				require.Error(t, err)
				require.ErrorIs(t, err, tc.expectedError)
				assert.Nil(t, settings)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, settings)
				assert.Len(t, settings, tc.expectedCount)
			}
		})
	}
}

func TestCreate(t *testing.T) {
	db := setupTestDB(t)

	testCases := []struct {
		name          string
		dbParam       *gorm.DB
		settingName   string
		settingValue  []byte
		seedData      []models.Setting
		expectedError error
	}{
		{
			name:          "nil database",
			dbParam:       nil,
			settingName:   "test",
			settingValue:  []byte("value"),
			expectedError: ErrDBNil,
		},
		{
			name:          "empty name",
			dbParam:       db,
			settingName:   "",
			settingValue:  []byte("value"),
			expectedError: ErrSettingNameEmpty,
		},
		{
			name:         "successful create",
			dbParam:      db,
			settingName:  "new_setting",
			settingValue: []byte("new_value"),
		},
		{
			name:         "duplicate setting",
			dbParam:      db,
			settingName:  "site_name",
			settingValue: []byte("Another Site"),
			seedData: []models.Setting{
				{Name: "site_name", Value: []byte("My Site")},
			},
			expectedError: ErrSettingAlreadyExists,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.dbParam != nil {
				tc.dbParam.Exec("DELETE FROM settings")
			}

			if tc.seedData != nil {
				seedSettings(t, tc.dbParam, tc.seedData)
			}

			setting, err := Create(tc.dbParam, tc.settingName, tc.settingValue)

			if tc.expectedError != nil {
				require.Error(t, err)
				require.ErrorIs(t, err, tc.expectedError)
				assert.Nil(t, setting)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, setting)
				assert.Equal(t, tc.settingName, setting.Name)
				assert.Equal(t, tc.settingValue, setting.Value)
				assert.NotZero(t, setting.ID)
			}
		})
	}
}

func TestSet(t *testing.T) {
	db := setupTestDB(t)

	testCases := []struct {
		name          string
		dbParam       *gorm.DB
		settingName   string
		settingValue  []byte
		seedData      []models.Setting
		expectedError error
		expectCreate  bool
	}{
		{
			name:          "nil database",
			dbParam:       nil,
			settingName:   "test",
			settingValue:  []byte("value"),
			expectedError: ErrDBNil,
		},
		{
			name:          "empty name",
			dbParam:       db,
			settingName:   "",
			settingValue:  []byte("value"),
			expectedError: ErrSettingNameEmpty,
		},
		{
			name:         "create new setting",
			dbParam:      db,
			settingName:  "new_setting",
			settingValue: []byte("new_value"),
			expectCreate: true,
		},
		{
			name:         "update existing setting",
			dbParam:      db,
			settingName:  "site_name",
			settingValue: []byte("Updated Site"),
			seedData: []models.Setting{
				{Name: "site_name", Value: []byte("My Site")},
			},
			expectCreate: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.dbParam != nil {
				tc.dbParam.Exec("DELETE FROM settings")
			}

			if tc.seedData != nil {
				seedSettings(t, tc.dbParam, tc.seedData)
			}

			setting, err := Set(tc.dbParam, tc.settingName, tc.settingValue)

			if tc.expectedError != nil {
				require.Error(t, err)
				require.ErrorIs(t, err, tc.expectedError)
				assert.Nil(t, setting)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, setting)
				assert.Equal(t, tc.settingName, setting.Name)
				assert.Equal(t, tc.settingValue, setting.Value)

				// Verify the setting was created or updated in the database
				var dbSetting models.Setting
				err = tc.dbParam.Where("name = ?", tc.settingName).First(&dbSetting).Error
				require.NoError(t, err)
				assert.Equal(t, tc.settingValue, dbSetting.Value)
			}
		})
	}
}

func TestUpdate(t *testing.T) {
	db := setupTestDB(t)

	testCases := []struct {
		name          string
		dbParam       *gorm.DB
		settingID     uint64
		settingValue  []byte
		seedData      []models.Setting
		expectedError error
	}{
		{
			name:          "nil database",
			dbParam:       nil,
			settingID:     1,
			settingValue:  []byte("value"),
			expectedError: ErrDBNil,
		},
		{
			name:          "setting not found",
			dbParam:       db,
			settingID:     999,
			settingValue:  []byte("value"),
			expectedError: ErrSettingNotFound,
		},
		{
			name:         "successful update",
			dbParam:      db,
			settingID:    1,
			settingValue: []byte("Updated Value"),
			seedData: []models.Setting{
				{Name: "site_name", Value: []byte("My Site")},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.dbParam != nil {
				tc.dbParam.Exec("DELETE FROM settings")
			}

			if tc.seedData != nil {
				seedSettings(t, tc.dbParam, tc.seedData)
			}

			setting, err := Update(tc.dbParam, tc.settingID, tc.settingValue)

			if tc.expectedError != nil {
				require.Error(t, err)
				require.ErrorIs(t, err, tc.expectedError)
				assert.Nil(t, setting)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, setting)
				assert.Equal(t, tc.settingValue, setting.Value)
			}
		})
	}
}

func TestUpdateByName(t *testing.T) {
	db := setupTestDB(t)

	testCases := []struct {
		name          string
		dbParam       *gorm.DB
		settingName   string
		settingValue  []byte
		seedData      []models.Setting
		expectedError error
	}{
		{
			name:          "nil database",
			dbParam:       nil,
			settingName:   "test",
			settingValue:  []byte("value"),
			expectedError: ErrDBNil,
		},
		{
			name:          "empty name",
			dbParam:       db,
			settingName:   "",
			settingValue:  []byte("value"),
			expectedError: ErrSettingNameEmpty,
		},
		{
			name:          "setting not found",
			dbParam:       db,
			settingName:   "nonexistent",
			settingValue:  []byte("value"),
			expectedError: ErrSettingNotFound,
		},
		{
			name:         "successful update",
			dbParam:      db,
			settingName:  "site_name",
			settingValue: []byte("Updated Site Name"),
			seedData: []models.Setting{
				{Name: "site_name", Value: []byte("My Site")},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.dbParam != nil {
				tc.dbParam.Exec("DELETE FROM settings")
			}

			if tc.seedData != nil {
				seedSettings(t, tc.dbParam, tc.seedData)
			}

			setting, err := UpdateByName(tc.dbParam, tc.settingName, tc.settingValue)

			if tc.expectedError != nil {
				require.Error(t, err)
				require.ErrorIs(t, err, tc.expectedError)
				assert.Nil(t, setting)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, setting)
				assert.Equal(t, tc.settingName, setting.Name)
				assert.Equal(t, tc.settingValue, setting.Value)
			}
		})
	}
}

func TestDelete(t *testing.T) {
	db := setupTestDB(t)

	testCases := []struct {
		name          string
		dbParam       *gorm.DB
		settingID     uint64
		seedData      []models.Setting
		expectedError error
	}{
		{
			name:          "nil database",
			dbParam:       nil,
			settingID:     1,
			expectedError: ErrDBNil,
		},
		{
			name:          "setting not found",
			dbParam:       db,
			settingID:     999,
			expectedError: ErrSettingNotFound,
		},
		{
			name:      "successful delete",
			dbParam:   db,
			settingID: 1,
			seedData: []models.Setting{
				{Name: "site_name", Value: []byte("My Site")},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.dbParam != nil {
				tc.dbParam.Exec("DELETE FROM settings")
			}

			if tc.seedData != nil {
				seedSettings(t, tc.dbParam, tc.seedData)
			}

			err := Delete(tc.dbParam, tc.settingID)

			if tc.expectedError != nil {
				require.Error(t, err)
				require.ErrorIs(t, err, tc.expectedError)
			} else {
				require.NoError(t, err)

				// Verify the setting was deleted
				var count int64
				tc.dbParam.Model(&models.Setting{}).Where("id = ?", tc.settingID).Count(&count)
				assert.Zero(t, count)
			}
		})
	}
}

func TestDeleteByName(t *testing.T) {
	db := setupTestDB(t)

	testCases := []struct {
		name          string
		dbParam       *gorm.DB
		settingName   string
		seedData      []models.Setting
		expectedError error
	}{
		{
			name:          "nil database",
			dbParam:       nil,
			settingName:   "test",
			expectedError: ErrDBNil,
		},
		{
			name:          "empty name",
			dbParam:       db,
			settingName:   "",
			expectedError: ErrSettingNameEmpty,
		},
		{
			name:          "setting not found",
			dbParam:       db,
			settingName:   "nonexistent",
			expectedError: ErrSettingNotFound,
		},
		{
			name:        "successful delete",
			dbParam:     db,
			settingName: "site_name",
			seedData: []models.Setting{
				{Name: "site_name", Value: []byte("My Site")},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.dbParam != nil {
				tc.dbParam.Exec("DELETE FROM settings")
			}

			if tc.seedData != nil {
				seedSettings(t, tc.dbParam, tc.seedData)
			}

			err := DeleteByName(tc.dbParam, tc.settingName)

			if tc.expectedError != nil {
				require.Error(t, err)
				require.ErrorIs(t, err, tc.expectedError)
			} else {
				require.NoError(t, err)

				// Verify the setting was deleted
				var count int64
				tc.dbParam.Model(&models.Setting{}).Where("name = ?", tc.settingName).Count(&count)
				assert.Zero(t, count)
			}
		})
	}
}

func TestIntegration(t *testing.T) {
	db := setupTestDB(t)

	// Create a setting
	setting, err := Create(db, "test_setting", []byte("initial_value"))
	require.NoError(t, err)
	require.NotNil(t, setting)
	assert.Equal(t, "test_setting", setting.Name)
	assert.Equal(t, []byte("initial_value"), setting.Value)

	// Get the setting by name
	retrieved, err := Get(db, "test_setting")
	require.NoError(t, err)
	assert.Equal(t, setting.ID, retrieved.ID)
	assert.Equal(t, []byte("initial_value"), retrieved.Value)

	// Get the setting by ID
	retrievedByID, err := GetByID(db, setting.ID)
	require.NoError(t, err)
	assert.Equal(t, "test_setting", retrievedByID.Name)
	assert.Equal(t, []byte("initial_value"), retrievedByID.Value)

	// Update the setting
	updated, err := UpdateByName(db, "test_setting", []byte("updated_value"))
	require.NoError(t, err)
	assert.Equal(t, []byte("updated_value"), updated.Value)

	// Verify the update
	retrieved, err = Get(db, "test_setting")
	require.NoError(t, err)
	assert.Equal(t, []byte("updated_value"), retrieved.Value)

	// Test Set (upsert) on existing setting
	upserted, err := Set(db, "test_setting", []byte("upserted_value"))
	require.NoError(t, err)
	assert.Equal(t, []byte("upserted_value"), upserted.Value)

	// Test Set (upsert) on new setting
	newSetting, err := Set(db, "another_setting", []byte("another_value"))
	require.NoError(t, err)
	assert.Equal(t, "another_setting", newSetting.Name)
	assert.Equal(t, []byte("another_value"), newSetting.Value)

	// Get all settings
	allSettings, err := GetAll(db)
	require.NoError(t, err)
	assert.Len(t, allSettings, 2)

	// Delete by name
	err = DeleteByName(db, "test_setting")
	require.NoError(t, err)

	// Verify deletion
	_, err = Get(db, "test_setting")
	require.Error(t, err)
	require.ErrorIs(t, err, ErrSettingNotFound)

	// Delete by ID
	err = Delete(db, newSetting.ID)
	require.NoError(t, err)

	// Verify all settings deleted
	allSettings, err = GetAll(db)
	require.NoError(t, err)
	assert.Empty(t, allSettings)
}
