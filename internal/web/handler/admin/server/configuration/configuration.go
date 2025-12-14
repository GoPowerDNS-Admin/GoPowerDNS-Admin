package configuration

import (
	"context"
	"strconv"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"

	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/auth"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/config"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/powerdns"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web/handler"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web/handler/dashboard"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web/navigation"
)

const (
	// Path is the base path for server configuration handlers.
	Path = "admin/server/configuration"

	// TemplateName is the name of the server configuration template.
	TemplateName = "admin/server/configuration"

	// DefaultPageSize is the default number of items per page.
	DefaultPageSize = 25

	defaultTimeout = 30 * time.Second
)

// Service is the server configuration handler service.
type Service struct {
	handler.Service
	cfg       *config.Config
	db        *gorm.DB
	validator *validator.Validate
}

// Data represents the data passed to the template.
type Data struct {
	Settings    []ConfigSetting
	CurrentPage int
	PageSize    int
	TotalItems  int
	TotalPages  int
	HasPrevPage bool
	HasNextPage bool
	PrevPage    int
	NextPage    int
	SearchQuery string
	FilterType  string
}

// ConfigSetting represents a PowerDNS configuration setting.
type ConfigSetting struct {
	Name  string
	Type  string
	Value string
}

var (
	// Handler is the server configuration handler.
	Handler = Service{}
)

// Init initializes the server configuration handler.
func (s *Service) Init(app *fiber.App, cfg *config.Config, db *gorm.DB, authService *auth.Service) {
	if app == nil || cfg == nil || db == nil {
		log.Fatal().Msg(handler.ErrNilACDFatalLogMsg)
		return
	}

	s.db = db
	s.cfg = cfg
	s.validator = validator.New()

	// register routes with permission checks
	app.Get("/"+Path,
		auth.RequirePermission(authService, auth.PermAdminServerConfig),
		s.Get,
	)
}

// Get handles the server configuration page rendering with pagination.
func (s *Service) Get(c *fiber.Ctx) error {
	// Create navigation context
	nav := navigation.NewContext("PowerDNS Server Configuration", "server", "configuration").
		AddBreadcrumb("Home", "/"+dashboard.Path, false).
		AddBreadcrumb("Server", "#", false).
		AddBreadcrumb("Configuration", "/server/configuration", true)

	// Check if PowerDNS client is initialized
	if powerdns.Engine.Client == nil {
		log.Error().Msg(powerdns.ErrMsgClientNotInitialized)

		return c.Status(fiber.StatusInternalServerError).Render(TemplateName, fiber.Map{
			"Navigation": nav,
			"Error":      powerdns.ErrMsgClientNotInitializedDetailed,
		}, handler.BaseLayout)
	}

	// Parse query params
	page, pageSize := getPaginationParams(c)
	searchQuery, filterType := getSearchAndFilter(c)

	// Fetch configuration from PowerDNS API and map
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	rawSettings, err := powerdns.Engine.Config.List(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to fetch PowerDNS configuration")

		return c.Status(fiber.StatusInternalServerError).Render(Path, fiber.Map{
			"Navigation": nav,
			"Error":      "Failed to fetch PowerDNS configuration: " + err.Error(),
		}, handler.BaseLayout)
	}

	// Map and filter
	settings := make([]ConfigSetting, 0, len(rawSettings))
	for _, setting := range rawSettings {
		if setting.Name == nil {
			continue
		}

		cs := ConfigSetting{Name: *setting.Name}
		if setting.Type != nil {
			cs.Type = *setting.Type
		}

		if setting.Value != nil {
			cs.Value = *setting.Value
		}

		if includeSetting(cs, searchQuery, filterType) {
			settings = append(settings, cs)
		}
	}

	// Pagination
	totalItems := len(settings)
	totalPages, page := computeTotalPagesAndAdjust(totalItems, pageSize, page)
	startIdx, endIdx := pageSliceBounds(totalItems, pageSize, page)
	paginated := settings[startIdx:endIdx]

	data := buildData(paginated, page, pageSize, totalItems, totalPages, searchQuery, filterType)

	log.Info().
		Int("total_settings", totalItems).
		Int("page", page).
		Int("page_size", pageSize).
		Str("search", searchQuery).
		Str("filter_type", filterType).
		Msg("PowerDNS configuration retrieved successfully")

	return c.Render(TemplateName, fiber.Map{
		"Navigation": nav,
		"Data":       data,
	}, handler.BaseLayout)
}

// getPaginationParams parses and normalizes page and pageSize query parameters.
func getPaginationParams(c *fiber.Ctx) (int, int) {
	page := c.QueryInt("page", 1)
	if page < 1 {
		page = 1
	}

	pageSize := c.QueryInt("pageSize", DefaultPageSize)
	if pageSize < 1 || pageSize > 100 {
		pageSize = DefaultPageSize
	}

	return page, pageSize
}

// getSearchAndFilter extracts search and type filter from the request.
func getSearchAndFilter(c *fiber.Ctx) (string, string) {
	return c.Query("search", ""), c.Query("type", "")
}

// includeSetting returns true if the setting matches search and filter criteria.
func includeSetting(cs ConfigSetting, searchQuery, filterType string) bool {
	if searchQuery != "" {
		if !contains(cs.Name, searchQuery) && !contains(cs.Value, searchQuery) {
			return false
		}
	}

	if filterType != "" && cs.Type != filterType {
		return false
	}

	return true
}

// computeTotalPagesAndAdjust computes total pages and adjusts the page into range.
func computeTotalPagesAndAdjust(totalItems, pageSize, page int) (int, int) {
	totalPages := (totalItems + pageSize - 1) / pageSize
	if totalPages < 1 {
		totalPages = 1
	}

	if page > totalPages {
		page = totalPages
	}

	return totalPages, page
}

// pageSliceBounds calculates start and end indices for slicing a page.
func pageSliceBounds(totalItems, pageSize, page int) (int, int) {
	startIdx := (page - 1) * pageSize

	endIdx := startIdx + pageSize
	if endIdx > totalItems {
		endIdx = totalItems
	}

	if startIdx < 0 {
		startIdx = 0
	}

	if startIdx > endIdx {
		startIdx = endIdx
	}

	return startIdx, endIdx
}

// buildData constructs the Data struct for the template.
func buildData(
	settings []ConfigSetting,
	page,
	pageSize,
	totalItems,
	totalPages int,
	searchQuery,
	filterType string) Data {
	return Data{
		Settings:    settings,
		CurrentPage: page,
		PageSize:    pageSize,
		TotalItems:  totalItems,
		TotalPages:  totalPages,
		HasPrevPage: page > 1,
		HasNextPage: page < totalPages,
		PrevPage:    page - 1,
		NextPage:    page + 1,
		SearchQuery: searchQuery,
		FilterType:  filterType,
	}
}

// contains checks if a string contains a substring (case-insensitive).
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		strconv.QuoteToASCII(s) != strconv.QuoteToASCII(substr) &&
			substr != "" && indexIgnoreCase(s, substr) >= 0)
}

// indexIgnoreCase returns the index of substr in s, case-insensitive.
func indexIgnoreCase(s, substr string) int {
	sLower := toLower(s)
	substrLower := toLower(substr)

	for i := 0; i <= len(sLower)-len(substrLower); i++ {
		if sLower[i:i+len(substrLower)] == substrLower {
			return i
		}
	}

	return -1
}

// toLower converts a string to lowercase.
func toLower(s string) string {
	result := make([]byte, len(s))
	for i := range len(s) {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			result[i] = c + 32
		} else {
			result[i] = c
		}
	}

	return string(result)
}
