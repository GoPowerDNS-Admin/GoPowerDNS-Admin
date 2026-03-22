// Package zonetag provides the admin handler for assigning tags to DNS zones.
package zonetag

import (
	"context"
	"encoding/json"
	"html/template"
	"slices"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"

	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/auth"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/config"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/db/models"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/powerdns"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web/handler"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web/navigation"
)

const (
	// PathList is the path for the zone-tag list.
	PathList = handler.RootPath + "admin/zone-tag"
	// PathEdit is the path for editing a zone's tags.
	PathEdit = handler.RootPath + "admin/zone-tag/:zoneName/edit"

	templateList = "admin/zonetag/list"
	templateForm = "admin/zonetag/form"

	defaultTimeout = 30 * time.Second
)

// Service is the zone-tag handler service.
type Service struct {
	handler.Service
	cfg         *config.Config
	db          *gorm.DB
	authService *auth.Service
}

// Handler is the zone-tag handler.
var Handler = Service{}

// Init initializes the zone-tag handler.
func (s *Service) Init(app *fiber.App, cfg *config.Config, db *gorm.DB, authService *auth.Service) {
	s.cfg = cfg
	s.db = db
	s.authService = authService

	app.Get(PathList, auth.RequirePermission(authService, auth.PermAdminZoneTags), s.List)
	app.Get(PathEdit, auth.RequirePermission(authService, auth.PermAdminZoneTags), s.Edit)
	app.Post(PathEdit, auth.RequirePermission(authService, auth.PermAdminZoneTags), s.Update)
}

// List renders the zone-tag list showing all zones and their tag counts.
func (s *Service) List(c fiber.Ctx) error {
	nav := navigation.NewContext("Zone Tags", "admin", "zone-tags").
		AddBreadcrumb("Home", "/", false).
		AddBreadcrumb("Admin", "/admin", false).
		AddBreadcrumb("Zone Tags", PathList, true)

	if powerdns.Engine.Client == nil {
		return c.Status(fiber.StatusInternalServerError).SendString(powerdns.ErrMsgClientNotInitializedDetailed)
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	apiZones, err := powerdns.Engine.Zones.List(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to fetch zones for zone-tag list")
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to fetch zones: " + err.Error())
	}

	// Load all zone-tag associations
	var zoneTags []models.ZoneTag
	s.db.Preload("Tag").Find(&zoneTags)

	// Build a map of zoneName → tag count
	tagCountByZone := make(map[string]int)
	for i := range zoneTags {
		tagCountByZone[zoneTags[i].ZoneID]++
	}

	type ZoneRow struct {
		Name     string
		TagCount int
	}

	rows := make([]ZoneRow, 0, len(apiZones))
	for i := range apiZones {
		if apiZones[i].Name == nil {
			continue
		}

		name := *apiZones[i].Name
		rows = append(rows, ZoneRow{
			Name:     name,
			TagCount: tagCountByZone[name],
		})
	}

	slices.SortFunc(rows, func(a, b ZoneRow) int {
		if a.Name < b.Name {
			return -1
		}

		if a.Name > b.Name {
			return 1
		}

		return 0
	})

	// Load all tags for display
	var allTags []models.Tag
	s.db.Order("name asc").Find(&allTags)

	zonesJSON, err := json.Marshal(rows)
	if err != nil {
		log.Error().Err(err).Msg("failed to marshal zone-tag rows")

		zonesJSON = []byte("[]")
	}

	return c.Render(templateList, fiber.Map{
		"Navigation": nav,
		"Zones":      rows,
		"ZonesJSON":  template.JS(zonesJSON), //nolint:gosec // safe: json.Marshal escapes HTML chars
		"AllTags":    allTags,
		"ZoneTags":   zoneTags,
	}, handler.BaseLayout)
}

// Edit renders the tag assignment form for a zone.
func (s *Service) Edit(c fiber.Ctx) error {
	zoneName := c.Params("zoneName")
	if zoneName == "" {
		return c.Status(fiber.StatusBadRequest).SendString("Zone name required")
	}

	nav := navigation.NewContext("Zone Tags", "admin", "zone-tags").
		AddBreadcrumb("Home", "/", false).
		AddBreadcrumb("Admin", "/admin", false).
		AddBreadcrumb("Zone Tags", PathList, false).
		AddBreadcrumb(zoneName, "", true)

	var allTags []models.Tag
	s.db.Order("name asc").Find(&allTags)

	var assigned []models.ZoneTag
	s.db.Where("zone_id = ?", zoneName).Find(&assigned)

	assignedSet := make(map[uint]bool)
	for i := range assigned {
		assignedSet[assigned[i].TagID] = true
	}

	return c.Render(templateForm, fiber.Map{
		"Navigation":  nav,
		"ZoneName":    zoneName,
		"AllTags":     allTags,
		"AssignedSet": assignedSet,
	}, handler.BaseLayout)
}

// Update handles the tag assignment form submission for a zone.
func (s *Service) Update(c fiber.Ctx) error {
	zoneName := c.Params("zoneName")
	if zoneName == "" {
		return c.Status(fiber.StatusBadRequest).SendString("Zone name required")
	}

	// Parse selected tag IDs
	selectedIDs := parseTagIDs(c)

	err := s.db.Transaction(func(tx *gorm.DB) error {
		// Delete existing zone-tag associations
		if err := tx.Where("zone_id = ?", zoneName).Delete(&models.ZoneTag{}).Error; err != nil {
			return err
		}

		// Insert new associations
		for _, tagID := range selectedIDs {
			zt := models.ZoneTag{
				ZoneID: zoneName,
				TagID:  tagID,
			}
			if err := tx.Create(&zt).Error; err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		log.Error().Err(err).Str("zone", zoneName).Msg("failed to update zone tags")
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to update zone tags")
	}

	return c.Redirect().To(PathList)
}

// parseTagIDs parses the multi-value tag_ids form field.
func parseTagIDs(c fiber.Ctx) []uint {
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
