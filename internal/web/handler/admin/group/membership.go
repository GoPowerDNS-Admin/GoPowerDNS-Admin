package group

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"

	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/db/models"
)

// updateOrCreateGroupMembership updates or creates group memberships in the database.
func (s *Service) updateOrCreateGroupMembership(c *fiber.Ctx, tx *gorm.DB, groupID uint, input *formInput) error {
	// Delete existing group members
	if err := tx.Where("group_id = ?", groupID).Delete(&models.UserGroup{}).Error; err != nil {
		tx.Rollback()
		log.Error().Err(err).Msg("failed to delete existing group members")

		return c.Status(fiber.StatusInternalServerError).SendString("Failed to update group members")
	}

	// Create new user group memberships
	for _, userIDStr := range input.UserIDs {
		userID, err := strconv.ParseUint(userIDStr, 10, 64)
		if err != nil {
			continue // skip invalid IDs
		}

		userGroup := models.UserGroup{
			UserID:  userID,
			GroupID: groupID,
		}
		if err = tx.Create(&userGroup).Error; err != nil {
			tx.Rollback()
			log.Error().Err(err).Msg("failed to add user to group")

			return c.Status(fiber.StatusInternalServerError).SendString("Failed to add users to group")
		}
	}

	if err := tx.Commit().Error; err != nil {
		log.Error().Err(err).Msg("failed to commit transaction")
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to update group")
	}

	return nil
}
