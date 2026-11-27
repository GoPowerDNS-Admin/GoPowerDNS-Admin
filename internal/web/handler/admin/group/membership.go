package group

import (
	"strconv"

	"gorm.io/gorm"

	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/db/models"
)

// reconcileMembershipForUpdate applies the correct membership strategy for a group update:
//   - local group: replace members from the submitted form;
//   - local -> external conversion: clear members so the directory becomes the sole
//     source of truth (repopulated on next login by SyncUserGroups);
//   - already external: leave members untouched (owned by SyncUserGroups).
//
// It operates within the given transaction and does not commit or roll back.
func (s *Service) reconcileMembershipForUpdate(
	tx *gorm.DB, groupID uint, wasExternal, isExternal bool, input *formInput,
) error {
	if !isExternal {
		return s.updateOrCreateGroupMembership(tx, groupID, input)
	}

	// Group is (now) external. Only clear rows when it just converted from local;
	// a group that was already external keeps its synced membership untouched.
	if !wasExternal {
		if err := tx.Where("group_id = ?", groupID).Delete(&models.UserGroup{}).Error; err != nil {
			return &saveStepError{message: "Failed to update group members", err: err}
		}
	}

	return nil
}

// updateOrCreateGroupMembership replaces the membership rows for a group within the
// given transaction. It does not commit or roll back; the caller owns the
// transaction lifecycle.
//
// This must only be called for locally-managed groups. For external groups
// (LDAP/OIDC) membership is authoritative in the directory and reconciled on login
// by auth.Service.SyncUserGroups, so manual edits are intentionally ignored.
func (s *Service) updateOrCreateGroupMembership(tx *gorm.DB, groupID uint, input *formInput) error {
	if err := tx.Where("group_id = ?", groupID).Delete(&models.UserGroup{}).Error; err != nil {
		return &saveStepError{message: "Failed to update group members", err: err}
	}

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
			return &saveStepError{message: "Failed to add users to group", err: err}
		}
	}

	return nil
}
