// Package activity provides the admin handler for viewing the audit / activity log.
package activity

import (
	"encoding/json"

	"github.com/gofiber/fiber/v3"
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
	// UndoDetails is populated for record_undone entries.
	UndoDetails *activitylog.RecordUndoneDetails
	// ZoneSnapshot is populated for zone_deleted entries.
	ZoneSnapshot *activitylog.ZoneSnapshot
	// ZoneDeletedUndoneDetails is populated for zone_deleted_undone entries.
	ZoneDeletedUndoneDetails *activitylog.ZoneDeletedUndoneDetails
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
	cfg         *config.Config
	db          *gorm.DB
	authService *auth.Service
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
	s.authService = authService

	app.Get(Path,
		auth.RequirePermission(authService, auth.PermAdminActivityLog),
		s.List,
	)

	app.Post(Path+"/:id/undo",
		auth.RequirePermission(authService, auth.PermAdminActivityLogUndo),
		s.PostUndo,
	)
}

// List renders the paginated activity log with optional filters.
func (s *Service) List(c fiber.Ctx) error {
	nav := navigation.NewContext("Activity Log", "admin", "activity").
		AddBreadcrumb("Home", dashboard.Path, false).
		AddBreadcrumb("Admin", "#", false).
		AddBreadcrumb("Activity Log", Path, true)

	page := fiber.Query[int](c, "page", 1)
	if page < 1 {
		page = 1
	}

	pageSize := fiber.Query[int](c, "pageSize", DefaultPageSize)
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
		"Success":      c.Query("success"),
		"Error":        c.Query("error"),
		"CanUndo":      auth.HasPermissionInContext(c, s.authService, auth.PermAdminActivityLogUndo),
	}, handler.BaseLayout)
}

func parseActivityFilters(c fiber.Ctx) activityFilters {
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
		case activitylog.ActionRecordUndone:
			var ud activitylog.RecordUndoneDetails
			if err := json.Unmarshal([]byte(entries[i].Details), &ud); err == nil {
				views[i].UndoDetails = &ud
			}
		case activitylog.ActionZoneDeleted:
			var snap activitylog.ZoneSnapshot
			if err := json.Unmarshal([]byte(entries[i].Details), &snap); err == nil {
				views[i].ZoneSnapshot = &snap
			}
		case activitylog.ActionZoneDeletedUndone:
			var ud activitylog.ZoneDeletedUndoneDetails
			if err := json.Unmarshal([]byte(entries[i].Details), &ud); err == nil {
				views[i].ZoneDeletedUndoneDetails = &ud
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
