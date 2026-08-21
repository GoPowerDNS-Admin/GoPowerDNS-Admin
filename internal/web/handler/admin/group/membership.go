package group

import (
	"strconv"

	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"

	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/db/models"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web/handler"
)

// updateOrCreateGroupMembership replaces the membership rows for a group within the
// given transaction. It does not commit; the caller owns the transaction lifecycle.
//
// This must only be called for locally-managed groups. For external groups
// (LDAP/OIDC) membership is authoritative in the directory and reconciled on login
// by auth.Service.SyncUserGroups, so manual edits are intentionally ignored.
func (s *Service) updateOrCreateGroupMembership(c fiber.Ctx, tx *gorm.DB, groupID uint, input *formInput) error {
	// Delete existing group members
	if err := tx.Where("group_id = ?", groupID).Delete(&models.UserGroup{}).Error; err != nil {
		tx.Rollback()
		log.Error().Err(err).Msg("failed to delete existing group members")

		return handler.RenderError(c, fiber.StatusInternalServerError, "Save Failed", "Failed to update group members", nil)
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

			return handler.RenderError(c, fiber.StatusInternalServerError, "Save Failed", "Failed to add users to group", nil)
		}
	}

	return nil
}
