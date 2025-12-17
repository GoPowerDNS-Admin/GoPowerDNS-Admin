// Package user provides handlers for managing users (CRUD) in admin area.
package user

import (
	"errors"
	"strconv"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"

	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/auth"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/config"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/db/models"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web/handler"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web/handler/dashboard"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web/navigation"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web/session"
)

const (
	// Path is the base path for user management.
	Path = handler.RootPath + "admin/user"

	// TemplateList is the template for listing users.
	TemplateList = "admin/user/list"
	// TemplateForm is the template for creating/updating a user.
	TemplateForm = "admin/user/form"

	// DefaultPageSize for pagination.
	DefaultPageSize = 25
)

// Service provides CRUD operations for users.
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

	// Routes
	app.Get(Path,
		auth.RequirePermission(authService, auth.PermAdminUsers),
		s.List,
	)
	app.Get(Path+"/new",
		auth.RequirePermission(authService, auth.PermAdminUsers),
		s.New,
	)
	app.Post(Path,
		auth.RequirePermission(authService, auth.PermAdminUsers),
		s.Create,
	)
	app.Get(Path+"/:id/edit",
		auth.RequirePermission(authService, auth.PermAdminUsers),
		s.Edit,
	)
	app.Post(Path+"/:id",
		auth.RequirePermission(authService, auth.PermAdminUsers),
		s.Update,
	)
	app.Post(Path+"/:id/delete",
		auth.RequirePermission(authService, auth.PermAdminUsers),
		s.Delete,
	)
}

// listViewData and formViewData were initially planned as typed data holders but this project uses
// fiber.Map with handler.BaseLayout for rendering, mirroring existing handlers (e.g., Groups).

// List shows users with simple pagination and search.
func (s *Service) List(c *fiber.Ctx) error {
	nav := navigation.NewContext("Users", "admin", "user").
		AddBreadcrumb("Home", dashboard.Path, false).
		AddBreadcrumb("Admin", "#", false).
		AddBreadcrumb("Users", Path, true)

	page := c.QueryInt("page", 1)
	if page < 1 {
		page = 1
	}

	pageSize := c.QueryInt("pageSize", DefaultPageSize)
	if pageSize < 1 || pageSize > 100 {
		pageSize = DefaultPageSize
	}

	search := c.Query("search", "")

	var (
		users      []models.User
		totalCount int64
		tx         = s.db.Model(&models.User{})
	)

	if search != "" {
		like := "%" + search + "%"
		tx = tx.Where(
			"username ILIKE ? OR email ILIKE ? OR external_id ILIKE ? OR first_name ILIKE ? OR last_name ILIKE ?",
			like,
			like,
			like,
			like,
			like,
		)
	}

	if err := tx.Count(&totalCount).Error; err != nil {
		log.Error().Err(err).Msg("count users failed")

		return c.Status(fiber.StatusInternalServerError).Render(TemplateList, fiber.Map{
			"Navigation": nav,
			"Error":      "Failed to load users",
			"Search":     search,
		}, handler.BaseLayout)
	}

	totalPages := int((totalCount + int64(pageSize) - 1) / int64(pageSize))
	if totalPages == 0 {
		totalPages = 1
	}

	if page > totalPages {
		page = totalPages
	}

	offset := (page - 1) * pageSize
	if err := tx.Preload("Role").Order("id DESC").Limit(pageSize).Offset(offset).Find(&users).Error; err != nil {
		log.Error().Err(err).Msg("query users failed")

		return c.Status(fiber.StatusInternalServerError).Render(TemplateList, fiber.Map{
			"Navigation": nav,
			"Error":      "Failed to load users",
			"Search":     search,
		}, handler.BaseLayout)
	}

	// Get current user ID from session
	var currentUserID uint64

	if sessionID := c.Cookies("session"); sessionID != "" {
		sessionData := new(session.Data)
		if err := sessionData.Read(sessionID); err == nil {
			currentUserID = sessionData.User.ID
		}
	}

	return c.Render(TemplateList, fiber.Map{
		"Navigation":    nav,
		"Users":         users,
		"CurrentUserID": currentUserID,
		"Search":        search,
		"Page":          page,
		"PageSize":      pageSize,
		"TotalItems":    totalCount,
		"TotalPages":    totalPages,
		"HasPrev":       page > 1,
		"HasNext":       page < totalPages,
		"PrevPage":      page - 1,
		"NextPage":      page + 1,
	}, handler.BaseLayout)
}

