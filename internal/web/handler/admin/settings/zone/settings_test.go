package zone

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/config"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/db/models"
)

// setupTestDB creates an in-memory SQLite database for testing.
func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	var err error

	var db *gorm.DB
	db, err = gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err, "failed to create test database")

	// Migrate the schema
	err = db.AutoMigrate(&models.Setting{})
	require.NoError(t, err, "failed to migrate test database")

	return db
}

func TestRecordSettings_Save(t *testing.T) {
	var err error

	db := setupTestDB(t)

	settings := &RecordSettings{
		Records: config.Record{
			"A": config.RecordTypeSettings{
				Forward: true,
				Reverse: false,
			},
			"AAAA": config.RecordTypeSettings{
				Forward: true,
				Reverse: false,
			},
			"CNAME": config.RecordTypeSettings{
				Forward: true,
				Reverse: false,
			},
		},
	}

	err = settings.Save(db)
	require.NoError(t, err)

	// Verify the setting was saved
	var savedSetting models.Setting
	err = db.Where("name = ?", SettingKeyZoneRecords).First(&savedSetting).Error
	require.NoError(t, err)
	assert.Equal(t, SettingKeyZoneRecords, savedSetting.Name)
	assert.NotEmpty(t, savedSetting.Value)
}

func TestRecordSettings_Load(t *testing.T) {
	var err error

	db := setupTestDB(t)

	// First save a setting
	original := &RecordSettings{
		Records: config.Record{
			"A": config.RecordTypeSettings{
				Forward: true,
				Reverse: false,
			},
			"PTR": config.RecordTypeSettings{
				Forward: true,
				Reverse: true,
			},
			"MX": config.RecordTypeSettings{
				Forward: true,
				Reverse: false,
			},
		},
	}

	err = original.Save(db)
	require.NoError(t, err)

	// Now load it into a new struct
	loaded := &RecordSettings{}
	err = loaded.Load(db)
	require.NoError(t, err)

	// Verify all fields match
	assert.Len(t, loaded.Records, len(original.Records))
	assert.Equal(t, original.Records["A"].Forward, loaded.Records["A"].Forward)
	assert.Equal(t, original.Records["A"].Reverse, loaded.Records["A"].Reverse)
	assert.Equal(t, original.Records["PTR"].Forward, loaded.Records["PTR"].Forward)
	assert.Equal(t, original.Records["PTR"].Reverse, loaded.Records["PTR"].Reverse)
	assert.Equal(t, original.Records["MX"].Forward, loaded.Records["MX"].Forward)
	assert.Equal(t, original.Records["MX"].Reverse, loaded.Records["MX"].Reverse)
}

func TestRecordSettings_Load_NotFound(t *testing.T) {
	var err error

	db := setupTestDB(t)

	settings := &RecordSettings{}
	err = settings.Load(db)
	require.Error(t, err)
}

func TestRecordSettings_SaveAndLoadMultipleTimes(t *testing.T) {
	var err error

	db := setupTestDB(t)

	// First save
	settings1 := &RecordSettings{
		Records: config.Record{
			"A": config.RecordTypeSettings{
				Forward: true,
				Reverse: false,
			},
		},
	}
	err = settings1.Save(db)
	require.NoError(t, err)

	// Update and save again
	settings2 := &RecordSettings{
		Records: config.Record{
			"A": config.RecordTypeSettings{
				Forward: true,
				Reverse: true,
			},
			"AAAA": config.RecordTypeSettings{
				Forward: false,
				Reverse: false,
			},
		},
	}
	err = settings2.Save(db)
	require.NoError(t, err)

	// Load and verify it has the latest values
	loaded := &RecordSettings{}
	err = loaded.Load(db)
	require.NoError(t, err)

	assert.Equal(t, settings2.Records["A"].Forward, loaded.Records["A"].Forward)
	assert.Equal(t, settings2.Records["A"].Reverse, loaded.Records["A"].Reverse)
	assert.Equal(t, settings2.Records["AAAA"].Forward, loaded.Records["AAAA"].Forward)
	assert.Equal(t, settings2.Records["AAAA"].Reverse, loaded.Records["AAAA"].Reverse)

	// Verify only one setting exists in the database
	var count int64
	err = db.Model(&models.Setting{}).Where("name = ?", SettingKeyZoneRecords).Count(&count).Error
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
}

func TestRecordSettings_EmptyRecords(t *testing.T) {
	var err error

	db := setupTestDB(t)

	// Save with empty records
	settings := &RecordSettings{
		Records: config.Record{},
	}
	err = settings.Save(db)
	require.NoError(t, err)

	// Load and verify
	loaded := &RecordSettings{}
	err = loaded.Load(db)
	require.NoError(t, err)

	assert.Empty(t, loaded.Records)
}

func TestRecordSettings_NilDatabase(t *testing.T) {
	var err error

	settings := &RecordSettings{
		Records: config.Record{
			"A": config.RecordTypeSettings{
				Forward: true,
				Reverse: false,
			},
		},
	}

	err = settings.Save(nil)
	require.Error(t, err)

	err = settings.Load(nil)
	require.Error(t, err)
}

func TestRecordSettings_AllRecordTypes(t *testing.T) {
	var err error

	db := setupTestDB(t)

	// Create settings with all common DNS record types
	settings := &RecordSettings{
		Records: config.Record{
			"A":          config.RecordTypeSettings{Forward: true, Reverse: false},
			"AAAA":       config.RecordTypeSettings{Forward: true, Reverse: false},
			"CNAME":      config.RecordTypeSettings{Forward: true, Reverse: false},
			"MX":         config.RecordTypeSettings{Forward: true, Reverse: false},
			"TXT":        config.RecordTypeSettings{Forward: true, Reverse: true},
			"NS":         config.RecordTypeSettings{Forward: true, Reverse: true},
			"PTR":        config.RecordTypeSettings{Forward: true, Reverse: true},
			"SRV":        config.RecordTypeSettings{Forward: true, Reverse: false},
			"CAA":        config.RecordTypeSettings{Forward: true, Reverse: false},
			"SOA":        config.RecordTypeSettings{Forward: false, Reverse: false},
			"DNSKEY":     config.RecordTypeSettings{Forward: false, Reverse: false},
			"DS":         config.RecordTypeSettings{Forward: false, Reverse: false},
			"NSEC":       config.RecordTypeSettings{Forward: false, Reverse: false},
			"NSEC3":      config.RecordTypeSettings{Forward: false, Reverse: false},
			"NSEC3PARAM": config.RecordTypeSettings{Forward: false, Reverse: false},
		},
	}

	err = settings.Save(db)
	require.NoError(t, err)

	// Load and verify all record types
	loaded := &RecordSettings{}
	err = loaded.Load(db)
	require.NoError(t, err)

	assert.Len(t, loaded.Records, len(settings.Records))

	for recordType, originalSettings := range settings.Records {
		loadedSettings, exists := loaded.Records[recordType]
		assert.True(t, exists, "Record type %s should exist", recordType)
		assert.Equal(t, originalSettings.Forward, loadedSettings.Forward,
			"Record type %s Forward mismatch", recordType)
		assert.Equal(t, originalSettings.Reverse, loadedSettings.Reverse,
			"Record type %s Reverse mismatch", recordType)
	}
}
