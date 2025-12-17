// Package group provides handlers for managing user groups (CRUD) in admin area.
package group

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
)

const (
	// Path is the base path for group management.
	Path = handler.RootPath + "admin/group"

	// TemplateList is the template for listing groups.
	TemplateList = "admin/group/list"
	// TemplateForm is the template for creating/updating a group.
	TemplateForm = "admin/group/form"

	// DefaultPageSize for pagination.
	DefaultPageSize = 25
	// MaxPageSize clamps the page size upper bound.
	MaxPageSize = 100

	// NavSectionAdmin is the top-level navigation section name for admin screens.
	NavSectionAdmin = "admin"
	// NavEntityGroup is the navigation entity key used for groups in the admin area.
	NavEntityGroup = "group"

	// TitleGroups is the page title for the groups list.
	TitleGroups = "Groups"
	// TitleNewGroup is the page title for creating a new group.
	TitleNewGroup = "New Group"
	// TitleEditGroup is the page title for editing an existing group.
	TitleEditGroup = "Edit Group"

	// BreadcrumbHomeLbl is the label for the home breadcrumb.
	BreadcrumbHomeLbl = "Home"
	// BreadcrumbAdminLbl is the label for the admin breadcrumb.
	BreadcrumbAdminLbl = "Admin"
	// BreadcrumbGroupsLbl is the label for the groups list breadcrumb.
	BreadcrumbGroupsLbl = "Groups"
	// BreadcrumbNewLbl is the label for the "new" breadcrumb.
	BreadcrumbNewLbl = "New"
	// BreadcrumbEditLbl is the label for the "edit" breadcrumb.
	BreadcrumbEditLbl = "Edit"

	// HrefHash represents a non-navigating link target (placeholder "#").
	HrefHash = "#"

	// QueryPage is the query parameter name for the current page index.
	QueryPage = "page"
	// QueryPageSize is the query parameter name for the page size.
	QueryPageSize = "pageSize"
	// QuerySearch is the query parameter name for the search term.
	QuerySearch = "search"

	// ErrInvalidID is returned when the provided id parameter is invalid or non-positive.
	ErrInvalidID = "Invalid id"
	// ErrGroupNotFound is returned when a group with the given id does not exist.
	ErrGroupNotFound = "Group not found"
	// ErrFailedLoadGroup indicates an unexpected error occurred while loading a single group.
	ErrFailedLoadGroup = "Failed to load group"
	// ErrFailedLoadGroups indicates an unexpected error occurred while loading multiple groups.
	ErrFailedLoadGroups = "Failed to load groups"
	// ErrFailedCreateGroup indicates the create operation failed, e.g. due to uniqueness constraints.
	ErrFailedCreateGroup = "Failed to create group (possibly duplicate external id with same source)"
	// ErrFailedUpdateGroup indicates the update operation failed, e.g. due to uniqueness constraints.
	ErrFailedUpdateGroup = "Failed to update group (check uniqueness constraints)"
	// ErrFailedDeleteGroup indicates the delete operation failed.
	ErrFailedDeleteGroup = "Failed to delete group"
	// ErrValidationPrefix prefixes validation error messages shown to the user.
	ErrValidationPrefix = "Validation failed: "

	// RouteNew is the route for rendering the new group form.
	RouteNew = Path + "/new"
	// RouteEdit is the route for rendering the edit group form.
	RouteEdit = Path + "/:id/edit"
	// RouteUpdate is the route for submitting an update to an existing group.
	RouteUpdate = Path + "/:id"
	// RouteDelete is the route for deleting a group.
	RouteDelete = Path + "/:id/delete"
)

// Service provides CRUD operations for groups.
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
		auth.RequirePermission(authService, auth.PermAdminGroups),
		s.List,
	)
	app.Get(RouteNew,
		auth.RequirePermission(authService, auth.PermAdminGroups),
		s.New,
	)
	app.Post(Path,
		auth.RequirePermission(authService, auth.PermAdminGroups),
		s.Create,
	)
	app.Get(RouteEdit,
		auth.RequirePermission(authService, auth.PermAdminGroups),
		s.Edit,
	)
	app.Post(RouteUpdate,
		auth.RequirePermission(authService, auth.PermAdminGroups),
		s.Update,
	)
	app.Post(RouteDelete,
		auth.RequirePermission(authService, auth.PermAdminGroups),
		s.Delete,
	)
}

