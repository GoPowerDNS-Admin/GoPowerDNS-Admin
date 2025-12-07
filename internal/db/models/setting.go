// Package models contains database model definitions.
package models

// Setting represents a configuration setting stored in the database.
type Setting struct {
	ID    uint64 `gorm:"primaryKey"`
	Name  string `gorm:"unique"`
	Value []byte `gorm:"type:blob"`
}
