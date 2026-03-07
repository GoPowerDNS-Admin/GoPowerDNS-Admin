// Package role provides handlers for managing roles (CRUD) in admin area.
package role

import (
	"errors"
	"strconv"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"

	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/auth"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/config"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/db/models"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web/handler"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web/handler/dashboard"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web/navigation"
)

const (
	// Path is the base path for role management.
	Path = handler.RootPath + "admin/role"

	// TemplateList is the template for listing roles.
	TemplateList = "admin/role/list"
	// TemplateForm is the template for creating/updating a role.
	TemplateForm = "admin/role/form"

	// DefaultPageSize for pagination.
	DefaultPageSize = 25
)

// Service provides CRUD operations for roles.
type Service struct {
	handler.Service
	cfg       *config.Config
	db        *gorm.DB
	validator *validator.Validate
}

// Handler is the exported instance.
var Handler = Service{}

// Init registers routes.
func (s *Service) Init(app *fiber.App, cfg *config.Config, db *gorm.DB, authService *auth.Service) {
	if app == nil || cfg == nil || db == nil {
		log.Fatal().Msg(handler.ErrNilACDFatalLogMsg)
		return
	}

	s.db = db
	s.cfg = cfg
	s.validator = validator.New()

	app.Get(Path,
		auth.RequirePermission(authService, auth.PermAdminRoles),
		s.List,
	)
	app.Get(Path+"/new",
		auth.RequirePermission(authService, auth.PermAdminRoles),
		s.New,
	)
	app.Post(Path,
		auth.RequirePermission(authService, auth.PermAdminRoles),
		s.Create,
	)
	app.Get(Path+"/:id/edit",
		auth.RequirePermission(authService, auth.PermAdminRoles),
		s.Edit,
	)
	app.Post(Path+"/:id",
		auth.RequirePermission(authService, auth.PermAdminRoles),
		s.Update,
	)
	app.Post(Path+"/:id/delete",
		auth.RequirePermission(authService, auth.PermAdminRoles),
		s.Delete,
	)
}

// List shows all roles with their permission counts and user counts.
func (s *Service) List(c fiber.Ctx) error {
	nav := navigation.NewContext("Roles", "admin", "role").
		AddBreadcrumb("Home", dashboard.Path, false).
		AddBreadcrumb("Admin", "#", false).
		AddBreadcrumb("Roles", Path, true)

	var roles []models.Role
	if err := s.db.Order(handler.OrderNameASC).Find(&roles).Error; err != nil {
		log.Error().Err(err).Msg("query roles failed")

		return c.Status(fiber.StatusInternalServerError).Render(TemplateList, fiber.Map{
			"Navigation": nav,
			"Error":      "Failed to load roles",
		}, handler.BaseLayout)
	}

	// Load permission counts and user counts per role.
	permCounts := make(map[uint]int64)
	userCounts := make(map[uint]int64)

	for _, r := range roles {
		var pc int64
		if err := s.db.Model(&models.RolePermission{}).Where("role_id = ?", r.ID).Count(&pc).Error; err == nil {
			permCounts[r.ID] = pc
		}

		var uc int64
		if err := s.db.Model(&models.User{}).Where("role_id = ?", r.ID).Count(&uc).Error; err == nil {
			userCounts[r.ID] = uc
		}
	}

	return c.Render(TemplateList, fiber.Map{
		"Navigation": nav,
		"Roles":      roles,
		"PermCounts": permCounts,
		"UserCounts": userCounts,
	}, handler.BaseLayout)
}

// New shows the creation form.
func (s *Service) New(c fiber.Ctx) error {
	nav := navigation.NewContext("New Role", "admin", "role").
		AddBreadcrumb("Home", dashboard.Path, false).
		AddBreadcrumb("Admin", "#", false).
		AddBreadcrumb("Roles", Path, false).
		AddBreadcrumb("New", Path+"/new", true)

	permissions, err := s.loadPermissions()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).Render(TemplateForm, fiber.Map{
			"Navigation": nav,
			"Error":      "Failed to load permissions",
		}, handler.BaseLayout)
	}

	return c.Render(TemplateForm, fiber.Map{
		"Navigation":      nav,
		"Role":            models.Role{},
		"IsCreate":        true,
		"Permissions":     permissions,
		"SelectedPermIDs": map[uint]bool{},
	}, handler.BaseLayout)
}

