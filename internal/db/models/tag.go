package models

import "time"

// Tag is a label that can be assigned to zones, users, and groups
// to implement zone-level access control.
type Tag struct {
	ID          uint      `gorm:"primaryKey"`
	Name        string    `gorm:"unique;size:100;not null"`
	Description string    `gorm:"size:255"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// TableName overrides the default GORM table name.
func (Tag) TableName() string { return "tags" }

// ZoneTag links a PowerDNS zone (by its canonical name, e.g. "example.com.") to a tag.
type ZoneTag struct {
	ZoneID    string    `gorm:"primaryKey;column:zone_id;size:255"`
	TagID     uint      `gorm:"primaryKey;column:tag_id"`
	Tag       Tag       `gorm:"foreignKey:TagID;constraint:OnDelete:CASCADE"`
	CreatedAt time.Time
}

// TableName overrides the default GORM table name.
func (ZoneTag) TableName() string { return "zone_tags" }

// UserTag grants a user access to all zones carrying a specific tag.
type UserTag struct {
	UserID    uint64    `gorm:"primaryKey;column:user_id"`
	TagID     uint      `gorm:"primaryKey;column:tag_id"`
	User      User      `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	Tag       Tag       `gorm:"foreignKey:TagID;constraint:OnDelete:CASCADE"`
	CreatedAt time.Time
}

// TableName overrides the default GORM table name.
func (UserTag) TableName() string { return "user_tags" }

// GroupTag grants all members of a group access to zones carrying a specific tag.
type GroupTag struct {
	GroupID   uint      `gorm:"primaryKey;column:group_id"`
	TagID     uint      `gorm:"primaryKey;column:tag_id"`
	Group     Group     `gorm:"foreignKey:GroupID;constraint:OnDelete:CASCADE"`
	Tag       Tag       `gorm:"foreignKey:TagID;constraint:OnDelete:CASCADE"`
	CreatedAt time.Time
}

// TableName overrides the default GORM table name.
func (GroupTag) TableName() string { return "group_tags" }
