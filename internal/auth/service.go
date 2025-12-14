package auth

import (
	"fmt"

	"gorm.io/gorm"

	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/db/models"
)

// Service provides authentication and authorization functionality.
type Service struct {
	db *gorm.DB
}

// NewService creates a new auth service.
func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

// HasPermission checks if a user has a specific permission.
// This works by checking if the user's role has the permission assigned,
// or if any of the user's groups map to roles with that permission.
func (s *Service) HasPermission(userID uint64, permission string) (bool, error) {
	var count int64

	// Check permissions from user's direct role
	err := s.db.Table("permissions").
		Joins("JOIN role_permissions ON role_permissions.permission_id = permissions.id").
		Joins("JOIN users ON users.role_id = role_permissions.role_id").
		Where("users.id = ? AND permissions.name = ?", userID, permission).
		Count(&count).Error
	if err != nil {
		return false, fmt.Errorf("failed to check direct role permission: %w", err)
	}

	if count > 0 {
		return true, nil
	}

	// Check permissions from user's groups (via group mappings)
	err = s.db.Table("permissions").
		Joins("JOIN role_permissions ON role_permissions.permission_id = permissions.id").
		Joins("JOIN group_mappings ON group_mappings.role_id = role_permissions.role_id").
		Joins("JOIN user_groups ON user_groups.group_id = group_mappings.group_id").
		Where("user_groups.user_id = ? AND permissions.name = ?", userID, permission).
		Count(&count).Error
	if err != nil {
		return false, fmt.Errorf("failed to check group permission: %w", err)
	}

	return count > 0, nil
}

// HasAnyPermission checks if a user has at least one of the given permissions.
func (s *Service) HasAnyPermission(userID uint64, permissions []string) (bool, error) {
	if len(permissions) == 0 {
		return false, nil
	}

	for _, perm := range permissions {
		has, err := s.HasPermission(userID, perm)
		if err != nil {
			return false, err
		}

		if has {
			return true, nil
		}
	}

	return false, nil
}

// HasAllPermissions checks if a user has all of the given permissions.
func (s *Service) HasAllPermissions(userID uint64, permissions []string) (bool, error) {
	if len(permissions) == 0 {
		return true, nil
	}

	for _, perm := range permissions {
		has, err := s.HasPermission(userID, perm)
		if err != nil {
			return false, err
		}

		if !has {
			return false, nil
		}
	}

	return true, nil
}

// GetUserPermissions retrieves all permissions for a user (from direct role and groups).
func (s *Service) GetUserPermissions(userID uint64) ([]string, error) {
	var permissions []string

	// Get permissions from user's direct role
	err := s.db.Table("permissions").
		Select("DISTINCT permissions.name").
		Joins("JOIN role_permissions ON role_permissions.permission_id = permissions.id").
		Joins("JOIN users ON users.role_id = role_permissions.role_id").
		Where("users.id = ?", userID).
		Pluck("permissions.name", &permissions).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get user permissions: %w", err)
	}

	// Get permissions from user's groups
	var groupPermissions []string

	err = s.db.Table("permissions").
		Select("DISTINCT permissions.name").
		Joins("JOIN role_permissions ON role_permissions.permission_id = permissions.id").
		Joins("JOIN group_mappings ON group_mappings.role_id = role_permissions.role_id").
		Joins("JOIN user_groups ON user_groups.group_id = group_mappings.group_id").
		Where("user_groups.user_id = ?", userID).
		Pluck("permissions.name", &groupPermissions).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get group permissions: %w", err)
	}

	// Merge and deduplicate permissions
	permMap := make(map[string]bool)
	for _, perm := range permissions {
		permMap[perm] = true
	}

	for _, perm := range groupPermissions {
		permMap[perm] = true
	}

	// Convert back to slice
	result := make([]string, 0, len(permMap))
	for perm := range permMap {
		result = append(result, perm)
	}

	return result, nil
}

// GetUserGroups retrieves all groups a user belongs to.
func (s *Service) GetUserGroups(userID uint64) ([]models.Group, error) {
	var groups []models.Group

	err := s.db.Table("groups").
		Joins("JOIN user_groups ON user_groups.group_id = groups.id").
		Where("user_groups.user_id = ?", userID).
		Find(&groups).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get user groups: %w", err)
	}

	return groups, nil
}

// SyncUserGroups synchronizes a user's groups with external groups.
// This is called after OIDC or LDAP authentication to update group memberships.
func (s *Service) SyncUserGroups(userID uint64, externalGroups []string, source models.GroupSource) error {
	// Start a transaction
	return s.db.Transaction(func(tx *gorm.DB) error {
		// Get or create groups for external groups
		var groupIDs []uint

		for _, externalGroup := range externalGroups {
			var group models.Group

			err := tx.Where("external_id = ? AND source = ?", externalGroup, source).
				FirstOrCreate(&group, models.Group{
					Name:       externalGroup,
					ExternalID: externalGroup,
					Source:     source,
				}).Error
			if err != nil {
				return fmt.Errorf("failed to create/get group %s: %w", externalGroup, err)
			}

			groupIDs = append(groupIDs, group.ID)
		}

		// Remove old group memberships for this source
		if err := tx.Where("user_id = ?", userID).
			Where("group_id IN (SELECT id FROM groups WHERE source = ?)", source).
			Delete(&models.UserGroup{}).Error; err != nil {
			return fmt.Errorf("failed to remove old group memberships: %w", err)
		}

		// Add new group memberships
		for _, groupID := range groupIDs {
			if err := tx.Create(&models.UserGroup{
				UserID:  userID,
				GroupID: groupID,
			}).Error; err != nil {
				return fmt.Errorf("failed to add group membership: %w", err)
			}
		}

		return nil
	})
}

// AssignRoleToUser assigns a role to a user (for local users).
func (s *Service) AssignRoleToUser(userID uint64, roleID uint) error {
	return s.db.Model(&models.User{}).
		Where("id = ?", userID).
		Update("role_id", roleID).Error
}
