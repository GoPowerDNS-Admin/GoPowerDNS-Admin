// Package profile provides the handler for a user's own profile page.
package profile

import (
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
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web/session"
)

const (
	// Path is the route for the profile page.
	Path = handler.RootPath + "profile"

	// Template is the template name for the profile page.
	Template = "profile/profile"
)

// Service handles profile view and password change.
type Service struct {
	handler.Service
	db        *gorm.DB
	validator *validator.Validate
}

// Handler is the exported instance.
var Handler = Service{}

// Init registers routes. No permission required — any authenticated user may access their profile.
func (s *Service) Init(app *fiber.App, cfg *config.Config, db *gorm.DB, _ *auth.Service) {
	if app == nil || cfg == nil || db == nil {
		log.Fatal().Msg(handler.ErrNilACDFatalLogMsg)
		return
	}

	s.db = db
	s.validator = validator.New()

	app.Get(Path, s.View)
	app.Post(Path+"/password", s.ChangePassword)
}

// View renders the profile page for the currently logged-in user.
func (s *Service) View(c fiber.Ctx) error {
	user, ok := s.currentUser(c)
	if !ok {
		return c.Redirect().To("/login")
	}

	return c.Render(Template, fiber.Map{
		"Navigation": profileNav(),
		"User":       user,
		"Groups":     s.loadGroupMemberships(user.ID),
	}, handler.BaseLayout)
}

// ChangePassword handles a password change request. Only available for local users.
func (s *Service) ChangePassword(c fiber.Ctx) error {
	user, ok := s.currentUser(c)
	if !ok {
		return c.Redirect().To("/login")
	}

	// Silently redirect non-local users — the form is never shown to them.
	if user.AuthSource != models.AuthSourceLocal {
		return c.Redirect().To(Path)
	}

	groups := s.loadGroupMemberships(user.ID)

	renderErr := func(msg string) error {
		return c.Status(fiber.StatusBadRequest).Render(Template, fiber.Map{
			"Navigation": profileNav(),
			"User":       user,
			"Groups":     groups,
			"Error":      msg,
		}, handler.BaseLayout)
	}

	var in struct {
		CurrentPassword string `form:"current_password" validate:"required"`
		NewPassword     string `form:"new_password"     validate:"required,min=8"`
		ConfirmPassword string `form:"confirm_password" validate:"required"`
	}

	if err := c.Bind().Body(&in); err != nil {
		return renderErr("Invalid form data")
	}

	if err := s.validator.Struct(in); err != nil {
		return renderErr("New password must be at least 8 characters")
	}

	if in.NewPassword != in.ConfirmPassword {
		return renderErr("New passwords do not match")
	}

	if !user.VerifyPassword(in.CurrentPassword) {
		return renderErr("Current password is incorrect")
	}

	user.Password = models.HashPassword(in.NewPassword)
	if err := s.db.Save(&user).Error; err != nil {
		log.Error().Err(err).Msg("failed to update password")
		return renderErr("Failed to update password")
	}

	return c.Render(Template, fiber.Map{
		"Navigation": profileNav(),
		"User":       user,
		"Groups":     groups,
		"Success":    "Password updated successfully",
	}, handler.BaseLayout)
}

// GroupMembership pairs a group with its mapped role (nil when no mapping exists).
type GroupMembership struct {
	Group      models.Group
	MappedRole *models.Role
}

// loadGroupMemberships returns all groups the user belongs to, each with its role mapping.
func (s *Service) loadGroupMemberships(userID uint64) []GroupMembership {
	var userGroups []models.UserGroup
	s.db.Preload("Group").Where("user_id = ?", userID).Find(&userGroups)

	memberships := make([]GroupMembership, 0, len(userGroups))
	for i := range userGroups {
		m := GroupMembership{Group: userGroups[i].Group}

		var mapping models.GroupMapping
		if err := s.db.Preload("Role").Where("group_id = ?", userGroups[i].GroupID).First(&mapping).Error; err == nil {
			m.MappedRole = &mapping.Role
		}

		memberships = append(memberships, m)
	}

	return memberships
}

// currentUser loads a fresh copy of the logged-in user from the DB.
func (s *Service) currentUser(c fiber.Ctx) (models.User, bool) {
	sessionID := c.Cookies("session")
	if sessionID == "" {
		return models.User{}, false
	}

	sessData := new(session.Data)
	if err := sessData.Read(sessionID); err != nil || sessData.User.ID == 0 {
		return models.User{}, false
	}

	var user models.User
	if err := s.db.Preload("Role").First(&user, sessData.User.ID).Error; err != nil {
		return models.User{}, false
	}

	return user, true
}

func profileNav() *navigation.Context {
	return navigation.NewContext("Profile", "profile", "profile").
		AddBreadcrumb("Home", dashboard.Path, false).
		AddBreadcrumb("Profile", Path, true)
}
