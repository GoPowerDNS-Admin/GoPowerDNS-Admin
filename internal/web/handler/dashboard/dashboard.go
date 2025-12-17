// Package dashboard provides the dashboard handler for displaying DNS zones.
package dashboard

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	pdnsapi "github.com/joeig/go-powerdns/v3"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"

	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/auth"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/config"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/powerdns"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web/handler"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web/navigation"
)

const (
	// Path is the path to the dashboard page.
	Path = handler.RootPath + "dashboard"

	// TemplateName is the name of the dashboard template.
	TemplateName = "dashboard/dashboard"

	// DefaultPageSize is the default number of items per page.
	DefaultPageSize = 25

	defaultTimeout = 180 * time.Second

	// TabForward represents the forward zones tab.
	TabForward = "forward"

	// TabReverseV4 represents the reverse IPv4 zones tab.
	TabReverseV4 = "reverse-ipv4"

	// TabReverseV6 represents the reverse IPv6 zones tab.
	TabReverseV6 = "reverse-ipv6"

	desc = "desc"
)

// Zone represents a DNS zone for template rendering.
type Zone struct {
	Name        string
	Kind        string
	Serial      uint32
	RecordCount int
	Masters     []string
}

// QueryParams holds the query and pagination parameters.
type QueryParams struct {
	Page        int
	PageSize    int
	SearchQuery string
	FilterKind  string
	SortField   string
	SortOrder   string
}

// TabData represents pagination data for a single tab.
type TabData struct {
	Zones       []Zone
	CurrentPage int
	PageSize    int
	TotalItems  int
	TotalPages  int
	HasPrevPage bool
	HasNextPage bool
	PrevPage    int
	NextPage    int
	SearchQuery string
	FilterKind  string
	SortField   string
	SortOrder   string
}

// Data represents the complete dashboard data.
type Data struct {
	ActiveTab    string
	ForwardTab   TabData
	ReverseV4Tab TabData
	ReverseV6Tab TabData
}

// Service is the dashboard handler service.
type Service struct {
	handler.Service
	cfg *config.Config
	db  *gorm.DB
}

// Handler is the dashboard handler.
var Handler = Service{}

// Init initializes the dashboard handler.
func (s *Service) Init(app *fiber.App, cfg *config.Config, db *gorm.DB, authService *auth.Service) {
	if app == nil || cfg == nil || db == nil {
		log.Fatal().Msg(handler.ErrNilACDFatalLogMsg)
		return
	}

	s.db = db
	s.cfg = cfg

	// register routes with permission checks
	app.Get(Path,
		auth.RequirePermission(authService, auth.PermDashboardView),
		s.Get,
	)
}

// Get handles the dashboard page rendering.
func (s *Service) Get(c *fiber.Ctx) error {
	// Create navigation context
	nav := navigation.NewContext("Dashboard", "dashboard", "dashboard").
		AddBreadcrumb("Home", Path, false).
		AddBreadcrumb("Dashboard", Path, true)

	// Get active tab (default: forward)
	activeTab := c.Query("tab", TabForward)
	if activeTab != TabForward && activeTab != TabReverseV4 && activeTab != TabReverseV6 {
		activeTab = TabForward
	}

	// Parse query parameters
	params := QueryParams{
		Page:        c.QueryInt("page", 1),
		PageSize:    c.QueryInt("pageSize", DefaultPageSize),
		SearchQuery: c.Query("search", ""),
		FilterKind:  c.Query("kind", ""),
		SortField:   c.Query("sort", "name"),
		SortOrder:   c.Query("order", "asc"),
	}

	// Validate pagination parameters
	if params.Page < 1 {
		params.Page = 1
	}

	if params.PageSize < 1 || params.PageSize > 100 {
		params.PageSize = DefaultPageSize
	}

	// Check if PowerDNS client is initialized
	if powerdns.Engine.Client == nil {
		log.Error().Msg(powerdns.ErrMsgClientNotInitialized)

		return c.Status(fiber.StatusInternalServerError).SendString(powerdns.ErrMsgClientNotInitializedDetailed)
	}

	// Fetch all zones from PowerDNS API
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	apiZones, err := powerdns.Engine.Zones.List(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to fetch zones from PowerDNS")

		return c.Status(fiber.StatusInternalServerError).SendString("Failed to fetch zones: " + err.Error())
	}

	// Convert API zones to template zones and categorize
	forwardZones, reverseV4Zones, reverseV6Zones := categorizeZones(apiZones)

	// Select zones for active tab
	var zones []Zone

	switch activeTab {
	case TabReverseV4:
		zones = reverseV4Zones
	case TabReverseV6:
		zones = reverseV6Zones
	default:
		zones = forwardZones
	}

	// Apply filters and sorting
	zones = filterZones(zones, params.SearchQuery, params.FilterKind)
	sortZones(zones, params.SortField, params.SortOrder)

	// Paginate results
	paginatedZones, totalPages, actualPaginatedPage := paginateZones(zones, params.Page, params.PageSize)
	totalItems := len(zones)

	// Build TabData for active tab
	params.Page = actualPaginatedPage // Update page if it was adjusted
	tabData := buildTabData(paginatedZones, totalPages, &params)
	tabData.TotalItems = totalItems

	// Build Data struct
	data := Data{
		ActiveTab: activeTab,
	}

	// Populate active tab data and set counts for other tabs
	switch activeTab {
	case TabReverseV4:
		data.ReverseV4Tab = tabData
		data.ForwardTab.TotalItems = len(forwardZones)
		data.ReverseV6Tab.TotalItems = len(reverseV6Zones)
	case TabReverseV6:
		data.ReverseV6Tab = tabData
		data.ForwardTab.TotalItems = len(forwardZones)
		data.ReverseV4Tab.TotalItems = len(reverseV4Zones)
	default:
		data.ForwardTab = tabData
		data.ReverseV4Tab.TotalItems = len(reverseV4Zones)
		data.ReverseV6Tab.TotalItems = len(reverseV6Zones)
	}

	log.Debug().
		Int("total_zones", len(apiZones)).
		Int("forward_zones", len(forwardZones)).
		Int("reverse_v4_zones", len(reverseV4Zones)).
		Int("reverse_v6_zones", len(reverseV6Zones)).
		Str("active_tab", activeTab).
		Int("page", params.Page).
		Int("page_size", params.PageSize).
		Str("search", params.SearchQuery).
		Str("filter_kind", params.FilterKind).
		Str("sort_field", params.SortField).
		Str("sort_order", params.SortOrder).
		Msg("Dashboard zones retrieved successfully")

	return c.Render(TemplateName, fiber.Map{
		"Navigation": nav,
		"Data":       data,
	}, handler.BaseLayout)
}