// Create creates a new role.
func (s *Service) Create(c fiber.Ctx) error {
	nav := navigation.NewContext("New Role", "admin", "role").
		AddBreadcrumb("Home", dashboard.Path, false).
		AddBreadcrumb("Admin", "#", false).
		AddBreadcrumb("Roles", Path, false).
		AddBreadcrumb("New", Path+"/new", true)

	var in struct {
		Name        string `form:"name"        validate:"required,min=1,max=100"`
		Description string `form:"description" validate:"max=255"`
	}

	if err := c.Bind().Body(&in); err != nil {
		return c.Status(fiber.StatusBadRequest).Render(TemplateForm, fiber.Map{
			"Navigation": nav,
			"Error":      "Invalid form data",
			"IsCreate":   true,
		}, handler.BaseLayout)
	}

	if err := s.validator.Struct(in); err != nil {
		permissions, _ := s.loadPermissions() //nolint:errcheck // best-effort; permissions may be empty on DB error

		return c.Status(fiber.StatusBadRequest).Render(TemplateForm, fiber.Map{
			"Navigation":      nav,
			"Error":           "Validation failed: " + err.Error(),
			"Role":            models.Role{Name: in.Name, Description: in.Description},
			"IsCreate":        true,
			"Permissions":     permissions,
			"SelectedPermIDs": s.parseSelectedPermIDs(c),
		}, handler.BaseLayout)
	}

	role := models.Role{
		Name:        in.Name,
		Description: in.Description,
	}

	tx := s.db.Begin()

	if err := tx.Create(&role).Error; err != nil {
		tx.Rollback()
		log.Error().Err(err).Msg("failed to create role")

		permissions, _ := s.loadPermissions() //nolint:errcheck // best-effort; permissions may be empty on DB error

		return c.Status(fiber.StatusInternalServerError).Render(TemplateForm, fiber.Map{
			"Navigation":      nav,
			"Error":           "Failed to create role (name may already be taken)",
			"Role":            role,
			"IsCreate":        true,
			"Permissions":     permissions,
			"SelectedPermIDs": s.parseSelectedPermIDs(c),
		}, handler.BaseLayout)
	}

	if err := s.syncPermissions(tx, role.ID, s.parseSelectedPermIDs(c)); err != nil {
		tx.Rollback()
		log.Error().Err(err).Msg("failed to assign permissions")

		return c.Status(fiber.StatusInternalServerError).Render(TemplateForm, fiber.Map{
			"Navigation": nav,
			"Error":      "Failed to assign permissions",
			"IsCreate":   true,
		}, handler.BaseLayout)
	}

	if err := tx.Commit().Error; err != nil {
		log.Error().Err(err).Msg("failed to commit role creation")
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to save role")
	}

	return c.Redirect().To(Path)
}

// Edit shows the edit form for a role.
func (s *Service) Edit(c fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id <= 0 {
		return c.Redirect().To(Path)
	}

	var role models.Role
	if errLoad := s.db.First(&role, id).Error; errLoad != nil {
		if errors.Is(errLoad, gorm.ErrRecordNotFound) {
			return c.Redirect().To(Path)
		}

		return c.Status(fiber.StatusInternalServerError).Render(TemplateForm, fiber.Map{
			"Error": "Failed to load role",
		}, handler.BaseLayout)
	}

	nav := navigation.NewContext("Edit Role", "admin", "role").
		AddBreadcrumb("Home", dashboard.Path, false).
		AddBreadcrumb("Admin", "#", false).
		AddBreadcrumb("Roles", Path, false).
		AddBreadcrumb("Edit", Path+"/"+strconv.Itoa(id)+"/edit", true)

	permissions, err := s.loadPermissions()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).Render(TemplateForm, fiber.Map{
			"Navigation": nav,
			"Error":      "Failed to load permissions",
		}, handler.BaseLayout)
	}

	// Load currently assigned permissions for this role.
	var assigned []models.RolePermission
	s.db.Where("role_id = ?", role.ID).Find(&assigned)

	selectedPermIDs := make(map[uint]bool, len(assigned))
	for i := range assigned {
		selectedPermIDs[assigned[i].PermissionID] = true
	}

	return c.Render(TemplateForm, fiber.Map{
		"Navigation":      nav,
		"Role":            role,
		"IsCreate":        false,
		"Permissions":     permissions,
		"SelectedPermIDs": selectedPermIDs,
	}, handler.BaseLayout)
}