// List shows groups with simple pagination and search.
func (s *Service) List(c *fiber.Ctx) error {
	nav := navigation.NewContext(TitleGroups, NavSectionAdmin, NavEntityGroup).
		AddBreadcrumb(BreadcrumbHomeLbl, dashboard.Path, false).
		AddBreadcrumb(BreadcrumbAdminLbl, HrefHash, false).
		AddBreadcrumb(BreadcrumbGroupsLbl, Path, true)

	page := c.QueryInt(QueryPage, 1)
	if page < 1 {
		page = 1
	}

	pageSize := c.QueryInt(QueryPageSize, DefaultPageSize)
	if pageSize < 1 || pageSize > MaxPageSize {
		pageSize = DefaultPageSize
	}

	search := c.Query(QuerySearch, "")

	var (
		groups     []models.Group
		totalCount int64
		tx         = s.db.Model(&models.Group{})
	)

	if search != "" {
		like := "%" + search + "%"
		tx = tx.Where("name ILIKE ? OR external_id ILIKE ? OR description ILIKE ?", like, like, like)
	}

	if err := tx.Count(&totalCount).Error; err != nil {
		log.Error().Err(err).Msg("count groups failed")

		return c.Status(fiber.StatusInternalServerError).Render(TemplateList, fiber.Map{
			"Navigation": nav,
			"Error":      ErrFailedLoadGroups,
		}, handler.BaseLayout)
	}

	totalPages := int((totalCount + int64(pageSize) - 1) / int64(pageSize))
	if totalPages < 1 {
		totalPages = 1
	}

	if page > totalPages {
		page = totalPages
	}

	offset := (page - 1) * pageSize
	if err := tx.Order("id DESC").Limit(pageSize).Offset(offset).Find(&groups).Error; err != nil {
		log.Error().Err(err).Msg("query groups failed")

		return c.Status(fiber.StatusInternalServerError).Render(TemplateList, fiber.Map{
			"Navigation": nav,
			"Error":      ErrFailedLoadGroups,
		}, handler.BaseLayout)
	}

	// Load member counts and role mappings for each group
	memberCounts := make(map[uint]int64)
	roleMappings := make(map[uint]string) // group_id -> role_name

	for _, g := range groups {
		var count int64
		if err := s.db.Model(&models.UserGroup{}).Where("group_id = ?", g.ID).Count(&count).Error; err == nil {
			memberCounts[g.ID] = count
		}

		// Load role mapping
		var mapping models.GroupMapping
		if err := s.db.Preload("Role").Where("group_id = ?", g.ID).First(&mapping).Error; err == nil {
			roleMappings[g.ID] = mapping.Role.Name
		}
	}

	return c.Render(TemplateList, fiber.Map{
		"Navigation":   nav,
		"Groups":       groups,
		"MemberCounts": memberCounts,
		"RoleMappings": roleMappings,
		"Search":       search,
		"Page":         page,
		"PageSize":     pageSize,
		"TotalItems":   totalCount,
		"TotalPages":   totalPages,
		"HasPrev":      page > 1,
		"HasNext":      page < totalPages,
		"PrevPage":     page - 1,
		"NextPage":     page + 1,
	}, handler.BaseLayout)
}

