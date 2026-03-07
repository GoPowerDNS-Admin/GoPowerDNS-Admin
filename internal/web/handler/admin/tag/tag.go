// Package tag provides the admin handler for managing zone-access tags.
package tag

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"

	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/auth"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/config"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/db/models"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web/handler"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web/navigation"
)

const (
	// PathList is the path for the tag list.
	PathList = handler.RootPath + "admin/tag"
	// PathNew is the path for creating a new tag.
	PathNew = handler.RootPath + "admin/tag/new"
	// PathEdit is the path for editing a tag.
	PathEdit = handler.RootPath + "admin/tag/:id/edit"
	// PathDelete is the path for deleting a tag.
	PathDelete = handler.RootPath + "admin/tag/:id/delete"

	templateList = "admin/tag/list"
	templateForm = "admin/tag/form"
)

// Service is the tag handler service.
type Service struct {
	handler.Service
	cfg         *config.Config
	db          *gorm.DB
	authService *auth.Service
}

// Handler is the tag handler.
var Handler = Service{}

// Init initializes the tag handler.
func (s *Service) Init(app *fiber.App, cfg *config.Config, db *gorm.DB, authService *auth.Service) {
	s.cfg = cfg
	s.db = db
	s.authService = authService

	app.Get(PathList, auth.RequirePermission(authService, auth.PermAdminTags), s.List)
	app.Get(PathNew, auth.RequirePermission(authService, auth.PermAdminTags), s.New)
	app.Post(PathNew, auth.RequirePermission(authService, auth.PermAdminTags), s.Create)
	app.Get(PathEdit, auth.RequirePermission(authService, auth.PermAdminTags), s.Edit)
	app.Post(PathEdit, auth.RequirePermission(authService, auth.PermAdminTags), s.Update)
	app.Post(PathDelete, auth.RequirePermission(authService, auth.PermAdminTags), s.Delete)
}

// List renders the tag list page.
func (s *Service) List(c fiber.Ctx) error {
	nav := navigation.NewContext("Tags", "admin", "tags").
		AddBreadcrumb("Home", "/", false).
		AddBreadcrumb("Admin", "/admin", false).
		AddBreadcrumb("Tags", PathList, true)

	var tags []models.Tag
	if err := s.db.Order("name asc").Find(&tags).Error; err != nil {
		log.Error().Err(err).Msg("failed to list tags")
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to load tags")
	}

	return c.Render(templateList, fiber.Map{
		"Navigation": nav,
		"Tags":       tags,
	}, handler.BaseLayout)
}

// New renders the create tag form.
func (s *Service) New(c fiber.Ctx) error {
	nav := navigation.NewContext("New Tag", "admin", "tags").
		AddBreadcrumb("Home", "/", false).
		AddBreadcrumb("Admin", "/admin", false).
		AddBreadcrumb("Tags", PathList, false).
		AddBreadcrumb("New Tag", PathNew, true)

	return c.Render(templateForm, fiber.Map{
		"Navigation": nav,
		"IsCreate":   true,
		"Tag":        models.Tag{},
	}, handler.BaseLayout)
}

// Create handles the create tag form submission.
func (s *Service) Create(c fiber.Ctx) error {
	nav := navigation.NewContext("New Tag", "admin", "tags").
		AddBreadcrumb("Home", "/", false).
		AddBreadcrumb("Admin", "/admin", false).
		AddBreadcrumb("Tags", PathList, false).
		AddBreadcrumb("New Tag", PathNew, true)

	var in struct {
		Name        string `form:"name"`
		Description string `form:"description"`
	}

	if err := c.Bind().Body(&in); err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid form data")
	}

	if in.Name == "" {
		return c.Render(templateForm, fiber.Map{
			"Navigation": nav,
			"IsCreate":   true,
			"Tag":        models.Tag{Name: in.Name, Description: in.Description},
			"Error":      "Name is required",
		}, handler.BaseLayout)
	}

	tag := models.Tag{
		Name:        in.Name,
		Description: in.Description,
	}

	if err := s.db.Create(&tag).Error; err != nil {
		log.Error().Err(err).Msg("failed to create tag")

		return c.Render(templateForm, fiber.Map{
			"Navigation": nav,
			"IsCreate":   true,
			"Tag":        tag,
			"Error":      "Failed to create tag: " + err.Error(),
		}, handler.BaseLayout)
	}

	return c.Redirect().To(PathList)
}

// Edit renders the edit tag form.
func (s *Service) Edit(c fiber.Ctx) error {
	id := fiber.Params[uint](c, "id")

	var tag models.Tag
	if err := s.db.First(&tag, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.Status(fiber.StatusNotFound).SendString("Tag not found")
		}

		return c.Status(fiber.StatusInternalServerError).SendString("Failed to load tag")
	}

	nav := navigation.NewContext("Edit Tag", "admin", "tags").
		AddBreadcrumb("Home", "/", false).
		AddBreadcrumb("Admin", "/admin", false).
		AddBreadcrumb("Tags", PathList, false).
		AddBreadcrumb("Edit Tag", "", true)

	return c.Render(templateForm, fiber.Map{
		"Navigation": nav,
		"IsCreate":   false,
		"Tag":        tag,
	}, handler.BaseLayout)
}

// Update handles the edit tag form submission.
func (s *Service) Update(c fiber.Ctx) error {
	id := fiber.Params[uint](c, "id")

	var tag models.Tag
	if err := s.db.First(&tag, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.Status(fiber.StatusNotFound).SendString("Tag not found")
		}

		return c.Status(fiber.StatusInternalServerError).SendString("Failed to load tag")
	}

	nav := navigation.NewContext("Edit Tag", "admin", "tags").
		AddBreadcrumb("Home", "/", false).
		AddBreadcrumb("Admin", "/admin", false).
		AddBreadcrumb("Tags", PathList, false).
		AddBreadcrumb("Edit Tag", "", true)

	var in struct {
		Name        string `form:"name"`
		Description string `form:"description"`
	}

	if err := c.Bind().Body(&in); err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid form data")
	}

	if in.Name == "" {
		return c.Render(templateForm, fiber.Map{
			"Navigation": nav,
			"IsCreate":   false,
			"Tag":        tag,
			"Error":      "Name is required",
		}, handler.BaseLayout)
	}

	tag.Name = in.Name
	tag.Description = in.Description

	if err := s.db.Save(&tag).Error; err != nil {
		log.Error().Err(err).Msg("failed to update tag")

		return c.Render(templateForm, fiber.Map{
			"Navigation": nav,
			"IsCreate":   false,
			"Tag":        tag,
			"Error":      "Failed to update tag: " + err.Error(),
		}, handler.BaseLayout)
	}

	return c.Redirect().To(PathList)
}

// Delete handles tag deletion.
func (s *Service) Delete(c fiber.Ctx) error {
	idStr := c.Params("id")

	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid tag ID")
	}

	var tag models.Tag
	if err = s.db.First(&tag, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.Status(fiber.StatusNotFound).SendString("Tag not found")
		}

		return c.Status(fiber.StatusInternalServerError).SendString("Failed to load tag")
	}

	if err = s.db.Delete(&tag).Error; err != nil {
		log.Error().Err(err).Msg("failed to delete tag")
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to delete tag")
	}

	return c.Redirect().To(PathList)
}
