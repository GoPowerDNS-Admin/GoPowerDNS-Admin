package daemon

import (
	"gorm.io/gorm"

	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/config"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/db/models"
)

func seed(_ *config.Config, db *gorm.DB) {
	// Seed initial data if user table is empty

	var count int64
	db.Model(&models.User{}).Count(&count)
	if count == 0 {
		// Create default admin user
		// salt and hash password in production

		db.Create(
			&models.User{
				Username: "admin",
				Password: models.HashPassword("changeme"),
				Active:   true,
				RoleID:   1, // assuming 1 is admin role
			},
		)
	}
}
