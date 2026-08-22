package group

import (
	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"

	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/db/models"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web/handler"
)

// reconcileGroupMapping sets the group's role mapping to roleID, or removes any
// existing mapping when roleID is zero. It operates within the given transaction.
func (s *Service) reconcileGroupMapping(c fiber.Ctx, tx *gorm.DB, groupID, roleID uint) error {
	if roleID > 0 {
		return s.updateOrCreateGroupMapping(c, tx, groupID, roleID)
	}

	if err := tx.Where("group_id = ?", groupID).Delete(&models.GroupMapping{}).Error; err != nil {
		tx.Rollback()
		log.Error().Err(err).Msg("failed to remove group mapping")

		return handler.RenderError(c, fiber.StatusInternalServerError, "Save Failed", "Failed to remove group role", nil)
	}

	return nil
}

// updateOrCreateGroupMapping updates or creates a group-role mapping in the database.
func (s *Service) updateOrCreateGroupMapping(c fiber.Ctx, tx *gorm.DB, groupID, roleID uint) error {
	// Delete existing mapping
	if err := tx.Where("group_id = ?", groupID).Delete(&models.GroupMapping{}).Error; err != nil {
		tx.Rollback()
		log.Error().Err(err).Msg("failed to delete existing group mapping")

		return handler.RenderError(c, fiber.StatusInternalServerError, "Save Failed", "Failed to update group role", nil)
	}

	// Create new mapping
	groupMapping := models.GroupMapping{
		GroupID: groupID,
		RoleID:  roleID,
	}
	if err := tx.Create(&groupMapping).Error; err != nil {
		tx.Rollback()
		log.Error().Err(err).Msg("failed to create group mapping")

		return handler.RenderError(c, fiber.StatusInternalServerError, "Save Failed", "Failed to assign role to group", nil)
	}

	return nil
}