// New renders empty form.
func (s *Service) New(c *fiber.Ctx) error {
	nav := navigation.NewContext(TitleNewGroup, NavSectionAdmin, NavEntityGroup).
		AddBreadcrumb(BreadcrumbHomeLbl, dashboard.Path, false).
		AddBreadcrumb(BreadcrumbAdminLbl, HrefHash, false).
		AddBreadcrumb(BreadcrumbGroupsLbl, Path, false).
		AddBreadcrumb(BreadcrumbNewLbl, RouteNew, true)

	var users []models.User
	if err := s.db.Order("username ASC").Find(&users).Error; err != nil {
		log.Error().Err(err).Msg("failed to load users")

		return c.Status(fiber.StatusInternalServerError).Render(TemplateForm, fiber.Map{
			"Navigation": nav,
			"Error":      "Failed to load users",
		}, handler.BaseLayout)
	}

	var roles []models.Role
	if err := s.db.Order("name ASC").Find(&roles).Error; err != nil {
		log.Error().Err(err).Msg("failed to load roles")

		return c.Status(fiber.StatusInternalServerError).Render(TemplateForm, fiber.Map{
			"Navigation": nav,
			"Error":      "Failed to load roles",
		}, handler.BaseLayout)
	}

	return c.Render(TemplateForm, fiber.Map{
		"Navigation":  nav,
		"Group":       models.Group{Source: models.GroupSourceLocal},
		"IsCreate":    true,
		"Users":       users,
		"Roles":       roles,
		"SelectedIDs": []uint64{},
	}, handler.BaseLayout)
}

// Create handles form submission for creating a group.
func (s *Service) Create(c *fiber.Ctx) error {
	// Get user IDs from form
	userIDsBytes := c.Request().PostArgs().PeekMulti("user_ids")

	userIDs := make([]string, len(userIDsBytes))
	for i, b := range userIDsBytes {
		userIDs[i] = string(b)
	}

	var input = formInput{
		Name:        c.FormValue("name"),
		ExternalID:  c.FormValue("external_id"),
		Source:      c.FormValue("source", string(models.GroupSourceLocal)),
		Description: c.FormValue("description"),
		RoleID:      0,
		UserIDs:     userIDs,
	}

	// Parse role_id from form
	if roleIDStr := c.FormValue("role_id"); roleIDStr != "" {
		roleIDParsed, err := strconv.ParseUint(roleIDStr, 10, 32)
		if err == nil {
			input.RoleID = uint(roleIDParsed)
		}
	}

	if err := s.validator.Struct(input); err != nil {
		log.Warn().Err(err).Msg("validation failed for create group")

		nav := navigation.NewContext(TitleNewGroup, NavSectionAdmin, NavEntityGroup).
			AddBreadcrumb(BreadcrumbHomeLbl, dashboard.Path, false).
			AddBreadcrumb(BreadcrumbAdminLbl, HrefHash, false).
			AddBreadcrumb(BreadcrumbGroupsLbl, Path, false).
			AddBreadcrumb(BreadcrumbNewLbl, RouteNew, true)

		return c.Status(fiber.StatusBadRequest).Render(TemplateForm, fiber.Map{
			"Navigation": nav,
			"Error":      ErrValidationPrefix + err.Error(),
			"Group": models.Group{
				Name:        input.Name,
				ExternalID:  input.ExternalID,
				Source:      models.GroupSource(input.Source),
				Description: input.Description,
			},
			"IsCreate": true,
		}, handler.BaseLayout)
	}

	g := &models.Group{
		Name:        input.Name,
		ExternalID:  input.ExternalID,
		Source:      models.GroupSource(input.Source),
		Description: input.Description,
	}

	// Begin transaction
	tx := s.db.Begin()
	if err := tx.Create(g).Error; err != nil {
		tx.Rollback()
		log.Error().Err(err).Msg("failed to create group")

		nav := navigation.NewContext(TitleNewGroup, NavSectionAdmin, NavEntityGroup).
			AddBreadcrumb(BreadcrumbHomeLbl, dashboard.Path, false).
			AddBreadcrumb(BreadcrumbAdminLbl, HrefHash, false).
			AddBreadcrumb(BreadcrumbGroupsLbl, Path, false).
			AddBreadcrumb(BreadcrumbNewLbl, RouteNew, true)

		return c.Status(fiber.StatusInternalServerError).Render(TemplateForm, fiber.Map{
			"Navigation": nav,
			"Error":      ErrFailedCreateGroup,
			"Group":      g,
			"IsCreate":   true,
		}, handler.BaseLayout)
	}

	// Create group mapping to role
	if input.RoleID > 0 {
		groupMapping := models.GroupMapping{
			GroupID: g.ID,
			RoleID:  input.RoleID,
		}
		if err := tx.Create(&groupMapping).Error; err != nil {
			tx.Rollback()
			log.Error().Err(err).Msg("failed to create group mapping")

			return c.Status(fiber.StatusInternalServerError).SendString("Failed to assign role to group")
		}
	}

	// Create user group memberships
	for _, userIDStr := range input.UserIDs {
		userID, err := strconv.ParseUint(userIDStr, 10, 64)
		if err != nil {
			continue // skip invalid IDs
		}

		userGroup := models.UserGroup{
			UserID:  userID,
			GroupID: g.ID,
		}
		if err := tx.Create(&userGroup).Error; err != nil {
			tx.Rollback()
			log.Error().Err(err).Msg("failed to add user to group")

			return c.Status(fiber.StatusInternalServerError).SendString("Failed to add users to group")
		}
	}

	if err := tx.Commit().Error; err != nil {
		log.Error().Err(err).Msg("failed to commit transaction")
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to save group")
	}

	return c.Redirect(Path)
}

