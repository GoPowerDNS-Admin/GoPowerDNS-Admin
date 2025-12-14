package models

// RolePermission represents the many-to-many relationship between roles and permissions.
// This junction table maps which permissions are assigned to which roles.
// When a role is deleted, its permission assignments are automatically removed (CASCADE).
type RolePermission struct {
	// RoleID is the ID of the role in this mapping.
	RoleID uint `gorm:"primaryKey;column:role_id"`
	// PermissionID is the ID of the permission in this mapping.
	PermissionID uint `gorm:"primaryKey;column:permission_id"`
	// Role is the associated role (loaded via foreign key).
	Role Role `gorm:"foreignKey:RoleID;constraint:OnDelete:CASCADE"`
	// Permission is the associated permission (loaded via foreign key).
	Permission Permission `gorm:"foreignKey:PermissionID;constraint:OnDelete:CASCADE"`
}

// TableName specifies the database table name for the RolePermission model.
// This overrides GORM's default pluralized table naming.
func (RolePermission) TableName() string {
	return "role_permissions"
}