// Update updates a role.
func (s *Service) Update(c fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id <= 0 {
		return c.Redirect().To(Path)
	}

	var role models.Role
	if errLoad := s.db.First(&role, id).Error; errLoad != nil {
		if errors.Is(errLoad, gorm.ErrRecordNotFound) {
			return c.Redirect().To(Path)
		}

		return c.Status(fiber.StatusInternalServerError).Render(TemplateForm, fiber.Map{
			"Error": "Failed to load role",
		}, handler.BaseLayout)
	}

	nav := navigation.NewContext("Edit Role", "admin", "role").
		AddBreadcrumb("Home", dashboard.Path, false).
		AddBreadcrumb("Admin", "#", false).
		AddBreadcrumb("Roles", Path, false).
		AddBreadcrumb("Edit", Path+"/"+strconv.Itoa(id)+"/edit", true)

	var in struct {
		Name        string `form:"name"        validate:"required,min=1,max=100"`
		Description string `form:"description" validate:"max=255"`
	}

	if err := c.Bind().Body(&in); err != nil {
		return c.Status(fiber.StatusBadRequest).Render(TemplateForm, fiber.Map{
			"Navigation": nav,
			"Error":      "Invalid form data",
			"Role":       role,
			"IsCreate":   false,
		}, handler.BaseLayout)
	}

	if err := s.validator.Struct(in); err != nil {
		permissions, _ := s.loadPermissions() //nolint:errcheck // best-effort; permissions may be empty on DB error

		return c.Status(fiber.StatusBadRequest).Render(TemplateForm, fiber.Map{
			"Navigation":      nav,
			"Error":           "Validation failed: " + err.Error(),
			"Role":            role,
			"IsCreate":        false,
			"Permissions":     permissions,
			"SelectedPermIDs": s.parseSelectedPermIDs(c),
		}, handler.BaseLayout)
	}

	// System roles cannot be renamed.
	if !role.IsSystem {
		role.Name = in.Name
	}

	role.Description = in.Description

	tx := s.db.Begin()

	selectedPerms := s.parseSelectedPermIDs(c)

	// Protection: prevent stripping all permissions from the admin role.
	if role.IsSystem && role.Name == "admin" && len(selectedPerms) == 0 {
		tx.Rollback()

		permissions, _ := s.loadPermissions() //nolint:errcheck // best-effort; permissions may be empty on DB error

		return c.Status(fiber.StatusBadRequest).Render(TemplateForm, fiber.Map{
			"Navigation":      nav,
			"Error":           "Cannot remove all permissions from the admin role",
			"Role":            role,
			"IsCreate":        false,
			"Permissions":     permissions,
			"SelectedPermIDs": selectedPerms,
		}, handler.BaseLayout)
	}

	if err := tx.Save(&role).Error; err != nil {
		tx.Rollback()
		log.Error().Err(err).Msg("failed to update role")

		permissions, _ := s.loadPermissions() //nolint:errcheck // best-effort; permissions may be empty on DB error

		return c.Status(fiber.StatusInternalServerError).Render(TemplateForm, fiber.Map{
			"Navigation":      nav,
			"Error":           "Failed to update role",
			"Role":            role,
			"IsCreate":        false,
			"Permissions":     permissions,
			"SelectedPermIDs": selectedPerms,
		}, handler.BaseLayout)
	}

	if err := s.syncPermissions(tx, role.ID, selectedPerms); err != nil {
		tx.Rollback()
		log.Error().Err(err).Msg("failed to sync permissions")

		return c.Status(fiber.StatusInternalServerError).Render(TemplateForm, fiber.Map{
			"Navigation": nav,
			"Error":      "Failed to update permissions",
			"Role":       role,
			"IsCreate":   false,
		}, handler.BaseLayout)
	}

	if err := tx.Commit().Error; err != nil {
		log.Error().Err(err).Msg("failed to commit role update")
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to save role")
	}

	return c.Redirect().To(Path)
}