// Edit renders edit form for a group.
func (s *Service) Edit(c *fiber.Ctx) error {
	idStr := c.Params("id")

	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		return c.Status(fiber.StatusBadRequest).SendString(ErrInvalidID)
	}

	var g models.Group
	if err := s.db.First(&g, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.Status(fiber.StatusNotFound).SendString(ErrGroupNotFound)
		}

		log.Error().Err(err).Msg("load group failed")

		return c.Status(fiber.StatusInternalServerError).SendString(ErrFailedLoadGroup)
	}

	// Load all users
	var users []models.User
	if err := s.db.Order("username ASC").Find(&users).Error; err != nil {
		log.Error().Err(err).Msg("failed to load users")
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to load users")
	}

	// Load all roles
	var roles []models.Role
	if err := s.db.Order("name ASC").Find(&roles).Error; err != nil {
		log.Error().Err(err).Msg("failed to load roles")
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to load roles")
	}

	// Load current group members
	var userGroups []models.UserGroup
	if err := s.db.Where("group_id = ?", g.ID).Find(&userGroups).Error; err != nil {
		log.Error().Err(err).Msg("failed to load group members")
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to load group members")
	}

	// Load group mapping (role assignment)
	var groupMapping models.GroupMapping

	var mappedRoleID uint
	if err := s.db.Where("group_id = ?", g.ID).First(&groupMapping).Error; err == nil {
		mappedRoleID = groupMapping.RoleID
	}

	// Create a slice of selected user IDs
	selectedIDs := make([]uint64, 0, len(userGroups))
	for i := range userGroups {
		selectedIDs = append(selectedIDs, userGroups[i].UserID)
	}

	nav := navigation.NewContext(TitleEditGroup, NavSectionAdmin, NavEntityGroup).
		AddBreadcrumb(BreadcrumbHomeLbl, dashboard.Path, false).
		AddBreadcrumb(BreadcrumbAdminLbl, HrefHash, false).
		AddBreadcrumb(BreadcrumbGroupsLbl, Path, false).
		AddBreadcrumb(BreadcrumbEditLbl, Path+"/"+strconv.Itoa(int(g.ID))+"/edit", true)

	return c.Render(TemplateForm, fiber.Map{
		"Navigation":   nav,
		"Group":        g,
		"IsCreate":     false,
		"Users":        users,
		"Roles":        roles,
		"MappedRoleID": mappedRoleID,
		"SelectedIDs":  selectedIDs,
	}, handler.BaseLayout)
}

