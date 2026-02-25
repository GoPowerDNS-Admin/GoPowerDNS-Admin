// Package activity provides the admin handler for viewing the audit / activity log.
package activity

import (
	"encoding/json"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"

	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/activitylog"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/auth"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/config"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/db/models"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web/handler"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web/handler/dashboard"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web/navigation"
)

const (
	// Path is the base path for the activity log page.
	Path = handler.RootPath + "admin/activity"

	// TemplateList is the template used to render the activity log list.
	TemplateList = "admin/activity/list"

	// DefaultPageSize is the default number of entries per page.
	DefaultPageSize = 50
)

// EntryView wraps an ActivityLog row with its decoded diff so the template
// can render structured data before/after without parsing JSON itself.
type EntryView struct {
	models.ActivityLog
	// ZoneSettings is populated for zone_updated entries.
	ZoneSettings *activitylog.ZoneSettingsDiff
	// RecordsDiff is populated with record_changed entries.
	RecordsDiff *activitylog.RecordsDiff
}

// activityFilters holds the query parameters for filtering the activity log.
type activityFilters struct {
	User   string
	Action string
	From   string
	To     string
}

// Service provides the read-only activity log view.
type Service struct {
	handler.Service
	cfg *config.Config
	db  *gorm.DB
}

// Handler is the exported singleton instance.
var Handler = Service{}

// Init registers the handler routes.
func (s *Service) Init(app *fiber.App, cfg *config.Config, db *gorm.DB, authService *auth.Service) {
	if app == nil || cfg == nil || db == nil {
		log.Fatal().Msg(handler.ErrNilACDFatalLogMsg)
		return
	}

	s.cfg = cfg
	s.db = db

	app.Get(Path,
		auth.RequirePermission(authService, auth.PermAdminActivityLog),
		s.List,
	)
}

// List renders the paginated activity log with optional filters.
func (s *Service) List(c *fiber.Ctx) error {
	nav := navigation.NewContext("Activity Log", "admin", "activity").
		AddBreadcrumb("Home", dashboard.Path, false).
		AddBreadcrumb("Admin", "#", false).
		AddBreadcrumb("Activity Log", Path, true)

	page := c.QueryInt("page", 1)
	if page < 1 {
		page = 1
	}

	pageSize := c.QueryInt("pageSize", DefaultPageSize)
	if pageSize < 1 || pageSize > 200 {
		pageSize = DefaultPageSize
	}

	filters := parseActivityFilters(c)
	tx := buildActivityQuery(s.db, filters)

	var totalCount int64
	if err := tx.Count(&totalCount).Error; err != nil {
		log.Error().Err(err).Msg("failed to count activity log entries")

		return c.Status(fiber.StatusInternalServerError).Render(TemplateList, fiber.Map{
			"Navigation": nav,
			"Error":      "Failed to load activity log",
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

	var entries []models.ActivityLog
	if err := tx.Order("created_at DESC").Limit(pageSize).Offset(offset).Find(&entries).Error; err != nil {
		log.Error().Err(err).Msg("failed to query activity log entries")

		return c.Status(fiber.StatusInternalServerError).Render(TemplateList, fiber.Map{
			"Navigation": nav,
			"Error":      "Failed to load activity log",
		}, handler.BaseLayout)
	}

	views := getActivityViews(entries)
	actions := getDistinctActions(s.db)

	return c.Render(TemplateList, fiber.Map{
		"Navigation":   nav,
		"Entries":      views,
		"Actions":      actions,
		"FilterUser":   filters.User,
		"FilterAction": filters.Action,
		"FilterFrom":   filters.From,
		"FilterTo":     filters.To,
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

func parseActivityFilters(c *fiber.Ctx) activityFilters {
	return activityFilters{
		User:   c.Query("user", ""),
		Action: c.Query("action", ""),
		From:   c.Query("from", ""),
		To:     c.Query("to", ""),
	}
}

func buildActivityQuery(db *gorm.DB, filters activityFilters) *gorm.DB {
	tx := db.Model(&models.ActivityLog{})

	if filters.User != "" {
		tx = tx.Where("username LIKE ?", "%"+filters.User+"%")
	}

	if filters.Action != "" {
		tx = tx.Where("action = ?", filters.Action)
	}

	if filters.From != "" {
		tx = tx.Where("created_at >= ?", filters.From)
	}

	if filters.To != "" {
		// Include the full day by adding a day to the "to" date
		tx = tx.Where("created_at < DATE_ADD(?, INTERVAL 1 DAY)", filters.To)
	}

	return tx
}

func getActivityViews(entries []models.ActivityLog) []EntryView {
	views := make([]EntryView, len(entries))
	for i := range entries {
		views[i] = EntryView{ActivityLog: entries[i]}
		if entries[i].Details == "" {
			continue
		}

		switch entries[i].Action {
		case activitylog.ActionZoneUpdated:
			var diff activitylog.ZoneSettingsDiff
			if err := json.Unmarshal([]byte(entries[i].Details), &diff); err == nil {
				views[i].ZoneSettings = &diff
			}
		case activitylog.ActionRecordChanged:
			var diff activitylog.RecordsDiff
			if err := json.Unmarshal([]byte(entries[i].Details), &diff); err == nil {
				views[i].RecordsDiff = &diff
			}
		}
	}

	return views
}

func getDistinctActions(db *gorm.DB) []string {
	var actions []string
	db.Model(&models.ActivityLog{}).Distinct("action").Order("action ASC").Pluck("action", &actions)

	return actions
}