// Delete removes a role.
func (s *Service) Delete(c fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id <= 0 {
		return c.Redirect().To(Path)
	}

	var role models.Role
	if err := s.db.First(&role, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.Redirect().To(Path)
		}

		return c.Status(fiber.StatusInternalServerError).Render(TemplateList, fiber.Map{
			"Error": "Failed to load role",
		}, handler.BaseLayout)
	}

	if role.IsSystem {
		nav := navigation.NewContext("Roles", "admin", "role").
			AddBreadcrumb("Home", dashboard.Path, false).
			AddBreadcrumb("Admin", "#", false).
			AddBreadcrumb("Roles", Path, true)

		return c.Status(fiber.StatusForbidden).Render(TemplateList, fiber.Map{
			"Navigation": nav,
			"Error":      "Cannot delete system roles",
		}, handler.BaseLayout)
	}

	// Prevent deletion if users are still assigned to this role.
	var userCount int64
	if err := s.db.Model(&models.User{}).Where("role_id = ?", id).Count(&userCount).Error; err == nil && userCount > 0 {
		nav := navigation.NewContext("Roles", "admin", "role").
			AddBreadcrumb("Home", dashboard.Path, false).
			AddBreadcrumb("Admin", "#", false).
			AddBreadcrumb("Roles", Path, true)

		return c.Status(fiber.StatusBadRequest).Render(TemplateList, fiber.Map{
			"Navigation": nav,
			"Error":      "Cannot delete role: it is still assigned to users",
		}, handler.BaseLayout)
	}

	if err := s.db.Delete(&models.Role{}, id).Error; err != nil {
		log.Error().Err(err).Msg("failed to delete role")

		nav := navigation.NewContext("Roles", "admin", "role").
			AddBreadcrumb("Home", dashboard.Path, false).
			AddBreadcrumb("Admin", "#", false).
			AddBreadcrumb("Roles", Path, true)

		return c.Status(fiber.StatusInternalServerError).Render(TemplateList, fiber.Map{
			"Navigation": nav,
			"Error":      "Failed to delete role",
		}, handler.BaseLayout)
	}

	return c.Redirect().To(Path)
}

// loadPermissions returns all permissions ordered by resource then action.
func (s *Service) loadPermissions() ([]models.Permission, error) {
	var permissions []models.Permission
	if err := s.db.Order("resource ASC, action ASC").Find(&permissions).Error; err != nil {
		return nil, err
	}

	return permissions, nil
}

// parseSelectedPermIDs reads the perm_ids multi-value form field.
func (s *Service) parseSelectedPermIDs(c fiber.Ctx) map[uint]bool {
	selected := make(map[uint]bool)

	for _, raw := range c.Request().PostArgs().PeekMulti("perm_ids") {
		if id, err := strconv.ParseUint(string(raw), 10, 32); err == nil {
			selected[uint(id)] = true
		}
	}

	return selected
}

// syncPermissions replaces the role's permission assignments within the given transaction.
func (s *Service) syncPermissions(tx *gorm.DB, roleID uint, selected map[uint]bool) error {
	if err := tx.Where("role_id = ?", roleID).Delete(&models.RolePermission{}).Error; err != nil {
		return err
	}

	for permID := range selected {
		if err := tx.Create(&models.RolePermission{
			RoleID:       roleID,
			PermissionID: permID,
		}).Error; err != nil {
			return err
		}
	}

	return nil
}