// New shows the creation form.
func (s *Service) New(c *fiber.Ctx) error {
	nav := navigation.NewContext("New User", "admin", "user").
		AddBreadcrumb("Home", dashboard.Path, false).
		AddBreadcrumb("Admin", "#", false).
		AddBreadcrumb("Users", Path, false).
		AddBreadcrumb("New", Path+"/new", true)

	var roles []models.Role
	if err := s.db.Order("name ASC").Find(&roles).Error; err != nil {
		log.Error().Err(err).Msg("failed to load roles")

		return c.Status(fiber.StatusInternalServerError).Render(TemplateForm, fiber.Map{
			"Navigation": nav,
			"Error":      "Failed to load roles",
		}, handler.BaseLayout)
	}

	return c.Render(TemplateForm, fiber.Map{
		"Navigation": nav,
		"User":       models.User{AuthSource: models.AuthSourceLocal, Active: true},
		"IsCreate":   true,
		"Roles":      roles,
	}, handler.BaseLayout)
}

// Create creates a new user.
func (s *Service) Create(c *fiber.Ctx) error {
	var in struct {
		Username   string `form:"username"    validate:"required,min=3,max=100"`
		Email      string `form:"email"       validate:"required,email,max=255"`
		FirstName  string `form:"firstname"   validate:"max=100"`
		LastName   string `form:"lastname"    validate:"max=100"`
		AuthSource string `form:"source"      validate:"required,oneof=local oidc ldap"`
		ExternalID string `form:"external_id"`
		Password   string `form:"password"`
		Active     bool   `form:"active"`
		RoleID     uint   `form:"role_id"`
	}

	if err := c.BodyParser(&in); err != nil {
		nav := navigation.NewContext("Users", "admin", "user").
			AddBreadcrumb("Home", dashboard.Path, false).
			AddBreadcrumb("Admin", "#", false).
			AddBreadcrumb("Users", Path, true)

		return c.Status(fiber.StatusBadRequest).Render(TemplateList, fiber.Map{
			"Navigation": nav,
			"Error":      "Invalid form data",
		}, handler.BaseLayout)
	}

	if in.AuthSource != "local" {
		in.Password = "" // ignore for non-local
	}

	if err := s.validator.Struct(in); err != nil {
		nav := navigation.NewContext("Users", "admin", "user").
			AddBreadcrumb("Home", dashboard.Path, false).
			AddBreadcrumb("Admin", "#", false).
			AddBreadcrumb("Users", Path, true)

		return c.Status(fiber.StatusBadRequest).Render(TemplateList, fiber.Map{
			"Navigation": nav,
			"Error":      "Please correct the highlighted errors",
		}, handler.BaseLayout)
	}

	user := models.User{
		Username:   in.Username,
		Email:      in.Email,
		FirstName:  in.FirstName,
		LastName:   in.LastName,
		AuthSource: models.AuthSource(in.AuthSource),
		ExternalID: in.ExternalID,
		Active:     in.Active,
		RoleID:     in.RoleID,
	}
	if user.RoleID == 0 {
		var userRole models.Role
		if err := s.db.Where(models.WhereNameIs, "user").First(&userRole).Error; err == nil && userRole.ID != 0 {
			user.RoleID = userRole.ID
		}
	}

	if in.AuthSource == string(models.AuthSourceLocal) && in.Password != "" {
		user.Password = models.HashPassword(in.Password)
	}

	if err := s.db.Create(&user).Error; err != nil {
		// Unique constraint errors etc.
		nav := navigation.NewContext("Users", "admin", "user").
			AddBreadcrumb("Home", dashboard.Path, false).
			AddBreadcrumb("Admin", "#", false).
			AddBreadcrumb("Users", Path, true)

		return c.Status(fiber.StatusBadRequest).Render(TemplateList, fiber.Map{
			"Navigation": nav,
			"Error":      "Failed to create user: " + err.Error(),
		}, handler.BaseLayout)
	}

	return c.Redirect(Path)
}

// Edit shows the edit form for a user.
func (s *Service) Edit(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id <= 0 {
		return c.Redirect(Path)
	}

	var user models.User
	if err := s.db.First(&user, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.Redirect(Path)
		}

		return c.Status(fiber.StatusInternalServerError).Render(TemplateForm, fiber.Map{
			"Error": "Failed to load user",
		}, handler.BaseLayout)
	}

	var roles []models.Role
	if err := s.db.Order("name ASC").Find(&roles).Error; err != nil {
		log.Error().Err(err).Msg("failed to load roles")

		return c.Status(fiber.StatusInternalServerError).Render(TemplateForm, fiber.Map{
			"Error": "Failed to load roles",
		}, handler.BaseLayout)
	}

	nav := navigation.NewContext("Edit User", "admin", "user").
		AddBreadcrumb("Home", dashboard.Path, false).
		AddBreadcrumb("Admin", "#", false).
		AddBreadcrumb("Users", Path, false).
		AddBreadcrumb("Edit", Path+"/"+strconv.Itoa(id)+"/edit", true)

	return c.Render(TemplateForm, fiber.Map{
		"Navigation": nav,
		"User":       user,
		"IsCreate":   false,
		"Roles":      roles,
	}, handler.BaseLayout)
}

