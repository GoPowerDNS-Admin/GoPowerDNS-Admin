package daemon

import (
	"errors"

	"github.com/rs/zerolog/log"
	"gorm.io/gorm"

	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/config"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/db/models"
)

func seed(_ *config.Config, db *gorm.DB) {
	// Seed roles
	seedRoles(db)

	// Seed permissions
	seedPermissions(db)

	// Seed role-permission mappings
	seedRolePermissions(db)

	// Seed default admin user
	seedUsers(db)
}

// seedRoles creates default roles.
func seedRoles(db *gorm.DB) {
	roles := []models.Role{
		{
			Name:        "admin",
			Description: "Full system access with all permissions",
			IsSystem:    true,
		},
		{
			Name:        "user",
			Description: "Standard user with zone management permissions",
			IsSystem:    true,
		},
		{
			Name:        "viewer",
			Description: "Read-only access to zones and dashboards",
			IsSystem:    true,
		},
	}

	for _, role := range roles {
		var existingRole models.Role

		err := db.Where(models.WhereNameIs, role.Name).First(&existingRole).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if err = db.Create(&role).Error; err != nil {
				log.Error().Err(err).Str("role", role.Name).Msg("Failed to create role")
			} else {
				log.Info().Str("role", role.Name).Msg("Created role")
			}
		}
	}
}

// seedPermissions creates default permissions.
func seedPermissions(db *gorm.DB) {
	permissions := []models.Permission{
		// Dashboard permissions
		{
			Name:        "dashboard.view",
			Resource:    "dashboard",
			Action:      "view",
			Description: "View dashboard",
		},

		// Zone permissions
		{
			Name:        "zone.create",
			Resource:    "zone",
			Action:      "create",
			Description: "Create DNS zones",
		},
		{
			Name:        "zone.read",
			Resource:    "zone",
			Action:      "read",
			Description: "View DNS zones",
		},
		{
			Name:        "zone.update",
			Resource:    "zone",
			Action:      "update",
			Description: "Update DNS zones",
		},
		{
			Name:        "zone.delete",
			Resource:    "zone",
			Action:      "delete",
			Description: "Delete DNS zones",
		},
		{
			Name:        "zone.list",
			Resource:    "zone",
			Action:      "list",
			Description: "List DNS zones",
		},

		// Admin permissions
		{
			Name:        "admin.settings",
			Resource:    "admin",
			Action:      "settings",
			Description: "Manage application settings",
		},
		{
			Name:        "admin.server.config",
			Resource:    "admin",
			Action:      "server.config",
			Description: "View server configuration",
		},
		{
			Name:        "admin.pdns.server",
			Resource:    "admin",
			Action:      "pdns.server",
			Description: "Manage PowerDNS server settings",
		},
		{
			Name:        "admin.zone.records",
			Resource:    "admin",
			Action:      "zone.records",
			Description: "Manage zone record type settings",
		},
		{
			Name:        "admin.users",
			Resource:    "admin",
			Action:      "users",
			Description: "Manage users",
		},
		{
			Name:        "admin.roles",
			Resource:    "admin",
			Action:      "roles",
			Description: "Manage roles",
		},
		{
			Name:        "admin.groups",
			Resource:    "admin",
			Action:      "groups",
			Description: "Manage groups",
		},
		{
			Name:        "admin.group.mappings",
			Resource:    "admin",
			Action:      "group.mappings",
			Description: "Manage group-to-role mappings",
		},
	}

	for _, perm := range permissions {
		var existingPerm models.Permission

		err := db.Where(models.WhereNameIs, perm.Name).First(&existingPerm).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if err = db.Create(&perm).Error; err != nil {
				log.Error().Err(err).Str("permission", perm.Name).Msg("Failed to create permission")
			} else {
				log.Debug().Str("permission", perm.Name).Msg("Created permission")
			}
		}
	}
}

// seedRolePermissions creates role-permission mappings.
func seedRolePermissions(db *gorm.DB) {
	// Get roles
	var adminRole, userRole, viewerRole models.Role
	db.Where(models.WhereNameIs, "admin").First(&adminRole)
	db.Where(models.WhereNameIs, "user").First(&userRole)
	db.Where(models.WhereNameIs, "viewer").First(&viewerRole)

	// Get all permissions
	var allPermissions []models.Permission
	db.Find(&allPermissions)

	// Admin gets all permissions
	for _, perm := range allPermissions {
		var existing models.RolePermission

		err := db.Where("role_id = ? AND permission_id = ?", adminRole.ID, perm.ID).
			First(&existing).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			db.Create(&models.RolePermission{
				RoleID:       adminRole.ID,
				PermissionID: perm.ID,
			})
		}
	}

	// User gets zone and dashboard permissions
	userPermissions := []string{
		"dashboard.view",
		"zone.create",
		"zone.read",
		"zone.update",
		"zone.delete",
		"zone.list",
	}
	assignPermissionsToRole(db, userRole.ID, userPermissions)

	// Viewer gets read-only permissions
	viewerPermissions := []string{
		"dashboard.view",
		"zone.read",
		"zone.list",
		"admin.server.config",
	}
	assignPermissionsToRole(db, viewerRole.ID, viewerPermissions)

	log.Info().Msg("Role-permission mappings created")
}

// assignPermissionsToRole assigns a list of permission names to a role.
func assignPermissionsToRole(db *gorm.DB, roleID uint, permissionNames []string) {
	for _, permName := range permissionNames {
		var perm models.Permission
		if err := db.Where(models.WhereNameIs, permName).First(&perm).Error; err == nil {
			var existing models.RolePermission

			err := db.Where("role_id = ? AND permission_id = ?", roleID, perm.ID).
				First(&existing).Error
			if errors.Is(err, gorm.ErrRecordNotFound) {
				db.Create(&models.RolePermission{
					RoleID:       roleID,
					PermissionID: perm.ID,
				})
			}
		}
	}
}

// seedUsers creates the default admin user.
func seedUsers(db *gorm.DB) {
	var count int64
	db.Model(&models.User{}).Count(&count)

	if count == 0 {
		// Get admin role
		var adminRole models.Role
		db.Where(models.WhereNameIs, "admin").First(&adminRole)

		// Create default admin user
		user := &models.User{
			Username:   "admin",
			Email:      "admin@localhost",
			Password:   models.HashPassword("changeme"),
			Active:     true,
			RoleID:     adminRole.ID,
			AuthSource: models.AuthSourceLocal,
			FirstName:  "System",
			LastName:   "Administrator",
		}

		if err := db.Create(user).Error; err != nil {
			log.Error().Err(err).Msg("Failed to create default admin user")
		} else {
			log.Info().Msg("Created default admin user (username: admin, password: changeme)")
			log.Warn().Msg("SECURITY WARNING: Please change the default admin password immediately!")
		}
	}
}
