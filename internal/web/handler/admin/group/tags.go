package group

import (
	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"

	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/db/models"
)

// parseGroupTagIDs reads the multi-value tag_ids form field and returns a slice of uint values.
func parseGroupTagIDs(c fiber.Ctx) []uint {
	vals := c.Request().PostArgs().PeekMulti("tag_ids")

	result := make([]uint, 0, len(vals))
	for _, v := range vals {
		n := 0
		ok := true

		for _, b := range v {
			if b < '0' || b > '9' {
				ok = false
				break
			}

			n = n*10 + int(b-'0')
		}

		if ok && n > 0 {
			result = append(result, uint(n))
		}
	}

	return result
}

// syncGroupTags replaces the GroupTag entries for the given group with the provided tag IDs.
func syncGroupTags(db *gorm.DB, groupID uint, tagIDs []uint) {
	db.Where("group_id = ?", groupID).Delete(&models.GroupTag{})

	for _, tagID := range tagIDs {
		db.Create(&models.GroupTag{GroupID: groupID, TagID: tagID})
	}
}