// Update handles updating an existing group.
func (s *Service) Update(c *fiber.Ctx) error {
	idStr := c.Params("id")

	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		return c.Status(fiber.StatusBadRequest).SendString(ErrInvalidID)
	}

	var g models.Group
	if err = s.db.First(&g, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.Status(fiber.StatusNotFound).SendString(ErrGroupNotFound)
		}

		log.Error().Err(err).Msg("load group failed")

		return c.Status(fiber.StatusInternalServerError).SendString(ErrFailedLoadGroup)
	}

	// Get user IDs from the form
	userIDsBytes := c.Request().PostArgs().PeekMulti("user_ids")

	userIDs := make([]string, len(userIDsBytes))
	for i, b := range userIDsBytes {
		userIDs[i] = string(b)
	}

	var input = formInput{
		Name:        c.FormValue("name"),
		ExternalID:  c.FormValue("external_id"),
		Source:      c.FormValue("source", string(models.GroupSourceLocal)),
		Description: c.FormValue("description"),
		RoleID:      0,
		UserIDs:     userIDs,
	}

	// Parse role_id from form
	if roleIDStr := c.FormValue("role_id"); roleIDStr != "" {
		roleIDParsed, errParse := strconv.ParseUint(roleIDStr, 10, 32)
		if errParse == nil {
			input.RoleID = uint(roleIDParsed)
		}
	}

	if errValidator := s.validator.Struct(input); errValidator != nil {
		log.Warn().Err(errValidator).Msg("validation failed for update group")

		nav := navigation.NewContext(TitleEditGroup, NavSectionAdmin, NavEntityGroup).
			AddBreadcrumb(BreadcrumbHomeLbl, dashboard.Path, false).
			AddBreadcrumb(BreadcrumbAdminLbl, HrefHash, false).
			AddBreadcrumb(BreadcrumbGroupsLbl, Path, false).
			AddBreadcrumb(BreadcrumbEditLbl, Path+"/"+strconv.Itoa(int(g.ID))+"/edit", true)

		// keep old values mixed with submitted
		g.Name = input.Name
		g.ExternalID = input.ExternalID
		g.Source = models.GroupSource(input.Source)
		g.Description = input.Description

		return c.Status(fiber.StatusBadRequest).Render(TemplateForm, fiber.Map{
			"Navigation": nav,
			"Error":      ErrValidationPrefix + errValidator.Error(),
			"Group":      g,
			"IsCreate":   false,
		}, handler.BaseLayout)
	}

	g.Name = input.Name
	g.ExternalID = input.ExternalID
	g.Source = models.GroupSource(input.Source)
	g.Description = input.Description

	// Begin transaction
	tx := s.db.Begin()

	if errSave := tx.Save(&g).Error; errSave != nil {
		tx.Rollback()
		log.Error().Err(errSave).Msg("failed to update group")

		nav := navigation.NewContext(TitleEditGroup, NavSectionAdmin, NavEntityGroup).
			AddBreadcrumb(BreadcrumbHomeLbl, dashboard.Path, false).
			AddBreadcrumb(BreadcrumbAdminLbl, HrefHash, false).
			AddBreadcrumb(BreadcrumbGroupsLbl, Path, false).
			AddBreadcrumb(BreadcrumbEditLbl, Path+"/"+strconv.Itoa(int(g.ID))+"/edit", true)

		return c.Status(fiber.StatusInternalServerError).Render(TemplateForm, fiber.Map{
			"Navigation": nav,
			"Error":      ErrFailedUpdateGroup,
			"Group":      g,
			"IsCreate":   false,
		}, handler.BaseLayout)
	}

	// Update or create group mapping
	if input.RoleID > 0 {
		if errUoCGM := s.updateOrCreateGroupMapping(c, tx, g.ID, input.RoleID); errUoCGM != nil {
			return errUoCGM
		}
	}

	if errGMS := s.updateOrCreateGroupMembership(c, tx, g.ID, &input); errGMS != nil {
		return errGMS
	}

	return c.Redirect(Path)
}

// Delete removes a group.
func (s *Service) Delete(c *fiber.Ctx) error {
	idStr := c.Params("id")

	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		return c.Status(fiber.StatusBadRequest).SendString(ErrInvalidID)
	}

	if err := s.db.Delete(&models.Group{}, id).Error; err != nil {
		log.Error().Err(err).Msg("failed to delete group")
		return c.Status(fiber.StatusInternalServerError).SendString(ErrFailedDeleteGroup)
	}

	return c.Redirect(Path)
}