// categorizeZones converts API zones to template zones and categorizes them by type.
func categorizeZones(apiZones []pdnsapi.Zone) (forward, reverseV4, reverseV6 []Zone) {
	forward = make([]Zone, 0)
	reverseV4 = make([]Zone, 0)
	reverseV6 = make([]Zone, 0)

	// Use index iteration to avoid copying potentially large pdnsapi.Zone values
	for i := range apiZones {
		apiZone := &apiZones[i]

		if apiZone.Name == nil {
			continue
		}

		zone := Zone{
			Name:        *apiZone.Name,
			RecordCount: len(apiZone.RRsets),
			Masters:     apiZone.Masters,
		}

		if apiZone.Kind != nil {
			zone.Kind = string(*apiZone.Kind)
		}

		if apiZone.Serial != nil {
			zone.Serial = *apiZone.Serial
		}

		switch {
		case strings.HasSuffix(zone.Name, ".in-addr.arpa."):
			reverseV4 = append(reverseV4, zone)
		case strings.HasSuffix(zone.Name, ".ip6.arpa."):
			reverseV6 = append(reverseV6, zone)
		default:
			forward = append(forward, zone)
		}
	}

	return forward, reverseV4, reverseV6
}

// filterZones applies search and kind filters to zones.
func filterZones(zones []Zone, searchQuery, filterKind string) []Zone {
	// Apply search filter
	if searchQuery != "" {
		filtered := make([]Zone, 0)

		for _, zone := range zones {
			if strings.Contains(strings.ToLower(zone.Name), strings.ToLower(searchQuery)) {
				filtered = append(filtered, zone)
			}
		}

		zones = filtered
	}

	// Apply kind filter
	if filterKind != "" {
		filtered := make([]Zone, 0)

		for _, zone := range zones {
			if zone.Kind == filterKind {
				filtered = append(filtered, zone)
			}
		}

		zones = filtered
	}

	return zones
}

// sortZones sorts zones by the specified field and order.
func sortZones(zones []Zone, sortField, sortOrder string) {
	switch sortField {
	case "name":
		sort.Slice(zones, func(i, j int) bool {
			if sortOrder == desc {
				return strings.ToLower(zones[i].Name) > strings.ToLower(zones[j].Name)
			}

			return strings.ToLower(zones[i].Name) < strings.ToLower(zones[j].Name)
		})
	case "kind":
		sort.Slice(zones, func(i, j int) bool {
			if sortOrder == desc {
				return strings.ToLower(zones[i].Kind) > strings.ToLower(zones[j].Kind)
			}

			return strings.ToLower(zones[i].Kind) < strings.ToLower(zones[j].Kind)
		})
	case "serial":
		sort.Slice(zones, func(i, j int) bool {
			if sortOrder == desc {
				return zones[i].Serial > zones[j].Serial
			}

			return zones[i].Serial < zones[j].Serial
		})
	}
}

// paginateZones calculates pagination and returns paginated zones.
func paginateZones(zones []Zone, page, pageSize int) (paginatedZones []Zone, totalPages, actualPage int) {
	var (
		totalItems = len(zones)
	)

	totalPages = (totalItems + pageSize - 1) / pageSize

	if totalPages < 1 {
		totalPages = 1
	}

	if page > totalPages {
		page = totalPages
	}

	var (
		startIdx = (page - 1) * pageSize
		endIdx   = startIdx + pageSize
	)

	if endIdx > totalItems {
		endIdx = totalItems
	}

	if startIdx < totalItems {
		paginatedZones = zones[startIdx:endIdx]
	} else {
		paginatedZones = []Zone{}
	}

	return paginatedZones, totalPages, page
}

// buildTabData creates TabData with pagination information.
func buildTabData(zones []Zone, totalPages int, params *QueryParams) TabData {
	return TabData{
		Zones:       zones,
		CurrentPage: params.Page,
		PageSize:    params.PageSize,
		TotalItems:  len(zones),
		TotalPages:  totalPages,
		HasPrevPage: params.Page > 1,
		HasNextPage: params.Page < totalPages,
		PrevPage:    params.Page - 1,
		NextPage:    params.Page + 1,
		SearchQuery: params.SearchQuery,
		FilterKind:  params.FilterKind,
		SortField:   params.SortField,
		SortOrder:   params.SortOrder,
	}
}
