package group

import (
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"

	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/db/models"
)

// updateOrCreateGroupMapping updates or creates a group-role mapping in the database.
func (s *Service) updateOrCreateGroupMapping(c *fiber.Ctx, tx *gorm.DB, groupID, roleID uint) error {
	// Delete existing mapping
	if err := tx.Where("group_id = ?", groupID).Delete(&models.GroupMapping{}).Error; err != nil {
		tx.Rollback()
		log.Error().Err(err).Msg("failed to delete existing group mapping")

		return c.Status(fiber.StatusInternalServerError).SendString("Failed to update group role")
	}

	// Create new mapping
	groupMapping := models.GroupMapping{
		GroupID: groupID,
		RoleID:  roleID,
	}
	if err := tx.Create(&groupMapping).Error; err != nil {
		tx.Rollback()
		log.Error().Err(err).Msg("failed to create group mapping")

		return c.Status(fiber.StatusInternalServerError).SendString("Failed to assign role to group")
	}

	return nil
}
