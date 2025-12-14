// Package models contains database model definitions.
package models

// Setting represents a configuration setting stored in the database.
// Settings provide a key-value store for application configuration that can be
// modified at runtime without requiring configuration file changes or restarts.
type Setting struct {
	// ID is the unique identifier for the setting.
	ID uint64 `gorm:"primaryKey"`
	// Name is the unique key for the setting (e.g., "smtp.host", "app.title").
	Name string `gorm:"unique"`
	// Value is the setting value stored as a binary blob to support any data type.
	// Values should be serialized (e.g., JSON) before storage and deserialized after retrieval.
	Value []byte `gorm:"type:blob"`
}