// Update updates a user.
func (s *Service) Update(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id <= 0 {
		return c.Redirect(Path)
	}

	var in struct {
		Username   string `form:"username"    validate:"required,min=3,max=100"`
		Email      string `form:"email"       validate:"required,email,max=255"`
		FirstName  string `form:"firstname"   validate:"max=100"`
		LastName   string `form:"lastname"    validate:"max=100"`
		AuthSource string `form:"source"      validate:"required,oneof=local oidc ldap"`
		ExternalID string `form:"external_id"`
		Password   string `form:"password"`
		Active     bool   `form:"active"`
		RoleID     uint   `form:"role_id"`
	}
	if err := c.BodyParser(&in); err != nil {
		return c.Status(fiber.StatusBadRequest).Render(TemplateForm, fiber.Map{
			"Error": "Invalid form data",
		}, handler.BaseLayout)
	}

	if in.AuthSource != "local" {
		in.Password = ""
	}

	if err := s.validator.Struct(in); err != nil {
		return c.Status(fiber.StatusBadRequest).Render(TemplateForm, fiber.Map{
			"Error": "Please correct the highlighted errors",
		}, handler.BaseLayout)
	}

	var user models.User
	if err := s.db.First(&user, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.Redirect(Path)
		}

		return c.Status(fiber.StatusInternalServerError).Render(TemplateForm, fiber.Map{
			"Error": "Failed to load user",
		}, handler.BaseLayout)
	}

	user.Username = in.Username
	user.Email = in.Email
	user.FirstName = in.FirstName
	user.LastName = in.LastName
	user.AuthSource = models.AuthSource(in.AuthSource)
	user.ExternalID = in.ExternalID
	user.Active = in.Active

	user.RoleID = in.RoleID
	if in.AuthSource == string(models.AuthSourceLocal) && in.Password != "" {
		user.Password = models.HashPassword(in.Password)
	}

	if err := s.db.Save(&user).Error; err != nil {
		return c.Status(fiber.StatusBadRequest).Render(TemplateForm, fiber.Map{
			"Error": "Failed to update user: " + err.Error(),
		}, handler.BaseLayout)
	}

	return c.Redirect(Path)
}

// Delete removes a user.
func (s *Service) Delete(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id <= 0 {
		return c.Redirect(Path)
	}

	// Load the user to check if they can be deleted
	var user models.User
	if err := s.db.Preload("Role").First(&user, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.Redirect(Path)
		}

		nav := navigation.NewContext("Users", "admin", "user").
			AddBreadcrumb("Home", dashboard.Path, false).
			AddBreadcrumb("Admin", "#", false).
			AddBreadcrumb("Users", Path, true)

		return c.Status(fiber.StatusInternalServerError).Render(TemplateList, fiber.Map{
			"Navigation": nav,
			"Error":      "Failed to load user.",
		}, handler.BaseLayout)
	}

	// Prevent deleting admin users
	if user.Role.Name == "admin" {
		nav := navigation.NewContext("Users", "admin", "user").
			AddBreadcrumb("Home", dashboard.Path, false).
			AddBreadcrumb("Admin", "#", false).
			AddBreadcrumb("Users", Path, true)

		return c.Status(fiber.StatusForbidden).Render(TemplateList, fiber.Map{
			"Navigation": nav,
			"Error":      "Cannot delete admin users.",
		}, handler.BaseLayout)
	}

	// Prevent a user (including admin) from deleting themselves
	// Read current session and compare target id with logged-in user id
	if sessionID := c.Cookies("session"); sessionID != "" {
		current := new(session.Data)
		if errSess := current.Read(sessionID); errSess == nil {
			if current.User.ID == uint64(id) {
				nav := navigation.NewContext("Users", "admin", "user").
					AddBreadcrumb("Home", dashboard.Path, false).
					AddBreadcrumb("Admin", "#", false).
					AddBreadcrumb("Users", Path, true)

				return c.Status(fiber.StatusBadRequest).Render(TemplateList, fiber.Map{
					"Navigation": nav,
					"Error":      "You cannot delete your own account.",
				}, handler.BaseLayout)
			}
		}
	}

	if err := s.db.Delete(&models.User{}, id).Error; err != nil {
		nav := navigation.NewContext("Users", "admin", "user").
			AddBreadcrumb("Home", dashboard.Path, false).
			AddBreadcrumb("Admin", "#", false).
			AddBreadcrumb("Users", Path, true)

		return c.Status(fiber.StatusBadRequest).Render(TemplateList, fiber.Map{
			"Navigation": nav,
			"Error":      "Failed to delete user: " + err.Error(),
		}, handler.BaseLayout)
	}

	return c.Redirect(Path)
}

// helper methods not needed; using inline rendering consistent with other handlers
