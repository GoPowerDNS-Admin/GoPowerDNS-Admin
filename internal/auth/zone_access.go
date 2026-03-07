package auth

import (
	"fmt"

	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/db/models"
)

// GetAccessibleZoneIDs returns the set of zone IDs the user may access.
//
// Returns nil when access is unrestricted (admin role, or the user/groups have
// no tag assignments at all — backward-compatible default).
// Returns a non-nil map (possibly empty) when tag restrictions are in effect;
// only zones whose ID appears in the map are accessible.
func (s *Service) GetAccessibleZoneIDs(userID uint64) (map[string]bool, error) {
	// Admin role always has unrestricted access.
	var user models.User
	if err := s.db.Preload("Role").First(&user, userID).Error; err != nil {
		return nil, fmt.Errorf("zone access: load user: %w", err)
	}

	if user.Role.Name == "admin" {
		return nil, nil //nolint:nilnil // nil map intentionally signals unrestricted access
	}

	// Count direct user-tag and group-tag assignments.
	var directCount int64
	if err := s.db.Model(&models.UserTag{}).Where("user_id = ?", userID).Count(&directCount).Error; err != nil {
		return nil, fmt.Errorf("zone access: count user tags: %w", err)
	}

	var groupCount int64
	if err := s.db.Table("group_tags").
		Joins("JOIN user_groups ON user_groups.group_id = group_tags.group_id").
		Where("user_groups.user_id = ?", userID).
		Count(&groupCount).Error; err != nil {
		return nil, fmt.Errorf("zone access: count group tags: %w", err)
	}

	// No assignments at all → unrestricted (backward compatible).
	if directCount == 0 && groupCount == 0 {
		return nil, nil //nolint:nilnil // nil map intentionally signals unrestricted access
	}

	// Collect all tag IDs the user has access to.
	tagSet := make(map[uint]struct{})

	var userTags []models.UserTag
	if err := s.db.Where("user_id = ?", userID).Find(&userTags).Error; err != nil {
		return nil, fmt.Errorf("zone access: load user tags: %w", err)
	}

	for i := range userTags {
		tagSet[userTags[i].TagID] = struct{}{}
	}

	type row struct{ TagID uint }

	var groupRows []row
	if err := s.db.Table("group_tags").
		Select("group_tags.tag_id").
		Joins("JOIN user_groups ON user_groups.group_id = group_tags.group_id").
		Where("user_groups.user_id = ?", userID).
		Scan(&groupRows).Error; err != nil {
		return nil, fmt.Errorf("zone access: load group tags: %w", err)
	}

	for _, r := range groupRows {
		tagSet[r.TagID] = struct{}{}
	}

	if len(tagSet) == 0 {
		// Has assignments but none resolved — deny everything.
		return map[string]bool{}, nil
	}

	tagIDs := make([]uint, 0, len(tagSet))
	for id := range tagSet {
		tagIDs = append(tagIDs, id)
	}

	// Collect zone IDs covered by these tags.
	var zoneTags []models.ZoneTag
	if err := s.db.Where("tag_id IN ?", tagIDs).Find(&zoneTags).Error; err != nil {
		return nil, fmt.Errorf("zone access: load zone tags: %w", err)
	}

	accessible := make(map[string]bool, len(zoneTags))
	for i := range zoneTags {
		accessible[zoneTags[i].ZoneID] = true
	}

	return accessible, nil
}
