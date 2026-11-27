package group

import (
	"gorm.io/gorm"

	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/db/models"
)

// reconcileGroupMapping sets the group's role mapping to roleID, or removes any
// existing mapping when roleID is zero. It operates within the given
// transaction and leaves rollback to the caller.
func (s *Service) reconcileGroupMapping(tx *gorm.DB, groupID, roleID uint) error {
	if roleID > 0 {
		return s.updateOrCreateGroupMapping(tx, groupID, roleID)
	}

	if err := tx.Where("group_id = ?", groupID).Delete(&models.GroupMapping{}).Error; err != nil {
		return &saveStepError{message: "Failed to remove group role", err: err}
	}

	return nil
}

// updateOrCreateGroupMapping replaces the group-role mapping within the given
// transaction and leaves rollback to the caller.
func (s *Service) updateOrCreateGroupMapping(tx *gorm.DB, groupID, roleID uint) error {
	if err := tx.Where("group_id = ?", groupID).Delete(&models.GroupMapping{}).Error; err != nil {
		return &saveStepError{message: "Failed to update group role", err: err}
	}

	groupMapping := models.GroupMapping{
		GroupID: groupID,
		RoleID:  roleID,
	}
	if err := tx.Create(&groupMapping).Error; err != nil {
		return &saveStepError{message: "Failed to assign role to group", err: err}
	}

	return nil
}
